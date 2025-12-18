package main

import (
	"bytes"
	"errors"
	"os"
	"testing"
	"time"

	"golang-https-rev/pkg/compression"
	"golang-https-rev/pkg/protocol"
)

// TestCompressDecompressRoundTrip verifies that data can be compressed to hex and decompressed back identically
func TestCompressDecompressRoundTrip(t *testing.T) {
	testData := []byte("Hello, this is test data for compression! " + string(bytes.Repeat([]byte("x"), 1000)))

	// Compress
	compressed, err := compression.CompressToHex(testData)
	if err != nil {
		t.Fatalf("CompressToHex failed: %v", err)
	}

	// Verify compressed is not empty
	if compressed == "" {
		t.Fatal("compressed hex should not be empty")
	}

	// Decompress
	decompressed, err := compression.DecompressHex(compressed)
	if err != nil {
		t.Fatalf("DecompressHex failed: %v", err)
	}

	// Verify round-trip
	if !bytes.Equal(decompressed, testData) {
		t.Fatalf("decompressed data does not match original: got %d bytes, expected %d bytes", len(decompressed), len(testData))
	}
}

// TestCompressEmptyData handles edge case of compressing empty data
func TestCompressEmptyData(t *testing.T) {
	testData := []byte{}

	compressed, err := compression.CompressToHex(testData)
	if err != nil {
		t.Fatalf("CompressToHex failed on empty data: %v", err)
	}

	decompressed, err := compression.DecompressHex(compressed)
	if err != nil {
		t.Fatalf("DecompressHex failed on empty data: %v", err)
	}

	if !bytes.Equal(decompressed, testData) {
		t.Fatal("decompressed empty data should match original")
	}
}

// TestCompressLargeData ensures compression works with large payloads
func TestCompressLargeData(t *testing.T) {
	// Create 5MB of repetitive data
	testData := bytes.Repeat([]byte("large data payload "), 262144)

	compressed, err := compression.CompressToHex(testData)
	if err != nil {
		t.Fatalf("CompressToHex failed on large data: %v", err)
	}

	decompressed, err := compression.DecompressHex(compressed)
	if err != nil {
		t.Fatalf("DecompressHex failed on large data: %v", err)
	}

	if !bytes.Equal(decompressed, testData) {
		t.Fatalf("large data round-trip failed: got %d bytes, expected %d bytes", len(decompressed), len(testData))
	}
}

// TestDecompressInvalidHex verifies that invalid hex input is handled gracefully
func TestDecompressInvalidHex(t *testing.T) {
	_, err := compression.DecompressHex("invalid!@#$%hex")
	if err == nil {
		t.Fatal("DecompressHex should return error for invalid hex input")
	}
}

// TestDecompressCorruptedGzip verifies that corrupted gzip data is detected
func TestDecompressCorruptedGzip(t *testing.T) {
	// Create valid hex that doesn't contain valid gzip data
	invalidGzip := "deadbeef"

	_, err := compression.DecompressHex(invalidGzip)
	if err == nil {
		t.Fatal("DecompressHex should return error for corrupted gzip data")
	}
}

func TestRunListenerArgValidation(t *testing.T) {
	if err := runListener([]string{}); err == nil {
		t.Fatal("expected error for missing args")
	}
	if err := runListener([]string{"8443"}); err == nil {
		t.Fatal("expected error for too few args")
	}
}

func TestListClientsEmpty(t *testing.T) {
	ml := &mockListener{clients: []string{}}
	listClients(ml)
}

func TestListClientsMultiple(t *testing.T) {
	ml := &mockListener{clients: []string{"192.168.1.2:1234", "10.0.0.5:5678"}}
	listClients(ml)
}

func TestUseClientValid(t *testing.T) {
	ml := &mockListener{clients: []string{"192.168.1.2:1234", "10.0.0.5:5678"}}
	result := useClient(ml, []string{"use", "1"})
	if result != "192.168.1.2:1234" {
		t.Fatalf("expected first client, got %s", result)
	}
}

