package ocpp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/lorenzodonini/ocpp-go/ocpp"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/lorenzodonini/ocpp-go/ocppj"
	"github.com/lorenzodonini/ocpp-go/ws"
	"go.uber.org/zap"
)

// Server handles OCPP 1.6J connections
type Server struct {
	db      *sql.DB
	logger  *zap.Logger
	cs      *ocppj.Server
	running bool
}

// New creates a new OCPP server
func New(db *sql.DB, logger *zap.Logger) *Server {
	// Create WebSocket server
	wsServer := ws.NewServer()

	// Create OCPP server with Core profile
	cs := ocppj.NewServer(wsServer, nil, nil, core.Profile)

	s := &Server{
		db:      db,
		logger:  logger,
		cs:      cs,
		running: true, // Server starts running by default
	}

	// Register handlers
	cs.SetRequestHandler(s.handleRequest)
	cs.SetNewClientHandler(s.handleNewClient)

	return s
}

// Mount mounts the OCPP server on the HTTP router
func (s *Server) Mount(r chi.Router) {
	// Create a simple HTTP handler that serves the OCPP WebSocket endpoint
	r.HandleFunc("/ocpp16/{id}", s.handleOCPPConnection)
	s.logger.Info("OCPP 1.6J server mounted at /ocpp16")
}

// Start starts the OCPP server
func (s *Server) Start() {
	s.running = true
	s.logger.Info("OCPP server started")
}

// Stop stops the OCPP server
func (s *Server) Stop() {
	s.running = false
	s.logger.Info("OCPP server stopped")
}

// IsRunning returns whether the server is currently running
func (s *Server) IsRunning() bool {
	return s.running
}

// handleOCPPConnection handles OCPP WebSocket connections
func (s *Server) handleOCPPConnection(w http.ResponseWriter, r *http.Request) {
	if !s.running {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error": "OCPP server is stopped"}`))
		return
	}

	// Get the charge point ID from the URL path
	chargePointId := chi.URLParam(r, "id")
	if chargePointId == "" {
		http.Error(w, "Missing charge point ID", http.StatusBadRequest)
		return
	}

	s.logger.Info("OCPP WebSocket connection attempt", zap.String("charge_point_id", chargePointId))

	// Upgrade HTTP connection to WebSocket
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for now
		},
		Subprotocols: []string{"ocpp1.6"},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade to WebSocket", zap.Error(err))
		return
	}
	defer conn.Close()

	s.logger.Info("OCPP WebSocket connection established", zap.String("charge_point_id", chargePointId))

	// Handle WebSocket messages
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Error("WebSocket connection closed unexpectedly", zap.Error(err))
			}
			s.logger.Info("OCPP WebSocket connection closed", zap.String("charge_point_id", chargePointId))
			break
		}

		// Process OCPP message
		response, err := s.processOCPPMessage(chargePointId, message)
		if err != nil {
			s.logger.Error("Failed to process OCPP message", zap.Error(err))
			continue
		}

		// Send response back to client
		if response != nil {
			if err := conn.WriteMessage(websocket.TextMessage, response); err != nil {
				s.logger.Error("Failed to send OCPP response", zap.Error(err))
				break
			}
		}
	}
}

// handleNewClient handles new client connections
func (s *Server) handleNewClient(client ws.Channel) {
	s.logger.Info("New OCPP client connected", zap.String("charge_point_id", client.ID()))
}

