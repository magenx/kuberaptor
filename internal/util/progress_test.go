// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package util

import (
	"testing"
	"time"
)

func TestNewProgressBar(t *testing.T) {
	pb := NewProgressBar(100, "test")
	if pb == nil {
		t.Fatal("NewProgressBar returned nil")
	}
	if pb.total != 100 {
		t.Errorf("expected total 100, got %d", pb.total)
	}
	if pb.current != 0 {
		t.Errorf("expected current 0, got %d", pb.current)
	}
	if pb.prefix != "test" {
		t.Errorf("expected prefix 'test', got %q", pb.prefix)
	}
	if pb.width != 50 {
		t.Errorf("expected width 50, got %d", pb.width)
	}
}

func TestProgressBar_Increment(t *testing.T) {
	pb := NewProgressBar(10, "test")

	pb.Increment()
	pb.mu.Lock()
	current := pb.current
	pb.mu.Unlock()

	if current != 1 {
		t.Errorf("expected current 1 after one Increment, got %d", current)
	}

	for i := 0; i < 5; i++ {
		pb.Increment()
	}
	pb.mu.Lock()
	current = pb.current
	pb.mu.Unlock()

	if current != 6 {
		t.Errorf("expected current 6 after 6 increments, got %d", current)
	}
}

func TestProgressBar_Add(t *testing.T) {
	pb := NewProgressBar(100, "test")

	pb.Add(10)
	pb.mu.Lock()
	current := pb.current
	pb.mu.Unlock()

	if current != 10 {
		t.Errorf("expected current 10, got %d", current)
	}

	pb.Add(25)
	pb.mu.Lock()
	current = pb.current
	pb.mu.Unlock()

	if current != 35 {
		t.Errorf("expected current 35, got %d", current)
	}
}

func TestProgressBar_Add_CapsAtTotal(t *testing.T) {
	pb := NewProgressBar(10, "test")

	// Add beyond total
	pb.Add(100)
	pb.mu.Lock()
	current := pb.current
	pb.mu.Unlock()

	if current != 10 {
		t.Errorf("expected current capped at total=10, got %d", current)
	}
}

func TestProgressBar_SetCurrent(t *testing.T) {
	pb := NewProgressBar(100, "test")

	pb.SetCurrent(42)
	pb.mu.Lock()
	current := pb.current
	pb.mu.Unlock()

	if current != 42 {
		t.Errorf("expected current 42, got %d", current)
	}
}

func TestProgressBar_SetCurrent_CapsAtTotal(t *testing.T) {
	pb := NewProgressBar(50, "test")

	pb.SetCurrent(200)
	pb.mu.Lock()
	current := pb.current
	pb.mu.Unlock()

	if current != 50 {
		t.Errorf("expected current capped at total=50, got %d", current)
	}
}

func TestProgressBar_Finish(t *testing.T) {
	pb := NewProgressBar(10, "test")
	pb.Add(3)

	pb.Finish()

	pb.mu.Lock()
	current := pb.current
	pb.mu.Unlock()

	if current != 10 {
		t.Errorf("expected current==total after Finish, got current=%d total=%d", current, pb.total)
	}
}

func TestNewSpinner(t *testing.T) {
	s := NewSpinner("Loading...", "test-scope")
	if s == nil {
		t.Fatal("NewSpinner returned nil")
	}
	if s.message != "Loading..." {
		t.Errorf("expected message 'Loading...', got %q", s.message)
	}
	if s.scope != "test-scope" {
		t.Errorf("expected scope 'test-scope', got %q", s.scope)
	}
	if s.active {
		t.Error("expected spinner to be inactive initially")
	}
	if len(s.frames) == 0 {
		t.Error("expected non-empty frames")
	}
}

func TestSpinner_StartStop(t *testing.T) {
	s := NewSpinner("Working...", "scope")

	s.Start()

	// Give goroutine a moment to start
	time.Sleep(10 * time.Millisecond)

	s.mu.Lock()
	active := s.active
	s.mu.Unlock()

	if !active {
		t.Error("expected spinner to be active after Start()")
	}

	s.Stop(true)

	// Give goroutine a moment to stop
	time.Sleep(20 * time.Millisecond)

	s.mu.Lock()
	active = s.active
	s.mu.Unlock()

	if active {
		t.Error("expected spinner to be inactive after Stop()")
	}
}

func TestSpinner_DoubleStart(t *testing.T) {
	s := NewSpinner("Working...", "scope")

	s.Start()
	time.Sleep(10 * time.Millisecond)

	// Double start should be a no-op (idempotent)
	s.Start()
	time.Sleep(10 * time.Millisecond)

	s.mu.Lock()
	active := s.active
	s.mu.Unlock()

	if !active {
		t.Error("expected spinner to still be active after double Start()")
	}

	s.Stop(true)
}

func TestSpinner_StopWhenNotStarted(t *testing.T) {
	s := NewSpinner("Working...", "scope")

	// Stop without starting should be a no-op
	// This should not block or panic
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.Stop(false)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("Stop() on non-started spinner blocked unexpectedly")
	}
}

func TestSpinner_UpdateMessage(t *testing.T) {
	s := NewSpinner("Initial message", "scope")

	s.UpdateMessage("Updated message")

	s.mu.Lock()
	msg := s.message
	s.mu.Unlock()

	if msg != "Updated message" {
		t.Errorf("expected message 'Updated message', got %q", msg)
	}
}

func TestSpinner_UpdateMessage_WhileRunning(t *testing.T) {
	s := NewSpinner("Initial message", "scope")
	s.Start()
	defer s.Stop(true)

	time.Sleep(10 * time.Millisecond)
	s.UpdateMessage("Runtime message")
	time.Sleep(10 * time.Millisecond)

	s.mu.Lock()
	msg := s.message
	s.mu.Unlock()

	if msg != "Runtime message" {
		t.Errorf("expected message 'Runtime message', got %q", msg)
	}
}

func TestSpinner_StartStopCycle(t *testing.T) {
	s := NewSpinner("Cycling...", "")

	for i := 0; i < 3; i++ {
		s.Start()
		time.Sleep(10 * time.Millisecond)
		s.Stop(false)
		time.Sleep(10 * time.Millisecond)
	}

	s.mu.Lock()
	active := s.active
	s.mu.Unlock()

	if active {
		t.Error("expected spinner to be inactive after start/stop cycle")
	}
}
