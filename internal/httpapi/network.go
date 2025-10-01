package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// NetworkAPI handles simple network information endpoints
type NetworkAPI struct {
	logger *zap.Logger
}

// NewNetworkAPI creates a new network API instance
func NewNetworkAPI(logger *zap.Logger) *NetworkAPI {
	return &NetworkAPI{
		logger: logger,
	}
}

// Routes defines the network API routes
func (api *NetworkAPI) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/hotspot", api.getHotspotInfo)

	return r
}

// HotspotInfo represents the hotspot network information
type HotspotInfo struct {
	IPAddress   string `json:"ip_address"`
	SubnetMask  string `json:"subnet_mask"`
	IsReachable bool   `json:"is_reachable"`
}

// getHotspotInfo returns static hotspot information
func (api *NetworkAPI) getHotspotInfo(w http.ResponseWriter, r *http.Request) {
	api.logger.Info("Hotspot info requested")

	// Return static hotspot information
	hotspotInfo := HotspotInfo{
		IPAddress:   "192.168.137.1",
		SubnetMask:  "255.255.255.0",
		IsReachable: true, // Assume reachable for static display
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(hotspotInfo); err != nil {
		api.logger.Error("Failed to encode hotspot info", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	api.logger.Info("Hotspot info returned", zap.String("ip", hotspotInfo.IPAddress))
}
