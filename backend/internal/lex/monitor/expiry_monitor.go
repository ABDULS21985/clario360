package monitor

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/lex/dto"
	lexmetrics "github.com/clario360/platform/internal/lex/metrics"
	"github.com/clario360/platform/internal/lex/model"
	"github.com/clario360/platform/internal/lex/repository"
	"github.com/clario360/platform/internal/lex/service"
)

var systemActorID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

type ExpiryMonitor struct {
	db              *pgxpool.Pool
	contracts       *repository.ContractRepository
	alerts          *repository.AlertRepository
	contractService *service.ContractService
	metrics         *lexmetrics.Metrics
	publisher       service.Publisher
	topic           string
	interval        time.Duration
	logger          zerolog.Logger
	now             func() time.Time
	listDueFunc     func(context.Context, int) ([]model.Contract, error)
	listExpiredFunc func(context.Context) ([]model.Contract, error)
	notifyFunc      func(context.Context, *model.Contract, int) error
	expireFunc      func(context.Context, *model.Contract) error
	autoRenewFunc   func(context.Context, *model.Contract) error
}

func NewExpiryMonitor(
	db *pgxpool.Pool,
	contracts *repository.ContractRepository,
	alerts *repository.AlertRepository,
	contractService *service.ContractService,
	metrics *lexmetrics.Metrics,
	publisher service.Publisher,
	topic string,
	interval time.Duration,
	logger zerolog.Logger,
) *ExpiryMonitor {
	if interval <= 0 {
		interval = time.Hour
	}
	return &ExpiryMonitor{
		db:              db,
		contracts:       contracts,
		alerts:          alerts,
		contractService: contractService,
		metrics:         metrics,
		publisher:       publisher,
		topic:           topic,
		interval:        interval,
		logger:          logger.With().Str("component", "lex-expiry-monitor").Logger(),
		now:             time.Now,
		listDueFunc: func(ctx context.Context, horizon int) ([]model.Contract, error) {
			return contracts.ListDueForExpiryBucket(ctx, -1, horizon)
		},
		listExpiredFunc: contracts.ListExpiredActive,
		notifyFunc:      nil,
		expireFunc:      nil,
		autoRenewFunc:   nil,
	}
}

func (m *ExpiryMonitor) Run(ctx context.Context) error {
	if err := m.RunOnce(ctx); err != nil && ctx.Err() == nil {
		m.logger.Error().Err(err).Msg("expiry monitor iteration failed")
	}

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := m.RunOnce(ctx); err != nil && ctx.Err() == nil {
				m.logger.Error().Err(err).Msg("expiry monitor iteration failed")
			}
		}
	}
}

func (m *ExpiryMonitor) RunOnce(ctx context.Context) error {
	if m.notifyFunc == nil {
		m.notifyFunc = m.recordExpiryNotification
	}
	if m.expireFunc == nil {
		m.expireFunc = m.expireContract
	}
	if m.autoRenewFunc == nil {
		m.autoRenewFunc = m.autoRenewContract
	}

	var errs []error
	for _, horizon := range []int{90, 60, 30, 7, 0} {
		if err := m.processHorizon(ctx, horizon); err != nil {
			errs = append(errs, fmt.Errorf("process expiry horizon %d: %w", horizon, err))
		}
	}
	if err := m.processExpiredContracts(ctx); err != nil {
		errs = append(errs, fmt.Errorf("process expired contracts: %w", err))
	}
	return errors.Join(errs...)
}

func (m *ExpiryMonitor) processHorizon(ctx context.Context, horizon int) error {
	contracts, err := m.listDueFunc(ctx, horizon)
	if err != nil {
		return err
	}
	if m.metrics != nil {
		m.metrics.ExpiringContracts.WithLabelValues(strconv.Itoa(horizon)).Set(float64(len(contracts)))
	}

	var errs []error
	for _, contract := range contracts {
		if contract.ExpiryDate == nil {
			continue
		}
		if err := m.notifyFunc(ctx, &contract, horizon); err != nil {
			errs = append(errs, fmt.Errorf("contract %s horizon %d: %w", contract.ID, horizon, err))
		}
	}
	return errors.Join(errs...)
}

