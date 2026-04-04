package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/events"
	iamdto "github.com/clario360/platform/internal/iam/dto"
	iammodel "github.com/clario360/platform/internal/iam/model"
	iamrepo "github.com/clario360/platform/internal/iam/repository"
	onboardingmodel "github.com/clario360/platform/internal/onboarding/model"
)

var (
	emailRegex     = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	htmlTagRegex   = regexp.MustCompile(`<[^>]+>`)
	countryRegex   = regexp.MustCompile(`^[A-Z]{2}$`)
	hexColorRegex  = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	disposableMail = map[string]struct{}{
		"mailinator.com": {}, "guerrillamail.com": {}, "10minutemail.com": {},
		"yopmail.com": {}, "temp-mail.org": {}, "dispostable.com": {},
		"trashmail.com": {}, "sharklasers.com": {}, "maildrop.cc": {},
	}
)

type issuedTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func normalizeCountry(country string) string {
	return strings.ToUpper(strings.TrimSpace(country))
}

func maskEmail(email string) string {
	parts := strings.Split(normalizeEmail(email), "@")
	if len(parts) != 2 || len(parts[0]) == 0 {
		return "***"
	}
	local := parts[0]
	if len(local) == 1 {
		return local + "***@" + parts[1]
	}
	return local[:1] + "***@" + parts[1]
}

func maskedEventEmail(email string) string {
	parts := strings.Split(normalizeEmail(email), "@")
	if len(parts) != 2 {
		return "***"
	}
	local := parts[0]
	domain := strings.Split(parts[1], ".")
	domainLabel := "***"
	tld := ""
	if len(domain) > 0 && len(domain[0]) > 0 {
		domainLabel = domain[0][:1] + "***"
	}
	if len(domain) > 1 {
		tld = "." + domain[len(domain)-1]
	}
	return local[:1] + "***@" + domainLabel + tld
}

func hashPII(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(strings.ToLower(value))))
	return hex.EncodeToString(sum[:])
}

func validateRegistrationInput(reqEmail, password, orgName, country string) error {
	if !emailRegex.MatchString(normalizeEmail(reqEmail)) {
		return fmt.Errorf("invalid email format: %w", iammodel.ErrValidation)
	}
	parts := strings.Split(normalizeEmail(reqEmail), "@")
	if len(parts) != 2 {
		return fmt.Errorf("invalid email format: %w", iammodel.ErrValidation)
	}
	if _, blocked := disposableMail[parts[1]]; blocked {
		return fmt.Errorf("disposable email addresses are not allowed: %w", iammodel.ErrValidation)
	}
	if err := validatePassword(password); err != nil {
		return err
	}
	trimmedName := strings.TrimSpace(orgName)
	if len(trimmedName) < 2 || len(trimmedName) > 100 {
		return fmt.Errorf("organization name must be between 2 and 100 characters: %w", iammodel.ErrValidation)
	}
	if trimmedName != html.EscapeString(trimmedName) || htmlTagRegex.MatchString(trimmedName) {
		return fmt.Errorf("organization name contains invalid characters: %w", iammodel.ErrValidation)
	}
	if !countryRegex.MatchString(normalizeCountry(country)) {
		return fmt.Errorf("country must be a valid ISO 3166-1 alpha-2 code: %w", iammodel.ErrValidation)
	}
	return nil
}

func validateIndustry(industry string) error {
	normalized := onboardingmodel.OrgIndustry(strings.TrimSpace(strings.ToLower(industry)))
	if normalized == "" {
		return nil
	}
	if _, ok := onboardingmodel.ValidOrgIndustries[normalized]; !ok {
		return fmt.Errorf("invalid organization industry: %w", iammodel.ErrValidation)
	}
	return nil
}

func validateOrganizationDetails(name, country string) error {
	trimmedName := strings.TrimSpace(name)
	if len(trimmedName) < 2 || len(trimmedName) > 100 {
		return fmt.Errorf("organization name must be between 2 and 100 characters: %w", iammodel.ErrValidation)
	}
	if trimmedName != html.EscapeString(trimmedName) || htmlTagRegex.MatchString(trimmedName) {
		return fmt.Errorf("organization name contains invalid characters: %w", iammodel.ErrValidation)
	}
	if !countryRegex.MatchString(normalizeCountry(country)) {
		return fmt.Errorf("country must be a valid ISO 3166-1 alpha-2 code: %w", iammodel.ErrValidation)
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < 12 || len(password) > 128 {
		return fmt.Errorf("password must be between 12 and 128 characters: %w", iammodel.ErrValidation)
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
		return fmt.Errorf("password must include uppercase, lowercase, digit, and special character: %w", iammodel.ErrValidation)
	}
	return nil
}

func validateHexColor(value string) error {
	if value == "" {
		return nil
	}
	if !hexColorRegex.MatchString(value) {
		return fmt.Errorf("invalid hex color: %w", iammodel.ErrValidation)
	}
	return nil
}

func generateTenantSlug(orgName string) (string, error) {
	base := strings.ToLower(strings.TrimSpace(orgName))
	base = strings.Map(func(r rune) rune {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			return r
		case unicode.IsSpace(r), r == '-', r == '_':
			return '-'
		default:
			return -1
		}
	}, base)
	base = regexp.MustCompile(`-+`).ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "tenant"
	}
	suffix := make([]byte, 2)
	if _, err := rand.Read(suffix); err != nil {
		return "", fmt.Errorf("random slug suffix: %w", err)
	}
	return fmt.Sprintf("%s-%s", base, hex.EncodeToString(suffix)), nil
}

