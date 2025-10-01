package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Station represents a charging station
type Station struct {
	ID             int64      `json:"id"`
	Identity       string     `json:"identity"`
	Name           *string    `json:"name"`
	Model          *string    `json:"model"`
	Vendor         *string    `json:"vendor"`
	MaxOutputKW    *float64   `json:"max_output_kw"`
	TotalEnergyWh  *int64     `json:"total_energy_wh"`
	TotalEnergyKwh *float64   `json:"total_energy_kwh"`
	Firmware       *string    `json:"firmware"`
	LastSeen       *time.Time `json:"last_seen"`
}

// CreateStationRequest represents the request to create a station
type CreateStationRequest struct {
	Identity    string   `json:"identity"`
	Name        *string  `json:"name"`
	Model       *string  `json:"model"`
	Vendor      *string  `json:"vendor"`
	MaxOutputKW *float64 `json:"max_output_kw"`
}

// UpdateStationRequest represents the request to update a station
type UpdateStationRequest struct {
	Name        *string  `json:"name"`
	Model       *string  `json:"model"`
	Vendor      *string  `json:"vendor"`
	MaxOutputKW *float64 `json:"max_output_kw"`
}

// StationsAPI handles station-related HTTP endpoints
type StationsAPI struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewStationsAPI creates a new stations API
func NewStationsAPI(db *sql.DB, logger *zap.Logger) *StationsAPI {
	return &StationsAPI{
		db:     db,
		logger: logger,
	}
}

// Routes returns the routes for the stations API
func (api *StationsAPI) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", api.ListStations)
	r.Post("/", api.CreateStation)
	r.Put("/{id}", api.UpdateStation)
	r.Delete("/{id}", api.DeleteStation)
	return r
}

// ListStations handles GET /api/stations
func (api *StationsAPI) ListStations(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, identity, name, model, vendor, max_output_kw, total_energy_wh, firmware, last_seen
		FROM chargers
		ORDER BY id ASC
	`

	rows, err := api.db.QueryContext(r.Context(), query)
	if err != nil {
		api.logger.Error("Failed to query stations", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var stations []Station
	for rows.Next() {
		var station Station
		err := rows.Scan(
			&station.ID,
			&station.Identity,
			&station.Name,
			&station.Model,
			&station.Vendor,
			&station.MaxOutputKW,
			&station.TotalEnergyWh,
			&station.Firmware,
			&station.LastSeen,
		)
		if err != nil {
			api.logger.Error("Failed to scan station", zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Calculate total_energy_kwh = total_energy_wh/1000 (float with 3 decimals)
		if station.TotalEnergyWh != nil {
			kwh := float64(*station.TotalEnergyWh) / 1000.0
			// Round to 3 decimal places
			station.TotalEnergyKwh = &kwh
		}

		stations = append(stations, station)
	}

	if err = rows.Err(); err != nil {
		api.logger.Error("Row iteration error", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Ensure we always return an array, never null
	if stations == nil {
		stations = []Station{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stations)
}

// CreateStation handles POST /api/stations
func (api *StationsAPI) CreateStation(w http.ResponseWriter, r *http.Request) {
	var req CreateStationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate identity
	if err := validateIdentity(req.Identity); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate max_output_kw
	if req.MaxOutputKW != nil && *req.MaxOutputKW < 0 {
		http.Error(w, "max_output_kw must be >= 0", http.StatusBadRequest)
		return
	}

	query := `
		INSERT INTO chargers (identity, name, model, vendor, max_output_kw)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := api.db.ExecContext(r.Context(), query,
		req.Identity,
		req.Name,
		req.Model,
		req.Vendor,
		req.MaxOutputKW,
	)

	if err != nil {
		api.logger.Error("Failed to create station", zap.Error(err))
		// Check for unique constraint violation
		if isUniqueConstraintError(err) {
			http.Error(w, "Station with this identity already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		api.logger.Error("Failed to get last insert ID", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Fetch the created station
	station, err := api.getStationByID(r.Context(), id)
	if err != nil {
		api.logger.Error("Failed to fetch created station", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(station)
}

// UpdateStation handles PUT /api/stations/{id}
func (api *StationsAPI) UpdateStation(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid station ID", http.StatusBadRequest)
		return
	}

	var req UpdateStationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate max_output_kw
	if req.MaxOutputKW != nil && *req.MaxOutputKW < 0 {
		http.Error(w, "max_output_kw must be >= 0", http.StatusBadRequest)
		return
	}

	query := `
		UPDATE chargers 
		SET name = ?, model = ?, vendor = ?, max_output_kw = ?
		WHERE id = ?
	`

	result, err := api.db.ExecContext(r.Context(), query,
		req.Name,
		req.Model,
		req.Vendor,
		req.MaxOutputKW,
		id,
	)

	if err != nil {
		api.logger.Error("Failed to update station", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		api.logger.Error("Failed to get rows affected", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Station not found", http.StatusNotFound)
		return
	}

	// Fetch the updated station
	station, err := api.getStationByID(r.Context(), id)
	if err != nil {
		api.logger.Error("Failed to fetch updated station", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(station)
}

// DeleteStation handles DELETE /api/stations/{id}
func (api *StationsAPI) DeleteStation(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid station ID", http.StatusBadRequest)
		return
	}

	query := `DELETE FROM chargers WHERE id = ?`
	result, err := api.db.ExecContext(r.Context(), query, id)
	if err != nil {
		api.logger.Error("Failed to delete station", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		api.logger.Error("Failed to get rows affected", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Station not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// getStationByID fetches a station by ID
func (api *StationsAPI) getStationByID(ctx context.Context, id int64) (*Station, error) {
	query := `
		SELECT id, identity, name, model, vendor, max_output_kw, total_energy_wh, firmware, last_seen
		FROM chargers
		WHERE id = ?
	`

	var station Station
	err := api.db.QueryRowContext(ctx, query, id).Scan(
		&station.ID,
		&station.Identity,
		&station.Name,
		&station.Model,
		&station.Vendor,
		&station.MaxOutputKW,
		&station.TotalEnergyWh,
		&station.Firmware,
		&station.LastSeen,
	)

	if err != nil {
		return nil, err
	}

	// Calculate total_energy_kwh = total_energy_wh/1000 (float with 3 decimals)
	if station.TotalEnergyWh != nil {
		kwh := float64(*station.TotalEnergyWh) / 1000.0
		// Round to 3 decimal places
		station.TotalEnergyKwh = &kwh
	}

	return &station, nil
}

// validateIdentity validates the station identity
func validateIdentity(identity string) error {
	if identity == "" {
		return fmt.Errorf("identity is required")
	}

	if len(identity) > 64 {
		return fmt.Errorf("identity must be <= 64 characters")
	}

	// Allow A-Z, a-z, 0-9, _, -, .
	matched, err := regexp.MatchString(`^[A-Za-z0-9_.-]+$`, identity)
	if err != nil {
		return fmt.Errorf("invalid identity format")
	}

	if !matched {
		return fmt.Errorf("identity can only contain A-Z, a-z, 0-9, _, -, .")
	}

	return nil
}

// isUniqueConstraintError checks if the error is a unique constraint violation
func isUniqueConstraintError(err error) bool {
	// SQLite returns "UNIQUE constraint failed" for unique violations
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
