package httpapi

import (
	"database/sql"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// OCPPServer interface for controlling OCPP server
type OCPPServer interface {
	Start()
	Stop()
	IsRunning() bool
}

// API holds the API dependencies
type API struct {
	db         *sql.DB
	logger     *zap.Logger
	ocppServer OCPPServer
}

// New creates a new API instance
func New(db *sql.DB, logger *zap.Logger, ocppServer OCPPServer) *API {
	return &API{
		db:         db,
		logger:     logger,
		ocppServer: ocppServer,
	}
}

// Routes defines the API routes
func (a *API) Routes() chi.Router {
	r := chi.NewRouter()

	// Mount sub-APIs
	r.Mount("/stations", NewStationsAPI(a.db, a.logger).Routes())
	r.Mount("/seed", NewSeedAPI(a.db, a.logger).Routes())
	r.Mount("/dev", NewDevAPI(a.db, a.logger).Routes())
	r.Mount("/settings", NewSettingsAPI(a.db, a.logger, a.ocppServer).Routes())

	return r
}
