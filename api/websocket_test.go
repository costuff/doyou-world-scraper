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
	// Create a test job
	job := NewJob("test_job", fiber.Map{"test": "data"})

	t.Run("Broadcast without panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("BroadcastJobUpdate panicked: %v", r)
			}
		}()

		BroadcastJobUpdate(job)
		time.Sleep(10 * time.Millisecond) // Give time for broadcast to process
	})
}
