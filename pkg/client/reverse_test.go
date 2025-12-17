package client

import (
	"testing"
)

// TestReverseClientCreation tests creating a new reverse client
func TestReverseClientCreation(t *testing.T) {
	client := NewReverseClient("127.0.0.1:8080")
	if client == nil {
		t.Fatal("Failed to create reverse client")
	}

	if client.target != "127.0.0.1:8080" {
		t.Fatalf("Expected target 127.0.0.1:8080, got %s", client.target)
	}

	t.Log("✓ Reverse client created successfully")
}

// TestIsConnected tests the IsConnected method
func TestIsConnected(t *testing.T) {
	client := NewReverseClient("127.0.0.1:8080")
	if client.IsConnected() {
		t.Fatal("Client should not be connected initially")
	}

	t.Log("✓ IsConnected works correctly")
}

// TestExecuteCommand tests command execution
func TestExecuteCommand(t *testing.T) {
	client := NewReverseClient("127.0.0.1:8080")
	output := client.ExecuteCommand("echo hello")

	if !contains(output, "hello") {
		t.Fatalf("Expected 'hello' in output, got '%s'", output)
	}

	t.Log("✓ ExecuteCommand works")
}

// TestExecuteMultipleCommands tests executing multiple commands
func TestExecuteMultipleCommands(t *testing.T) {
	client := NewReverseClient("127.0.0.1:8080")

	commands := []string{
		"echo test1",
		"echo test2",
		"whoami",
	}

	for i, cmd := range commands {
		output := client.ExecuteCommand(cmd)
		if len(output) == 0 {
			t.Fatalf("Command %d (%s) produced empty output", i+1, cmd)
		}
		t.Logf("Command %d executed: %s", i+1, cmd)
	}

	t.Log("✓ Multiple commands execute successfully")
}

// TestExecuteCommandWithPath tests command execution with path output
func TestExecuteCommandWithPath(t *testing.T) {
	client := NewReverseClient("127.0.0.1:8080")
	output := client.ExecuteCommand("pwd")

	if len(output) == 0 {
		t.Fatal("pwd produced empty output")
	}

	if output[0] != '/' {
		t.Fatalf("Expected path starting with '/', got '%s'", output)
	}

	t.Log("✓ Path commands work correctly")
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
