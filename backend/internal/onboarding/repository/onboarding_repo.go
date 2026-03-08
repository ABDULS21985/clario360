package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	iammodel "github.com/clario360/platform/internal/iam/model"
	onboardingmodel "github.com/clario360/platform/internal/onboarding/model"
)

type CreateRegistrationParams struct {
	TenantID       uuid.UUID
	TenantName     string
	TenantSlug     string
	AdminUserID    uuid.UUID
	AdminEmail     string
	FirstName      string
	LastName       string
	PasswordHash   string
	Country        string
	Industry       onboardingmodel.OrgIndustry
	ReferralSource *string
	OTPHash        string
	OTPExpiresAt   time.Time
	IPAddress      *string
	UserAgent      *string
	RolePermissions []string
}

type ActivationResult struct {
	OnboardingID uuid.UUID
	TenantID     uuid.UUID
	UserID       uuid.UUID
}

type CreateTenantUserParams struct {
	UserID       uuid.UUID
	TenantID     uuid.UUID
	RoleID       uuid.UUID
	Email        string
	FirstName    string
	LastName     string
	PasswordHash string
	CreatedBy    *uuid.UUID
}

type OnboardingRepository struct {
	pool *pgxpool.Pool
}

func NewOnboardingRepository(pool *pgxpool.Pool) *OnboardingRepository {
	return &OnboardingRepository{pool: pool}
}

func (r *OnboardingRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM users
			WHERE lower(email) = lower($1)
			  AND deleted_at IS NULL
		)`, strings.TrimSpace(strings.ToLower(email)),
	).Scan(&exists)
	return exists, err
}

func (r *OnboardingRepository) OrganizationNameExists(ctx context.Context, name string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM tenants
			WHERE lower(name) = lower($1)
		)`, strings.TrimSpace(name),
	).Scan(&exists)
	return exists, err
}

func (r *OnboardingRepository) CreateRegistration(ctx context.Context, params CreateRegistrationParams) error {
	return runInTx(ctx, r.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			INSERT INTO tenants (id, name, slug, settings, status, subscription_tier)
			VALUES ($1, $2, $3, '{}'::jsonb, $4, $5)`,
			params.TenantID,
			params.TenantName,
			params.TenantSlug,
			iammodel.TenantStatusOnboarding,
			iammodel.TierEnterprise,
		); err != nil {
			return fmt.Errorf("insert tenant: %w", err)
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO users (
				id, tenant_id, email, password_hash, first_name, last_name, status, created_by
			) VALUES ($1, $2, $3, $4, $5, $6, $7, NULL)`,
			params.AdminUserID,
			params.TenantID,
			strings.ToLower(strings.TrimSpace(params.AdminEmail)),
			params.PasswordHash,
			params.FirstName,
			params.LastName,
			iammodel.UserStatusPendingVerification,
		); err != nil {
			return fmt.Errorf("insert admin user: %w", err)
		}

		var roleID uuid.UUID
		if err := tx.QueryRow(ctx, `
			INSERT INTO roles (tenant_id, name, slug, description, is_system_role, permissions)
			VALUES ($1, 'Tenant Admin', 'tenant-admin', 'Tenant administrator', true, $2::jsonb)
			ON CONFLICT (tenant_id, slug)
			DO UPDATE SET name = EXCLUDED.name
			RETURNING id`,
			params.TenantID,
			marshalJSON(params.RolePermissions),
		).Scan(&roleID); err != nil {
			return fmt.Errorf("ensure tenant-admin role: %w", err)
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_id, tenant_id, assigned_by)
			VALUES ($1, $2, $3, $1)
			ON CONFLICT (user_id, role_id) DO NOTHING`,
			params.AdminUserID,
			roleID,
			params.TenantID,
		); err != nil {
			return fmt.Errorf("assign tenant-admin role: %w", err)
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO tenant_onboarding (
				tenant_id, admin_user_id, admin_email, current_step,
				org_name, org_industry, org_country, referral_source
			) VALUES ($1, $2, $3, 0, $4, $5, $6, $7)`,
			params.TenantID,
			params.AdminUserID,
			strings.ToLower(strings.TrimSpace(params.AdminEmail)),
			params.TenantName,
			params.Industry,
			params.Country,
			params.ReferralSource,
		); err != nil {
			return fmt.Errorf("insert onboarding row: %w", err)
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO email_verifications (
				email, otp_hash, purpose, expires_at, ip_address, user_agent
			) VALUES ($1, $2, 'registration', $3, $4, $5)`,
			strings.ToLower(strings.TrimSpace(params.AdminEmail)),
			params.OTPHash,
			params.OTPExpiresAt,
			params.IPAddress,
			params.UserAgent,
		); err != nil {
			return fmt.Errorf("insert email verification: %w", err)
		}

		return nil
	})
}

func (r *OnboardingRepository) GetLatestEmailVerification(ctx context.Context, email, purpose string) (*onboardingmodel.EmailVerification, error) {
	item := &onboardingmodel.EmailVerification{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, otp_hash, purpose, verified, attempts, max_attempts, locked_at,
		       expires_at, verified_at, ip_address, user_agent, created_at
		FROM email_verifications
		WHERE lower(email) = lower($1) AND purpose = $2
		ORDER BY created_at DESC
		LIMIT 1`,
		strings.ToLower(strings.TrimSpace(email)),
		purpose,
	).Scan(
		&item.ID,
		&item.Email,
		&item.OTPHash,
		&item.Purpose,
		&item.Verified,
		&item.Attempts,
		&item.MaxAttempts,
		&item.LockedAt,
		&item.ExpiresAt,
		&item.VerifiedAt,
		&item.IPAddress,
		&item.UserAgent,
		&item.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *OnboardingRepository) CreateEmailVerification(ctx context.Context, email, otpHash string, expiresAt time.Time, ipAddress, userAgent *string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO email_verifications (email, otp_hash, purpose, expires_at, ip_address, user_agent)
		VALUES ($1, $2, 'registration', $3, $4, $5)`,
		strings.ToLower(strings.TrimSpace(email)),
		otpHash,
		expiresAt,
		ipAddress,
		userAgent,
	)
	return err
}

func (r *OnboardingRepository) IncrementVerificationAttempts(ctx context.Context, verificationID uuid.UUID) (int, error) {
	var remaining int
	err := r.pool.QueryRow(ctx, `
		UPDATE email_verifications
		SET attempts = attempts + 1,
		    locked_at = CASE WHEN attempts + 1 >= max_attempts THEN now() ELSE locked_at END
		WHERE id = $1
		RETURNING GREATEST(max_attempts - attempts, 0) - 1`,
		verificationID,
	).Scan(&remaining)
	return remaining, err
}

func (r *OnboardingRepository) MarkEmailVerificationVerified(ctx context.Context, verificationID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE email_verifications
		SET verified = true, verified_at = now()
		WHERE id = $1`,
		verificationID,
	)
	return err
}

