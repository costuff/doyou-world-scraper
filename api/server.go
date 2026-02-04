package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
)

func main() {
	// Initialize WebSocket hub
	InitWebSocketHub()

	app := fiber.New(fiber.Config{
		AppName: "DoYou World Radio Scraper API",
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New())

	// Routes
	api := app.Group("/api/v1")

	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "DoYou World Radio Scraper API is running",
		})
	})

	// Scraping endpoints
	scrapeGroup := api.Group("/scrape")
	scrapeGroup.Post("/date", handleScrapeDate)
	scrapeGroup.Post("/month", handleScrapeMonth)
	scrapeGroup.Post("/year", handleScrapeYear)

	// Download endpoints
	downloadGroup := api.Group("/download")
	downloadGroup.Post("/csv", handleDownloadFromCSV)
	downloadGroup.Post("/all", handleDownloadAll)

	// Utility endpoints
	utilGroup := api.Group("/util")
	utilGroup.Post("/deduplicate", handleDeduplicate)
	utilGroup.Post("/move-to-nas", handleMoveToNAS)
	utilGroup.Post("/match", handleMatchDownloads)
	utilGroup.Post("/clear-csvs", handleClearCSVs)
	utilGroup.Post("/clear-downloads", handleClearDownloads)
	utilGroup.Get("/stats", handleStats)

	// Track browsing endpoints
	api.Get("/tracks", handleGetTracks)
	api.Get("/tracks/stream", handleStreamTrack)
	api.Get("/tracks/status", handleGetTracksStatus)

	// Jobs endpoint
	api.Get("/jobs", handleGetJobs)
	api.Get("/jobs/:id", handleGetJob)
	api.Post("/jobs/:id/cancel", handleCancelJob)

	// WebSocket endpoint
	api.Get("/ws", websocket.New(HandleWebSocket))

	// Get port from env or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001" // Changed from 3000 to avoid conflict with VS Code port forwarding
	}

	log.Printf("Starting server on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatal(err)
	}
}
