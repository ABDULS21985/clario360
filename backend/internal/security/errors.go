package security

import "errors"

// Sentinel errors for security violations.
var (
	// Input validation errors
	ErrMaliciousInput         = errors.New("security: malicious input detected")
	ErrInvalidJSON            = errors.New("security: invalid JSON")
	ErrJSONTooLarge           = errors.New("security: JSON payload exceeds maximum size")
	ErrJSONTooDeep            = errors.New("security: JSON nesting exceeds maximum depth")
	ErrStringTooLong          = errors.New("security: string exceeds maximum length")
	ErrPathTraversalDetected  = errors.New("security: path traversal detected")
	ErrDangerousFileExtension = errors.New("security: dangerous file extension")
	ErrHiddenFile             = errors.New("security: hidden files not allowed")

	// XSS errors
	ErrXSSDetected = errors.New("security: XSS pattern detected")

	// CSRF errors
	ErrCSRFMissing    = errors.New("security: CSRF token cookie not found")
	ErrCSRFHeaderMissing = errors.New("security: CSRF header not found")
	ErrCSRFMismatch   = errors.New("security: CSRF token mismatch")

	// Auth errors
	ErrForbidden       = errors.New("security: insufficient permissions")
	ErrNotFound        = errors.New("security: resource not found")
	ErrInvalidTable    = errors.New("security: invalid table name")
	ErrForbiddenField  = errors.New("security: forbidden field in request")
	ErrUnknownField    = errors.New("security: unknown field in request")
	ErrAccountLocked   = errors.New("security: account temporarily locked")

	// Rate limit errors
	ErrRateLimited = errors.New("security: rate limit exceeded")

	// Session errors
	ErrSessionExpired       = errors.New("security: session expired")
	ErrSessionFixation      = errors.New("security: session fixation detected")
	ErrConcurrentSession    = errors.New("security: concurrent session limit exceeded")

	// File upload errors
	ErrFileTooLarge       = errors.New("security: file exceeds maximum size")
	ErrInvalidMIMEType    = errors.New("security: file type not allowed")
	ErrMagicByteMismatch  = errors.New("security: file content does not match declared type")
	ErrMalwareDetected    = errors.New("security: malware detected in file")

	// SSRF errors
	ErrSSRFBlocked       = errors.New("security: request to private/restricted address blocked")
	ErrSSRFDNSRebinding  = errors.New("security: DNS rebinding attack detected")
	ErrSSRFRedirectChain = errors.New("security: excessive redirect chain detected")

	// API security errors
	ErrInvalidContentType = errors.New("security: invalid content type")
	ErrInvalidUUID        = errors.New("security: invalid UUID format")
	ErrPaginationExceeded = errors.New("security: pagination limit exceeded")
)
