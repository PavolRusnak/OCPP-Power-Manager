package httpapi

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// SeedAPI handles seeding endpoints
type SeedAPI struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewSeedAPI creates a new seed API
func NewSeedAPI(db *sql.DB, logger *zap.Logger) *SeedAPI {
	return &SeedAPI{
		db:     db,
		logger: logger,
	}
}

// Routes returns the routes for the seed API
func (api *SeedAPI) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/stations", api.SeedStations)
	return r
}

// SeedStations handles POST /api/seed/stations
func (api *SeedAPI) SeedStations(w http.ResponseWriter, r *http.Request) {
	// Check if any stations already exist
	var count int
	err := api.db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM chargers").Scan(&count)
	if err != nil {
		api.logger.Error("Failed to count existing stations", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if count > 0 {
		http.Error(w, "Stations already exist", http.StatusConflict)
		return
	}

	// Insert demo stations
	stations := []struct {
		identity    string
		name        string
		model       string
		vendor      string
		maxOutputKW float64
	}{
		{"CP-001", "Demo Station 1", "Demo-7kW", "DemoVendor", 7.2},
		{"CP-002", "Demo Station 2", "Demo-22kW", "DemoVendor", 22.0},
		{"CP-003", "Demo Station 3", "Demo-50kW", "DemoVendor", 50.0},
	}

	query := `
		INSERT INTO chargers (identity, name, model, vendor, max_output_kw)
		VALUES (?, ?, ?, ?, ?)
	`

	for _, station := range stations {
		_, err := api.db.ExecContext(r.Context(), query,
			station.identity,
			station.name,
			station.model,
			station.vendor,
			station.maxOutputKW,
		)
		if err != nil {
			api.logger.Error("Failed to seed station", zap.Error(err), zap.String("identity", station.identity))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	api.logger.Info("Seeded demo stations", zap.Int("count", len(stations)))
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"message": "Demo stations created successfully"}`))
}
