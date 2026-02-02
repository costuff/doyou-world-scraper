package main

import (
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func TestWebSocketHub(t *testing.T) {
	// Initialize the hub before testing
	InitWebSocketHub()

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	t.Run("Hub initialization", func(t *testing.T) {
		if hub == nil {
			t.Fatal("Hub should be initialized")
		}

		if hub.clients == nil {
			t.Error("Hub clients map should be initialized")
		}

		if hub.broadcast == nil {
			t.Error("Hub broadcast channel should be initialized")
		}

		if hub.register == nil {
			t.Error("Hub register channel should be initialized")
		}

		if hub.unregister == nil {
			t.Error("Hub unregister channel should be initialized")
		}
	})
}

func TestBroadcastJobUpdate(t *testing.T) {
	// Reinitialize hub for this test
	InitWebSocketHub()
	time.Sleep(10 * time.Millisecond)

	// Create a test job
	job := NewJob("test_job", fiber.Map{"test": "data"})

	t.Run("Broadcast without panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("BroadcastJobUpdate panicked: %v", r)
			}
		}()

		// Should not panic even with no clients
		BroadcastJobUpdate(job)

		// Give time for broadcast to be processed
		time.Sleep(10 * time.Millisecond)
	})

	t.Run("Broadcast with nil hub", func(t *testing.T) {
		oldHub := hub
		hub = nil

		defer func() {
			hub = oldHub
			if r := recover(); r != nil {
				t.Errorf("BroadcastJobUpdate with nil hub panicked: %v", r)
			}
		}()

		// Should not panic with nil hub
		BroadcastJobUpdate(job)
	})

	t.Run("Multiple broadcasts don't block", func(t *testing.T) {
		done := make(chan bool, 1)
		
		go func() {
			for i := 0; i < 100; i++ {
				job := NewJob("test_broadcast", fiber.Map{"index": i})
				BroadcastJobUpdate(job)
			}
			done <- true
		}()

		select {
		case <-done:
			// Success - broadcasts completed
		case <-time.After(2 * time.Second):
			t.Error("Multiple broadcasts blocked or took too long")
		}
	})
}
