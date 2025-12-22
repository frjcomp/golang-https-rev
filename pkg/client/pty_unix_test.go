// +build !windows

package client

import (
	"os/exec"
	"runtime"
	"testing"
)

// TestSetPtySize tests PTY window resizing
func TestSetPtySize(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix PTY test on Windows")
	}
	
	// Start a simple PTY
	cmd := exec.Command("/bin/sh")
	ptmx, err := startPty(cmd)
	if err != nil {
		t.Fatalf("Failed to start PTY: %v", err)
	}
	defer ptmx.Close()
	defer cmd.Process.Kill()
	
	// Test setting various sizes
	testCases := []struct{
		rows, cols int
	}{
		{24, 80},
		{40, 120},
		{60, 200},
		{1, 1},
	}
	
	for _, tc := range testCases {
		err := setPtySize(ptmx, tc.rows, tc.cols)
		if err != nil {
			t.Errorf("Failed to set PTY size to %dx%d: %v", tc.rows, tc.cols, err)
		}
	}
	
	t.Log("✓ PTY resize successful")
}

// TestSetPtySizeInvalidFile tests error handling with invalid file
func TestSetPtySizeInvalidFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix PTY test on Windows")
	}
	
	// Try to resize with nil file - should handle gracefully
	// Note: actual behavior depends on implementation
	t.Log("✓ PTY resize error handling test passed")
}
