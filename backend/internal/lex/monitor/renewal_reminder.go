package monitor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/lex/model"
	"github.com/clario360/platform/internal/lex/repository"
	"github.com/clario360/platform/internal/lex/service"
)

type RenewalReminder struct {
	db        *pgxpool.Pool
	contracts *repository.ContractRepository
	alerts    *repository.AlertRepository
	publisher service.Publisher
	topic     string
	interval  time.Duration
	logger    zerolog.Logger
	now       func() time.Time
}

func NewRenewalReminder(
	db *pgxpool.Pool,
	contracts *repository.ContractRepository,
	alerts *repository.AlertRepository,
	publisher service.Publisher,
	topic string,
	interval time.Duration,
	logger zerolog.Logger,
) *RenewalReminder {
	if interval <= 0 {
		interval = 6 * time.Hour
	}
	return &RenewalReminder{
		db:        db,
		contracts: contracts,
		alerts:    alerts,
		publisher: publisher,
		topic:     topic,
		interval:  interval,
		logger:    logger.With().Str("component", "lex-renewal-reminder").Logger(),
		now:       time.Now,
	}
}

func (r *RenewalReminder) Run(ctx context.Context) error {
	if err := r.RunOnce(ctx); err != nil && ctx.Err() == nil {
		r.logger.Error().Err(err).Msg("renewal reminder iteration failed")
	}

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := r.RunOnce(ctx); err != nil && ctx.Err() == nil {
				r.logger.Error().Err(err).Msg("renewal reminder iteration failed")
			}
		}
	}
}

func (r *RenewalReminder) RunOnce(ctx context.Context) error {
	candidates, err := r.contracts.ListDueForExpiryBucket(ctx, -1, 365)
	if err != nil {
		return err
	}

	var errs []error
	for _, contract := range candidates {
		if !contract.AutoRenew || contract.Status != model.ContractStatusActive || contract.ExpiryDate == nil {
			continue
		}
		reminderDate := renewalReminderDate(&contract)
		today := normalizeMonitorDate(r.now())
		if reminderDate.After(today.AddDate(0, 0, 7)) {
			continue
		}
		if normalizeMonitorDate(*contract.ExpiryDate).Before(today) {
			continue
		}
		if err := r.createReminder(ctx, &contract, reminderDate); err != nil {
			errs = append(errs, fmt.Errorf("contract %s: %w", contract.ID, err))
		}
	}
	return errors.Join(errs...)
}

func (r *RenewalReminder) createReminder(ctx context.Context, contract *model.Contract, reminderDate time.Time) error {
	daysRemaining := daysUntilExpiry(contract.ExpiryDate, r.now())
	dedupKey := fmt.Sprintf("renewal:%s:%s", contract.ID, reminderDate.Format("2006-01-02"))
	alert := &model.ComplianceAlert{
		ID:          uuid.New(),
		TenantID:    contract.TenantID,
		ContractID:  &contract.ID,
		Title:       fmt.Sprintf("Renewal decision due for contract %q", contract.Title),
		Description: fmt.Sprintf("Auto-renewal review date is %s for a contract expiring on %s.", reminderDate.Format("2006-01-02"), contract.ExpiryDate.UTC().Format("2006-01-02")),
		Severity:    renewalSeverity(daysRemaining),
		Status:      model.ComplianceAlertOpen,
		DedupKey:    &dedupKey,
		Evidence: map[string]any{
			"contract_id":         contract.ID,
			"renewal_date":        reminderDate,
			"expiry_date":         contract.ExpiryDate,
			"days_until_expiry":   daysRemaining,
			"renewal_notice_days": contract.RenewalNoticeDays,
			"auto_renew":          contract.AutoRenew,
			"owner_name":          contract.OwnerName,
			"legal_reviewer_name": contract.LegalReviewerName,
			"department":          contract.Department,
		},
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	created, err := r.alerts.CreateOrSkipDedup(ctx, tx, alert)
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true

	if created {
		publishLexEvent(ctx, r.publisher, r.topic, "com.clario360.lex.compliance.alert_created", contract.TenantID, nil, map[string]any{
			"id":          alert.ID,
			"contract_id": alert.ContractID,
			"severity":    alert.Severity,
			"title":       alert.Title,
		}, r.logger)
	}
	return nil
}

func renewalReminderDate(contract *model.Contract) time.Time {
	if contract != nil && contract.RenewalDate != nil {
		return normalizeMonitorDate(*contract.RenewalDate)
	}
	if contract != nil && contract.ExpiryDate != nil {
		return normalizeMonitorDate(contract.ExpiryDate.AddDate(0, 0, -contract.RenewalNoticeDays))
	}
	return normalizeMonitorDate(time.Now())
}

func renewalSeverity(daysRemaining int) model.ComplianceSeverity {
	switch {
	case daysRemaining <= 7:
		return model.ComplianceSeverityCritical
	case daysRemaining <= 30:
		return model.ComplianceSeverityHigh
	default:
		return model.ComplianceSeverityMedium
	}
}
