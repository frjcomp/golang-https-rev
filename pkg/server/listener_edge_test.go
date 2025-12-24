package server

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/frjcomp/gots/pkg/certs"
	"github.com/frjcomp/gots/pkg/protocol"
)

// TestAuthenticationNoAuthCommandSent tests when client doesn't send AUTH
func TestAuthenticationNoAuthCommandSent(t *testing.T) {
	cert, _, err := certs.GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("Failed to generate cert: %v", err)
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}

	// Create listener with shared secret
	listener := NewListener("0", "127.0.0.1", tlsConfig, "test-secret-key")
	netListener, err := listener.Start()
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer netListener.Close()

	// Connect client but send wrong command
	conn, err := tls.Dial("tcp", netListener.Addr().String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send wrong command instead of AUTH
	writer := bufio.NewWriter(conn)
	writer.WriteString("PING\n")
	writer.Flush()

	// Expect AUTH_FAILED response
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if !strings.Contains(response, protocol.CmdAuthFailed) {
		t.Errorf("Expected AUTH_FAILED, got: %s", response)
	}

	t.Log("✓ Authentication no AUTH command test passed")
}

// TestAuthenticationWrongSecret tests authentication with wrong secret
func TestAuthenticationWrongSecret(t *testing.T) {
	cert, _, err := certs.GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("Failed to generate cert: %v", err)
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}

	listener := NewListener("0", "127.0.0.1", tlsConfig, "correct-secret")
	netListener, err := listener.Start()
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer netListener.Close()

	conn, err := tls.Dial("tcp", netListener.Addr().String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send wrong secret
	writer := bufio.NewWriter(conn)
	writer.WriteString(protocol.CmdAuth + " wrong-secret\n")
	writer.Flush()

	// Expect AUTH_FAILED
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if !strings.Contains(response, protocol.CmdAuthFailed) {
		t.Errorf("Expected AUTH_FAILED, got: %s", response)
	}

	// Client should be disconnected, verify not in connection list
	time.Sleep(100 * time.Millisecond)
	clients := listener.GetClients()
	if len(clients) > 0 {
		t.Errorf("Expected 0 clients after auth failure, got %d", len(clients))
	}

	t.Log("✓ Authentication wrong secret test passed")
}

// TestAuthenticationReadError tests auth failure when read fails
func TestAuthenticationReadError(t *testing.T) {
	cert, _, err := certs.GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("Failed to generate cert: %v", err)
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}

	listener := NewListener("0", "127.0.0.1", tlsConfig, "test-secret")
	netListener, err := listener.Start()
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer netListener.Close()

	// Connect and immediately close to cause read error
	conn, err := tls.Dial("tcp", netListener.Addr().String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	conn.Close() // Close immediately to cause read error

	// Give time for listener to process
	time.Sleep(100 * time.Millisecond)

	// Should have no clients
	clients := listener.GetClients()
	if len(clients) > 0 {
		t.Errorf("Expected 0 clients after read error, got %d", len(clients))
	}

	t.Log("✓ Authentication read error test passed")
}

// TestResponseBufferExceedsMax tests response buffer overflow handling
func TestResponseBufferExceedsMax(t *testing.T) {
	cert, _, err := certs.GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("Failed to generate cert: %v", err)
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}

	listener := NewListener("0", "127.0.0.1", tlsConfig, "")
	netListener, err := listener.Start()
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer netListener.Close()

	conn, err := tls.Dial("tcp", netListener.Addr().String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	time.Sleep(100 * time.Millisecond)

	clients := listener.GetClients()
	if len(clients) == 0 {
		t.Fatal("Expected at least 1 client")
	}

	// Send a very large response without newline to trigger buffer reset
	writer := bufio.NewWriter(conn)
	largeData := strings.Repeat("A", protocol.MaxBufferSize+1000)
	writer.WriteString(largeData)
	writer.Flush()

	// Give time to process
	time.Sleep(200 * time.Millisecond)

	// Client should still be connected (buffer resets, doesn't disconnect)
	clients = listener.GetClients()
	if len(clients) == 0 {
		t.Error("Client should still be connected after buffer reset")
	}

	t.Log("✓ Response buffer exceeds max test passed")
}

// TestClientDisconnectCleanup tests proper cleanup when client disconnects
func TestClientDisconnectCleanup(t *testing.T) {
	cert, _, err := certs.GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("Failed to generate cert: %v", err)
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}

	listener := NewListener("0", "127.0.0.1", tlsConfig, "")
	netListener, err := listener.Start()
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer netListener.Close()

	conn, err := tls.Dial("tcp", netListener.Addr().String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	clients := listener.GetClients()
	if len(clients) != 1 {
		t.Fatalf("Expected 1 client, got %d", len(clients))
	}

	clientAddr := clients[0]

	// Enter PTY mode to ensure cleanup handles it
	_, err = listener.EnterPtyMode(clientAddr)
	if err != nil {
		t.Fatalf("Failed to enter PTY mode: %v", err)
	}

	// Close connection
	conn.Close()

	// Give time for cleanup
	time.Sleep(200 * time.Millisecond)

	// Verify client removed from all maps
	clients = listener.GetClients()
	if len(clients) != 0 {
		t.Errorf("Expected 0 clients after disconnect, got %d", len(clients))
	}

	// Verify PTY data cleaned up
	listener.mutex.Lock()
	_, ptyExists := listener.clientPtyData[clientAddr]
	_, modeExists := listener.clientPtyMode[clientAddr]
	listener.mutex.Unlock()

	if ptyExists {
		t.Error("PTY data channel should be cleaned up")
	}
	if modeExists {
		t.Error("PTY mode flag should be cleaned up")
	}

	t.Log("✓ Client disconnect cleanup test passed")
}