// handleRequest handles incoming OCPP requests
func (s *Server) handleRequest(client ws.Channel, request ocpp.Request, requestId string, action string) {
	chargePointId := client.ID()

	switch action {
	case "BootNotification":
		if req, ok := request.(*core.BootNotificationRequest); ok {
			confirmation, err := s.OnBootNotification(chargePointId, req)
			if err != nil {
				s.logger.Error("BootNotification error", zap.Error(err))
				return
			}
			s.cs.SendResponse(chargePointId, requestId, confirmation)
		}
	case "StatusNotification":
		if req, ok := request.(*core.StatusNotificationRequest); ok {
			confirmation, err := s.OnStatusNotification(chargePointId, req)
			if err != nil {
				s.logger.Error("StatusNotification error", zap.Error(err))
				return
			}
			s.cs.SendResponse(chargePointId, requestId, confirmation)
		}
	case "MeterValues":
		if req, ok := request.(*core.MeterValuesRequest); ok {
			confirmation, err := s.OnMeterValues(chargePointId, req)
			if err != nil {
				s.logger.Error("MeterValues error", zap.Error(err))
				return
			}
			s.cs.SendResponse(chargePointId, requestId, confirmation)
		}
	case "StartTransaction":
		if req, ok := request.(*core.StartTransactionRequest); ok {
			confirmation, err := s.OnStartTransaction(chargePointId, req)
			if err != nil {
				s.logger.Error("StartTransaction error", zap.Error(err))
				return
			}
			s.cs.SendResponse(chargePointId, requestId, confirmation)
		}
	case "StopTransaction":
		if req, ok := request.(*core.StopTransactionRequest); ok {
			confirmation, err := s.OnStopTransaction(chargePointId, req)
			if err != nil {
				s.logger.Error("StopTransaction error", zap.Error(err))
				return
			}
			s.cs.SendResponse(chargePointId, requestId, confirmation)
		}
	case "Authorize":
		if req, ok := request.(*core.AuthorizeRequest); ok {
			confirmation, err := s.OnAuthorize(chargePointId, req)
			if err != nil {
				s.logger.Error("Authorize error", zap.Error(err))
				return
			}
			s.cs.SendResponse(chargePointId, requestId, confirmation)
		}
	default:
		s.logger.Info("Unhandled OCPP action", zap.String("action", action))
	}
}

// OnBootNotification handles BootNotification requests
func (s *Server) OnBootNotification(chargePointId string, request *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
	s.logger.Info("BootNotification received",
		zap.String("charge_point_id", chargePointId),
		zap.String("model", request.ChargePointModel),
		zap.String("vendor", request.ChargePointVendor),
		zap.String("firmware", request.FirmwareVersion),
	)

	// Upsert charger
	query := `
		INSERT INTO chargers (identity, name, model, vendor, firmware, last_seen)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(identity) DO UPDATE SET
			model = excluded.model,
			vendor = excluded.vendor,
			firmware = excluded.firmware,
			last_seen = excluded.last_seen
	`

	_, err := s.db.ExecContext(context.Background(), query,
		chargePointId,
		chargePointId, // Use identity as name if not provided
		request.ChargePointModel,
		request.ChargePointVendor,
		request.FirmwareVersion,
		time.Now(),
	)

	if err != nil {
		s.logger.Error("Failed to upsert charger", zap.Error(err))
		return &core.BootNotificationConfirmation{
			Status:      core.RegistrationStatusRejected,
			CurrentTime: types.NewDateTime(time.Now()),
			Interval:    300,
		}, nil
	}

	return &core.BootNotificationConfirmation{
		Status:      core.RegistrationStatusAccepted,
		CurrentTime: types.NewDateTime(time.Now()),
		Interval:    300,
	}, nil
}

// OnStatusNotification handles StatusNotification requests
func (s *Server) OnStatusNotification(chargePointId string, request *core.StatusNotificationRequest) (*core.StatusNotificationConfirmation, error) {
	s.logger.Info("StatusNotification received",
		zap.String("charge_point_id", chargePointId),
		zap.String("status", string(request.Status)),
		zap.Int("connector_id", request.ConnectorId),
	)

	// Update last_seen
	query := `UPDATE chargers SET last_seen = ? WHERE identity = ?`
	_, err := s.db.ExecContext(context.Background(), query, time.Now(), chargePointId)
	if err != nil {
		s.logger.Error("Failed to update charger last_seen", zap.Error(err))
	}

	return &core.StatusNotificationConfirmation{}, nil
}

