package httpapi

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// SettingsAPI handles settings-related HTTP endpoints
type SettingsAPI struct {
	db         *sql.DB
	logger     *zap.Logger
	ocppServer OCPPServer
}

// NewSettingsAPI creates a new settings API
func NewSettingsAPI(db *sql.DB, logger *zap.Logger, ocppServer OCPPServer) *SettingsAPI {
	return &SettingsAPI{
		db:         db,
		logger:     logger,
		ocppServer: ocppServer,
	}
}

// Routes returns the routes for the settings API
func (api *SettingsAPI) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", api.GetSettings)
	r.Put("/", api.UpdateSettings)
	r.Get("/status", api.GetServerStatus)
	r.Post("/server/start", api.StartOCPPServer)
	r.Post("/server/stop", api.StopOCPPServer)
	return r
}

// Settings represents the application settings
type Settings struct {
	HeartbeatInterval string `json:"heartbeat_interval"`
	LogLevel          string `json:"log_level"`
}

// GetSettings handles GET /api/settings
func (api *SettingsAPI) GetSettings(w http.ResponseWriter, r *http.Request) {
	query := `SELECT key, value FROM app_settings`
	rows, err := api.db.QueryContext(r.Context(), query)
	if err != nil {
		api.logger.Error("Failed to query settings", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	settings := Settings{
		HeartbeatInterval: "300",  // default
		LogLevel:          "info", // default
	}

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			api.logger.Error("Failed to scan setting", zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		switch key {
		case "heartbeat_interval":
			settings.HeartbeatInterval = value
		case "log_level":
			settings.LogLevel = value
		}
	}

	if err = rows.Err(); err != nil {
		api.logger.Error("Row iteration error", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

// UpdateSettings handles PUT /api/settings
func (api *SettingsAPI) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var settings Settings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate settings
	if settings.HeartbeatInterval == "" {
		http.Error(w, "heartbeat_interval is required", http.StatusBadRequest)
		return
	}
	if settings.LogLevel == "" {
		http.Error(w, "log_level is required", http.StatusBadRequest)
		return
	}

	// Update settings in database
	updateQuery := `
		INSERT INTO app_settings (key, value) 
		VALUES (?, ?) 
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`

	// Update heartbeat_interval
	_, err := api.db.ExecContext(r.Context(), updateQuery, "heartbeat_interval", settings.HeartbeatInterval)
	if err != nil {
		api.logger.Error("Failed to update heartbeat_interval", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Update log_level
	_, err = api.db.ExecContext(r.Context(), updateQuery, "log_level", settings.LogLevel)
	if err != nil {
		api.logger.Error("Failed to update log_level", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	api.logger.Info("Settings updated",
		zap.String("heartbeat_interval", settings.HeartbeatInterval),
		zap.String("log_level", settings.LogLevel),
	)

	// Return updated settings
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

// ServerStatus represents the server status
type ServerStatus struct {
	OCPPServerRunning bool   `json:"ocppServerRunning"`
	HTTPAddr          string `json:"httpAddr"`
	OCPPEndpoint      string `json:"ocppEndpoint"`
}

// GetServerStatus handles GET /api/settings/status
func (api *SettingsAPI) GetServerStatus(w http.ResponseWriter, r *http.Request) {
	status := ServerStatus{
		OCPPServerRunning: api.ocppServer.IsRunning(),
		HTTPAddr:          ":8080",
		OCPPEndpoint:      "/ocpp16",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// StartOCPPServer handles POST /api/settings/server/start
func (api *SettingsAPI) StartOCPPServer(w http.ResponseWriter, r *http.Request) {
	api.logger.Info("OCPP server start requested")

	api.ocppServer.Start()

	response := map[string]string{
		"message": "OCPP server started successfully",
		"status":  "running",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// StopOCPPServer handles POST /api/settings/server/stop
func (api *SettingsAPI) StopOCPPServer(w http.ResponseWriter, r *http.Request) {
	api.logger.Info("OCPP server stop requested")

	api.ocppServer.Stop()

	response := map[string]string{
		"message": "OCPP server stopped successfully",
		"status":  "stopped",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
