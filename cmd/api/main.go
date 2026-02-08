package main

import (
	"doyou-world-scraper/internal/api"
	"doyou-world-scraper/internal/config"
	"doyou-world-scraper/internal/jobs"
	"doyou-world-scraper/pkg/logger"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger.Init(cfg.LogLevel, cfg.IsProduction())
	logger.Infof("Starting DoYou World Radio Scraper API in %s mode", cfg.Environment)

	// Ensure required directories exist
	if err := cfg.EnsureDirectories(); err != nil {
		logger.Fatalf("Failed to create directories: %v", err)
	}
	logger.Infof("Data directory: %s", cfg.DataDir)

	// Initialize WebSocket hub
	jobs.InitWebSocketHub()
	logger.Info("WebSocket hub initialized")

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName: "DoYou World Radio Scraper API",
	})

	// Middleware
	app.Use(fiberlogger.New())
	app.Use(cors.New())

	// Routes
	apiGroup := app.Group("/api/v1")

	// Health check
	apiGroup.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "DoYou World Radio Scraper API is running",
			"config": fiber.Map{
				"environment": cfg.Environment,
				"data_dir":    cfg.DataDir,
			},
		})
	})

	// Initialize handlers with config
	handlers := api.NewHandlers(cfg)

	// Version endpoint
	apiGroup.Get("/version", handlers.HandleVersion)

	// Scraping endpoints
	scrapeGroup := apiGroup.Group("/scrape")
	scrapeGroup.Post("/date", handlers.HandleScrapeDate)
	scrapeGroup.Post("/month", handlers.HandleScrapeMonth)
	scrapeGroup.Post("/year", handlers.HandleScrapeYear)

	// Download endpoints
	downloadGroup := apiGroup.Group("/download")
	downloadGroup.Post("/csv", handlers.HandleDownloadFromCSV)
	downloadGroup.Post("/all", handlers.HandleDownloadAll)

	// Utility endpoints
	utilGroup := apiGroup.Group("/util")
	utilGroup.Post("/deduplicate", handlers.HandleDeduplicate)
	utilGroup.Post("/move-to-nas", handlers.HandleMoveToNAS)
	utilGroup.Post("/match", handlers.HandleMatchDownloads)
	utilGroup.Post("/clear-csvs", handlers.HandleClearCSVs)
	utilGroup.Post("/clear-downloads", handlers.HandleClearDownloads)
	utilGroup.Get("/stats", handlers.HandleStats)

	// Track browsing endpoints
	apiGroup.Get("/tracks", handlers.HandleGetTracks)
	apiGroup.Get("/tracks/stream", handlers.HandleStreamTrack)
	apiGroup.Get("/tracks/status", handlers.HandleGetTracksStatus)

	// Audio analysis endpoints
	apiGroup.Post("/analyze/track", handlers.HandleAnalyzeTrack)
	apiGroup.Post("/analyze/all", handlers.HandleAnalyzeAll)

	// Jobs endpoint
	apiGroup.Get("/jobs", handlers.HandleGetJobs)
	apiGroup.Get("/jobs/:id", handlers.HandleGetJob)
	apiGroup.Post("/jobs/:id/cancel", handlers.HandleCancelJob)

	// WebSocket endpoint
	apiGroup.Get("/ws", websocket.New(jobs.HandleWebSocket))

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Port)
	logger.Infof("Starting server on port %d", cfg.Port)
	if err := app.Listen(addr); err != nil {
		logger.Fatalf("Failed to start server: %v", err)
	}
}