func TestUseClientInvalidID(t *testing.T) {
	ml := &mockListener{clients: []string{"192.168.1.2:1234"}}
	result := useClient(ml, []string{"use", "5"})
	if result != "" {
		t.Fatalf("expected empty for out-of-range ID, got %s", result)
	}
}

func TestUseClientNonNumericID(t *testing.T) {
	ml := &mockListener{clients: []string{"192.168.1.2:1234"}}
	result := useClient(ml, []string{"use", "abc"})
	if result != "" {
		t.Fatalf("expected empty for non-numeric ID, got %s", result)
	}
}

func TestUseClientMissingArg(t *testing.T) {
	ml := &mockListener{clients: []string{"192.168.1.2:1234"}}
	result := useClient(ml, []string{"use"})
	if result != "" {
		t.Fatalf("expected empty when missing arg, got %s", result)
	}
}

type mockListener struct {
	clients       []string
	sentCommands  []string
	responses     []string
	responseIdx   int
	sendErr       error
	sendErrs      []error // Multiple send errors for different calls
	getErr        error
}

func (m *mockListener) GetClients() []string {
	return m.clients
}

func (m *mockListener) SendCommand(client, cmd string) error {
	// Use sendErrs if available for per-call errors
	if len(m.sendErrs) > 0 {
		callNum := len(m.sentCommands)
		if callNum < len(m.sendErrs) && m.sendErrs[callNum] != nil {
			return m.sendErrs[callNum]
		}
	}
	if m.sendErr != nil {
		return m.sendErr
	}
	m.sentCommands = append(m.sentCommands, cmd)
	return nil
}

func (m *mockListener) GetResponse(client string, timeout time.Duration) (string, error) {
	if m.getErr != nil {
		return "", m.getErr
	}
	if m.responseIdx < len(m.responses) {
		resp := m.responses[m.responseIdx]
		m.responseIdx++
		return resp, nil
	}
	return "", nil
}

func TestSendShellCommandSuccess(t *testing.T) {
	ml := &mockListener{responses: []string{"output" + protocol.EndOfOutputMarker}}
	if !sendShellCommand(ml, "192.168.1.2:1234", "ls") {
		t.Fatal("expected success")
	}
	if len(ml.sentCommands) != 1 || ml.sentCommands[0] != "ls" {
		t.Fatalf("unexpected commands: %v", ml.sentCommands)
	}
}

func TestSendShellCommandSendError(t *testing.T) {
	ml := &mockListener{sendErr: bytes.ErrTooLarge}
	if sendShellCommand(ml, "192.168.1.2:1234", "ls") {
		t.Fatal("expected failure when send fails")
	}
}

func TestSendShellCommandGetError(t *testing.T) {
	ml := &mockListener{getErr: bytes.ErrTooLarge}
	if sendShellCommand(ml, "192.168.1.2:1234", "ls") {
		t.Fatal("expected failure when get response fails")
	}
}

func TestPrintHelp(t *testing.T) {
	// Just call it to increase coverage - it only prints output
	printHelp()
}

func TestPrintHeader(t *testing.T) {
	// Call it to increase coverage - it only prints output
	printHeader()
}

func TestHandleUploadGetResponseError(t *testing.T) {
	// Create a temp file
	tmpfile := t.TempDir() + "/test.txt"
	if err := os.WriteFile(tmpfile, []byte("test data"), 0644); err != nil {
		t.Fatal(err)
	}

	ml := &mockListener{
		getErr: bytes.ErrTooLarge,
	}
	result := handleUpload(ml, "192.168.1.2:1234", []string{"upload", tmpfile, "/remote/path.txt"})
	if result {
		t.Fatal("expected false when get response fails")
	}
}

func TestHandleDownloadGetResponseError(t *testing.T) {
	ml := &mockListener{getErr: bytes.ErrTooLarge}
	tmpfile := t.TempDir() + "/out.txt"
	result := handleDownload(ml, "192.168.1.2:1234", []string{"download", "/remote/file.txt", tmpfile})
	if result {
		t.Fatal("expected false when get response fails")
	}
}

