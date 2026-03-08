package security_test

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	security "github.com/clario360/platform/internal/security"
)

// mockClamdServer starts a TCP listener that emulates clamd INSTREAM protocol.
// The handler receives data and returns the configured response.
func mockClamdServer(t *testing.T, response string) (addr string, close func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock clamd: %v", err)
	}

	done := make(chan struct{})
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-done:
					return
				default:
					continue
				}
			}
			go handleClamdConn(conn, response)
		}
	}()

	return listener.Addr().String(), func() {
		close(done)
		listener.Close()
	}
}

func handleClamdConn(conn net.Conn, response string) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return
	}

	cmd := strings.TrimRight(string(buf[:n]), "\x00")

	switch {
	case cmd == "zPING":
		conn.Write([]byte("PONG\x00"))
	case cmd == "zINSTREAM":
		// Drain all chunks until we get a zero-length chunk
		for {
			// Read 4-byte length
			lenBuf := make([]byte, 4)
			_, err := conn.Read(lenBuf)
			if err != nil {
				return
			}
			chunkLen := int(lenBuf[0])<<24 | int(lenBuf[1])<<16 | int(lenBuf[2])<<8 | int(lenBuf[3])
			if chunkLen == 0 {
				break
			}
			// Drain chunk data
			remaining := chunkLen
			drain := make([]byte, 4096)
			for remaining > 0 {
				toRead := remaining
				if toRead > len(drain) {
					toRead = len(drain)
				}
				nr, err := conn.Read(drain[:toRead])
				if err != nil {
					return
				}
				remaining -= nr
			}
		}
		// Send configured response
		conn.Write([]byte(response + "\x00"))
	default:
		conn.Write([]byte(fmt.Sprintf("UNKNOWN COMMAND: %s\x00", cmd)))
	}
}

func TestClamAV_ScanCleanFile(t *testing.T) {
	addr, cleanup := mockClamdServer(t, "stream: OK")
	defer cleanup()

	cfg := &security.ClamAVConfig{
		Addr:      addr,
		Timeout:   5 * time.Second,
		MaxSize:   10 * 1024 * 1024,
		ChunkSize: 1024,
	}

	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	scanner := security.NewClamAVScanner(cfg, metrics, zerolog.Nop())

	// Give the background ping a moment
	time.Sleep(100 * time.Millisecond)

	err := scanner.Scan([]byte("clean file content"), "test.pdf")
	if err != nil {
		t.Fatalf("expected clean scan, got error: %v", err)
	}
}

func TestClamAV_ScanMalwareDetected(t *testing.T) {
	addr, cleanup := mockClamdServer(t, "stream: Win.Test.EICAR_HDB-1 FOUND")
	defer cleanup()

	cfg := &security.ClamAVConfig{
		Addr:      addr,
		Timeout:   5 * time.Second,
		MaxSize:   10 * 1024 * 1024,
		ChunkSize: 1024,
	}

	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	scanner := security.NewClamAVScanner(cfg, metrics, zerolog.Nop())
	time.Sleep(100 * time.Millisecond)

	// EICAR test string
	eicar := []byte(`X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*`)
	err := scanner.Scan(eicar, "malware.exe")
	if err == nil {
		t.Fatal("expected malware error, got nil")
	}

	if !strings.Contains(err.Error(), "malware detected") {
		t.Errorf("expected ErrMalwareDetected, got: %v", err)
	}
	if !strings.Contains(err.Error(), "EICAR") {
		t.Errorf("expected virus name in error, got: %v", err)
	}
}

func TestClamAV_ScanFileTooLarge(t *testing.T) {
	addr, cleanup := mockClamdServer(t, "stream: OK")
	defer cleanup()

	cfg := &security.ClamAVConfig{
		Addr:      addr,
		Timeout:   5 * time.Second,
		MaxSize:   100, // tiny limit
		ChunkSize: 1024,
	}

	scanner := security.NewClamAVScanner(cfg, nil, zerolog.Nop())
	time.Sleep(100 * time.Millisecond)

	data := make([]byte, 200)
	err := scanner.Scan(data, "large.bin")
	if err == nil {
		t.Fatal("expected size limit error, got nil")
	}
	if !strings.Contains(err.Error(), "file size") {
		t.Errorf("expected size limit error, got: %v", err)
	}
}

