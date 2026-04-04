package security

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// ClamAVScanner provides real-time virus scanning by connecting to a ClamAV daemon
// (clamd) over TCP using the INSTREAM protocol.
//
// Protocol reference (clamd man page):
//  1. Send "zINSTREAM\0"
//  2. Send data in chunks: 4-byte big-endian length + data (max 25 MB per chunk)
//  3. End stream with 4 zero bytes
//  4. Read response: "stream: OK\0" or "stream: <virus_name> FOUND\0"
type ClamAVScanner struct {
	addr       string // host:port of clamd (default "localhost:3310")
	timeout    time.Duration
	maxSize    int64 // max stream size, must not exceed clamd's StreamMaxLength
	chunkSize  int
	logger     zerolog.Logger
	metrics    *Metrics
	mu         sync.RWMutex
	available  bool
	lastCheck  time.Time
}

// ClamAVConfig configures the ClamAV scanner connection.
type ClamAVConfig struct {
	Addr      string        // clamd address, e.g., "localhost:3310"
	Timeout   time.Duration // per-scan timeout
	MaxSize   int64         // max file size to scan (bytes)
	ChunkSize int           // chunk size for INSTREAM protocol
}

// DefaultClamAVConfig returns production defaults matching docker-compose.yml (port 3310).
func DefaultClamAVConfig() *ClamAVConfig {
	return &ClamAVConfig{
		Addr:      "localhost:3310",
		Timeout:   30 * time.Second,
		MaxSize:   25 * 1024 * 1024, // 25 MB — clamd default StreamMaxLength
		ChunkSize: 8192,
	}
}

// ScanResult represents the outcome of a ClamAV scan.
type ScanResult struct {
	Clean     bool   // true if no malware detected
	VirusName string // populated only if malware is found
	Error     error  // non-nil if scan failed (connection error, timeout, etc.)
}

// NewClamAVScanner creates a scanner connected to the specified clamd instance.
func NewClamAVScanner(cfg *ClamAVConfig, metrics *Metrics, logger zerolog.Logger) *ClamAVScanner {
	if cfg == nil {
		cfg = DefaultClamAVConfig()
	}
	if cfg.ChunkSize <= 0 {
		cfg.ChunkSize = 8192
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 25 * 1024 * 1024
	}

	s := &ClamAVScanner{
		addr:      cfg.Addr,
		timeout:   cfg.Timeout,
		maxSize:   cfg.MaxSize,
		chunkSize: cfg.ChunkSize,
		logger:    logger.With().Str("component", "clamav").Logger(),
		metrics:   metrics,
	}

	// Perform initial connectivity check (non-blocking best-effort)
	go s.ping()

	return s
}

// Scan scans file data via the clamd INSTREAM command.
// Returns ErrMalwareDetected if malware is found.
// This method is safe for concurrent use.
func (s *ClamAVScanner) Scan(data []byte, filename string) error {
	if int64(len(data)) > s.maxSize {
		return fmt.Errorf("%w: file size %d exceeds ClamAV limit %d", ErrFileTooLarge, len(data), s.maxSize)
	}

	result := s.scanINSTREAM(data)

	if result.Error != nil {
		s.logger.Error().
			Err(result.Error).
			Str("filename", filename).
			Int("size", len(data)).
			Msg("ClamAV scan failed")
		// Do NOT fail open for virus scanning — return the error so the caller
		// decides whether to reject or quarantine
		return fmt.Errorf("virus scan unavailable: %w", result.Error)
	}

	if !result.Clean {
		s.logger.Warn().
			Str("filename", filename).
			Str("virus", result.VirusName).
			Int("size", len(data)).
			Msg("malware detected in uploaded file")
		if s.metrics != nil {
			s.metrics.FileUploadBlocked.WithLabelValues("malware").Inc()
		}
		return fmt.Errorf("%w: %s", ErrMalwareDetected, result.VirusName)
	}

	s.logger.Debug().
		Str("filename", filename).
		Int("size", len(data)).
		Msg("file passed ClamAV scan")

	return nil
}

// ScanHook returns a function compatible with WithVirusScanHook for use
// with FileUploadValidator.
func (s *ClamAVScanner) ScanHook() func(data []byte, filename string) error {
	return s.Scan
}

// Ping checks if clamd is reachable and responsive.
func (s *ClamAVScanner) Ping() error {
	return s.ping()
}

// IsAvailable returns the cached availability status from the last health check.
func (s *ClamAVScanner) IsAvailable() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.available
}

