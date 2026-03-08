package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	onboardingmodel "github.com/clario360/platform/internal/onboarding/model"
)

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func runInTx(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func marshalJSON(value any) []byte {
	if value == nil {
		return []byte("{}")
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return payload
}

func scanOnboarding(row interface{ Scan(...any) error }) (*onboardingmodel.OnboardingStatus, error) {
	item := &onboardingmodel.OnboardingStatus{}
	var (
		orgIndustry       *string
		orgSize           *string
		stepsCompleted    []int32
		activeSuites      []string
		logoFileID        *uuid.UUID
		primaryColor      *string
		accentColor       *string
		provisioningErr   *string
		referralSource    *string
		emailVerifiedAt   *time.Time
		wizardCompletedAt *time.Time
		provStartedAt     *time.Time
		provCompletedAt   *time.Time
		orgName           *string
		orgCity           *string
	)
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.AdminUserID,
		&item.AdminEmail,
		&item.EmailVerified,
		&emailVerifiedAt,
		&item.CurrentStep,
		&stepsCompleted,
		&item.WizardCompleted,
		&wizardCompletedAt,
		&orgName,
		&orgIndustry,
		&item.OrgCountry,
		&orgCity,
		&orgSize,
		&logoFileID,
		&primaryColor,
		&accentColor,
		&activeSuites,
		&item.ProvisioningStatus,
		&provStartedAt,
		&provCompletedAt,
		&provisioningErr,
		&referralSource,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	item.OrgName = orgName
	item.OrgCity = orgCity
	item.EmailVerifiedAt = emailVerifiedAt
	item.WizardCompletedAt = wizardCompletedAt
	item.LogoFileID = logoFileID
	item.PrimaryColor = primaryColor
	item.AccentColor = accentColor
	item.ProvisioningStartedAt = provStartedAt
	item.ProvisioningCompletedAt = provCompletedAt
	item.ProvisioningError = provisioningErr
	item.ReferralSource = referralSource
	item.ActiveSuites = activeSuites
	item.StepsCompleted = int32SliceToInts(stepsCompleted)
	if orgIndustry != nil {
		value := onboardingmodel.OrgIndustry(*orgIndustry)
		item.OrgIndustry = &value
	}
	if orgSize != nil {
		value := onboardingmodel.OrgSize(*orgSize)
		item.OrgSize = &value
	}
	return item, nil
}

func scanProvisioningStep(row interface{ Scan(...any) error }) (*onboardingmodel.ProvisioningStep, error) {
	item := &onboardingmodel.ProvisioningStep{}
	var metadata []byte
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.OnboardingID,
		&item.StepNumber,
		&item.StepName,
		&item.Status,
		&item.StartedAt,
		&item.CompletedAt,
		&item.DurationMS,
		&item.ErrorMessage,
		&item.RetryCount,
		&item.IdempotencyKey,
		&metadata,
		&item.CreatedAt,
	); err != nil {
		return nil, err
	}
	item.Metadata = map[string]any{}
	if len(metadata) > 0 {
		_ = json.Unmarshal(metadata, &item.Metadata)
	}
	return item, nil
}

func scanInvitation(row interface{ Scan(...any) error }) (*onboardingmodel.Invitation, error) {
	item := &onboardingmodel.Invitation{}
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.Email,
		&item.RoleSlug,
		&item.TokenHash,
		&item.TokenPrefix,
		&item.Status,
		&item.InvitedBy,
		&item.InvitedByName,
		&item.AcceptedAt,
		&item.AcceptedBy,
		&item.ExpiresAt,
		&item.Message,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return item, nil
}

func int32SliceToInts(values []int32) []int {
	if len(values) == 0 {
		return []int{}
	}
	out := make([]int, 0, len(values))
	for _, value := range values {
		out = append(out, int(value))
	}
	return out
}

func nextWizardStep(completed []int) int {
	seen := map[int]struct{}{}
	for _, step := range completed {
		seen[step] = struct{}{}
	}
	for step := 1; step <= 5; step++ {
		if _, ok := seen[step]; !ok {
			return step
		}
	}
	return 5
}
