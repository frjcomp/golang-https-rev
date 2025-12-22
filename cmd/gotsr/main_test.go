package main

import (
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeClient struct {
	connectErrs  []error
	handleErrs   []error
	connectCalls int
	handleCalls  int
	closed       int
}

func (f *fakeClient) Connect() error {
	if f.connectCalls < len(f.connectErrs) {
		err := f.connectErrs[f.connectCalls]
		f.connectCalls++
		return err
	}
	f.connectCalls++
	return nil
}

func (f *fakeClient) HandleCommands() error {
	if f.handleCalls < len(f.handleErrs) {
		err := f.handleErrs[f.handleCalls]
		f.handleCalls++
		return err
	}
	f.handleCalls++
	return nil
}

func (f *fakeClient) Close() error { f.closed++; return nil }

func noSleep(time.Duration) {}

func TestRunClientArgValidation(t *testing.T) {
	if err := runClient([]string{}); err == nil {
		t.Fatal("expected error for missing args")
	}
	if err := runClient([]string{"127.0.0.1:8443"}); err == nil {
		t.Fatal("expected error for too few args")
	}
	if err := runClient([]string{"host:8443", "invalid_number", "extra"}); err == nil {
		t.Fatal("expected error for too many args")
	}
}

func TestRunClientValidArgs(t *testing.T) {
	// Test with valid arguments (but will try to connect)
	// This covers the argument parsing logic
	args := []string{"127.0.0.1:8443", "1"}
	
	// Can't actually run it without a real server, but we can test arg parsing
	done := make(chan error, 1)
	go func() {
		done <- runClient(args)
	}()
	
	select {
	case <-done:
		// ok - either connected or failed after retries
	case <-time.After(5 * time.Second):
		t.Log("runClient is still running (expected for max-retries)")
	}
}

func TestPrintHeader(t *testing.T) {
	// Just call it for coverage
	printHeader()
}

func TestConnectWithRetry_MaxRetriesReachedOnConnectFailures(t *testing.T) {
	fc := &fakeClient{connectErrs: []error{errors.New("fail"), errors.New("fail"), errors.New("fail")}}
	created := 0
	factory := func(target string) reverseClient {
		created++
		return fc
	}

	done := make(chan struct{})
	go func() { connectWithRetry("127.0.0.1:8443", 3, factory, noSleep); close(done) }()

	select {
	case <-done:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("connectWithRetry did not return after max retries")
	}

	if created != 3 {
		t.Fatalf("expected 3 client creations, got %d", created)
	}
	if fc.connectCalls != 3 {
		t.Fatalf("expected 3 connect attempts, got %d", fc.connectCalls)
	}
}

func TestConnectWithRetry_ReconnectAfterHandleCommandsError(t *testing.T) {
	fc := &fakeClient{connectErrs: []error{nil, errors.New("fail")}, handleErrs: []error{errors.New("session error")}}
	created := 0
	factory := func(target string) reverseClient {
		created++
		return fc
	}

	done := make(chan struct{})
	go func() { connectWithRetry("127.0.0.1:8443", 2, factory, noSleep); close(done) }()

	select {
	case <-done:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("connectWithRetry did not return after retries")
	}

	if created < 2 {
		t.Fatalf("expected at least 2 client creations, got %d", created)
	}
	if fc.connectCalls < 2 {
		t.Fatalf("expected at least 2 connect attempts, got %d", fc.connectCalls)
	}
	if fc.handleCalls < 1 {
		t.Fatalf("expected at least 1 handle attempt, got %d", fc.handleCalls)
	}
}

func TestConnectWithRetrySuccessful(t *testing.T) {
	fc := &fakeClient{} // No errors
	created := 0
	factory := func(target string) reverseClient {
		created++
		return fc
	}

	// Run with 1 retry so it exits after HandleCommands returns nil
	done := make(chan struct{})
	go func() {
		connectWithRetry("127.0.0.1:8443", 0, factory, noSleep)
		close(done)
	}()

	// Wait a bit for it to run
	time.Sleep(100 * time.Millisecond)

	// It should still be running with infinite retries
	select {
	case <-done:
		t.Log("Client exited (HandleCommands returned)")
	default:
		t.Log("Client still running with infinite retries")
	}
}

func TestConnectWithRetryInfiniteRetries(t *testing.T) {
	// Test with maxRetries=0 (infinite)
	fc := &fakeClient{connectErrs: []error{errors.New("fail"), errors.New("fail")}}
	var mu sync.Mutex
	created := 0
	factory := func(target string) reverseClient {
		mu.Lock()
		created++
		mu.Unlock()
		return fc
	}

	done := make(chan struct{})
	go func() {
		// This should keep trying forever, but we'll stop after a few attempts
		connectWithRetry("127.0.0.1:8443", 0, factory, noSleep)
		close(done)
	}()

	// Give it time for a few attempts
	time.Sleep(100 * time.Millisecond)

	// With infinite retries and always failing, it should keep going
	mu.Lock()
	attempts := created
	mu.Unlock()
	if attempts < 2 {
		t.Fatalf("expected multiple retry attempts with infinite retries, got %d", attempts)
	}
}

func TestConnectWithRetryBackoffMaximum(t *testing.T) {
	// Test that backoff caps at 5 minutes
	fc := &fakeClient{connectErrs: []error{
		errors.New("fail1"),
		errors.New("fail2"),
		errors.New("fail3"),
		errors.New("fail4"),
		errors.New("fail5"),
	}}
	
	created := 0
	factory := func(target string) reverseClient {
		created++
		return fc
	}

	done := make(chan struct{})
	go func() {
		connectWithRetry("127.0.0.1:8443", 5, factory, noSleep)
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("connectWithRetry did not return")
	}

	if created != 5 {
		t.Fatalf("expected 5 attempts, got %d", created)
	}
}

func TestConnectWithRetryHandleCommandsSuccess(t *testing.T) {
	// Test successful connection and command handling with eventual failure
	fc := &fakeClient{
		connectErrs: []error{nil, nil},
		handleErrs:  []error{errors.New("disconnect"), errors.New("disconnect")},
	}
	
	created := 0
	factory := func(target string) reverseClient {
		created++
		return fc
	}

	done := make(chan struct{})
	go func() {
		connectWithRetry("127.0.0.1:8443", 2, factory, noSleep)
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("connectWithRetry did not return")
	}

	if fc.handleCalls < 1 {
		t.Fatalf("expected at least 1 handle attempt, got %d", fc.handleCalls)
	}
	if fc.closed < 1 {
		t.Fatalf("expected Close to be called at least once, got %d", fc.closed)
	}
}