func (r *OnboardingRepository) ActivateRegistration(ctx context.Context, email string) (*ActivationResult, error) {
	result := &ActivationResult{}
	err := runInTx(ctx, r.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			SELECT id, tenant_id, admin_user_id
			FROM tenant_onboarding
			WHERE lower(admin_email) = lower($1)
			ORDER BY created_at DESC
			LIMIT 1`,
			strings.ToLower(strings.TrimSpace(email)),
		).Scan(&result.OnboardingID, &result.TenantID, &result.UserID); err != nil {
			return fmt.Errorf("load onboarding for activation: %w", err)
		}

		if _, err := tx.Exec(ctx, `
			UPDATE users
			SET status = $2, updated_at = now()
			WHERE id = $1`,
			result.UserID,
			iammodel.UserStatusActive,
		); err != nil {
			return fmt.Errorf("activate user: %w", err)
		}

		if _, err := tx.Exec(ctx, `
			UPDATE tenant_onboarding
			SET email_verified = true,
			    email_verified_at = now(),
			    current_step = CASE WHEN current_step < 1 THEN 1 ELSE current_step END,
			    updated_at = now()
			WHERE id = $1`,
			result.OnboardingID,
		); err != nil {
			return fmt.Errorf("mark onboarding verified: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *OnboardingRepository) CreateTenantUserWithRole(ctx context.Context, params CreateTenantUserParams) error {
	return runInTx(ctx, r.pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO users (
				id, tenant_id, email, password_hash, first_name, last_name, status, created_by
			) VALUES ($1, $2, $3, $4, $5, $6, 'active', $7)`,
			params.UserID,
			params.TenantID,
			strings.ToLower(strings.TrimSpace(params.Email)),
			params.PasswordHash,
			params.FirstName,
			params.LastName,
			params.CreatedBy,
		)
		if err != nil {
			return fmt.Errorf("insert invited user: %w", err)
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_id, tenant_id, assigned_by)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id, role_id) DO NOTHING`,
			params.UserID,
			params.RoleID,
			params.TenantID,
			params.CreatedBy,
		)
		if err != nil {
			return fmt.Errorf("assign invited user role: %w", err)
		}
		return nil
	})
}

func (r *OnboardingRepository) GetOnboardingByTenantID(ctx context.Context, tenantID uuid.UUID) (*onboardingmodel.OnboardingStatus, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, admin_user_id, admin_email, email_verified, email_verified_at,
		       current_step, steps_completed, wizard_completed, wizard_completed_at,
		       org_name, org_industry, org_country, org_city, org_size,
		       logo_file_id, primary_color, accent_color, active_suites,
		       provisioning_status, provisioning_started_at, provisioning_completed_at,
		       provisioning_error, referral_source, created_at, updated_at
		FROM tenant_onboarding
		WHERE tenant_id = $1`,
		tenantID,
	)
	return scanOnboarding(row)
}

