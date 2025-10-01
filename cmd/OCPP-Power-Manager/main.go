package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"

	"OCPP-Power-Manager/internal/config"
	"OCPP-Power-Manager/internal/db"
	"OCPP-Power-Manager/internal/httpapi"
	"OCPP-Power-Manager/internal/ocpp"
)

func main() {
	// Initialize logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	logger.Info("🚀 Starting OCPP Power Manager - EV Charging Station Management System")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	logger.Info("⚙️ Configuration loaded successfully",
		zap.String("http_addr", cfg.HTTPAddr),
		zap.String("db_driver", cfg.DBDriver),
		zap.String("db_dsn", maskDSN(cfg.DBDSN)),
	)

	// Open database connection
	ctx := context.Background()
	database, err := db.Open(ctx, cfg.DBDriver, cfg.DBDSN)
	if err != nil {
		logger.Fatal("Failed to open database", zap.Error(err))
	}
	defer func() {
		if err := db.Close(database); err != nil {
			logger.Error("Failed to close database", zap.Error(err))
		}
	}()

	logger.Info("🗄️ Database connection established successfully")

	// Run migrations if requested
	if os.Getenv("RUN_MIGRATIONS") == "1" {
		logger.Info("Running database migrations")
		if err := goose.SetDialect("sqlite3"); err != nil {
			logger.Fatal("Failed to set goose dialect", zap.Error(err))
		}
		if err := goose.Up(database, "migrations"); err != nil {
			logger.Fatal("Failed to run migrations", zap.Error(err))
		}
		logger.Info("Database migrations completed")
	}

	// Setup HTTP router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Mount OCPP server
	ocppServer := ocpp.New(database, logger)
	ocppServer.Mount(r)

	// Create API instance with OCPP server
	api := httpapi.New(database, logger, ocppServer)

	// Logs scheduler temporarily disabled
	// logsScheduler := httpapi.NewLogsScheduler(database, logger)
	// logsScheduler.Start()
	// defer logsScheduler.Stop()

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Mount("/", api.Routes())
	})

	// Serve static files (React app)
	r.Handle("/*", httpapi.StaticHandler())

	// Start HTTP server
	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("🌐 Starting HTTP server", zap.String("addr", cfg.HTTPAddr), zap.String("url", "http://localhost:8080"))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	logger.Info("✅ OCPP Power Manager is running! Open http://localhost:8080 in your browser")
	logger.Info("📱 Web interface ready - Manage your EV charging stations")
	logger.Info("🔌 OCPP server ready - Stations can connect to ws://localhost:8080/ocpp16/{station_id}")
	logger.Info("⏹️ Press Ctrl+C to stop the server")
	
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

// maskDSN masks sensitive information in DSN for logging
func maskDSN(dsn string) string {
	// Simple masking - replace password-like patterns
	if len(dsn) > 20 {
		return dsn[:10] + "***" + dsn[len(dsn)-7:]
	}
	return "***"
}