func (m *ExpiryMonitor) recordExpiryNotification(ctx context.Context, contract *model.Contract, horizon int) error {
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	recorded, err := m.contracts.RecordExpiryNotification(ctx, tx, contract.TenantID, contract.ID, horizon)
	if err != nil {
		return err
	}
	if !recorded {
		return nil
	}

	alert := buildExpiryAlert(contract, horizon, m.now())
	alertCreated, err := m.alerts.CreateOrSkipDedup(ctx, tx, alert)
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true

	if alertCreated {
		publishLexEvent(ctx, m.publisher, m.topic, "com.clario360.lex.compliance.alert_created", contract.TenantID, nil, map[string]any{
			"id":          alert.ID,
			"contract_id": alert.ContractID,
			"severity":    alert.Severity,
			"title":       alert.Title,
		}, m.logger)
	}

	publishLexEvent(ctx, m.publisher, m.topic, "com.clario360.lex.contract.expiring", contract.TenantID, nil, map[string]any{
		"id":                contract.ID,
		"title":             contract.Title,
		"expiry_date":       contract.ExpiryDate,
		"days_until_expiry": daysUntilExpiry(contract.ExpiryDate, m.now()),
		"horizon":           horizon,
		"recipients":        expiryRecipients(contract, horizon),
		"owner_user_id":     contract.OwnerUserID,
		"legal_reviewer_id": contract.LegalReviewerID,
		"party_name":        contract.PartyBName,
	}, m.logger)
	return nil
}

func (m *ExpiryMonitor) processExpiredContracts(ctx context.Context) error {
	contracts, err := m.listExpiredFunc(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, contract := range contracts {
		if contract.AutoRenew && autoRenewEligible(&contract, m.now()) {
			if err := m.autoRenewFunc(ctx, &contract); err != nil {
				errs = append(errs, fmt.Errorf("auto-renew contract %s: %w", contract.ID, err))
			}
			continue
		}
		if err := m.expireFunc(ctx, &contract); err != nil {
			errs = append(errs, fmt.Errorf("expire contract %s: %w", contract.ID, err))
		}
	}
	return errors.Join(errs...)
}

func (m *ExpiryMonitor) expireContract(ctx context.Context, contract *model.Contract) error {
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	now := normalizeMonitorDate(m.now())
	prev := contract.Status
	if err := m.contracts.UpdateStatus(ctx, tx, contract.TenantID, contract.ID, &prev, model.ContractStatusExpired, nil, now, nil); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true

	publishLexEvent(ctx, m.publisher, m.topic, "com.clario360.lex.contract.expired", contract.TenantID, nil, map[string]any{
		"id":          contract.ID,
		"title":       contract.Title,
		"expiry_date": contract.ExpiryDate,
	}, m.logger)
	return nil
}

func (m *ExpiryMonitor) autoRenewContract(ctx context.Context, contract *model.Contract) error {
	if m.contractService == nil {
		return fmt.Errorf("contract service is required for auto-renewal")
	}

	newEffectiveDate, newExpiryDate := nextRenewalTerm(contract, m.now())
	req := dto.RenewContractRequest{
		NewEffectiveDate: &newEffectiveDate,
		NewExpiryDate:    newExpiryDate,
		ChangeSummary:    "Automatically renewed by expiry monitor.",
	}
	if contract.TotalValue != nil {
		value := *contract.TotalValue
		req.NewValue = &value
	}

	_, err := m.contractService.RenewContract(ctx, contract.TenantID, systemActorID, contract.ID, req)
	return err
}

func buildExpiryAlert(contract *model.Contract, horizon int, now time.Time) *model.ComplianceAlert {
	title := fmt.Sprintf("Contract %q expiring in %d days", contract.Title, horizon)
	if horizon == 0 {
		title = fmt.Sprintf("Contract %q expiring in 0 days", contract.Title)
	}
	description := fmt.Sprintf(
		"Contract expires on %s and has reached the %d-day notification horizon.",
		contract.ExpiryDate.UTC().Format("2006-01-02"),
		horizon,
	)
	dedupKey := fmt.Sprintf("expiry:%s:%d", contract.ID, horizon)
	return &model.ComplianceAlert{
		ID:          uuid.New(),
		TenantID:    contract.TenantID,
		ContractID:  &contract.ID,
		Title:       title,
		Description: description,
		Severity:    expirySeverity(horizon),
		Status:      model.ComplianceAlertOpen,
		DedupKey:    &dedupKey,
		Evidence: map[string]any{
			"contract_id":         contract.ID,
			"contract_number":     contract.ContractNumber,
			"expiry_date":         contract.ExpiryDate,
			"days_until_expiry":   daysUntilExpiry(contract.ExpiryDate, now),
			"horizon_days":        horizon,
			"owner_name":          contract.OwnerName,
			"legal_reviewer_name": contract.LegalReviewerName,
			"recipients":          expiryRecipients(contract, horizon),
		},
	}
}

func expirySeverity(horizon int) model.ComplianceSeverity {
	switch horizon {
	case 90:
		return model.ComplianceSeverityLow
	case 60:
		return model.ComplianceSeverityMedium
	case 30:
		return model.ComplianceSeverityHigh
	default:
		return model.ComplianceSeverityCritical
	}
}

func daysUntilExpiry(expiryDate *time.Time, now time.Time) int {
	if expiryDate == nil {
		return 0
	}
	return int(normalizeMonitorDate(*expiryDate).Sub(normalizeMonitorDate(now)).Hours() / 24)
}

func expiryRecipients(contract *model.Contract, horizon int) []string {
	recipients := []string{}
	appendUnique := func(values ...string) {
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			seen := false
			for _, existing := range recipients {
				if existing == value {
					seen = true
					break
				}
			}
			if !seen {
				recipients = append(recipients, value)
			}
		}
	}

	appendUnique(contract.OwnerName)
	if horizon <= 60 && contract.LegalReviewerName != nil {
		appendUnique(*contract.LegalReviewerName)
	}
	if horizon <= 30 && contract.Department != nil {
		appendUnique(*contract.Department + " department head")
	}
	if horizon <= 7 {
		appendUnique("management")
	}
	if horizon == 0 {
		appendUnique("compliance team")
	}
	return recipients
}

