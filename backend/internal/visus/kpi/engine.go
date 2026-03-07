package kpi

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	visusmetrics "github.com/clario360/platform/internal/visus/metrics"
	"github.com/clario360/platform/internal/visus/model"
	"github.com/clario360/platform/internal/visus/repository"
)

type KPIAlertGenerator interface {
	GenerateFromKPI(ctx context.Context, tenantID uuid.UUID, kpi *model.KPIDefinition, value float64, status model.KPIStatus) error
}

type KPIEngine struct {
	fetcher      *KPIFetcher
	calculator   *KPICalculator
	threshold    *ThresholdEvaluator
	snapshotRepo *repository.KPISnapshotRepository
	kpiRepo      *repository.KPIRepository
	alertGen     KPIAlertGenerator
	metrics      *visusmetrics.Metrics
	logger       zerolog.Logger
}

func NewEngine(fetcher *KPIFetcher, calculator *KPICalculator, threshold *ThresholdEvaluator, snapshotRepo *repository.KPISnapshotRepository, kpiRepo *repository.KPIRepository, alertGen KPIAlertGenerator, metrics *visusmetrics.Metrics, logger zerolog.Logger) *KPIEngine {
	return &KPIEngine{
		fetcher:      fetcher,
		calculator:   calculator,
		threshold:    threshold,
		snapshotRepo: snapshotRepo,
		kpiRepo:      kpiRepo,
		alertGen:     alertGen,
		metrics:      metrics,
		logger:       logger.With().Str("component", "visus_kpi_engine").Logger(),
	}
}

func (e *KPIEngine) TakeSnapshots(ctx context.Context, tenantID uuid.UUID) error {
	definitions, err := e.kpiRepo.ListEnabled(ctx, tenantID)
	if err != nil {
		return err
	}
	evaluated := 0
	breaches := 0
	for idx := range definitions {
		kpi := &definitions[idx]
		if !snapshotDue(kpi.LastSnapshotAt, kpi.SnapshotFrequency, time.Now().UTC()) {
			continue
		}
		evaluated++

		history, _ := e.snapshotRepo.ListByKPI(ctx, tenantID, kpi.ID, model.KPIQuery{Limit: 60})
		var previous *model.KPISnapshot
		if len(history) > 0 {
			previous = &history[0]
		}

		rawValue, latency, fetchErr := e.fetcher.Fetch(ctx, tenantID, kpi)
		fetchSuccess := fetchErr == nil
		value := rawValue
		status := model.KPIStatusUnknown
		if fetchSuccess {
			value = e.calculator.Calculate(kpi, rawValue, history)
			status = e.threshold.Evaluate(kpi, value)
		} else if previous != nil {
			value = previous.Value
		}

		now := time.Now().UTC()
		snapshot := &model.KPISnapshot{
			TenantID:     tenantID,
			KPIID:        kpi.ID,
			Value:        value,
			Status:       status,
			PeriodStart:  snapshotPeriodStart(now, kpi.SnapshotFrequency),
			PeriodEnd:    now,
			FetchSuccess: fetchSuccess,
		}
		if previous != nil {
			snapshot.PreviousValue = &previous.Value
			delta := value - previous.Value
			snapshot.Delta = &delta
			if previous.Value != 0 {
				deltaPct := (delta / previous.Value) * 100
				snapshot.DeltaPercent = &deltaPct
			}
		}
		if latency > 0 {
			latencyMS := int(latency / time.Millisecond)
			snapshot.FetchLatencyMS = &latencyMS
		}
		if fetchErr != nil {
			message := fetchErr.Error()
			snapshot.FetchError = &message
		}
		if _, err := e.snapshotRepo.Create(ctx, snapshot); err != nil {
			return err
		}
		if err := e.kpiRepo.UpdateSnapshotState(ctx, tenantID, kpi.ID, now, value, status); err != nil {
			return err
		}
		if e.metrics != nil && e.metrics.KPISnapshotsTotal != nil {
			e.metrics.KPISnapshotsTotal.WithLabelValues(string(kpi.Suite), string(status)).Inc()
		}
		if e.metrics != nil && e.metrics.KPISnapshotDurationSeconds != nil {
			e.metrics.KPISnapshotDurationSeconds.WithLabelValues(string(kpi.Suite)).Observe(latency.Seconds())
		}
		if fetchSuccess && (status == model.KPIStatusWarning || status == model.KPIStatusCritical) && e.alertGen != nil {
			breaches++
			if e.metrics != nil && e.metrics.KPIThresholdBreachesTotal != nil {
				e.metrics.KPIThresholdBreachesTotal.WithLabelValues(string(kpi.Suite), string(status)).Inc()
			}
			if err := e.alertGen.GenerateFromKPI(ctx, tenantID, kpi, value, status); err != nil {
				e.logger.Error().Err(err).Str("kpi_id", kpi.ID.String()).Msg("failed to generate alert for kpi breach")
			}
		}
	}
	e.logger.Info().Str("tenant_id", tenantID.String()).Int("evaluated", evaluated).Int("breaches", breaches).Msg("kpi snapshot complete")
	return nil
}

func snapshotDue(last *time.Time, frequency model.KPISnapshotFrequency, now time.Time) bool {
	if last == nil {
		return true
	}
	return last.Add(IntervalForFrequency(frequency)).Before(now)
}