func rolePermissionsBySlug(slug string) []string {
	for _, role := range iammodel.SystemRoles {
		if role.Slug == slug {
			return append([]string(nil), role.Permissions...)
		}
	}
	return []string{"*:read"}
}

func issueAuthTokens(
	ctx context.Context,
	user *iammodel.User,
	sessionRepo iamrepo.SessionRepository,
	jwtMgr *auth.JWTManager,
	refreshTTL time.Duration,
	ip,
	userAgent string,
) (*issuedTokens, error) {
	// Pre-generate the session ID so it can be embedded in the JWT "sid" claim.
	sessionID := uuid.New().String()

	tokenPair, err := jwtMgr.GenerateTokenPair(user.ID, user.TenantID, user.Email, user.RoleSlugs(), sessionID)
	if err != nil {
		return nil, fmt.Errorf("generate jwt tokens: %w", err)
	}

	tokenHash := sha256.Sum256([]byte(tokenPair.RefreshToken))
	hashString := hex.EncodeToString(tokenHash[:])

	var ipPtr, uaPtr *string
	if strings.TrimSpace(ip) != "" {
		ipCopy := ip
		ipPtr = &ipCopy
	}
	if strings.TrimSpace(userAgent) != "" {
		uaCopy := userAgent
		uaPtr = &uaCopy
	}

	if err := sessionRepo.Create(ctx, &iammodel.Session{
		ID:               sessionID,
		UserID:           user.ID,
		TenantID:         user.TenantID,
		RefreshTokenHash: hashString,
		IPAddress:        ipPtr,
		UserAgent:        uaPtr,
		ExpiresAt:        time.Now().Add(refreshTTL),
	}); err != nil {
		return nil, fmt.Errorf("create auth session: %w", err)
	}

	return &issuedTokens{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}, nil
}

func publishOnboardingEvent(ctx context.Context, producer *events.Producer, eventType string, tenantID uuid.UUID, userID *uuid.UUID, payload map[string]any, logger zerolog.Logger) {
	if producer == nil {
		return
	}
	evt, err := events.NewEvent(eventType, "iam-service", tenantID.String(), payload)
	if err != nil {
		logger.Error().Err(err).Str("event_type", eventType).Msg("create onboarding event")
		return
	}
	if userID != nil {
		evt.UserID = userID.String()
	}
	if err := producer.Publish(ctx, events.Topics.OnboardingEvents, evt); err != nil {
		logger.Error().Err(err).Str("event_type", eventType).Msg("publish onboarding event")
	}
}

func throttleKey(prefix, value string) string {
	return prefix + ":" + hashPII(value)
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

func consumeThrottle(ctx context.Context, redisClient *redis.Client, key string, limit int64, ttl time.Duration) (bool, int64, error) {
	if redisClient == nil {
		return true, limit, nil
	}
	pipe := redisClient.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return true, limit, err
	}
	current := incr.Val()
	remaining := limit - current
	if remaining < 0 {
		remaining = 0
	}
	return current <= limit, remaining, nil
}

func userToAuthResponse(user *iammodel.User) iamdto.UserResponse {
	return iamdto.UserToResponse(user)
}

func ensureActiveSuites(activeSuites []string) ([]string, error) {
	valid := map[string]struct{}{
		"cyber": {}, "data": {}, "acta": {}, "lex": {}, "visus": {},
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(activeSuites))
	for _, suite := range activeSuites {
		suite = strings.TrimSpace(strings.ToLower(suite))
		if _, ok := valid[suite]; !ok {
			return nil, fmt.Errorf("invalid suite %q: %w", suite, iammodel.ErrValidation)
		}
		if _, ok := seen[suite]; ok {
			continue
		}
		seen[suite] = struct{}{}
		out = append(out, suite)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("at least one suite must be selected: %w", iammodel.ErrValidation)
	}
	return out, nil
}

func toWizardStepLabel(step int) string {
	switch step {
	case 1:
		return "organization"
	case 2:
		return "branding"
	case 3:
		return "team"
	case 4:
		return "suites"
	case 5:
		return "complete"
	default:
		return "unknown"
	}
}

func buildProvisioningStatus(steps []onboardingmodel.ProvisioningStep) onboardingmodel.ProvisioningStatus {
	status := onboardingmodel.ProvisioningStatus{
		Steps:      steps,
		TotalSteps: len(steps),
	}
	for idx := range steps {
		if steps[idx].Status == onboardingmodel.ProvisioningStepCompleted {
			status.CompletedStep++
		}
	}
	if status.TotalSteps > 0 {
		status.ProgressPct = int(float64(status.CompletedStep) / float64(status.TotalSteps) * 100)
	}
	return status
}
