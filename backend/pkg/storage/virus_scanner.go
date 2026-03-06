package storage

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

const (
	clamdChunkSize = 2048
)

// ScanStatus represents the result of a virus scan.
type ScanStatus string

const (
	ScanClean    ScanStatus = "clean"
	ScanInfected ScanStatus = "infected"
	ScanError    ScanStatus = "error"
	ScanSkipped  ScanStatus = "skipped"
)

// ScanResult holds the outcome of a virus scan.
type ScanResult struct {
	Status    ScanStatus    `json:"status"`
	VirusName string        `json:"virus_name,omitempty"`
	Reason    string        `json:"reason,omitempty"`
	Duration  time.Duration `json:"duration_ms"`
	FileSize  int64         `json:"file_size"`
}

// VirusScanner scans files using the ClamAV clamd INSTREAM protocol over TCP.
type VirusScanner struct {
	address    string
	timeout    time.Duration
	maxScanSize int64
}

// NewVirusScanner creates a ClamAV scanner.
func NewVirusScanner(address string, timeout time.Duration, maxScanSizeMB int) *VirusScanner {
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	maxBytes := int64(maxScanSizeMB) * 1024 * 1024
	if maxBytes <= 0 {
		maxBytes = 100 * 1024 * 1024
	}
	return &VirusScanner{
		address:    address,
		timeout:    timeout,
		maxScanSize: maxBytes,
	}
}

// Ping checks if ClamAV is reachable and responding.
func (v *VirusScanner) Ping() error {
	conn, err := net.DialTimeout("tcp", v.address, 5*time.Second)
	if err != nil {
		return fmt.Errorf("clamd: dial failed: %w", err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return fmt.Errorf("clamd: set deadline: %w", err)
	}

	// Send zPING command
	if _, err := conn.Write([]byte("zPING\x00")); err != nil {
		return fmt.Errorf("clamd: write PING: %w", err)
	}

	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		resp := strings.TrimSpace(scanner.Text())
		if resp == "PONG" {
			return nil
		}
		return fmt.Errorf("clamd: unexpected PING response: %q", resp)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("clamd: read PING response: %w", err)
	}
	return fmt.Errorf("clamd: no response to PING")
}

// Scan scans the content for viruses using the INSTREAM protocol.
// If the content exceeds maxScanSize, returns ScanSkipped.
func (v *VirusScanner) Scan(content io.Reader, fileSize int64) (*ScanResult, error) {
	start := time.Now()

	// Check size limit
	if fileSize > v.maxScanSize {
		return &ScanResult{
			Status:   ScanSkipped,
			Reason:   fmt.Sprintf("file size %d exceeds max scan size %d", fileSize, v.maxScanSize),
			Duration: time.Since(start),
			FileSize: fileSize,
		}, nil
	}

	conn, err := net.DialTimeout("tcp", v.address, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("clamd: dial failed: %w", err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(v.timeout)); err != nil {
		return nil, fmt.Errorf("clamd: set deadline: %w", err)
	}

	// Send zINSTREAM command
	if _, err := conn.Write([]byte("zINSTREAM\x00")); err != nil {
		return nil, fmt.Errorf("clamd: write INSTREAM: %w", err)
	}

	// Stream content in chunks: 4-byte big-endian length + chunk data
	buf := make([]byte, clamdChunkSize)
	var totalRead int64
	for {
		n, readErr := content.Read(buf)
		if n > 0 {
			totalRead += int64(n)
			// Write chunk length (4 bytes big-endian)
			lenBuf := make([]byte, 4)
			binary.BigEndian.PutUint32(lenBuf, uint32(n))
			if _, err := conn.Write(lenBuf); err != nil {
				return nil, fmt.Errorf("clamd: write chunk length: %w", err)
			}
			if _, err := conn.Write(buf[:n]); err != nil {
				return nil, fmt.Errorf("clamd: write chunk data: %w", err)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("clamd: reading content: %w", readErr)
		}
	}

	// Terminate stream with zero-length chunk
	terminator := make([]byte, 4)
	if _, err := conn.Write(terminator); err != nil {
		return nil, fmt.Errorf("clamd: write terminator: %w", err)
	}

	// Read response
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("clamd: read response: %w", err)
		}
		return nil, fmt.Errorf("clamd: no response received")
	}

	response := strings.TrimSpace(scanner.Text())
	result := &ScanResult{
		Duration: time.Since(start),
		FileSize: totalRead,
	}

	return parseResponse(response, result), nil
}

// parseResponse parses the ClamAV response string.
func parseResponse(response string, result *ScanResult) *ScanResult {
	switch {
	case strings.HasSuffix(response, "OK"):
		result.Status = ScanClean
	case strings.HasSuffix(response, "FOUND"):
		result.Status = ScanInfected
		// Extract virus name: "stream: VirusName FOUND"
		resp := strings.TrimPrefix(response, "stream: ")
		resp = strings.TrimSuffix(resp, " FOUND")
		result.VirusName = resp
	case strings.HasSuffix(response, "ERROR"):
		result.Status = ScanError
		result.Reason = response
	default:
		result.Status = ScanError
		result.Reason = fmt.Sprintf("unexpected response: %s", response)
	}
	return result
}
