package httpapi

import (
	"database/sql"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// API holds the API dependencies
type API struct {
	db     *sql.DB
	logger *zap.Logger
}

// New creates a new API instance
func New(db *sql.DB, logger *zap.Logger) *API {
	return &API{
		db:     db,
		logger: logger,
	}
}

// Routes defines the API routes
func (a *API) Routes() chi.Router {
	r := chi.NewRouter()

	// Mount sub-APIs
	r.Mount("/stations", NewStationsAPI(a.db, a.logger).Routes())
	r.Mount("/seed", NewSeedAPI(a.db, a.logger).Routes())

	return r
}