// ping sends a PING command to clamd and expects "PONG\n" back.
func (s *ClamAVScanner) ping() error {
	conn, err := net.DialTimeout("tcp", s.addr, 5*time.Second)
	if err != nil {
		s.mu.Lock()
		s.available = false
		s.lastCheck = time.Now()
		s.mu.Unlock()
		return fmt.Errorf("clamd connection failed: %w", err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}

	// Send PING command (null-terminated)
	if _, err := conn.Write([]byte("zPING\x00")); err != nil {
		s.mu.Lock()
		s.available = false
		s.lastCheck = time.Now()
		s.mu.Unlock()
		return fmt.Errorf("clamd PING write failed: %w", err)
	}

	resp := make([]byte, 64)
	n, err := conn.Read(resp)
	if err != nil {
		s.mu.Lock()
		s.available = false
		s.lastCheck = time.Now()
		s.mu.Unlock()
		return fmt.Errorf("clamd PING read failed: %w", err)
	}

	response := strings.TrimRight(string(resp[:n]), "\x00\n")
	if response != "PONG" {
		s.mu.Lock()
		s.available = false
		s.lastCheck = time.Now()
		s.mu.Unlock()
		return fmt.Errorf("clamd unexpected PING response: %q", response)
	}

	s.mu.Lock()
	s.available = true
	s.lastCheck = time.Now()
	s.mu.Unlock()

	return nil
}

// scanINSTREAM implements the clamd INSTREAM protocol.
func (s *ClamAVScanner) scanINSTREAM(data []byte) ScanResult {
	conn, err := net.DialTimeout("tcp", s.addr, s.timeout)
	if err != nil {
		return ScanResult{Error: fmt.Errorf("clamd connect: %w", err)}
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(s.timeout)); err != nil {
		return ScanResult{Error: fmt.Errorf("clamd set deadline: %w", err)}
	}

	// 1. Send INSTREAM command (null-terminated for z-protocol)
	if _, err := conn.Write([]byte("zINSTREAM\x00")); err != nil {
		return ScanResult{Error: fmt.Errorf("clamd INSTREAM write: %w", err)}
	}

	// 2. Send data in chunks: [4-byte big-endian length][data]
	reader := bytes.NewReader(data)
	chunk := make([]byte, s.chunkSize)
	lenBuf := make([]byte, 4)

	for {
		n, readErr := reader.Read(chunk)
		if n > 0 {
			binary.BigEndian.PutUint32(lenBuf, uint32(n))
			if _, err := conn.Write(lenBuf); err != nil {
				return ScanResult{Error: fmt.Errorf("clamd chunk length write: %w", err)}
			}
			if _, err := conn.Write(chunk[:n]); err != nil {
				return ScanResult{Error: fmt.Errorf("clamd chunk data write: %w", err)}
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return ScanResult{Error: fmt.Errorf("reading scan data: %w", readErr)}
		}
	}

	// 3. End stream with 4 zero bytes
	binary.BigEndian.PutUint32(lenBuf, 0)
	if _, err := conn.Write(lenBuf); err != nil {
		return ScanResult{Error: fmt.Errorf("clamd end stream: %w", err)}
	}

	// 4. Read response
	var respBuf bytes.Buffer
	if _, err := io.Copy(&respBuf, conn); err != nil {
		return ScanResult{Error: fmt.Errorf("clamd read response: %w", err)}
	}

	return parseINSTREAMResponse(respBuf.String())
}

// parseINSTREAMResponse parses the clamd response.
// Expected formats:
//
//	"stream: OK" — no malware
//	"stream: <VirusName> FOUND" — malware detected
//	"INSTREAM size limit exceeded" — file too large
func parseINSTREAMResponse(response string) ScanResult {
	response = strings.TrimRight(response, "\x00\n\r ")

	if response == "stream: OK" {
		return ScanResult{Clean: true}
	}

	if strings.HasSuffix(response, "FOUND") {
		// "stream: Win.Test.EICAR_HDB-1 FOUND"
		virus := strings.TrimPrefix(response, "stream: ")
		virus = strings.TrimSuffix(virus, " FOUND")
		return ScanResult{Clean: false, VirusName: virus}
	}

	if strings.Contains(response, "size limit exceeded") {
		return ScanResult{Error: fmt.Errorf("clamd: %s", response)}
	}

	if strings.Contains(response, "ERROR") {
		return ScanResult{Error: fmt.Errorf("clamd: %s", response)}
	}

	// Unexpected response
	return ScanResult{Error: fmt.Errorf("clamd unexpected response: %q", response)}
}

// ClamAVHealthCheck returns a health check function for use with the bootstrap health checker.
func (s *ClamAVScanner) ClamAVHealthCheck() func() error {
	return func() error {
		return s.ping()
	}
}