func (r *OnboardingRepository) GetOnboardingByAdminEmail(ctx context.Context, email string) (*onboardingmodel.OnboardingStatus, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, admin_user_id, admin_email, email_verified, email_verified_at,
		       current_step, steps_completed, wizard_completed, wizard_completed_at,
		       org_name, org_industry, org_country, org_city, org_size,
		       logo_file_id, primary_color, accent_color, active_suites,
		       provisioning_status, provisioning_started_at, provisioning_completed_at,
		       provisioning_error, referral_source, created_at, updated_at
		FROM tenant_onboarding
		WHERE lower(admin_email) = lower($1)
		ORDER BY created_at DESC
		LIMIT 1`,
		strings.ToLower(strings.TrimSpace(email)),
	)
	return scanOnboarding(row)
}

func (r *OnboardingRepository) UpdateOrganization(ctx context.Context, tenantID uuid.UUID, name string, industry onboardingmodel.OrgIndustry, country string, city *string, size onboardingmodel.OrgSize) (*onboardingmodel.WizardProgress, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE tenant_onboarding
		SET org_name = $2,
		    org_industry = $3,
		    org_country = $4,
		    org_city = $5,
		    org_size = $6,
		    current_step = 2,
		    steps_completed = (
		        SELECT ARRAY(
		            SELECT DISTINCT value
		            FROM unnest(steps_completed || ARRAY[1]::int[]) AS value
		            ORDER BY value
		        )
		    ),
		    updated_at = now()
		WHERE tenant_id = $1
		RETURNING tenant_id, current_step, steps_completed, wizard_completed, email_verified,
		          org_name, org_industry, org_country, org_city, org_size,
		          logo_file_id, primary_color, accent_color, active_suites,
		          provisioning_status, provisioning_started_at, provisioning_completed_at, provisioning_error`,
		tenantID,
		name,
		industry,
		country,
		city,
		size,
	)
	return scanWizardProgress(row)
}

func (r *OnboardingRepository) UpdateBranding(ctx context.Context, tenantID uuid.UUID, logoFileID *uuid.UUID, primaryColor, accentColor *string) (*onboardingmodel.WizardProgress, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE tenant_onboarding
		SET logo_file_id = COALESCE($2, logo_file_id),
		    primary_color = COALESCE($3, primary_color),
		    accent_color = COALESCE($4, accent_color),
		    current_step = CASE WHEN current_step < 3 THEN 3 ELSE current_step END,
		    steps_completed = (
		        SELECT ARRAY(
		            SELECT DISTINCT value
		            FROM unnest(steps_completed || ARRAY[2]::int[]) AS value
		            ORDER BY value
		        )
		    ),
		    updated_at = now()
		WHERE tenant_id = $1
		RETURNING tenant_id, current_step, steps_completed, wizard_completed, email_verified,
		          org_name, org_industry, org_country, org_city, org_size,
		          logo_file_id, primary_color, accent_color, active_suites,
		          provisioning_status, provisioning_started_at, provisioning_completed_at, provisioning_error`,
		tenantID,
		logoFileID,
		primaryColor,
		accentColor,
	)
	return scanWizardProgress(row)
}

func (r *OnboardingRepository) MarkTeamStepCompleted(ctx context.Context, tenantID uuid.UUID) (*onboardingmodel.WizardProgress, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE tenant_onboarding
		SET current_step = CASE WHEN current_step < 4 THEN 4 ELSE current_step END,
		    steps_completed = (
		        SELECT ARRAY(
		            SELECT DISTINCT value
		            FROM unnest(steps_completed || ARRAY[3]::int[]) AS value
		            ORDER BY value
		        )
		    ),
		    updated_at = now()
		WHERE tenant_id = $1
		RETURNING tenant_id, current_step, steps_completed, wizard_completed, email_verified,
		          org_name, org_industry, org_country, org_city, org_size,
		          logo_file_id, primary_color, accent_color, active_suites,
		          provisioning_status, provisioning_started_at, provisioning_completed_at, provisioning_error`,
		tenantID,
	)
	return scanWizardProgress(row)
}

