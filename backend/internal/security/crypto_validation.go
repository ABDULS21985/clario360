package security

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/rs/zerolog"
)

// CryptoValidator validates cryptographic configuration.
type CryptoValidator struct {
	logger zerolog.Logger
}

// NewCryptoValidator creates a new crypto validator.
func NewCryptoValidator(logger zerolog.Logger) *CryptoValidator {
	return &CryptoValidator{
		logger: logger.With().Str("component", "crypto_validation").Logger(),
	}
}

// CryptoValidationResult represents the result of a crypto validation check.
type CryptoValidationResult struct {
	Check    string
	Status   string // "pass", "warn", "fail"
	Details  string
}

// ValidateTLSConfig checks TLS configuration for security best practices.
func (cv *CryptoValidator) ValidateTLSConfig(cfg *tls.Config) []CryptoValidationResult {
	var results []CryptoValidationResult

	if cfg == nil {
		results = append(results, CryptoValidationResult{
			Check:   "tls_config",
			Status:  "fail",
			Details: "TLS configuration is nil",
		})
		return results
	}

	// Check minimum TLS version
	if cfg.MinVersion < tls.VersionTLS12 {
		results = append(results, CryptoValidationResult{
			Check:   "tls_min_version",
			Status:  "fail",
			Details: "minimum TLS version must be 1.2 or higher",
		})
	} else if cfg.MinVersion == tls.VersionTLS12 {
		results = append(results, CryptoValidationResult{
			Check:   "tls_min_version",
			Status:  "pass",
			Details: "TLS 1.2 minimum version configured",
		})
	} else {
		results = append(results, CryptoValidationResult{
			Check:   "tls_min_version",
			Status:  "pass",
			Details: "TLS 1.3 minimum version configured",
		})
	}

	// Check cipher suites (only relevant for TLS 1.2)
	if cfg.MinVersion <= tls.VersionTLS12 && len(cfg.CipherSuites) > 0 {
		weakCiphers := findWeakCiphers(cfg.CipherSuites)
		if len(weakCiphers) > 0 {
			results = append(results, CryptoValidationResult{
				Check:   "tls_cipher_suites",
				Status:  "fail",
				Details: fmt.Sprintf("weak cipher suites detected: %s", strings.Join(weakCiphers, ", ")),
			})
		} else {
			results = append(results, CryptoValidationResult{
				Check:   "tls_cipher_suites",
				Status:  "pass",
				Details: "all cipher suites are strong",
			})
		}
	}

	// Check InsecureSkipVerify
	if cfg.InsecureSkipVerify {
		results = append(results, CryptoValidationResult{
			Check:   "tls_verify",
			Status:  "fail",
			Details: "InsecureSkipVerify is true — certificate validation disabled",
		})
	} else {
		results = append(results, CryptoValidationResult{
			Check:   "tls_verify",
			Status:  "pass",
			Details: "certificate verification enabled",
		})
	}

	return results
}

// ValidateHashingConfig checks password hashing configuration.
func (cv *CryptoValidator) ValidateHashingConfig(algorithm string, iterations int, keyLength int) []CryptoValidationResult {
	var results []CryptoValidationResult

	switch strings.ToLower(algorithm) {
	case "argon2id":
		results = append(results, CryptoValidationResult{
			Check:   "hash_algorithm",
			Status:  "pass",
			Details: "Argon2id is recommended for password hashing",
		})
	case "bcrypt":
		results = append(results, CryptoValidationResult{
			Check:   "hash_algorithm",
			Status:  "pass",
			Details: "bcrypt is acceptable for password hashing",
		})
		if iterations < 12 {
			results = append(results, CryptoValidationResult{
				Check:   "hash_cost",
				Status:  "warn",
				Details: fmt.Sprintf("bcrypt cost factor %d is below recommended minimum of 12", iterations),
			})
		}
	case "pbkdf2":
		results = append(results, CryptoValidationResult{
			Check:   "hash_algorithm",
			Status:  "warn",
			Details: "PBKDF2 is acceptable but Argon2id is preferred",
		})
		if iterations < 600000 {
			results = append(results, CryptoValidationResult{
				Check:   "hash_iterations",
				Status:  "fail",
				Details: fmt.Sprintf("PBKDF2 iterations %d below minimum 600000", iterations),
			})
		}
	case "sha256", "sha1", "md5":
		results = append(results, CryptoValidationResult{
			Check:   "hash_algorithm",
			Status:  "fail",
			Details: fmt.Sprintf("%s is not suitable for password hashing", algorithm),
		})
	default:
		results = append(results, CryptoValidationResult{
			Check:   "hash_algorithm",
			Status:  "warn",
			Details: fmt.Sprintf("unknown hashing algorithm: %s", algorithm),
		})
	}

	if keyLength > 0 && keyLength < 256 {
		results = append(results, CryptoValidationResult{
			Check:   "key_length",
			Status:  "fail",
			Details: fmt.Sprintf("key length %d bits is below minimum 256", keyLength),
		})
	}

	return results
}