func autoRenewEligible(contract *model.Contract, now time.Time) bool {
	if contract == nil || !contract.AutoRenew {
		return false
	}
	if contract.RenewalDate != nil {
		return !normalizeMonitorDate(*contract.RenewalDate).After(normalizeMonitorDate(now))
	}
	return true
}

func nextRenewalTerm(contract *model.Contract, now time.Time) (time.Time, time.Time) {
	start := normalizeMonitorDate(now)
	if contract != nil && contract.ExpiryDate != nil {
		start = normalizeMonitorDate(contract.ExpiryDate.AddDate(0, 0, 1))
	}

	termDays := 365
	if contract != nil && contract.EffectiveDate != nil && contract.ExpiryDate != nil {
		calculated := int(normalizeMonitorDate(*contract.ExpiryDate).Sub(normalizeMonitorDate(*contract.EffectiveDate)).Hours()/24) + 1
		if calculated > 0 {
			termDays = calculated
		}
	}
	end := normalizeMonitorDate(start.AddDate(0, 0, termDays-1))
	return start, end
}

func normalizeMonitorDate(value time.Time) time.Time {
	utc := value.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

func publishLexEvent(ctx context.Context, publisher service.Publisher, topic, eventType string, tenantID uuid.UUID, userID *uuid.UUID, payload any, logger zerolog.Logger) {
	if publisher == nil || strings.TrimSpace(topic) == "" {
		return
	}
	event, err := events.NewEvent(eventType, "lex-service", tenantID.String(), payload)
	if err != nil {
		logger.Error().Err(err).Str("event_type", eventType).Msg("build lex monitor event")
		return
	}
	if userID != nil {
		event.UserID = userID.String()
	}
	if err := publisher.Publish(ctx, topic, event); err != nil {
		logger.Error().Err(err).Str("event_type", eventType).Str("topic", topic).Msg("publish lex monitor event")
	}
}
