package service

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/config"
	iammodel "github.com/clario360/platform/internal/iam/model"
	iamrepository "github.com/clario360/platform/internal/iam/repository"
	onboardingmodel "github.com/clario360/platform/internal/onboarding/model"
	onboardingrepo "github.com/clario360/platform/internal/onboarding/repository"
)

func newServiceTestJWTManager(t *testing.T) *auth.JWTManager {
	t.Helper()

	mgr, err := auth.NewJWTManager(config.AuthConfig{
		JWTIssuer:       "test-issuer",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("NewJWTManager failed: %v", err)
	}

	return mgr
}

func newDiscardLogger() zerolog.Logger {
	return zerolog.New(io.Discard)
}

type fakeEmailSender struct {
	verificationEmails []verificationEmail
	invitationEmails   []invitationEmail
	welcomeEmails      []welcomeEmail
}

type verificationEmail struct {
	email     string
	orgName   string
	adminName string
	otp       string
}

type invitationEmail struct {
	email            string
	organizationName string
	inviterName      string
	roleName         string
	rawToken         string
	message          *string
	expiresAt        time.Time
}

type welcomeEmail struct {
	email            string
	organizationName string
	firstName        string
}

func (f *fakeEmailSender) SendVerificationEmail(ctx context.Context, email, orgName, adminName, otp string) error {
	f.verificationEmails = append(f.verificationEmails, verificationEmail{
		email:     email,
		orgName:   orgName,
		adminName: adminName,
		otp:       otp,
	})
	return nil
}

func (f *fakeEmailSender) SendInvitationEmail(ctx context.Context, email, organizationName, inviterName, roleName, rawToken string, message *string, expiresAt time.Time) error {
	f.invitationEmails = append(f.invitationEmails, invitationEmail{
		email:            email,
		organizationName: organizationName,
		inviterName:      inviterName,
		roleName:         roleName,
		rawToken:         rawToken,
		message:          message,
		expiresAt:        expiresAt,
	})
	return nil
}

func (f *fakeEmailSender) SendWelcomeEmail(ctx context.Context, email, organizationName, firstName string) error {
	f.welcomeEmails = append(f.welcomeEmails, welcomeEmail{
		email:            email,
		organizationName: organizationName,
		firstName:        firstName,
	})
	return nil
}

type fakeUserRepo struct {
	byID    map[string]*iammodel.User
	byEmail map[string]*iammodel.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		byID:    make(map[string]*iammodel.User),
		byEmail: make(map[string]*iammodel.User),
	}
}

func (f *fakeUserRepo) key(tenantID, email string) string {
	return tenantID + ":" + strings.ToLower(strings.TrimSpace(email))
}

func (f *fakeUserRepo) put(user *iammodel.User) {
	f.byID[user.ID] = user
	f.byEmail[f.key(user.TenantID, user.Email)] = user
}

func (f *fakeUserRepo) Create(ctx context.Context, user *iammodel.User) error {
	if user.ID == "" {
		user.ID = uuid.NewString()
	}
	user.CreatedAt = time.Now()
	user.UpdatedAt = user.CreatedAt
	f.put(user)
	return nil
}

func (f *fakeUserRepo) GetByID(ctx context.Context, id string) (*iammodel.User, error) {
	user, ok := f.byID[id]
	if !ok {
		return nil, iammodel.ErrNotFound
	}
	return user, nil
}

func (f *fakeUserRepo) GetByEmail(ctx context.Context, tenantID, email string) (*iammodel.User, error) {
	user, ok := f.byEmail[f.key(tenantID, email)]
	if !ok {
		return nil, iammodel.ErrNotFound
	}
	return user, nil
}

func (f *fakeUserRepo) List(ctx context.Context, tenantID string, filter iamrepository.UserFilter) ([]iammodel.User, int, error) {
	out := make([]iammodel.User, 0)
	for _, user := range f.byID {
		if user.TenantID == tenantID {
			out = append(out, *user)
		}
	}
	return out, len(out), nil
}

func (f *fakeUserRepo) Update(ctx context.Context, user *iammodel.User) error {
	f.put(user)
	return nil
}

func (f *fakeUserRepo) SoftDelete(ctx context.Context, id, deletedBy string) error {
	return nil
}