func TestClamAV_Ping(t *testing.T) {
	addr, cleanup := mockClamdServer(t, "stream: OK")
	defer cleanup()

	cfg := &security.ClamAVConfig{
		Addr:    addr,
		Timeout: 5 * time.Second,
	}

	scanner := security.NewClamAVScanner(cfg, nil, zerolog.Nop())
	time.Sleep(100 * time.Millisecond)

	if err := scanner.Ping(); err != nil {
		t.Fatalf("Ping() failed: %v", err)
	}

	if !scanner.IsAvailable() {
		t.Error("expected IsAvailable() == true after successful ping")
	}
}

func TestClamAV_PingUnreachable(t *testing.T) {
	cfg := &security.ClamAVConfig{
		Addr:    "127.0.0.1:1", // nobody listens here
		Timeout: 1 * time.Second,
	}

	scanner := security.NewClamAVScanner(cfg, nil, zerolog.Nop())
	time.Sleep(200 * time.Millisecond) // let background ping fail

	if err := scanner.Ping(); err == nil {
		t.Fatal("expected Ping() error for unreachable server")
	}

	if scanner.IsAvailable() {
		t.Error("expected IsAvailable() == false for unreachable server")
	}
}

func TestClamAV_ScanConnectionRefused(t *testing.T) {
	cfg := &security.ClamAVConfig{
		Addr:    "127.0.0.1:1",
		Timeout: 1 * time.Second,
	}

	scanner := security.NewClamAVScanner(cfg, nil, zerolog.Nop())

	err := scanner.Scan([]byte("test"), "test.txt")
	if err == nil {
		t.Fatal("expected scan error when clamd unreachable")
	}
	if !strings.Contains(err.Error(), "virus scan unavailable") {
		t.Errorf("expected 'virus scan unavailable' error, got: %v", err)
	}
}

func TestClamAV_ScanHookIntegration(t *testing.T) {
	addr, cleanup := mockClamdServer(t, "stream: OK")
	defer cleanup()

	cfg := &security.ClamAVConfig{
		Addr:      addr,
		Timeout:   5 * time.Second,
		MaxSize:   10 * 1024 * 1024,
		ChunkSize: 1024,
	}

	scanner := security.NewClamAVScanner(cfg, nil, zerolog.Nop())
	time.Sleep(100 * time.Millisecond)

	// The ScanHook() method returns a func([]byte, string) error
	// compatible with WithVirusScanHook
	hook := scanner.ScanHook()
	err := hook([]byte("clean data"), "document.pdf")
	if err != nil {
		t.Fatalf("ScanHook() returned error for clean file: %v", err)
	}
}

func TestClamAV_ClamdError(t *testing.T) {
	addr, cleanup := mockClamdServer(t, "INSTREAM size limit exceeded. ERROR")
	defer cleanup()

	cfg := &security.ClamAVConfig{
		Addr:      addr,
		Timeout:   5 * time.Second,
		MaxSize:   10 * 1024 * 1024,
		ChunkSize: 1024,
	}

	scanner := security.NewClamAVScanner(cfg, nil, zerolog.Nop())
	time.Sleep(100 * time.Millisecond)

	err := scanner.Scan([]byte("data"), "test.bin")
	if err == nil {
		t.Fatal("expected error for clamd error response")
	}
	if !strings.Contains(err.Error(), "virus scan unavailable") {
		t.Errorf("expected scan error, got: %v", err)
	}
}

func TestClamAV_LargeFileChunked(t *testing.T) {
	addr, cleanup := mockClamdServer(t, "stream: OK")
	defer cleanup()

	cfg := &security.ClamAVConfig{
		Addr:      addr,
		Timeout:   10 * time.Second,
		MaxSize:   5 * 1024 * 1024,
		ChunkSize: 512, // small chunks to test chunking
	}

	scanner := security.NewClamAVScanner(cfg, nil, zerolog.Nop())
	time.Sleep(100 * time.Millisecond)

	// Create a 10 KB file to verify multi-chunk transfer
	data := make([]byte, 10*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	err := scanner.Scan(data, "largefile.bin")
	if err != nil {
		t.Fatalf("expected clean scan of chunked file, got: %v", err)
	}
}

func TestClamAV_HealthCheck(t *testing.T) {
	addr, cleanup := mockClamdServer(t, "stream: OK")
	defer cleanup()

	cfg := &security.ClamAVConfig{
		Addr:    addr,
		Timeout: 5 * time.Second,
	}

	scanner := security.NewClamAVScanner(cfg, nil, zerolog.Nop())
	time.Sleep(100 * time.Millisecond)

	healthFn := scanner.ClamAVHealthCheck()
	if err := healthFn(); err != nil {
		t.Fatalf("health check failed: %v", err)
	}
}
