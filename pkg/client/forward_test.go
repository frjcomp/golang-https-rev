package client

import (
	"testing"
)

func TestForwardHandler_New(t *testing.T) {
	sendFunc := func(msg string) {}
	fh := NewForwardHandler(sendFunc)
	
	if fh == nil {
		t.Fatal("NewForwardHandler returned nil")
	}
	
	if fh.connections == nil {
		t.Error("connections map not initialized")
	}
}

func TestForwardHandler_HandleForwardStop(t *testing.T) {
	sendFunc := func(msg string) {}
	fh := NewForwardHandler(sendFunc)
	
	// Should not panic even if forward doesn't exist
	fh.HandleForwardStop("nonexistent")
}

func TestForwardHandler_Close(t *testing.T) {
	sendFunc := func(msg string) {}
	fh := NewForwardHandler(sendFunc)
	
	// Should not panic on close
	fh.Close()
}