func (r *OnboardingRepository) UpdateSuites(ctx context.Context, tenantID uuid.UUID, activeSuites []string) (*onboardingmodel.WizardProgress, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE tenant_onboarding
		SET active_suites = $2,
		    current_step = CASE WHEN current_step < 5 THEN 5 ELSE current_step END,
		    steps_completed = (
		        SELECT ARRAY(
		            SELECT DISTINCT value
		            FROM unnest(steps_completed || ARRAY[4]::int[]) AS value
		            ORDER BY value
		        )
		    ),
		    updated_at = now()
		WHERE tenant_id = $1
		RETURNING tenant_id, current_step, steps_completed, wizard_completed, email_verified,
		          org_name, org_industry, org_country, org_city, org_size,
		          logo_file_id, primary_color, accent_color, active_suites,
		          provisioning_status, provisioning_started_at, provisioning_completed_at, provisioning_error`,
		tenantID,
		activeSuites,
	)
	return scanWizardProgress(row)
}

func (r *OnboardingRepository) CompleteWizard(ctx context.Context, tenantID uuid.UUID) (*onboardingmodel.WizardProgress, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE tenant_onboarding
		SET wizard_completed = true,
		    wizard_completed_at = now(),
		    current_step = 5,
		    steps_completed = (
		        SELECT ARRAY(
		            SELECT DISTINCT value
		            FROM unnest(steps_completed || ARRAY[5]::int[]) AS value
		            ORDER BY value
		        )
		    ),
		    updated_at = now()
		WHERE tenant_id = $1
		RETURNING tenant_id, current_step, steps_completed, wizard_completed, email_verified,
		          org_name, org_industry, org_country, org_city, org_size,
		          logo_file_id, primary_color, accent_color, active_suites,
		          provisioning_status, provisioning_started_at, provisioning_completed_at, provisioning_error`,
		tenantID,
	)
	return scanWizardProgress(row)
}

func (r *OnboardingRepository) GetTenantIdentity(ctx context.Context, tenantID uuid.UUID) (string, string, iammodel.TenantStatus, *time.Time, error) {
	var (
		name        string
		slug        string
		status      iammodel.TenantStatus
		retainUntil *time.Time
	)
	err := r.pool.QueryRow(ctx, `
		SELECT name, slug, status, retain_until
		FROM tenants
		WHERE id = $1`,
		tenantID,
	).Scan(&name, &slug, &status, &retainUntil)
	return name, slug, status, retainUntil, err
}

func scanWizardProgress(row interface{ Scan(...any) error }) (*onboardingmodel.WizardProgress, error) {
	item := &onboardingmodel.WizardProgress{}
	var (
		orgIndustry       *string
		orgSize           *string
		stepsCompleted    []int32
		activeSuites      []string
		logoFileID        *uuid.UUID
		orgName           *string
		orgCity           *string
		primaryColor      *string
		accentColor       *string
		provStartedAt     *time.Time
		provCompletedAt   *time.Time
		provisioningError *string
	)
	if err := row.Scan(
		&item.TenantID,
		&item.CurrentStep,
		&stepsCompleted,
		&item.WizardCompleted,
		&item.EmailVerified,
		&orgName,
		&orgIndustry,
		&item.Country,
		&orgCity,
		&orgSize,
		&logoFileID,
		&primaryColor,
		&accentColor,
		&activeSuites,
		&item.ProvisioningStatus,
		&provStartedAt,
		&provCompletedAt,
		&provisioningError,
	); err != nil {
		return nil, err
	}
	item.StepsCompleted = int32SliceToInts(stepsCompleted)
	item.OrganizationName = orgName
	item.City = orgCity
	item.LogoFileID = logoFileID
	item.PrimaryColor = primaryColor
	item.AccentColor = accentColor
	item.ActiveSuites = activeSuites
	item.ProvisioningStartedAt = provStartedAt
	item.ProvisioningCompletedAt = provCompletedAt
	item.ProvisioningError = provisioningError
	if orgIndustry != nil {
		value := onboardingmodel.OrgIndustry(*orgIndustry)
		item.Industry = &value
	}
	if orgSize != nil {
		value := onboardingmodel.OrgSize(*orgSize)
		item.OrganizationSize = &value
	}
	return item, nil
}
