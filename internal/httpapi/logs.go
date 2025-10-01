package httpapi

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// LogsAPI handles CSV log export endpoints
type LogsAPI struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewLogsAPI creates a new logs API instance
func NewLogsAPI(db *sql.DB, logger *zap.Logger) *LogsAPI {
	return &LogsAPI{
		db:     db,
		logger: logger,
	}
}

// Routes defines the logs API routes
func (api *LogsAPI) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/config", api.getLogsConfig)
	r.Put("/config", api.updateLogsConfig)
	r.Post("/download", api.downloadCSV)
	r.Post("/browse", api.browseDirectory)

	return r
}

// LogsConfig represents the logs configuration
type LogsConfig struct {
	Enabled        bool      `json:"enabled"`
	Directory      string    `json:"directory"`
	Frequency      string    `json:"frequency"`       // minutes, hours, days
	FrequencyValue int       `json:"frequency_value"` // the numeric value for the frequency
	LastExport     time.Time `json:"last_export"`
}

// getLogsConfig returns the current logs configuration
func (api *LogsAPI) getLogsConfig(w http.ResponseWriter, r *http.Request) {
	api.logger.Info("Logs configuration requested")

	// Get configuration from database
	config, err := api.getLogsConfigFromDB()
	if err != nil {
		api.logger.Error("Failed to get logs config from database", zap.Error(err))
		http.Error(w, "Failed to get configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(config); err != nil {
		api.logger.Error("Failed to encode logs config", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	api.logger.Info("Logs configuration returned")
}

// updateLogsConfig updates the logs configuration
func (api *LogsAPI) updateLogsConfig(w http.ResponseWriter, r *http.Request) {
	api.logger.Info("Logs configuration update requested")

	var config LogsConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		api.logger.Error("Failed to decode logs config", zap.Error(err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate configuration
	if config.Enabled && config.Directory == "" {
		http.Error(w, "Directory is required when logging is enabled", http.StatusBadRequest)
		return
	}

	if config.Frequency != "seconds" && config.Frequency != "minutes" && config.Frequency != "hours" {
		http.Error(w, "Invalid frequency. Must be seconds, minutes, or hours", http.StatusBadRequest)
		return
	}

	if config.FrequencyValue < 1 {
		http.Error(w, "Frequency value must be at least 1", http.StatusBadRequest)
		return
	}

	// No limits - user can set any frequency they want

	// Save configuration to database
	if err := api.saveLogsConfigToDB(&config); err != nil {
		api.logger.Error("Failed to save logs config to database", zap.Error(err))
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	api.logger.Info("Logs configuration updated",
		zap.Bool("enabled", config.Enabled),
		zap.String("directory", config.Directory),
		zap.String("frequency", config.Frequency),
		zap.Int("frequency_value", config.FrequencyValue))

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{"message": "Configuration updated successfully"}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		api.logger.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// downloadCSV generates and downloads a CSV file with station data
func (api *LogsAPI) downloadCSV(w http.ResponseWriter, r *http.Request) {
	api.logger.Info("CSV download requested")

	// Get all stations from the database
	rows, err := api.db.Query(`
		SELECT id, identity, model, total_energy_wh
		FROM chargers 
		ORDER BY id
	`)
	if err != nil {
		api.logger.Error("Failed to query chargers", zap.Error(err))
		http.Error(w, "Failed to query chargers", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Create CSV content
	var csvData [][]string
	csvData = append(csvData, []string{"ID", "OCPP Identity", "Model", "Total Energy (kWh)", "Timestamp"})

	// Get current timestamp for all records
	currentTime := time.Now().Format("2006-01-02 15:04:05")

	for rows.Next() {
		var id int
		var identity, model string
		var totalEnergyWh sql.NullInt64

		err := rows.Scan(&id, &identity, &model, &totalEnergyWh)
		if err != nil {
			api.logger.Error("Failed to scan charger row", zap.Error(err))
			continue
		}

		// Convert Wh to kWh and format
		totalEnergy := "0.000"
		if totalEnergyWh.Valid {
			kwh := float64(totalEnergyWh.Int64) / 1000.0
			totalEnergy = fmt.Sprintf("%.3f", kwh)
		}

		csvData = append(csvData, []string{
			fmt.Sprintf("%d", id),
			identity,
			model,
			totalEnergy,
			currentTime,
		})
	}

	if err := rows.Err(); err != nil {
		api.logger.Error("Error iterating charger rows", zap.Error(err))
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("stations_log_%s.csv", timestamp)

	// Set response headers for file download
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// Write CSV data
	writer := csv.NewWriter(w)
	defer writer.Flush()

	for _, row := range csvData {
		if err := writer.Write(row); err != nil {
			api.logger.Error("Failed to write CSV row", zap.Error(err))
			return
		}
	}

	api.logger.Info("CSV download completed", zap.String("filename", filename), zap.Int("rows", len(csvData)-1))
}

// saveCSVToFile saves CSV data to a file (for scheduled exports)
func (api *LogsAPI) saveCSVToFile(directory, filename string) error {
	// Get all stations from the database
	rows, err := api.db.Query(`
		SELECT id, identity, model, total_energy_wh
		FROM chargers 
		ORDER BY id
	`)
	if err != nil {
		return fmt.Errorf("failed to query chargers: %w", err)
	}
	defer rows.Close()

	// Create directory if it doesn't exist
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file
	filePath := filepath.Join(directory, filename)
	api.logger.Info("Creating CSV file",
		zap.String("directory", directory),
		zap.String("filename", filename),
		zap.String("full_path", filePath),
		zap.Bool("is_absolute", filepath.IsAbs(directory)))
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write CSV data
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"ID", "OCPP Identity", "Model", "Total Energy (kWh)", "Timestamp"}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Get current timestamp for all records
	currentTime := time.Now().Format("2006-01-02 15:04:05")

	// Write data rows
	for rows.Next() {
		var id int
		var identity, model string
		var totalEnergyWh sql.NullInt64

		err := rows.Scan(&id, &identity, &model, &totalEnergyWh)
		if err != nil {
			api.logger.Error("Failed to scan charger row", zap.Error(err))
			continue
		}

		// Convert Wh to kWh and format
		totalEnergy := "0.000"
		if totalEnergyWh.Valid {
			kwh := float64(totalEnergyWh.Int64) / 1000.0
			totalEnergy = fmt.Sprintf("%.3f", kwh)
		}

		row := []string{
			fmt.Sprintf("%d", id),
			identity,
			model,
			totalEnergy,
			currentTime,
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating charger rows: %w", err)
	}

	api.logger.Info("CSV file saved", zap.String("filepath", filePath))
	return nil
}

// getLogsConfigFromDB retrieves the logs configuration from the database
func (api *LogsAPI) getLogsConfigFromDB() (*LogsConfig, error) {
	query := `
		SELECT enabled, directory, frequency, frequency_value, last_export
		FROM logs_config
		ORDER BY id DESC
		LIMIT 1
	`

	var config LogsConfig
	var lastExport sql.NullTime

	err := api.db.QueryRow(query).Scan(
		&config.Enabled,
		&config.Directory,
		&config.Frequency,
		&config.FrequencyValue,
		&lastExport,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Return default configuration if no config exists
			return &LogsConfig{
				Enabled:        false,
				Directory:      "",
				Frequency:      "hours",
				FrequencyValue: 1,
				LastExport:     time.Time{},
			}, nil
		}
		return nil, fmt.Errorf("failed to query logs config: %w", err)
	}

	if lastExport.Valid {
		config.LastExport = lastExport.Time
	}

	return &config, nil
}

// saveLogsConfigToDB saves the logs configuration to the database
func (api *LogsAPI) saveLogsConfigToDB(config *LogsConfig) error {
	// First, try to update existing record
	updateQuery := `
		UPDATE logs_config 
		SET enabled = ?, directory = ?, frequency = ?, frequency_value = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = (SELECT id FROM logs_config ORDER BY id DESC LIMIT 1)
	`

	result, err := api.db.Exec(updateQuery,
		config.Enabled,
		config.Directory,
		config.Frequency,
		config.FrequencyValue,
	)

	if err != nil {
		return fmt.Errorf("failed to update logs config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// If no rows were updated, insert a new record
	if rowsAffected == 0 {
		insertQuery := `
			INSERT INTO logs_config (enabled, directory, frequency, frequency_value, last_export)
			VALUES (?, ?, ?, ?, ?)
		`

		var lastExport interface{}
		if config.LastExport.IsZero() {
			lastExport = nil
		} else {
			lastExport = config.LastExport
		}

		_, err = api.db.Exec(insertQuery,
			config.Enabled,
			config.Directory,
			config.Frequency,
			config.FrequencyValue,
			lastExport,
		)

		if err != nil {
			return fmt.Errorf("failed to insert logs config: %w", err)
		}
	}

	return nil
}

// updateLastExportTime updates the last export time in the database
func (api *LogsAPI) updateLastExportTime() error {
	query := `
		UPDATE logs_config 
		SET last_export = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = (SELECT id FROM logs_config ORDER BY id DESC LIMIT 1)
	`

	_, err := api.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to update last export time: %w", err)
	}

	return nil
}

// browseDirectory handles directory browsing requests
func (api *LogsAPI) browseDirectory(w http.ResponseWriter, r *http.Request) {
	api.logger.Info("Directory browse requested")

	// For now, just return a simple response
	// In a real implementation, this would interact with the file system
	response := map[string]interface{}{
		"success": true,
		"message": "Directory browsing not fully implemented yet. Please type the full path manually.",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
