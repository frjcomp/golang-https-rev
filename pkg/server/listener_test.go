package server

import (
	"crypto/tls"
	"testing"
	"time"

	"golang-https-rev/pkg/certs"
)

// TestListenerCreation tests creating a new listener
func TestListenerCreation(t *testing.T) {
	cert, _, err := certs.GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("Failed to generate certificate: %v", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	listener := NewListener("19000", "127.0.0.1", tlsConfig, "")
	if listener == nil {
		t.Fatal("Failed to create listener")
	}

	if listener.port != "19000" {
		t.Fatalf("Expected port 19000, got %s", listener.port)
	}

	t.Log("✓ Listener created successfully")
}

// TestGetClients tests getting client list
func TestGetClients(t *testing.T) {
	cert, _, _ := certs.GenerateSelfSignedCert()
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	listener := NewListener("19001", "127.0.0.1", tlsConfig, "")
	clients := listener.GetClients()

	if len(clients) != 0 {
		t.Fatalf("Expected 0 clients, got %d", len(clients))
	}

	t.Log("✓ GetClients works for empty list")
}

// TestSendCommand tests sending a command
func TestSendCommand(t *testing.T) {
	cert, _, _ := certs.GenerateSelfSignedCert()
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	listener := NewListener("19002", "127.0.0.1", tlsConfig, "")

	// Try sending to non-existent client
	err := listener.SendCommand("127.0.0.1:9999", "test")
	if err == nil {
		t.Fatal("Expected error for non-existent client")
	}

	t.Log("✓ SendCommand properly rejects non-existent clients")
}

// TestGetResponse tests getting a response
func TestGetResponse(t *testing.T) {
	cert, _, _ := certs.GenerateSelfSignedCert()
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	listener := NewListener("19003", "127.0.0.1", tlsConfig, "")

	// Try getting response from non-existent client
	_, err := listener.GetResponse("127.0.0.1:9999", 1*time.Second)
	if err == nil {
		t.Fatal("Expected error for non-existent client")
	}

	t.Log("✓ GetResponse properly rejects non-existent clients")
}

// TestEnterPtyMode tests entering PTY mode for a client
func TestEnterPtyMode(t *testing.T) {
	cert, _, _ := certs.GenerateSelfSignedCert()
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	listener := NewListener("19004", "127.0.0.1", tlsConfig, "")
	
	// Add a mock client
	clientAddr := "127.0.0.1:5000"
	listener.clientConnections[clientAddr] = make(chan string)
	
	// Test entering PTY mode
	ptyDataChan, err := listener.EnterPtyMode(clientAddr)
	if err != nil {
		t.Fatalf("Failed to enter PTY mode: %v", err)
	}
	
	if ptyDataChan == nil {
		t.Fatal("PTY data channel should not be nil")
	}
	
	if !listener.IsInPtyMode(clientAddr) {
		t.Error("Client should be in PTY mode")
	}
	
	t.Log("✓ Enter PTY mode successful")
}

// TestEnterPtyModeNonExistentClient tests entering PTY mode for non-existent client
func TestEnterPtyModeNonExistentClient(t *testing.T) {
	cert, _, _ := certs.GenerateSelfSignedCert()
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	listener := NewListener("19005", "127.0.0.1", tlsConfig, "")
	
	_, err := listener.EnterPtyMode("127.0.0.1:9999")
	if err == nil {
		t.Fatal("Expected error for non-existent client")
	}
	
	t.Log("✓ Non-existent client rejected")
}

// TestEnterPtyModeAlreadyInPtyMode tests entering PTY mode when already in it
func TestEnterPtyModeAlreadyInPtyMode(t *testing.T) {
	cert, _, _ := certs.GenerateSelfSignedCert()
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	listener := NewListener("19006", "127.0.0.1", tlsConfig, "")
	
	clientAddr := "127.0.0.1:5001"
	listener.clientConnections[clientAddr] = make(chan string)
	
	// Enter PTY mode first time
	_, err := listener.EnterPtyMode(clientAddr)
	if err != nil {
		t.Fatalf("First entry failed: %v", err)
	}
	
	// Try entering again
	_, err = listener.EnterPtyMode(clientAddr)
	if err == nil {
		t.Fatal("Expected error when already in PTY mode")
	}
	
	t.Log("✓ Duplicate PTY mode entry rejected")
}

// TestExitPtyMode tests exiting PTY mode
func TestExitPtyMode(t *testing.T) {
	cert, _, _ := certs.GenerateSelfSignedCert()
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	listener := NewListener("19007", "127.0.0.1", tlsConfig, "")
	
	clientAddr := "127.0.0.1:5002"
	listener.clientConnections[clientAddr] = make(chan string)
	
	// Enter PTY mode
	_, err := listener.EnterPtyMode(clientAddr)
	if err != nil {
		t.Fatalf("Failed to enter PTY mode: %v", err)
	}
	
	// Exit PTY mode
	err = listener.ExitPtyMode(clientAddr)
	if err != nil {
		t.Errorf("Failed to exit PTY mode: %v", err)
	}
	
	if listener.IsInPtyMode(clientAddr) {
		t.Error("Client should not be in PTY mode after exit")
	}
	
	t.Log("✓ Exit PTY mode successful")
}

// TestExitPtyModeNotInPtyMode tests exiting when not in PTY mode
func TestExitPtyModeNotInPtyMode(t *testing.T) {
	cert, _, _ := certs.GenerateSelfSignedCert()
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	listener := NewListener("19008", "127.0.0.1", tlsConfig, "")
	
	clientAddr := "127.0.0.1:5003"
	
	// Try exiting without entering
	err := listener.ExitPtyMode(clientAddr)
	if err != nil {
		t.Errorf("Exit without PTY mode should not error, got: %v", err)
	}
	
	t.Log("✓ Exit without PTY mode handled gracefully")
}

// TestIsInPtyMode tests checking PTY mode status
func TestIsInPtyMode(t *testing.T) {
	cert, _, _ := certs.GenerateSelfSignedCert()
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	listener := NewListener("19009", "127.0.0.1", tlsConfig, "")
	
	clientAddr := "127.0.0.1:5004"
	listener.clientConnections[clientAddr] = make(chan string)
	
	// Should not be in PTY mode initially
	if listener.IsInPtyMode(clientAddr) {
		t.Error("Client should not be in PTY mode initially")
	}
	
	// Enter PTY mode
	listener.EnterPtyMode(clientAddr)
	
	// Should be in PTY mode now
	if !listener.IsInPtyMode(clientAddr) {
		t.Error("Client should be in PTY mode")
	}
	
	t.Log("✓ IsInPtyMode works correctly")
}

// TestGetPtyDataChan tests getting PTY data channel
func TestGetPtyDataChan(t *testing.T) {
	cert, _, _ := certs.GenerateSelfSignedCert()
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	listener := NewListener("19010", "127.0.0.1", tlsConfig, "")
	
	clientAddr := "127.0.0.1:5005"
	listener.clientConnections[clientAddr] = make(chan string)
	
	// Should not exist initially
	_, exists := listener.GetPtyDataChan(clientAddr)
	if exists {
		t.Error("PTY data channel should not exist initially")
	}
	
	// Enter PTY mode
	listener.EnterPtyMode(clientAddr)
	
	// Should exist now
	ch, exists := listener.GetPtyDataChan(clientAddr)
	if !exists {
		t.Error("PTY data channel should exist after entering PTY mode")
	}
	if ch == nil {
		t.Error("PTY data channel should not be nil")
	}
	
	t.Log("✓ GetPtyDataChan works correctly")
}

