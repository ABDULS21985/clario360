package security_test

import (
	"bufio"
	"encoding/binary"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	security "github.com/clario360/platform/internal/security"
)

// mockClamdServer starts a TCP listener that emulates clamd INSTREAM protocol.
func mockClamdServer(t *testing.T, response string) (addr string, cleanup func()) {
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

// readNullTerminated reads bytes until a null byte is found, returning the command string.
func readNullTerminated(r *bufio.Reader) (string, error) {
	var cmd []byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		if b == 0 {
			break
		}
		cmd = append(cmd, b)
	}
	return string(cmd), nil
}

func handleClamdConn(conn net.Conn, response string) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	reader := bufio.NewReader(conn)

	// Read null-terminated command
	cmd, err := readNullTerminated(reader)
	if err != nil {
		return
	}

	switch cmd {
	case "zPING":
		conn.Write([]byte("PONG\x00"))

	case "zINSTREAM":
		// Drain all chunks: each is [4-byte big-endian length][data]
		// Stream ends with 4 zero bytes (length == 0)
		for {
			var chunkLen uint32
			if err := binary.Read(reader, binary.BigEndian, &chunkLen); err != nil {
				return
			}
			if chunkLen == 0 {
				break
			}
			// Drain chunk data
			if _, err := io.CopyN(io.Discard, reader, int64(chunkLen)); err != nil {
				return
			}
		}
		// Send configured response
		conn.Write([]byte(response + "\x00"))

	default:
		conn.Write([]byte("UNKNOWN COMMAND\x00"))
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
		Addr:    "127.0.0.1:1",
		Timeout: 1 * time.Second,
	}

	scanner := security.NewClamAVScanner(cfg, nil, zerolog.Nop())
	time.Sleep(200 * time.Millisecond)

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