func (f *fakeUserRepo) UpdateStatus(ctx context.Context, id string, status iammodel.UserStatus, updatedBy string) error {
	user, err := f.GetByID(ctx, id)
	if err != nil {
		return err
	}
	user.Status = status
	return nil
}

func (f *fakeUserRepo) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	user, err := f.GetByID(ctx, id)
	if err != nil {
		return err
	}
	user.PasswordHash = passwordHash
	return nil
}

func (f *fakeUserRepo) UpdateMFA(ctx context.Context, id string, enabled bool, secret *string) error {
	user, err := f.GetByID(ctx, id)
	if err != nil {
		return err
	}
	user.MFAEnabled = enabled
	user.MFASecret = secret
	return nil
}

func (f *fakeUserRepo) UpdateLastLogin(ctx context.Context, id string) error {
	return nil
}

func (f *fakeUserRepo) CountByTenant(ctx context.Context, tenantID string) (int, error) {
	count := 0
	for _, user := range f.byID {
		if user.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

type fakeRoleRepo struct {
	byID        map[string]*iammodel.Role
	byTenantKey map[string]*iammodel.Role
	userRoles   map[string][]iammodel.Role
}

func newFakeRoleRepo() *fakeRoleRepo {
	return &fakeRoleRepo{
		byID:        make(map[string]*iammodel.Role),
		byTenantKey: make(map[string]*iammodel.Role),
		userRoles:   make(map[string][]iammodel.Role),
	}
}

func (f *fakeRoleRepo) key(tenantID, slug string) string {
	return tenantID + ":" + strings.TrimSpace(slug)
}

func (f *fakeRoleRepo) addRole(tenantID, slug, name string, permissions []string) *iammodel.Role {
	role := &iammodel.Role{
		ID:           uuid.NewString(),
		TenantID:     tenantID,
		Name:         name,
		Slug:         slug,
		IsSystemRole: true,
		Permissions:  append([]string(nil), permissions...),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	f.byID[role.ID] = role
	f.byTenantKey[f.key(tenantID, slug)] = role
	return role
}

func (f *fakeRoleRepo) assignUserRole(userID string, role iammodel.Role) {
	existing := f.userRoles[userID]
	for _, current := range existing {
		if current.ID == role.ID {
			return
		}
	}
	f.userRoles[userID] = append(existing, role)
}

func (f *fakeRoleRepo) Create(ctx context.Context, role *iammodel.Role) error {
	if role.ID == "" {
		role.ID = uuid.NewString()
	}
	role.CreatedAt = time.Now()
	role.UpdatedAt = role.CreatedAt
	f.byID[role.ID] = role
	f.byTenantKey[f.key(role.TenantID, role.Slug)] = role
	return nil
}

func (f *fakeRoleRepo) GetByID(ctx context.Context, id string) (*iammodel.Role, error) {
	role, ok := f.byID[id]
	if !ok {
		return nil, iammodel.ErrNotFound
	}
	return role, nil
}

func (f *fakeRoleRepo) GetBySlug(ctx context.Context, tenantID, slug string) (*iammodel.Role, error) {
	role, ok := f.byTenantKey[f.key(tenantID, slug)]
	if !ok {
		return nil, iammodel.ErrNotFound
	}
	return role, nil
}

func (f *fakeRoleRepo) List(ctx context.Context, tenantID string) ([]iammodel.Role, error) {
	out := make([]iammodel.Role, 0)
	for _, role := range f.byTenantKey {
		if role.TenantID == tenantID {
			out = append(out, *role)
		}
	}
	return out, nil
}

func (f *fakeRoleRepo) Update(ctx context.Context, role *iammodel.Role) error {
	f.byID[role.ID] = role
	f.byTenantKey[f.key(role.TenantID, role.Slug)] = role
	return nil
}

func (f *fakeRoleRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func (f *fakeRoleRepo) AssignToUser(ctx context.Context, userID, roleID, tenantID, assignedBy string) error {
	role, err := f.GetByID(ctx, roleID)
	if err != nil {
		return err
	}
	f.assignUserRole(userID, *role)
	return nil
}

func (f *fakeRoleRepo) RemoveFromUser(ctx context.Context, userID, roleID string) error {
	return nil
}

func (f *fakeRoleRepo) GetUserRoles(ctx context.Context, userID string) ([]iammodel.Role, error) {
	return append([]iammodel.Role(nil), f.userRoles[userID]...), nil
}

func (f *fakeRoleRepo) ListUserIDsByRole(ctx context.Context, tenantID, roleSlug string) ([]string, error) {
	return nil, nil
}

func (f *fakeRoleRepo) SeedSystemRoles(ctx context.Context, tenantID string) error {
	for _, role := range iammodel.SystemRoles {
		roleCopy := role
		roleCopy.TenantID = tenantID
		if err := f.Create(ctx, &roleCopy); err != nil {
			return err
		}
	}
	return nil
}

type fakeSessionRepo struct {
	sessions []*iammodel.Session
}

func (f *fakeSessionRepo) Create(ctx context.Context, session *iammodel.Session) error {
	if session.ID == "" {
		session.ID = uuid.NewString()
	}
	session.CreatedAt = time.Now()
	f.sessions = append(f.sessions, session)
	return nil
}

func (f *fakeSessionRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*iammodel.Session, error) {
	for _, session := range f.sessions {
		if session.RefreshTokenHash == tokenHash {
			return session, nil
		}
	}
	return nil, iammodel.ErrNotFound
}

func (f *fakeSessionRepo) GetByUserID(ctx context.Context, userID string) ([]iammodel.Session, error) {
	out := make([]iammodel.Session, 0)
	for _, session := range f.sessions {
		if session.UserID == userID {
			out = append(out, *session)
		}
	}
	return out, nil
}

func (f *fakeSessionRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func (f *fakeSessionRepo) DeleteByUserID(ctx context.Context, userID string) error {
	return nil
}

func (f *fakeSessionRepo) DeleteExpired(ctx context.Context) (int64, error) {
	return 0, nil
}

type fakeTenantIdentity struct {
	name        string
	slug        string
	status      iammodel.TenantStatus
	retainUntil *time.Time
}

type createdEmailVerification struct {
	email     string
	otpHash   string
	expiresAt time.Time
}

type fakeOnboardingRepo struct {
	userRepo                *fakeUserRepo
	roleRepo                *fakeRoleRepo
	invitationRepo          *fakeInvitationRepo
	emailExists             map[string]bool
	organizationExists      map[string]bool
	tenantIdentities        map[uuid.UUID]fakeTenantIdentity
	onboardingByEmail       map[string]*onboardingmodel.OnboardingStatus
	onboardingByTenant      map[uuid.UUID]*onboardingmodel.OnboardingStatus
	verificationsByEmail    map[string]*onboardingmodel.EmailVerification
	createdRegistrations    []onboardingrepo.CreateRegistrationParams
	createdVerifications    []createdEmailVerification
	incrementAttemptsCalled int
}

func newFakeOnboardingRepo() *fakeOnboardingRepo {
	return &fakeOnboardingRepo{
		emailExists:          make(map[string]bool),
		organizationExists:   make(map[string]bool),
		tenantIdentities:     make(map[uuid.UUID]fakeTenantIdentity),
		onboardingByEmail:    make(map[string]*onboardingmodel.OnboardingStatus),
		onboardingByTenant:   make(map[uuid.UUID]*onboardingmodel.OnboardingStatus),
		verificationsByEmail: make(map[string]*onboardingmodel.EmailVerification),
	}
}

func (f *fakeOnboardingRepo) EmailExists(ctx context.Context, email string) (bool, error) {
	return f.emailExists[normalizeEmail(email)], nil
}

func (f *fakeOnboardingRepo) OrganizationNameExists(ctx context.Context, name string) (bool, error) {
	return f.organizationExists[strings.TrimSpace(strings.ToLower(name))], nil
}

func (f *fakeOnboardingRepo) CreateRegistration(ctx context.Context, params onboardingrepo.CreateRegistrationParams) error {
	f.createdRegistrations = append(f.createdRegistrations, params)

	orgName := params.TenantName
	industry := params.Industry
	onboarding := &onboardingmodel.OnboardingStatus{
		ID:                 uuid.New(),
		TenantID:           params.TenantID,
		AdminUserID:        params.AdminUserID,
		AdminEmail:         normalizeEmail(params.AdminEmail),
		CurrentStep:        0,
		OrgName:            &orgName,
		OrgIndustry:        &industry,
		OrgCountry:         params.Country,
		ActiveSuites:       []string{"cyber", "data", "visus"},
		ProvisioningStatus: onboardingmodel.OnboardingProvisioningPending,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	f.onboardingByEmail[normalizeEmail(params.AdminEmail)] = onboarding
	f.onboardingByTenant[params.TenantID] = onboarding
	f.tenantIdentities[params.TenantID] = fakeTenantIdentity{
		name:   params.TenantName,
		slug:   params.TenantSlug,
		status: iammodel.TenantStatusOnboarding,
	}
	f.verificationsByEmail[normalizeEmail(params.AdminEmail)] = &onboardingmodel.EmailVerification{
		ID:          uuid.New(),
		Email:       normalizeEmail(params.AdminEmail),
		OTPHash:     params.OTPHash,
		Purpose:     registrationPurpose,
		MaxAttempts: 5,
		ExpiresAt:   params.OTPExpiresAt,
		CreatedAt:   time.Now(),
	}

	if f.userRepo != nil {
		f.userRepo.put(&iammodel.User{
			ID:           params.AdminUserID.String(),
			TenantID:     params.TenantID.String(),
			Email:        normalizeEmail(params.AdminEmail),
			PasswordHash: params.PasswordHash,
			FirstName:    params.FirstName,
			LastName:     params.LastName,
			Status:       iammodel.UserStatusPendingVerification,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		})
	}
	if f.roleRepo != nil {
		role := f.roleRepo.addRole(params.TenantID.String(), "tenant-admin", "Tenant Admin", params.RolePermissions)
		f.roleRepo.assignUserRole(params.AdminUserID.String(), *role)
	}

	return nil
}

func (f *fakeOnboardingRepo) GetLatestEmailVerification(ctx context.Context, email, purpose string) (*onboardingmodel.EmailVerification, error) {
	verification, ok := f.verificationsByEmail[normalizeEmail(email)]
	if !ok {
		return nil, iammodel.ErrNotFound
	}
	return verification, nil
}

func (f *fakeOnboardingRepo) IncrementVerificationAttempts(ctx context.Context, verificationID uuid.UUID) (int, error) {
	f.incrementAttemptsCalled++
	for _, verification := range f.verificationsByEmail {
		if verification.ID != verificationID {
			continue
		}
		verification.Attempts++
		if verification.Attempts >= verification.MaxAttempts {
			now := time.Now()
			verification.LockedAt = &now
		}
		return verification.MaxAttempts - verification.Attempts, nil
	}
	return 0, iammodel.ErrNotFound
}

func (f *fakeOnboardingRepo) MarkEmailVerificationVerified(ctx context.Context, verificationID uuid.UUID) error {
	for _, verification := range f.verificationsByEmail {
		if verification.ID == verificationID {
			verification.Verified = true
			now := time.Now()
			verification.VerifiedAt = &now
			return nil
		}
	}
	return iammodel.ErrNotFound
}

func (f *fakeOnboardingRepo) ActivateRegistration(ctx context.Context, email string) (*onboardingrepo.ActivationResult, error) {
	onboarding, ok := f.onboardingByEmail[normalizeEmail(email)]
	if !ok {
		return nil, iammodel.ErrNotFound
	}
	onboarding.EmailVerified = true
	now := time.Now()
	onboarding.EmailVerifiedAt = &now
	if onboarding.CurrentStep < 1 {
		onboarding.CurrentStep = 1
	}
	if f.userRepo != nil {
		if user, err := f.userRepo.GetByID(ctx, onboarding.AdminUserID.String()); err == nil {
			user.Status = iammodel.UserStatusActive
		}
	}
	return &onboardingrepo.ActivationResult{
		OnboardingID: onboarding.ID,
		TenantID:     onboarding.TenantID,
		UserID:       onboarding.AdminUserID,
	}, nil
}

func (f *fakeOnboardingRepo) GetOnboardingByAdminEmail(ctx context.Context, email string) (*onboardingmodel.OnboardingStatus, error) {
	onboarding, ok := f.onboardingByEmail[normalizeEmail(email)]
	if !ok {
		return nil, iammodel.ErrNotFound
	}
	return onboarding, nil
}

func (f *fakeOnboardingRepo) CreateEmailVerification(ctx context.Context, email, otpHash string, expiresAt time.Time, ipAddress, userAgent *string) error {
	f.createdVerifications = append(f.createdVerifications, createdEmailVerification{
		email:     normalizeEmail(email),
		otpHash:   otpHash,
		expiresAt: expiresAt,
	})
	f.verificationsByEmail[normalizeEmail(email)] = &onboardingmodel.EmailVerification{
		ID:          uuid.New(),
		Email:       normalizeEmail(email),
		OTPHash:     otpHash,
		Purpose:     registrationPurpose,
		MaxAttempts: 5,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
	}
	return nil
}

func (f *fakeOnboardingRepo) GetTenantIdentity(ctx context.Context, tenantID uuid.UUID) (string, string, iammodel.TenantStatus, *time.Time, error) {
	identity, ok := f.tenantIdentities[tenantID]
	if !ok {
		return "", "", "", nil, iammodel.ErrNotFound
	}
	return identity.name, identity.slug, identity.status, identity.retainUntil, nil
}

func (f *fakeOnboardingRepo) CreateTenantUserWithRole(ctx context.Context, params onboardingrepo.CreateTenantUserParams) error {
	if f.userRepo != nil {
		f.userRepo.put(&iammodel.User{
			ID:           params.UserID.String(),
			TenantID:     params.TenantID.String(),
			Email:        normalizeEmail(params.Email),
			PasswordHash: params.PasswordHash,
			FirstName:    params.FirstName,
			LastName:     params.LastName,
			Status:       iammodel.UserStatusActive,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		})
	}
	if f.roleRepo != nil {
		if role, err := f.roleRepo.GetByID(ctx, params.RoleID.String()); err == nil {
			f.roleRepo.assignUserRole(params.UserID.String(), *role)
		}
	}
	if params.InvitationID != nil && f.invitationRepo != nil {
		if invitation, ok := f.invitationRepo.byID[*params.InvitationID]; ok {
			now := time.Now()
			invitation.Status = onboardingmodel.InvitationStatusAccepted
			invitation.AcceptedAt = &now
			invitation.AcceptedBy = &params.UserID
			invitation.UpdatedAt = now
		}
	}
	return nil
}

func (f *fakeOnboardingRepo) GetOnboardingByTenantID(ctx context.Context, tenantID uuid.UUID) (*onboardingmodel.OnboardingStatus, error) {
	onboarding, ok := f.onboardingByTenant[tenantID]
	if !ok {
		return nil, iammodel.ErrNotFound
	}
	return onboarding, nil
}

type fakeInvitationRepo struct {
	byID map[uuid.UUID]*onboardingmodel.Invitation
}

func newFakeInvitationRepo() *fakeInvitationRepo {
	return &fakeInvitationRepo{byID: make(map[uuid.UUID]*onboardingmodel.Invitation)}
}

func (f *fakeInvitationRepo) CountPending(ctx context.Context, tenantID uuid.UUID) (int, error) {
	count := 0
	for _, invitation := range f.byID {
		if invitation.TenantID == tenantID && invitation.Status == onboardingmodel.InvitationStatusPending {
			count++
		}
	}
	return count, nil
}

func (f *fakeInvitationRepo) Create(ctx context.Context, invitation *onboardingmodel.Invitation) error {
	for _, existing := range f.byID {
		if existing.TenantID == invitation.TenantID && existing.Email == invitation.Email && existing.Status == onboardingmodel.InvitationStatusPending {
			return fmt.Errorf("pending invitation already exists")
		}
	}
	if invitation.ID == uuid.Nil {
		invitation.ID = uuid.New()
	}
	now := time.Now()
	if invitation.CreatedAt.IsZero() {
		invitation.CreatedAt = now
	}
	invitation.UpdatedAt = now
	f.byID[invitation.ID] = invitation
	return nil
}

func (f *fakeInvitationRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]onboardingmodel.Invitation, error) {
	out := make([]onboardingmodel.Invitation, 0)
	for _, invitation := range f.byID {
		if invitation.TenantID == tenantID {
			out = append(out, *invitation)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (f *fakeInvitationRepo) GetByID(ctx context.Context, tenantID, invitationID uuid.UUID) (*onboardingmodel.Invitation, error) {
	invitation, ok := f.byID[invitationID]
	if !ok || invitation.TenantID != tenantID {
		return nil, iammodel.ErrNotFound
	}
	return invitation, nil
}

func (f *fakeInvitationRepo) ListByPrefix(ctx context.Context, tokenPrefix string) ([]onboardingmodel.Invitation, error) {
	out := make([]onboardingmodel.Invitation, 0)
	for _, invitation := range f.byID {
		if invitation.TokenPrefix == tokenPrefix {
			out = append(out, *invitation)
		}
	}
	return out, nil
}

func (f *fakeInvitationRepo) UpdateStatus(ctx context.Context, tenantID, invitationID uuid.UUID, status onboardingmodel.InvitationStatus) error {
	invitation, err := f.GetByID(ctx, tenantID, invitationID)
	if err != nil {
		return err
	}
	invitation.Status = status
	invitation.UpdatedAt = time.Now()
	return nil
}

func (f *fakeInvitationRepo) Refresh(ctx context.Context, tenantID, invitationID uuid.UUID, tokenHash, tokenPrefix string, expiresAt time.Time) error {
	invitation, err := f.GetByID(ctx, tenantID, invitationID)
	if err != nil {
		return err
	}
	invitation.TokenHash = tokenHash
	invitation.TokenPrefix = tokenPrefix
	invitation.ExpiresAt = expiresAt
	invitation.Status = onboardingmodel.InvitationStatusPending
	invitation.UpdatedAt = time.Now()
	return nil
}

func (f *fakeInvitationRepo) ExpirePastDue(ctx context.Context) error {
	now := time.Now()
	for _, invitation := range f.byID {
		if invitation.Status == onboardingmodel.InvitationStatusPending && invitation.ExpiresAt.Before(now) {
			invitation.Status = onboardingmodel.InvitationStatusExpired
			invitation.UpdatedAt = now
		}
	}
	return nil
}

type fakeProvisioningRepo struct {
	steps             map[uuid.UUID]map[int]*onboardingmodel.ProvisioningStep
	tenantStatus      map[uuid.UUID]string
	provisioningState map[uuid.UUID]onboardingmodel.OnboardingProvisioningStatus
	failureMessage    map[uuid.UUID]string
	startCounts       map[uuid.UUID]map[int]int
}

func newFakeProvisioningRepo() *fakeProvisioningRepo {
	return &fakeProvisioningRepo{
		steps:             make(map[uuid.UUID]map[int]*onboardingmodel.ProvisioningStep),
		tenantStatus:      make(map[uuid.UUID]string),
		provisioningState: make(map[uuid.UUID]onboardingmodel.OnboardingProvisioningStatus),
		failureMessage:    make(map[uuid.UUID]string),
		startCounts:       make(map[uuid.UUID]map[int]int),
	}
}

func (f *fakeProvisioningRepo) Initialize(ctx context.Context, tenantID, onboardingID uuid.UUID, stepNames []string) error {
	if _, ok := f.steps[tenantID]; !ok {
		f.steps[tenantID] = make(map[int]*onboardingmodel.ProvisioningStep)
	}
	for idx, name := range stepNames {
		stepNumber := idx + 1
		if _, exists := f.steps[tenantID][stepNumber]; exists {
			continue
		}
		f.steps[tenantID][stepNumber] = &onboardingmodel.ProvisioningStep{
			ID:           uuid.New(),
			TenantID:     tenantID,
			OnboardingID: onboardingID,
			StepNumber:   stepNumber,
			StepName:     name,
			Status:       onboardingmodel.ProvisioningStepPending,
			Metadata:     map[string]any{},
			CreatedAt:    time.Now(),
		}
	}
	f.provisioningState[tenantID] = onboardingmodel.OnboardingProvisioningProvisioning
	return nil
}

func (f *fakeProvisioningRepo) ListSteps(ctx context.Context, tenantID uuid.UUID) ([]onboardingmodel.ProvisioningStep, error) {
	tenantSteps := f.steps[tenantID]
	keys := make([]int, 0, len(tenantSteps))
	for key := range tenantSteps {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	out := make([]onboardingmodel.ProvisioningStep, 0, len(keys))
	for _, key := range keys {
		out = append(out, *tenantSteps[key])
	}
	return out, nil
}

func (f *fakeProvisioningRepo) StartStep(ctx context.Context, tenantID uuid.UUID, stepNumber int) error {
	step := f.steps[tenantID][stepNumber]
	if step == nil {
		return iammodel.ErrNotFound
	}
	step.Status = onboardingmodel.ProvisioningStepRunning
	now := time.Now()
	step.StartedAt = &now
	if _, ok := f.startCounts[tenantID]; !ok {
		f.startCounts[tenantID] = make(map[int]int)
	}
	f.startCounts[tenantID][stepNumber]++
	return nil
}

func (f *fakeProvisioningRepo) CompleteStep(ctx context.Context, tenantID uuid.UUID, stepNumber int, metadata map[string]any) error {
	step := f.steps[tenantID][stepNumber]
	if step == nil {
		return iammodel.ErrNotFound
	}
	step.Status = onboardingmodel.ProvisioningStepCompleted
	step.Metadata = metadata
	now := time.Now()
	step.CompletedAt = &now
	return nil
}

func (f *fakeProvisioningRepo) FailStep(ctx context.Context, tenantID uuid.UUID, stepNumber int, errMessage string, metadata map[string]any) error {
	step := f.steps[tenantID][stepNumber]
	if step == nil {
		return iammodel.ErrNotFound
	}
	step.Status = onboardingmodel.ProvisioningStepFailed
	step.Metadata = metadata
	now := time.Now()
	step.CompletedAt = &now
	step.ErrorMessage = &errMessage
	return nil
}

func (f *fakeProvisioningRepo) MarkFailed(ctx context.Context, tenantID uuid.UUID, errMessage string) error {
	f.provisioningState[tenantID] = onboardingmodel.OnboardingProvisioningFailed
	f.failureMessage[tenantID] = errMessage
	return nil
}

func (f *fakeProvisioningRepo) MarkCompleted(ctx context.Context, tenantID uuid.UUID) error {
	f.provisioningState[tenantID] = onboardingmodel.OnboardingProvisioningCompleted
	return nil
}

func (f *fakeProvisioningRepo) SetTenantStatus(ctx context.Context, tenantID uuid.UUID, status string) error {
	f.tenantStatus[tenantID] = status
	return nil
}

type auditRecord struct {
	tenantID uuid.UUID
	adminID  uuid.UUID
	action   string
	metadata map[string]any
}

type fakeLifecycleStore struct {
	suspendCalls           []uuid.UUID
	deleteSessionCalls     []uuid.UUID
	revokeAPIKeyCalls      []uuid.UUID
	softDeleteCalls        []uuid.UUID
	restoreCalls           []uuid.UUID
	markedDeprovisioned    []uuid.UUID
	markedActive           []uuid.UUID
	auditRecords           []auditRecord
	lastRetainUntil        time.Time
	lastDeprovisionAdminID uuid.UUID
}

func (f *fakeLifecycleStore) SuspendUsers(ctx context.Context, tenantID uuid.UUID) error {
	f.suspendCalls = append(f.suspendCalls, tenantID)
	return nil
}

func (f *fakeLifecycleStore) DeleteSessions(ctx context.Context, tenantID uuid.UUID) error {
	f.deleteSessionCalls = append(f.deleteSessionCalls, tenantID)
	return nil
}

func (f *fakeLifecycleStore) RevokeAPIKeys(ctx context.Context, tenantID uuid.UUID) error {
	f.revokeAPIKeyCalls = append(f.revokeAPIKeyCalls, tenantID)
	return nil
}

func (f *fakeLifecycleStore) SoftDeleteTenantRows(ctx context.Context, tenantID uuid.UUID) error {
	f.softDeleteCalls = append(f.softDeleteCalls, tenantID)
	return nil
}

func (f *fakeLifecycleStore) RestoreTenantRows(ctx context.Context, tenantID uuid.UUID) error {
	f.restoreCalls = append(f.restoreCalls, tenantID)
	return nil
}

func (f *fakeLifecycleStore) MarkTenantDeprovisioned(ctx context.Context, tenantID, adminID uuid.UUID, retainUntil time.Time) error {
	f.markedDeprovisioned = append(f.markedDeprovisioned, tenantID)
	f.lastRetainUntil = retainUntil
	f.lastDeprovisionAdminID = adminID
	return nil
}

func (f *fakeLifecycleStore) MarkTenantActive(ctx context.Context, tenantID uuid.UUID) error {
	f.markedActive = append(f.markedActive, tenantID)
	return nil
}

func (f *fakeLifecycleStore) InsertAuditLog(ctx context.Context, tenantID, adminID uuid.UUID, action string, metadata map[string]any) error {
	f.auditRecords = append(f.auditRecords, auditRecord{
		tenantID: tenantID,
		adminID:  adminID,
		action:   action,
		metadata: metadata,
	})
	return nil
}
