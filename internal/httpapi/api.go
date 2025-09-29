package httpapi

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// Station represents a charging station
type Station struct {
	ID       int       `json:"id"`
	Identity string    `json:"identity"`
	Firmware string    `json:"firmware"`
	LastSeen time.Time `json:"last_seen"`
}

// API holds the database connection
type API struct {
	db *sql.DB
}

// New creates a new API instance
func New(db *sql.DB) *API {
	return &API{db: db}
}

// Routes sets up the API routes
func (api *API) Routes() chi.Router {
	r := chi.NewRouter()
	
	r.Get("/stations", api.getStations)
	
	return r
}

// getStations returns a list of all charging stations
func (api *API) getStations(w http.ResponseWriter, r *http.Request) {
	// For now, return mock data since we don't have the chargers table yet
	stations := []Station{
		{
			ID:       1,
			Identity: "CP001",
			Firmware: "1.6.0",
			LastSeen: time.Now().Add(-5 * time.Minute),
		},
		{
			ID:       2,
			Identity: "CP002", 
			Firmware: "1.6.0",
			LastSeen: time.Now().Add(-2 * time.Minute),
		},
		{
			ID:       3,
			Identity: "CP003",
			Firmware: "1.5.9",
			LastSeen: time.Now().Add(-10 * time.Minute),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stations)
}
