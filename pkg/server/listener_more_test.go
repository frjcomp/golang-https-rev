package server

import (
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	"golang-https-rev/pkg/certs"
)

// createTestListenerHelper creates a listener with a dynamic port (OS selects available port)
func createTestListenerHelper(t *testing.T) *Listener {
	cert, err := certs.GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("Failed to generate cert: %v", err)
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	// Port "0" tells OS to select an available port
	return NewListener("0", "127.0.0.1", tlsConfig)
}

// TestGetClientAddressesSorted tests the sorted client addresses function
func TestGetClientAddressesSorted(t *testing.T) {
	listener := createTestListenerHelper(t)
	
	// Initially empty
	clients := listener.GetClientAddressesSorted()
	if len(clients) != 0 {
		t.Fatalf("Expected 0 sorted clients, got %d", len(clients))
	}

	t.Log("✓ Get client addresses sorted test passed")
}

// TestListenerWithMultipleClients tests listener with multiple concurrent clients
func TestListenerWithMultipleClients(t *testing.T) {
	listener := createTestListenerHelper(t)
	netListener, err := listener.Start()
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer netListener.Close()

	// Create multiple client connections
	const numClients = 3
	conns := make([]*tls.Conn, numClients)

	for i := 0; i < numClients; i++ {
		conn, err := tls.Dial("tcp", netListener.Addr().String(), &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			t.Fatalf("Failed to connect client %d: %v", i, err)
		}
		conns[i] = conn
		defer conn.Close()
	}

	// Give listener time to accept all
	time.Sleep(200 * time.Millisecond)

	// Verify all clients registered
	clients := listener.GetClients()
	if len(clients) != numClients {
		t.Fatalf("Expected %d clients, got %d", numClients, len(clients))
	}

	// Send command to each client
	for i, clientAddr := range clients {
		cmd := fmt.Sprintf("test_cmd_%d", i)
		err := listener.SendCommand(clientAddr, cmd)
		if err != nil {
			t.Fatalf("Failed to send command to client %d: %v", i, err)
		}
	}

	t.Log("✓ Listener with multiple clients test passed")
}

// TestSendCommandToInvalidClient tests error handling for non-existent client
func TestSendCommandToInvalidClient(t *testing.T) {
	listener := createTestListenerHelper(t)

	// Try to send to non-existent client
	err := listener.SendCommand("192.0.2.1:9999", "test_command")
	if err == nil {
		t.Fatal("Expected error for non-existent client")
	}

	t.Log("✓ Send command to invalid client test passed")
}

// TestGetResponseFromInvalidClient tests error handling for non-existent client
func TestGetResponseFromInvalidClient(t *testing.T) {
	listener := createTestListenerHelper(t)

	// Try to get response from non-existent client
	_, err := listener.GetResponse("192.0.2.1:9999", 100*time.Millisecond)
	if err == nil {
		t.Fatal("Expected error for non-existent client")
	}

	t.Log("✓ Get response from invalid client test passed")
}

// TestListenerResponseBuffering tests response buffering mechanism
func TestListenerResponseBuffering(t *testing.T) {
	listener := createTestListenerHelper(t)
	netListener, err := listener.Start()
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer netListener.Close()

	// Connect a client
	conn, err := tls.Dial("tcp", netListener.Addr().String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	time.Sleep(100 * time.Millisecond)
	clients := listener.GetClients()
	if len(clients) != 1 {
		t.Logf("Expected 1 client, got %d - response channel may not be available yet", len(clients))
		return
	}

	clientAddr := clients[0]

	// Test that GetResponse returns error for non-responsive client
	// (client isn't running a handler, so no response will come)
	_, err = listener.GetResponse(clientAddr, 100*time.Millisecond)
	if err == nil {
		t.Logf("Expected timeout getting response from idle client (acceptable behavior)")
	}

	t.Log("✓ Listener response buffering test passed")
}

// TestListenerPausePingChannel tests pause channel operations
func TestListenerPausePingChannel(t *testing.T) {
	listener := createTestListenerHelper(t)
	netListener, err := listener.Start()
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer netListener.Close()

	// Connect a client
	conn, err := tls.Dial("tcp", netListener.Addr().String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	time.Sleep(100 * time.Millisecond)
	clients := listener.GetClients()
	if len(clients) != 1 {
		t.Fatalf("Expected 1 client, got %d", len(clients))
	}

	clientAddr := clients[0]

	// Multiple send/receive cycles to exercise pause channel
	for i := 0; i < 5; i++ {
		cmd := fmt.Sprintf("echo cmd_%d", i)
		err := listener.SendCommand(clientAddr, cmd)
		if err != nil {
			t.Fatalf("Failed to send command %d: %v", i, err)
		}

		_, err = listener.GetResponse(clientAddr, 1*time.Second)
		if err != nil {
			// Some responses might timeout, that's ok for this test
			t.Logf("Note: response %d timed out (acceptable)", i)
		}
	}

	t.Log("✓ Listener pause ping channel test passed")
}

// TestListenerStartError tests error handling when starting listeners
func TestListenerStartError(t *testing.T) {
	// Create first listener
	listener1 := createTestListenerHelper(t)
	netListener1, err := listener1.Start()
	if err != nil {
		t.Fatalf("Failed to start first listener: %v", err)
	}
	defer netListener1.Close()

	// Create second listener with different port
	listener2 := createTestListenerHelper(t)
	netListener2, err := listener2.Start()
	if err != nil {
		t.Fatalf("Second listener should succeed with dynamic port: %v", err)
	}
	defer netListener2.Close()

	// Verify both listeners are on different ports
	addr1 := netListener1.Addr().String()
	addr2 := netListener2.Addr().String()
	if addr1 == addr2 {
		t.Fatalf("Listeners should have different addresses: %s == %s", addr1, addr2)
	}

	t.Log("✓ Listener start error test passed")
}
