package server

import (
	"testing"
)

func TestForwardManager_StartForward(t *testing.T) {
	fm := NewForwardManager()
	
	sendCalls := []string{}
	sendFunc := func(msg string) {
		sendCalls = append(sendCalls, msg)
	}
	
	err := fm.StartForward("test1", "0", "example.com:80", sendFunc)
	if err != nil {
		t.Fatalf("StartForward failed: %v", err)
	}
	
	forwards := fm.ListForwards()
	if len(forwards) != 1 {
		t.Errorf("Expected 1 forward, got %d", len(forwards))
	}
	
	if forwards[0].ID != "test1" {
		t.Errorf("Expected ID 'test1', got %s", forwards[0].ID)
	}
	
	if forwards[0].RemoteAddr != "example.com:80" {
		t.Errorf("Expected RemoteAddr 'example.com:80', got %s", forwards[0].RemoteAddr)
	}
}

func TestForwardManager_StopForward(t *testing.T) {
	fm := NewForwardManager()
	
	sendFunc := func(msg string) {}
	
	err := fm.StartForward("test1", "0", "example.com:80", sendFunc)
	if err != nil {
		t.Fatalf("StartForward failed: %v", err)
	}
	
	err = fm.StopForward("test1")
	if err != nil {
		t.Errorf("StopForward failed: %v", err)
	}
	
	forwards := fm.ListForwards()
	if len(forwards) != 0 {
		t.Errorf("Expected 0 forwards, got %d", len(forwards))
	}
}

func TestForwardManager_DuplicateID(t *testing.T) {
	fm := NewForwardManager()
	
	sendFunc := func(msg string) {}
	
	err := fm.StartForward("test1", "0", "example.com:80", sendFunc)
	if err != nil {
		t.Fatalf("First StartForward failed: %v", err)
	}
	
	err = fm.StartForward("test1", "0", "example.com:443", sendFunc)
	if err == nil {
		t.Error("Expected error for duplicate forward ID, got nil")
	}
}

func TestForwardManager_StopAll(t *testing.T) {
	fm := NewForwardManager()
	
	sendFunc := func(msg string) {}
	
	_ = fm.StartForward("test1", "0", "example.com:80", sendFunc)
	_ = fm.StartForward("test2", "0", "example.com:443", sendFunc)
	
	fm.StopAll()
	
	forwards := fm.ListForwards()
	if len(forwards) != 0 {
		t.Errorf("Expected 0 forwards after StopAll, got %d", len(forwards))
	}
}