// OnStartTransaction handles StartTransaction requests
func (s *Server) OnStartTransaction(chargePointId string, request *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
	s.logger.Info("StartTransaction received",
		zap.String("charge_point_id", chargePointId),
		zap.String("tx_id", request.IdTag),
		zap.Int("meter_start", request.MeterStart),
	)

	// Get charger ID
	var chargerID int64
	err := s.db.QueryRowContext(context.Background(), "SELECT id FROM chargers WHERE identity = ?", chargePointId).Scan(&chargerID)
	if err != nil {
		s.logger.Error("Failed to get charger ID", zap.Error(err))
		return &core.StartTransactionConfirmation{
			IdTagInfo: &types.IdTagInfo{
				Status: types.AuthorizationStatusInvalid,
			},
		}, nil
	}

	// Generate transaction ID
	txID := int(time.Now().Unix())

	// Insert transaction
	query := `
		INSERT INTO transactions (charger_id, tx_id, start_ts, start_meter_wh)
		VALUES (?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(context.Background(), query,
		chargerID,
		txID,
		request.Timestamp.Time,
		request.MeterStart,
	)

	if err != nil {
		s.logger.Error("Failed to insert transaction", zap.Error(err))
		return &core.StartTransactionConfirmation{
			IdTagInfo: &types.IdTagInfo{
				Status: types.AuthorizationStatusInvalid,
			},
		}, nil
	}

	return &core.StartTransactionConfirmation{
		TransactionId: txID,
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted,
		},
	}, nil
}

// OnStopTransaction handles StopTransaction requests
func (s *Server) OnStopTransaction(chargePointId string, request *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
	s.logger.Info("StopTransaction received",
		zap.String("charge_point_id", chargePointId),
		zap.Int("tx_id", request.TransactionId),
		zap.Int("meter_stop", request.MeterStop),
	)

	// Update transaction
	query := `
		UPDATE transactions 
		SET stop_ts = ?, stop_meter_wh = ?, energy_wh = MAX(0, ? - start_meter_wh)
		WHERE tx_id = ?
	`

	_, err := s.db.ExecContext(context.Background(), query,
		request.Timestamp.Time,
		request.MeterStop,
		request.MeterStop,
		request.TransactionId,
	)

	if err != nil {
		s.logger.Error("Failed to update transaction", zap.Error(err))
	}

	return &core.StopTransactionConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted,
		},
	}, nil
}

// OnMeterValues handles MeterValues requests
func (s *Server) OnMeterValues(chargePointId string, request *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
	s.logger.Info("MeterValues received",
		zap.String("charge_point_id", chargePointId),
		zap.Int("tx_id", *request.TransactionId),
		zap.Int("meter_values", len(request.MeterValue)),
	)

	if request.TransactionId == nil {
		return &core.MeterValuesConfirmation{}, nil
	}

	// Get transaction ID
	var transactionID int64
	err := s.db.QueryRowContext(context.Background(), "SELECT id FROM transactions WHERE tx_id = ?", *request.TransactionId).Scan(&transactionID)
	if err != nil {
		s.logger.Error("Failed to get transaction ID", zap.Error(err))
		return &core.MeterValuesConfirmation{}, nil
	}

	// Get charger ID for updating total_energy_wh
	var chargerID int64
	err = s.db.QueryRowContext(context.Background(), "SELECT charger_id FROM transactions WHERE id = ?", transactionID).Scan(&chargerID)
	if err != nil {
		s.logger.Error("Failed to get charger ID", zap.Error(err))
		return &core.MeterValuesConfirmation{}, nil
	}

	// Process meter values and update total_energy_wh
	for _, mv := range request.MeterValue {
		for _, sample := range mv.SampledValue {
			if sample.Measurand == "Energy.Active.Import.Register" {
				value, err := strconv.ParseFloat(sample.Value, 64)
				if err != nil {
					s.logger.Error("Failed to parse meter value", zap.Error(err))
					continue
				}

				// Convert to Wh if needed (assume kWh if value > 1000)
				valueWh := value
				if value > 1000 {
					valueWh = value * 1000
				}

				// Insert meter value
				insertQuery := `
					INSERT INTO meter_values (transaction_id, ts, measurand, value)
					VALUES (?, ?, ?, ?)
				`

				_, err = s.db.ExecContext(context.Background(), insertQuery,
					transactionID,
					mv.Timestamp.Time,
					sample.Measurand,
					valueWh,
				)

				if err != nil {
					s.logger.Error("Failed to insert meter value", zap.Error(err))
					continue
				}

				// Update total_energy_wh using monotonic cumulative register logic
				// Only update if new_value_wh >= current_total
				updateQuery := `
					UPDATE chargers 
					SET total_energy_wh = ? 
					WHERE id = ? AND (? >= total_energy_wh OR total_energy_wh IS NULL)
				`

				result, err := s.db.ExecContext(context.Background(), updateQuery,
					int64(valueWh),
					chargerID,
					int64(valueWh),
				)

				if err != nil {
					s.logger.Error("Failed to update total_energy_wh", zap.Error(err))
					continue
				}

				rowsAffected, _ := result.RowsAffected()
				if rowsAffected > 0 {
					s.logger.Info("Updated total_energy_wh",
						zap.String("charge_point_id", chargePointId),
						zap.Int64("charger_id", chargerID),
						zap.Float64("new_value_wh", valueWh),
					)
				} else {
					s.logger.Info("Ignored decreasing total_energy_wh",
						zap.String("charge_point_id", chargePointId),
						zap.Int64("charger_id", chargerID),
						zap.Float64("new_value_wh", valueWh),
					)
				}
			}
		}
	}

	return &core.MeterValuesConfirmation{}, nil
}

// OnAuthorize handles Authorize requests
func (s *Server) OnAuthorize(chargePointId string, request *core.AuthorizeRequest) (*core.AuthorizeConfirmation, error) {
	s.logger.Info("Authorize received",
		zap.String("charge_point_id", chargePointId),
		zap.String("id_tag", request.IdTag),
	)

	return &core.AuthorizeConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted,
		},
	}, nil
}

// OnDataTransfer handles DataTransfer requests
func (s *Server) OnDataTransfer(chargePointId string, request *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
	s.logger.Info("DataTransfer received",
		zap.String("charge_point_id", chargePointId),
		zap.String("vendor_id", request.VendorId),
		zap.String("message_id", request.MessageId),
	)

	return &core.DataTransferConfirmation{
		Status: core.DataTransferStatusAccepted,
	}, nil
}

// OnChangeAvailability handles ChangeAvailability requests
func (s *Server) OnChangeAvailability(chargePointId string, request *core.ChangeAvailabilityRequest) (*core.ChangeAvailabilityConfirmation, error) {
	s.logger.Info("ChangeAvailability received",
		zap.String("charge_point_id", chargePointId),
		zap.String("type", string(request.Type)),
	)

	return &core.ChangeAvailabilityConfirmation{
		Status: core.AvailabilityStatusAccepted,
	}, nil
}

// OnChangeConfiguration handles ChangeConfiguration requests
func (s *Server) OnChangeConfiguration(chargePointId string, request *core.ChangeConfigurationRequest) (*core.ChangeConfigurationConfirmation, error) {
	s.logger.Info("ChangeConfiguration received",
		zap.String("charge_point_id", chargePointId),
		zap.String("key", request.Key),
	)

	return &core.ChangeConfigurationConfirmation{
		Status: core.ConfigurationStatusAccepted,
	}, nil
}

// OnClearCache handles ClearCache requests
func (s *Server) OnClearCache(chargePointId string, request *core.ClearCacheRequest) (*core.ClearCacheConfirmation, error) {
	s.logger.Info("ClearCache received", zap.String("charge_point_id", chargePointId))

	return &core.ClearCacheConfirmation{
		Status: core.ClearCacheStatusAccepted,
	}, nil
}

// OnGetConfiguration handles GetConfiguration requests
func (s *Server) OnGetConfiguration(chargePointId string, request *core.GetConfigurationRequest) (*core.GetConfigurationConfirmation, error) {
	s.logger.Info("GetConfiguration received", zap.String("charge_point_id", chargePointId))

	return &core.GetConfigurationConfirmation{
		ConfigurationKey: []core.ConfigurationKey{},
		UnknownKey:       request.Key,
	}, nil
}

// OnRemoteStartTransaction handles RemoteStartTransaction requests
func (s *Server) OnRemoteStartTransaction(chargePointId string, request *core.RemoteStartTransactionRequest) (*core.RemoteStartTransactionConfirmation, error) {
	s.logger.Info("RemoteStartTransaction received",
		zap.String("charge_point_id", chargePointId),
		zap.String("id_tag", request.IdTag),
	)

	return &core.RemoteStartTransactionConfirmation{
		Status: types.RemoteStartStopStatusAccepted,
	}, nil
}

// OnRemoteStopTransaction handles RemoteStopTransaction requests
func (s *Server) OnRemoteStopTransaction(chargePointId string, request *core.RemoteStopTransactionRequest) (*core.RemoteStopTransactionConfirmation, error) {
	s.logger.Info("RemoteStopTransaction received",
		zap.String("charge_point_id", chargePointId),
		zap.Int("tx_id", request.TransactionId),
	)

	return &core.RemoteStopTransactionConfirmation{
		Status: types.RemoteStartStopStatusAccepted,
	}, nil
}

// OnReset handles Reset requests
func (s *Server) OnReset(chargePointId string, request *core.ResetRequest) (*core.ResetConfirmation, error) {
	s.logger.Info("Reset received",
		zap.String("charge_point_id", chargePointId),
		zap.String("type", string(request.Type)),
	)

	return &core.ResetConfirmation{
		Status: core.ResetStatusAccepted,
	}, nil
}

// OnUnlockConnector handles UnlockConnector requests
func (s *Server) OnUnlockConnector(chargePointId string, request *core.UnlockConnectorRequest) (*core.UnlockConnectorConfirmation, error) {
	s.logger.Info("UnlockConnector received",
		zap.String("charge_point_id", chargePointId),
		zap.Int("connector_id", request.ConnectorId),
	)

	return &core.UnlockConnectorConfirmation{
		Status: core.UnlockStatusUnlocked,
	}, nil
}

// processOCPPMessage processes incoming OCPP messages
func (s *Server) processOCPPMessage(chargePointId string, message []byte) ([]byte, error) {
	s.logger.Info("Processing OCPP message", 
		zap.String("charge_point_id", chargePointId),
		zap.String("message", string(message)))

	// Parse the OCPP message
	var ocppMessage []interface{}
	if err := json.Unmarshal(message, &ocppMessage); err != nil {
		return nil, err
	}

	if len(ocppMessage) < 3 {
		return nil, fmt.Errorf("invalid OCPP message format")
	}

	messageType, ok := ocppMessage[0].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid message type")
	}

	messageId, ok := ocppMessage[1].(string)
	if !ok {
		return nil, fmt.Errorf("invalid message ID")
	}

	action, ok := ocppMessage[2].(string)
	if !ok {
		return nil, fmt.Errorf("invalid action")
	}

	// Handle different message types
	switch int(messageType) {
	case 2: // CALL (request from charge point)
		return s.handleOCPPRequest(chargePointId, messageId, action, ocppMessage[3])
	case 3: // CALLRESULT (response from charge point)
		return nil, nil // We don't handle responses for now
	case 4: // CALLERROR (error from charge point)
		return nil, nil // We don't handle errors for now
	default:
		return nil, fmt.Errorf("unknown message type: %d", int(messageType))
	}
}

// handleOCPPRequest handles incoming OCPP requests
func (s *Server) handleOCPPRequest(chargePointId, messageId, action string, payload interface{}) ([]byte, error) {
	s.logger.Info("Handling OCPP request",
		zap.String("charge_point_id", chargePointId),
		zap.String("action", action),
		zap.String("message_id", messageId))

	var response interface{}

	switch action {
	case "BootNotification":
		response = s.handleBootNotificationRequest(chargePointId, payload)
	case "StatusNotification":
		response = s.handleStatusNotificationRequest(chargePointId, payload)
	case "MeterValues":
		response = s.handleMeterValuesRequest(chargePointId, payload)
	case "StartTransaction":
		response = s.handleStartTransactionRequest(chargePointId, payload)
	case "StopTransaction":
		response = s.handleStopTransactionRequest(chargePointId, payload)
	case "Authorize":
		response = s.handleAuthorizeRequest(chargePointId, payload)
	default:
		s.logger.Info("Unhandled OCPP action", zap.String("action", action))
		// Return a generic error response
		errorResponse := []interface{}{
			4, // CALLERROR
			messageId,
			"NotImplemented",
			"Action not implemented",
			"{}",
		}
		return json.Marshal(errorResponse)
	}

	// Create CALLRESULT response
	callResult := []interface{}{
		3, // CALLRESULT
		messageId,
		response,
	}

	return json.Marshal(callResult)
}

// handleBootNotificationRequest handles BootNotification requests
func (s *Server) handleBootNotificationRequest(chargePointId string, payload interface{}) interface{} {
	s.logger.Info("BootNotification received", zap.String("charge_point_id", chargePointId))

	// Parse payload
	payloadMap, ok := payload.(map[string]interface{})
	if !ok {
		s.logger.Error("Invalid BootNotification payload")
		return map[string]interface{}{
			"status": "Rejected",
		}
	}

	chargePointModel, _ := payloadMap["chargePointModel"].(string)
	chargePointVendor, _ := payloadMap["chargePointVendor"].(string)
	firmwareVersion, _ := payloadMap["firmwareVersion"].(string)

	// Upsert charger
	query := `
		INSERT INTO chargers (identity, name, model, vendor, firmware, last_seen)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(identity) DO UPDATE SET
			model = excluded.model,
			vendor = excluded.vendor,
			firmware = excluded.firmware,
			last_seen = excluded.last_seen
	`

	_, err := s.db.ExecContext(context.Background(), query,
		chargePointId,
		chargePointId, // Use identity as name if not provided
		chargePointModel,
		chargePointVendor,
		firmwareVersion,
		time.Now(),
	)

	if err != nil {
		s.logger.Error("Failed to upsert charger", zap.Error(err))
		return map[string]interface{}{
			"status": "Rejected",
		}
	}

	return map[string]interface{}{
		"status":      "Accepted",
		"currentTime": time.Now().Format(time.RFC3339),
		"interval":    300,
	}
}

// handleStatusNotificationRequest handles StatusNotification requests
func (s *Server) handleStatusNotificationRequest(chargePointId string, payload interface{}) interface{} {
	s.logger.Info("StatusNotification received", zap.String("charge_point_id", chargePointId))

	// Update last_seen
	query := `UPDATE chargers SET last_seen = ? WHERE identity = ?`
	_, err := s.db.ExecContext(context.Background(), query, time.Now(), chargePointId)
	if err != nil {
		s.logger.Error("Failed to update charger last_seen", zap.Error(err))
	}

	return map[string]interface{}{}
}

// handleMeterValuesRequest handles MeterValues requests
func (s *Server) handleMeterValuesRequest(chargePointId string, payload interface{}) interface{} {
	s.logger.Info("MeterValues received", zap.String("charge_point_id", chargePointId))

	// For now, just acknowledge the message
	// TODO: Implement proper meter values processing
	return map[string]interface{}{}
}

// handleStartTransactionRequest handles StartTransaction requests
func (s *Server) handleStartTransactionRequest(chargePointId string, payload interface{}) interface{} {
	s.logger.Info("StartTransaction received", zap.String("charge_point_id", chargePointId))

	// Parse payload
	payloadMap, ok := payload.(map[string]interface{})
	if !ok {
		s.logger.Error("Invalid StartTransaction payload")
		return map[string]interface{}{
			"idTagInfo": map[string]interface{}{
				"status": "Invalid",
			},
		}
	}

	_, _ = payloadMap["idTag"].(string) // idTag not used for now
	meterStart, _ := payloadMap["meterStart"].(float64)

	// Get charger ID
	var chargerID int64
	err := s.db.QueryRowContext(context.Background(), "SELECT id FROM chargers WHERE identity = ?", chargePointId).Scan(&chargerID)
	if err != nil {
		s.logger.Error("Failed to get charger ID", zap.Error(err))
		return map[string]interface{}{
			"idTagInfo": map[string]interface{}{
				"status": "Invalid",
			},
		}
	}

	// Generate transaction ID
	txID := int(time.Now().Unix())

	// Insert transaction
	query := `
		INSERT INTO transactions (charger_id, tx_id, start_ts, start_meter_wh)
		VALUES (?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(context.Background(), query,
		chargerID,
		txID,
		time.Now(),
		int64(meterStart),
	)

	if err != nil {
		s.logger.Error("Failed to insert transaction", zap.Error(err))
		return map[string]interface{}{
			"idTagInfo": map[string]interface{}{
				"status": "Invalid",
			},
		}
	}

	return map[string]interface{}{
		"transactionId": txID,
		"idTagInfo": map[string]interface{}{
			"status": "Accepted",
		},
	}
}

// handleStopTransactionRequest handles StopTransaction requests
func (s *Server) handleStopTransactionRequest(chargePointId string, payload interface{}) interface{} {
	s.logger.Info("StopTransaction received", zap.String("charge_point_id", chargePointId))

	// Parse payload
	payloadMap, ok := payload.(map[string]interface{})
	if !ok {
		s.logger.Error("Invalid StopTransaction payload")
		return map[string]interface{}{
			"idTagInfo": map[string]interface{}{
				"status": "Invalid",
			},
		}
	}

	transactionId, _ := payloadMap["transactionId"].(float64)
	meterStop, _ := payloadMap["meterStop"].(float64)

	// Update transaction
	query := `
		UPDATE transactions 
		SET stop_ts = ?, stop_meter_wh = ?, energy_wh = MAX(0, ? - start_meter_wh)
		WHERE tx_id = ?
	`

	_, err := s.db.ExecContext(context.Background(), query,
		time.Now(),
		int64(meterStop),
		int64(meterStop),
		int64(transactionId),
	)

	if err != nil {
		s.logger.Error("Failed to update transaction", zap.Error(err))
	}

	return map[string]interface{}{
		"idTagInfo": map[string]interface{}{
			"status": "Accepted",
		},
	}
}

// handleAuthorizeRequest handles Authorize requests
func (s *Server) handleAuthorizeRequest(chargePointId string, payload interface{}) interface{} {
	s.logger.Info("Authorize received", zap.String("charge_point_id", chargePointId))

	return map[string]interface{}{
		"idTagInfo": map[string]interface{}{
			"status": "Accepted",
		},
	}
}