// TestPingPauseDuringCommand tests ping pause mechanism
func TestPingPauseDuringCommand(t *testing.T) {
	cert, _, err := certs.GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("Failed to generate cert: %v", err)
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}

	listener := NewListener("0", "127.0.0.1", tlsConfig, "")
	netListener, err := listener.Start()
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer netListener.Close()

	conn, err := tls.Dial("tcp", netListener.Addr().String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	time.Sleep(100 * time.Millisecond)

	clients := listener.GetClients()
	if len(clients) == 0 {
		t.Fatal("Expected at least 1 client")
	}
	clientAddr := clients[0]

	// Verify pause ping channel exists
	listener.mutex.Lock()
	pauseChan, exists := listener.clientPausePing[clientAddr]
	listener.mutex.Unlock()

	if !exists {
		t.Fatal("Pause ping channel should exist")
	}

	// Send pause signal
	select {
	case pauseChan <- true:
		t.Log("✓ Pause signal sent successfully")
	case <-time.After(100 * time.Millisecond):
		t.Error("Failed to send pause signal")
	}

	t.Log("✓ Ping pause during command test passed")
}

// TestAcceptConnectionsError tests error handling in accept loop
func TestAcceptConnectionsError(t *testing.T) {
	cert, _, err := certs.GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("Failed to generate cert: %v", err)
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}

	listener := NewListener("0", "127.0.0.1", tlsConfig, "")
	netListener, err := listener.Start()
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}

	// Close listener immediately to trigger accept error
	netListener.Close()

	// Give time for error to be logged
	time.Sleep(100 * time.Millisecond)

	// Should handle gracefully (no panic)
	t.Log("✓ Accept connections error test passed")
}

// TestSendCommandChannelFull tests handling when command channel is full
func TestSendCommandChannelFull(t *testing.T) {
	cert, _, err := certs.GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("Failed to generate cert: %v", err)
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}

	listener := NewListener("0", "127.0.0.1", tlsConfig, "")
	netListener, err := listener.Start()
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer netListener.Close()

	conn, err := tls.Dial("tcp", netListener.Addr().String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	time.Sleep(100 * time.Millisecond)

	clients := listener.GetClients()
	if len(clients) == 0 {
		t.Fatal("Expected at least 1 client")
	}
	clientAddr := clients[0]

	// Try to fill the command channel (capacity is 10)
	for i := 0; i < 15; i++ {
		err := listener.SendCommand(clientAddr, fmt.Sprintf("cmd_%d", i))
		// Should handle gracefully even if channel fills
		if err != nil && i < 10 {
			t.Errorf("Unexpected error on command %d: %v", i, err)
		}
	}

	t.Log("✓ Send command channel full test passed")
}

// TestMultipleCommandsRapidFire tests sending many commands quickly
func TestMultipleCommandsRapidFire(t *testing.T) {
	cert, _, err := certs.GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("Failed to generate cert: %v", err)
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}

	listener := NewListener("0", "127.0.0.1", tlsConfig, "")
	netListener, err := listener.Start()
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer netListener.Close()

	conn, err := tls.Dial("tcp", netListener.Addr().String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	time.Sleep(100 * time.Millisecond)

	clients := listener.GetClients()
	if len(clients) == 0 {
		t.Fatal("Expected at least 1 client")
	}
	clientAddr := clients[0]

	// Send multiple commands rapidly
	const numCommands = 5
	for i := 0; i < numCommands; i++ {
		err := listener.SendCommand(clientAddr, fmt.Sprintf("rapid_cmd_%d", i))
		if err != nil {
			t.Errorf("Failed to send command %d: %v", i, err)
		}
	}

	// Read commands from connection
	reader := bufio.NewReader(conn)
	for i := 0; i < numCommands; i++ {
		cmd, err := reader.ReadString('\n')
		if err != nil {
			t.Errorf("Failed to read command %d: %v", i, err)
			break
		}
		if !strings.Contains(cmd, fmt.Sprintf("rapid_cmd_%d", i)) {
			t.Logf("Command %d: %s", i, cmd)
		}
	}

	t.Log("✓ Multiple commands rapid fire test passed")
}

// TestGetResponseTimeoutEdgeCase tests response timeout handling
func TestGetResponseTimeoutEdgeCase(t *testing.T) {
	cert, _, err := certs.GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("Failed to generate cert: %v", err)
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}

	listener := NewListener("0", "127.0.0.1", tlsConfig, "")
	netListener, err := listener.Start()
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer netListener.Close()

	conn, err := tls.Dial("tcp", netListener.Addr().String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	time.Sleep(100 * time.Millisecond)

	clients := listener.GetClients()
	if len(clients) == 0 {
		t.Fatal("Expected at least 1 client")
	}
	clientAddr := clients[0]

	// Try to get response without sending command (should timeout)
	start := time.Now()
	_, err = listener.GetResponse(clientAddr, 200*time.Millisecond)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Expected timeout error")
	}

	if elapsed < 150*time.Millisecond {
		t.Errorf("Timeout occurred too quickly: %v", elapsed)
	}

	t.Log("✓ Get response timeout test passed")
}
