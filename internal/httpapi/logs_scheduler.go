package httpapi

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// LogsScheduler handles automatic CSV log exports
type LogsScheduler struct {
	db     *sql.DB
	logger *zap.Logger
	api    *LogsAPI
	ctx    context.Context
	cancel context.CancelFunc
}

// NewLogsScheduler creates a new logs scheduler
func NewLogsScheduler(db *sql.DB, logger *zap.Logger) *LogsScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	api := NewLogsAPI(db, logger)

	return &LogsScheduler{
		db:     db,
		logger: logger,
		api:    api,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start begins the background scheduler
func (s *LogsScheduler) Start() {
	s.logger.Info("Starting logs scheduler")

	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("Scheduler panic recovered", zap.Any("panic", r))
			}
		}()
		s.run()
	}()
}

// Stop stops the background scheduler
func (s *LogsScheduler) Stop() {
	s.logger.Info("Stopping logs scheduler")
	s.cancel()
}

// run is the main scheduler loop
func (s *LogsScheduler) run() {
	s.logger.Info("Scheduler run() function started")
	ticker := time.NewTicker(1 * time.Minute) // Check every minute
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("Logs scheduler stopped")
			return
		case <-ticker.C:
			s.logger.Info("Scheduler ticker fired - calling checkAndExport")
			s.checkAndExport()
		}
	}
}

// checkAndExport checks if it's time to export logs and performs the export if needed
func (s *LogsScheduler) checkAndExport() {
	s.logger.Debug("Scheduler checkAndExport called")

	// Get current configuration
	config, err := s.api.getLogsConfigFromDB()
	if err != nil {
		s.logger.Error("Failed to get logs config for scheduler", zap.Error(err))
		return
	}

	s.logger.Debug("Scheduler config retrieved",
		zap.Bool("enabled", config.Enabled),
		zap.String("directory", config.Directory),
		zap.String("frequency", config.Frequency),
		zap.Int("frequency_value", config.FrequencyValue),
		zap.Time("last_export", config.LastExport))

	// Skip if logging is disabled
	if !config.Enabled {
		s.logger.Debug("Logging is disabled, skipping export")
		return
	}

	// Skip if no directory is configured
	if config.Directory == "" {
		s.logger.Debug("No directory configured, skipping export")
		return
	}

	// Check if it's time to export
	if !s.shouldExport(config) {
		s.logger.Debug("Not time to export yet, skipping")
		return
	}

	// Perform the export
	s.performExport(config)
}

// shouldExport determines if it's time to export based on the configuration
func (s *LogsScheduler) shouldExport(config *LogsConfig) bool {
	now := time.Now().UTC() // Use UTC for consistency with database

	s.logger.Debug("Checking if should export",
		zap.Time("now", now),
		zap.Time("last_export", config.LastExport),
		zap.String("frequency", config.Frequency),
		zap.Int("frequency_value", config.FrequencyValue))

	// If never exported before, export now
	if config.LastExport.IsZero() {
		s.logger.Info("First time export - performing export now")
		return true
	}

	// Calculate next export time based on frequency
	var nextExport time.Time
	switch config.Frequency {
	case "seconds":
		nextExport = config.LastExport.Add(time.Duration(config.FrequencyValue) * time.Second)
	case "minutes":
		nextExport = config.LastExport.Add(time.Duration(config.FrequencyValue) * time.Minute)
	case "hours":
		nextExport = config.LastExport.Add(time.Duration(config.FrequencyValue) * time.Hour)
	case "days":
		nextExport = config.LastExport.Add(time.Duration(config.FrequencyValue) * 24 * time.Hour)
	default:
		s.logger.Warn("Unknown frequency", zap.String("frequency", config.Frequency))
		return false
	}

	// Export if it's time
	shouldExport := now.After(nextExport)
	s.logger.Debug("Export check result",
		zap.Time("next_export", nextExport),
		zap.Bool("should_export", shouldExport))

	if shouldExport {
		s.logger.Info("Scheduled export time reached",
			zap.Time("last_export", config.LastExport),
			zap.Time("next_export", nextExport),
			zap.Time("now", now))
	}

	return shouldExport
}

// performExport performs the actual CSV export
func (s *LogsScheduler) performExport(config *LogsConfig) {
	s.logger.Info("Starting scheduled CSV export",
		zap.String("directory", config.Directory))

	// Generate filename with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("stations_log_%s.csv", timestamp)

	// Save CSV to file
	if err := s.api.saveCSVToFile(config.Directory, filename); err != nil {
		s.logger.Error("Failed to save CSV file during scheduled export",
			zap.Error(err),
			zap.String("directory", config.Directory),
			zap.String("filename", filename))
		return
	}

	// Update last export time
	if err := s.api.updateLastExportTime(); err != nil {
		s.logger.Error("Failed to update last export time",
			zap.Error(err))
		return
	}

	s.logger.Info("Scheduled CSV export completed successfully",
		zap.String("directory", config.Directory),
		zap.String("filename", filename))
}
