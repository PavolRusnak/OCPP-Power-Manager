package httpapi

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// DevAPI handles development/testing endpoints
type DevAPI struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewDevAPI creates a new dev API
func NewDevAPI(db *sql.DB, logger *zap.Logger) *DevAPI {
	return &DevAPI{
		db:     db,
		logger: logger,
	}
}

// Routes returns the routes for the dev API
func (api *DevAPI) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/meter", api.SimulateMeterValues)
	return r
}

// MeterTestRequest represents the request to simulate a MeterValues update
type MeterTestRequest struct {
	Identity string  `json:"identity"`
	Ts       string  `json:"ts"`
	ValueWh  float64 `json:"value_wh"`
}

// SimulateMeterValues handles POST /api/dev/meter
func (api *DevAPI) SimulateMeterValues(w http.ResponseWriter, r *http.Request) {
	var req MeterTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Identity == "" {
		http.Error(w, "identity is required", http.StatusBadRequest)
		return
	}
	if req.ValueWh < 0 {
		http.Error(w, "value_wh must be >= 0", http.StatusBadRequest)
		return
	}

	// Parse timestamp
	var timestamp time.Time
	if req.Ts != "" {
		var err error
		timestamp, err = time.Parse(time.RFC3339, req.Ts)
		if err != nil {
			http.Error(w, "Invalid timestamp format (use RFC3339)", http.StatusBadRequest)
			return
		}
	} else {
		timestamp = time.Now()
	}

	// Find charger by identity
	var chargerID int64
	err := api.db.QueryRowContext(r.Context(), "SELECT id FROM chargers WHERE identity = ?", req.Identity).Scan(&chargerID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Charger not found", http.StatusNotFound)
			return
		}
		api.logger.Error("Failed to find charger", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create a fake transaction for this meter value
	txID := int(time.Now().Unix())
	insertTxQuery := `
		INSERT INTO transactions (charger_id, tx_id, start_ts, start_meter_wh)
		VALUES (?, ?, ?, ?)
	`

	result, err := api.db.ExecContext(r.Context(), insertTxQuery,
		chargerID,
		txID,
		timestamp,
		0, // start with 0
	)

	if err != nil {
		api.logger.Error("Failed to create test transaction", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	transactionID, err := result.LastInsertId()
	if err != nil {
		api.logger.Error("Failed to get transaction ID", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Insert meter value
	insertMeterQuery := `
		INSERT INTO meter_values (transaction_id, ts, measurand, value)
		VALUES (?, ?, ?, ?)
	`

	_, err = api.db.ExecContext(r.Context(), insertMeterQuery,
		transactionID,
		timestamp,
		"Energy.Active.Import.Register",
		req.ValueWh,
	)

	if err != nil {
		api.logger.Error("Failed to insert meter value", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get the previous meter reading to calculate incremental energy
	var previousMeterReading float64
	err = api.db.QueryRowContext(r.Context(), 
		"SELECT value FROM meter_values WHERE transaction_id IN (SELECT id FROM transactions WHERE charger_id = ?) AND measurand = 'Energy.Active.Import.Register' ORDER BY ts DESC LIMIT 1", 
		chargerID).Scan(&previousMeterReading)
	
	// Calculate incremental energy (current reading - previous reading)
	incrementalEnergy := req.ValueWh - previousMeterReading
	if incrementalEnergy < 0 {
		// Meter reset or out-of-order reading, ignore
		api.logger.Info("Ignoring negative incremental energy (meter reset or out-of-order reading)",
			zap.String("identity", req.Identity),
			zap.Float64("current_reading", req.ValueWh),
			zap.Float64("previous_reading", previousMeterReading),
			zap.Float64("incremental_energy", incrementalEnergy))
		http.Error(w, "Negative incremental energy - meter reset or out-of-order reading", http.StatusBadRequest)
		return
	}

	// Add incremental energy to total_energy_wh
	updateQuery := `
		UPDATE chargers 
		SET total_energy_wh = COALESCE(total_energy_wh, 0) + ?
		WHERE id = ?
	`

	updateResult, err := api.db.ExecContext(r.Context(), updateQuery,
		int64(incrementalEnergy),
		chargerID,
	)

	if err != nil {
		api.logger.Error("Failed to update total_energy_wh", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := updateResult.RowsAffected()
	status := "updated"
	if rowsAffected == 0 {
		status = "no rows affected"
	}

	api.logger.Info("Simulated MeterValues update",
		zap.String("identity", req.Identity),
		zap.Int64("charger_id", chargerID),
		zap.Float64("current_meter_reading", req.ValueWh),
		zap.Float64("previous_meter_reading", previousMeterReading),
		zap.Float64("incremental_energy_wh", incrementalEnergy),
		zap.String("status", status),
	)

	// Return success response
	response := map[string]interface{}{
		"success":                true,
		"identity":               req.Identity,
		"charger_id":             chargerID,
		"current_meter_reading":  req.ValueWh,
		"previous_meter_reading": previousMeterReading,
		"incremental_energy_wh":  incrementalEnergy,
		"incremental_energy_kwh": incrementalEnergy / 1000.0,
		"timestamp":              timestamp.Format(time.RFC3339),
		"status":                 status,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
