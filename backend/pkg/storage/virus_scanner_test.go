package storage

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestScan_OversizedFile_Skipped(t *testing.T) {
	// Create a scanner with maxScanSize of 1 MB
	scanner := NewVirusScanner("127.0.0.1:3310", 10*time.Second, 1)

	// Report a file size of 2 MB (exceeds 1 MB max)
	content := bytes.NewReader(make([]byte, 100))
	result, err := scanner.Scan(content, 2*1024*1024)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.Status != ScanSkipped {
		t.Fatalf("expected ScanSkipped, got %s", result.Status)
	}
	if result.FileSize != 2*1024*1024 {
		t.Fatalf("expected file size 2MB, got %d", result.FileSize)
	}
	if result.Reason == "" {
		t.Fatal("expected non-empty reason for skipped scan")
	}
}

// EICAR test string - standard test pattern for antivirus testing
const eicarTestString = `X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*`

// startTestClamdServer starts a real TCP server that speaks the clamd INSTREAM protocol.
// It accepts "zINSTREAM\x00", reads chunks, and responds based on content:
// - If the content contains the EICAR test string, responds with "stream: Eicar-Signature FOUND\n"
// - Otherwise, responds with "stream: OK\n"
func startTestClamdServer(t *testing.T) (addr string, stop func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test clamd server: %v", err)
	}

	done := make(chan struct{})

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-done:
					return
				default:
					return
				}
			}
			go handleClamdConnection(conn)
		}
	}()

	return ln.Addr().String(), func() {
		close(done)
		ln.Close()
	}
}

func handleClamdConnection(conn net.Conn) {
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(10 * time.Second))

	// Read the command (null-terminated)
	cmdBuf := make([]byte, 256)
	n, err := conn.Read(cmdBuf)
	if err != nil {
		return
	}

	cmd := string(cmdBuf[:n])

	// Check for zINSTREAM command
	if !strings.HasPrefix(cmd, "zINSTREAM\x00") {
		fmt.Fprintf(conn, "UNKNOWN COMMAND\n")
		return
	}

	// After the command there might be data already in cmdBuf if
	// the client sent it in the same write. We need to handle the
	// stream protocol: 4-byte big-endian length + data, terminated
	// by a zero-length chunk.

	// We'll re-read from connection for chunk protocol
	// First, check if there's leftover data after the command
	leftover := cmdBuf[len("zINSTREAM\x00"):n]

	var allData []byte
	buf := bytes.NewBuffer(leftover)

	// readExactly reads exactly n bytes from the buffer or connection
	readExactly := func(need int) ([]byte, error) {
		for buf.Len() < need {
			tmp := make([]byte, 4096)
			nr, err := conn.Read(tmp)
			if nr > 0 {
				buf.Write(tmp[:nr])
			}
			if err != nil {
				return nil, err
			}
		}
		result := make([]byte, need)
		_, err := io.ReadFull(buf, result)
		return result, err
	}

	for {
		// Read 4-byte chunk length
		lenBytes, err := readExactly(4)
		if err != nil {
			return
		}
		chunkLen := binary.BigEndian.Uint32(lenBytes)

		// Zero-length chunk means end of stream
		if chunkLen == 0 {
			break
		}

		// Read chunk data
		chunkData, err := readExactly(int(chunkLen))
		if err != nil {
			return
		}
		allData = append(allData, chunkData...)
	}

	// Check if data contains EICAR test string
	if strings.Contains(string(allData), "EICAR-STANDARD-ANTIVIRUS-TEST-FILE") {
		fmt.Fprintf(conn, "stream: Eicar-Signature FOUND\n")
	} else {
		fmt.Fprintf(conn, "stream: OK\n")
	}
}

func TestScan_ClamdProtocol(t *testing.T) {
	addr, stop := startTestClamdServer(t)
	defer stop()

	scanner := NewVirusScanner(addr, 10*time.Second, 100)

	t.Run("clean file", func(t *testing.T) {
		content := bytes.NewReader([]byte("This is a clean file with no malware"))
		result, err := scanner.Scan(content, 36)
		if err != nil {
			t.Fatalf("Scan clean file: %v", err)
		}
		if result.Status != ScanClean {
			t.Fatalf("expected ScanClean, got %s (reason: %s, virus: %s)", result.Status, result.Reason, result.VirusName)
		}
	})

	t.Run("infected file (EICAR)", func(t *testing.T) {
		content := bytes.NewReader([]byte(eicarTestString))
		result, err := scanner.Scan(content, int64(len(eicarTestString)))
		if err != nil {
			t.Fatalf("Scan EICAR: %v", err)
		}
		if result.Status != ScanInfected {
			t.Fatalf("expected ScanInfected, got %s", result.Status)
		}
		if result.VirusName != "Eicar-Signature" {
			t.Fatalf("expected virus name 'Eicar-Signature', got %q", result.VirusName)
		}
	})

	t.Run("scan records duration", func(t *testing.T) {
		content := bytes.NewReader([]byte("timing test"))
		result, err := scanner.Scan(content, 11)
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		if result.Duration <= 0 {
			t.Fatal("expected positive Duration")
		}
	})
}
