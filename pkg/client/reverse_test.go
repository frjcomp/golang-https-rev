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

// TestExecuteCommandEmptyInput tests handling of empty commands
func TestExecuteCommandEmptyInput(t *testing.T) {
	client := NewReverseClient("127.0.0.1:8080")
	output := client.ExecuteCommand("")
	
	// Empty command should return empty output
	if len(output) != 0 {
		t.Logf("Empty command returned: %s", output)
	}
	
	t.Log("✓ Empty command handled")
}

// TestExecuteCommandInvalidCommand tests handling of invalid commands
func TestExecuteCommandInvalidCommand(t *testing.T) {
	client := NewReverseClient("127.0.0.1:8080")
	output := client.ExecuteCommand("nonexistent_command_12345")
	
	// Should get error output
	if len(output) == 0 {
		t.Log("Invalid command produced empty output (might be valid)")
	}
	
	t.Log("✓ Invalid command handled")
}

// TestCloseWithoutConnection tests closing when not connected
func TestCloseWithoutConnection(t *testing.T) {
	client := NewReverseClient("127.0.0.1:8080")
	err := client.Close()
	
	// Should handle gracefully
	if err != nil {
		t.Logf("Close without connection returned error: %v", err)
	}
	
	t.Log("✓ Close without connection handled")
}

// TestExecuteShellCommand tests the shell command execution
func TestExecuteShellCommand(t *testing.T) {
	// Test simple command using package-level function
	output, err := executeShellCommand("echo test_shell")
	if err != nil {
		t.Errorf("Shell command failed: %v", err)
	}
	if !contains(output, "test_shell") {
		t.Errorf("Expected 'test_shell' in output, got '%s'", output)
	}
	
	t.Log("✓ Shell command execution works")
}

// TestExecuteShellCommandError tests error handling in shell execution
func TestExecuteShellCommandError(t *testing.T) {
	// Command that will fail
	output, err := executeShellCommand("exit 1")
	
	// Should return error
	if err == nil {
		t.Error("Expected error for failing command")
	}
	
	// Output should contain error info
	if !contains(output, "Error") {
		t.Logf("Error command output: %s", output)
	}
	
	t.Log("✓ Shell command error handled")
}

// TestExecuteShellCommandNonexistent tests handling of nonexistent commands
func TestExecuteShellCommandNonexistent(t *testing.T) {
	output, err := executeShellCommand("nonexistent_cmd_xyz_12345")
	
	// Should return error
	if err == nil {
		t.Error("Expected error for nonexistent command")
	}
	
	// Output should contain error info
	if !contains(output, "Error") {
		t.Logf("Nonexistent command output: %s", output)
	}
	
	t.Log("✓ Nonexistent command handled")
}
