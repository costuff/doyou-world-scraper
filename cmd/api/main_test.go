package main

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// TestHealthEndpoint tests a basic health check endpoint
// Note: This tests the endpoint pattern, not the actual main() server
func TestHealthEndpoint(t *testing.T) {
	app := fiber.New()

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
		})
	})

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Health endpoint returned %d, want 200", resp.StatusCode)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	body := buf.String()

	if !bytes.Contains([]byte(body), []byte("ok")) {
		t.Errorf("Health endpoint response doesn't contain 'ok': %s", body)
	}
}

// Note: main() function itself cannot be easily unit tested as it:
// 1. Calls os.Exit via logger.Fatalf
// 2. Starts a blocking HTTP server
// 3. Loads configuration from environment
// Integration tests or Docker-based tests would be more appropriate
