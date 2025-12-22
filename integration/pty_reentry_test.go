package main

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestPtyReentry specifically tests that immediately re-entering the PTY after exit works.
func TestPtyReentry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	port := freePort(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	listenerBin := buildBinary(t, "gotsl", "./cmd/gotsl")
	reverseBin := buildBinary(t, "gotsr", "./cmd/gotsr")

	listener := startProcess(ctx, t, listenerBin, port, "127.0.0.1")
	t.Cleanup(listener.stop)
	waitForContains(t, listener, "Listener ready. Waiting for connections", 10*time.Second)

	reverse := startProcess(ctx, t, reverseBin, fmt.Sprintf("127.0.0.1:%s", port), "1")
	t.Cleanup(reverse.stop)
	waitForContains(t, reverse, "Connected to listener successfully", 10*time.Second)

	send(listener, "shell 1\n")
	waitForContains(t, listener, "PTY shell active", 5*time.Second)
	send(listener, "\x04") // Ctrl-D
	waitForContains(t, listener, "[Remote shell exited]", 5*time.Second)

	// Immediately try to re-enter the PTY
	send(listener, "shell 1\n")
	waitForContains(t, listener, "PTY shell active", 5*time.Second)
}
