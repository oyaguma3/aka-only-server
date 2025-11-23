package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"aka-server/internal/api"
	"aka-server/internal/config"
	"aka-server/internal/db"
	"aka-server/internal/logger"
	"aka-server/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	// Parse CLI Flags
	install := flag.Bool("install", false, "Install as systemd service")
	uninstall := flag.Bool("uninstall", false, "Uninstall systemd service")
	serviceName := flag.String("service-name", "aka-server", "Name of the systemd service")
	flag.Parse()

	if *install {
		if err := service.Install(*serviceName, "AKA API Server"); err != nil {
			fmt.Printf("Failed to install service: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Service installed successfully.")
		return
	}

	if *uninstall {
		if err := service.Uninstall(*serviceName); err != nil {
			fmt.Printf("Failed to uninstall service: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Service uninstalled successfully.")
		return
	}

	// Load Configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize Logger
	logger.InitLogger(cfg.LogFile, cfg.LogMaxSize, cfg.LogMaxBackups, cfg.LogMaxAge)
	slog.Info("Starting AKA Server...")

	// Initialize Database
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)
	repo, err := db.NewRepository(dbURL)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer repo.Close()
	slog.Info("Connected to database")

	// Initialize API Handler
	handler := api.NewHandler(repo, cfg)

	// Setup Router
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Custom Logger Middleware for Gin to use slog
	r.Use(func(c *gin.Context) {
		c.Next()
		slog.Info("Request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"ip", c.ClientIP(),
		)
	})

	handler.RegisterRoutes(r)

	// Start Server
	addr := ":" + cfg.APIPort
	slog.Info("Server listening", "addr", addr)
	if err := r.Run(addr); err != nil {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}