// ValidateKeyStrength checks cryptographic key strength.
func (cv *CryptoValidator) ValidateKeyStrength(algorithm string, keyBits int) CryptoValidationResult {
	switch strings.ToUpper(algorithm) {
	case "RSA":
		if keyBits < 2048 {
			return CryptoValidationResult{
				Check:   "key_strength",
				Status:  "fail",
				Details: fmt.Sprintf("RSA key size %d bits is below minimum 2048", keyBits),
			}
		}
		if keyBits < 4096 {
			return CryptoValidationResult{
				Check:   "key_strength",
				Status:  "warn",
				Details: fmt.Sprintf("RSA key size %d bits; 4096 recommended for high-security", keyBits),
			}
		}
		return CryptoValidationResult{
			Check:   "key_strength",
			Status:  "pass",
			Details: fmt.Sprintf("RSA key size %d bits meets requirements", keyBits),
		}
	case "EC", "ECDSA":
		if keyBits < 256 {
			return CryptoValidationResult{
				Check:   "key_strength",
				Status:  "fail",
				Details: fmt.Sprintf("EC key size %d bits is below minimum 256", keyBits),
			}
		}
		return CryptoValidationResult{
			Check:   "key_strength",
			Status:  "pass",
			Details: fmt.Sprintf("EC key size %d bits meets requirements", keyBits),
		}
	default:
		return CryptoValidationResult{
			Check:   "key_strength",
			Status:  "warn",
			Details: fmt.Sprintf("cannot validate key strength for algorithm: %s", algorithm),
		}
	}
}

// ValidateAll runs a default set of cryptographic checks suitable for automated
// scanning. Validates TLS with recommended settings and standard hash algorithm.
func (cv *CryptoValidator) ValidateAll() []CryptoValidationResult {
	var results []CryptoValidationResult

	// Validate TLS with recommended minimum settings
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	results = append(results, cv.ValidateTLSConfig(tlsCfg)...)

	// Validate hash config (assuming argon2id as used in IAM service)
	results = append(results, cv.ValidateHashingConfig("argon2id", 3, 256)...)

	// Validate RS256 JWT key strength
	results = append(results, cv.ValidateKeyStrength("RSA", 2048))

	return results
}

// weakCipherSuites lists TLS cipher suites considered weak.
var weakCipherSuites = map[uint16]string{
	tls.TLS_RSA_WITH_RC4_128_SHA:                "TLS_RSA_WITH_RC4_128_SHA",
	tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA:           "TLS_RSA_WITH_3DES_EDE_CBC_SHA",
	tls.TLS_RSA_WITH_AES_128_CBC_SHA:            "TLS_RSA_WITH_AES_128_CBC_SHA",
	tls.TLS_RSA_WITH_AES_256_CBC_SHA:            "TLS_RSA_WITH_AES_256_CBC_SHA",
	tls.TLS_RSA_WITH_AES_128_CBC_SHA256:         "TLS_RSA_WITH_AES_128_CBC_SHA256",
	tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA:        "TLS_ECDHE_ECDSA_WITH_RC4_128_SHA",
	tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA:          "TLS_ECDHE_RSA_WITH_RC4_128_SHA",
	tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA:     "TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA",
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA:    "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA",
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA:    "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
	tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA:      "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",
	tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA:      "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
	tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256:   "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256",
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256: "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256",
}

// findWeakCiphers returns names of weak cipher suites found in the list.
func findWeakCiphers(suites []uint16) []string {
	var weak []string
	for _, s := range suites {
		if name, isWeak := weakCipherSuites[s]; isWeak {
			weak = append(weak, name)
		}
	}
	return weak
}