func TestHandleDownloadInvalidResponse(t *testing.T) {
	ml := &mockListener{
		responses: []string{"INVALID_RESPONSE\n" + protocol.EndOfOutputMarker},
	}
	tmpfile := t.TempDir() + "/out.txt"
	result := handleDownload(ml, "192.168.1.2:1234", []string{"download", "/remote/file.txt", tmpfile})
	if !result {
		t.Fatal("expected true - error handled but connection maintained")
	}
}

func TestHandleDownloadInvalidHex(t *testing.T) {
	ml := &mockListener{
		responses: []string{protocol.DataPrefix + "INVALID_HEX!@#\n" + protocol.EndOfOutputMarker},
	}
	tmpfile := t.TempDir() + "/out.txt"
	result := handleDownload(ml, "192.168.1.2:1234", []string{"download", "/remote/file.txt", tmpfile})
	if !result {
		t.Fatal("expected true - error handled but connection maintained")
	}
}

func TestHandleDownloadWriteError(t *testing.T) {
	// Create a valid compressed hex payload
	testData := []byte("test data")
	compressed, err := compression.CompressToHex(testData)
	if err != nil {
		t.Fatal(err)
	}

	ml := &mockListener{
		responses: []string{protocol.DataPrefix + compressed + "\n" + protocol.EndOfOutputMarker},
	}

	// Try to write to an invalid path (directory doesn't exist)
	result := handleDownload(ml, "192.168.1.2:1234", []string{"download", "/remote/file.txt", "/nonexistent/dir/file.txt"})
	if !result {
		t.Fatal("expected true - write error handled but connection maintained")
	}
}

func TestHandleUploadBadStartResponse(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "upload-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("test content"))
	tmpFile.Close()

	ml := &mockListener{
		responses: []string{"ERROR" + protocol.EndOfOutputMarker},
	}

	result := handleUpload(ml, "client1", []string{"upload", tmpFile.Name(), "/remote/path.txt"})
	if result {
		t.Fatal("expected false when start response is not OK")
	}
}

func TestHandleUploadChunkSendError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "upload-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	// Write data larger than chunk size to trigger chunk loop
	tmpFile.Write(bytes.Repeat([]byte("x"), protocol.ChunkSize+1000))
	tmpFile.Close()

	ml := &mockListener{
		responses: []string{"OK" + protocol.EndOfOutputMarker},
		sendErrs:  []error{nil, errors.New("chunk send failed")}, // First OK, second fails
	}

	result := handleUpload(ml, "client1", []string{"upload", tmpFile.Name(), "/remote/path.txt"})
	if result {
		t.Fatal("expected false when chunk send fails")
	}
}

func TestHandleUploadChunkResponseError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "upload-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("test"))
	tmpFile.Close()

	ml := &mockListener{
		responses: []string{
			"OK" + protocol.EndOfOutputMarker,       // Start upload OK
			"CHUNK_ERROR" + protocol.EndOfOutputMarker, // Chunk response not OK
		},
	}

	result := handleUpload(ml, "client1", []string{"upload", tmpFile.Name(), "/remote/path.txt"})
	if result {
		t.Fatal("expected false when chunk response is not OK")
	}
}

func TestHandleUploadEndUploadError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "upload-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("test"))
	tmpFile.Close()

	ml := &mockListener{
		responses: []string{
			"OK" + protocol.EndOfOutputMarker, // Start upload OK
			"OK" + protocol.EndOfOutputMarker, // Chunk OK
		},
		sendErrs: []error{nil, nil, errors.New("end upload send failed")}, // First 2 OK, third fails
	}

	result := handleUpload(ml, "client1", []string{"upload", tmpFile.Name(), "/remote/path.txt"})
	if result {
		t.Fatal("expected false when end upload send fails")
	}
}
