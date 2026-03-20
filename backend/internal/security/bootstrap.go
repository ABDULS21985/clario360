package security

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// SecurityStack holds all initialized security components, ready for middleware wiring.
type SecurityStack struct {
	Config        *Config
	Metrics       *Metrics
	Logger        *SecurityLogger
	Sanitizer     *Sanitizer
	SessionMgr    *SessionManager
	AuthLimiter   *AuthRateLimiter
	APILimiter    *APIRateLimiter
	ClamAV        *ClamAVScanner
	SSRFValidator *SSRFValidator
}

// Bootstrap initializes the full security stack from config, Redis, and a Prometheus registry.
// All components are wired together and ready for use as middleware or service-level checks.
func Bootstrap(cfg *Config, rdb *redis.Client, reg prometheus.Registerer, logger zerolog.Logger) (*SecurityStack, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	secLogger := logger.With().Str("subsystem", "security").Logger()

	metrics := NewMetrics(reg)
	securityLogger := NewSecurityLogger(secLogger, metrics, cfg.EnableTamperProof)

	// Sanitizer
	sanitizer := NewSanitizer(
		WithMaxStringLength(cfg.MaxStringLength),
		WithMaxJSONDepth(cfg.MaxJSONDepth),
		WithMaxJSONSize(cfg.MaxJSONSize),
		WithMaxFilenameLength(cfg.MaxFilenameLength),
	)

	// Session manager
	sessionCfg := &SessionConfig{
		IdleTimeout:    cfg.SessionIdleTimeout,
		AbsoluteMaxAge: cfg.SessionAbsoluteMax,
		MaxConcurrent:  cfg.MaxConcurrentSessions,
		RotateOnAuth:   true,
		BindToFingerprint: true,
	}
	sessionMgr := NewSessionManager(rdb, sessionCfg, metrics, secLogger)

	// Auth rate limiter
	authCfg := &AuthRateLimitConfig{
		LoginPerEmail:         cfg.LoginPerEmail,
		LoginPerIP:            cfg.LoginPerIP,
		LoginWindow:           cfg.LoginWindow,
		RegisterPerIP:         cfg.RegisterPerIP,
		RegisterWindow:        cfg.RegisterWindow,
		PasswordResetPerEmail: cfg.PasswordResetPerEmail,
		PasswordResetPerIP:    cfg.PasswordResetPerIP,
		PasswordResetWindow:   cfg.PasswordResetWindow,
		MFAPerSession:         cfg.MFAPerSession,
		MFAWindow:             cfg.MFAWindow,
		LockoutThreshold:      cfg.LockoutThreshold,
		LockoutDuration:       cfg.LockoutDuration,
		EscalationThreshold:   cfg.EscalationThreshold,
		EscalationWindow:      cfg.EscalationWindow,
	}
	authLimiter := NewAuthRateLimiter(rdb, authCfg, metrics, secLogger)

	// API rate limiter
	apiCfg := &APIRateLimitConfig{
		DefaultPerMinute: cfg.APIDefaultPerMinute,
		BurstMultiplier:  cfg.APIBurstMultiplier,
		EndpointLimits:   DefaultAPIRateLimitConfig().EndpointLimits,
	}
	apiLimiter := NewAPIRateLimiter(rdb, apiCfg, metrics, secLogger)

	// ClamAV scanner
	var clamav *ClamAVScanner
	if cfg.VirusScanEnabled {
		clamavCfg := cfg.ClamAVConfigFromConfig()
		clamav = NewClamAVScanner(clamavCfg, metrics, secLogger)
	}

	// SSRF validator
	ssrfValidator := NewSSRFValidator(cfg.SSRFAllowedHosts, cfg.SSRFBlockPrivate, metrics, secLogger)

	secLogger.Info().
		Str("environment", cfg.Environment).
		Bool("virus_scan", cfg.VirusScanEnabled).
		Bool("ssrf_block_private", cfg.SSRFBlockPrivate).
		Bool("tamper_proof_logs", cfg.EnableTamperProof).
		Int("max_concurrent_sessions", cfg.MaxConcurrentSessions).
		Msg("security stack initialized")

	return &SecurityStack{
		Config:        cfg,
		Metrics:       metrics,
		Logger:        securityLogger,
		Sanitizer:     sanitizer,
		SessionMgr:    sessionMgr,
		AuthLimiter:   authLimiter,
		APILimiter:    apiLimiter,
		ClamAV:        clamav,
		SSRFValidator: ssrfValidator,
	}, nil
}

// Middleware returns the standard middleware chain for the security stack.
// The returned middlewares should be applied in order:
//  1. SecurityHeaders
//  2. CSRFProtection
//  3. APIRateLimitMiddleware
//  4. SanitizeRequestBody
//  5. APISecurityMiddleware
//  6. ContentTypeEnforcement
func (ss *SecurityStack) Middleware(logger zerolog.Logger) []func(http.Handler) http.Handler {
	headersCfg := HeadersConfigFromConfig(ss.Config)

	csrfCfg := &CSRFConfig{
		CookieName:     ss.Config.CSRFCookieName,
		HeaderName:     ss.Config.CSRFHeaderName,
		CookieDomain:   ss.Config.CSRFCookieDomain,
		CookieSecure:   ss.Config.CSRFCookieSecure,
		CookieSameSite: http.SameSiteStrictMode,
		MaxAge:         ss.Config.CSRFMaxAge,
		ExemptMethods:  []string{http.MethodGet, http.MethodHead, http.MethodOptions},
		ExemptPaths: []string{
			"/api/v1/webhooks/",
			"/api/v1/health",
			"/healthz",
			"/readyz",
		},
	}

	apiSecCfg := DefaultAPISecurityConfig()

	return []func(http.Handler) http.Handler{
		SecurityHeaders(headersCfg, logger, ss.Metrics),
		CSRFProtection(csrfCfg, ss.Logger, logger, ss.Metrics),
		APIRateLimitMiddleware(ss.APILimiter, ss.Logger),
		SanitizeRequestBody(ss.Sanitizer, ss.Logger, logger),
		APISecurityMiddleware(apiSecCfg, ss.Logger, logger, ss.Metrics),
		ContentTypeEnforcement(ss.Logger, ss.Metrics),
	}
}

// FileUploadValidator creates a FileUploadValidator wired with the stack's
// sanitizer, metrics, and optional ClamAV scanner.
func (ss *SecurityStack) FileUploadValidator(logger zerolog.Logger) *FileUploadValidator {
	var opts []FileUploadOption
	if ss.ClamAV != nil {
		opts = append(opts, WithVirusScanHook(ss.ClamAV.ScanHook()))
	}
	return NewFileUploadValidator(ss.Config, ss.Sanitizer, ss.Metrics, logger, opts...)
}
