package ocpp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
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

// Server manages communication with electric vehicle charging stations
// It handles WebSocket connections and processes OCPP messages from chargers
type Server struct {
	db          *sql.DB                    // Database connection for storing charger data
	logger      *zap.Logger                // Logger for recording events and errors
	cs          *ocppj.Server              // OCPP library server (not currently used)
	running     bool                       // Whether the server is active and accepting connections
	connections map[string]*websocket.Conn // Map of charger ID to WebSocket connection
}

// New creates a new server to handle charging station connections
// It sets up the basic structure but we use our own WebSocket handling instead of the OCPP library
func New(db *sql.DB, logger *zap.Logger) *Server {
	// Create WebSocket server (not used in our implementation)
	wsServer := ws.NewServer()

	// Create OCPP server with Core profile (not used in our implementation)
	cs := ocppj.NewServer(wsServer, nil, nil, core.Profile)

	s := &Server{
		db:          db,
		logger:      logger,
		cs:          cs,
		running:     true, // Server is ready to accept connections
		connections: make(map[string]*websocket.Conn),
	}

	// Register handlers (not used since we handle WebSocket manually)
	cs.SetRequestHandler(s.handleRequest)
	cs.SetNewClientHandler(s.handleNewClient)

	return s
}

// Mount sets up the WebSocket endpoint where charging stations can connect
// Chargers will connect to /ocpp16/{their_id} to start communicating
func (s *Server) Mount(r chi.Router) {
	// Set up the WebSocket endpoint for charger connections
	r.HandleFunc("/ocpp16/{id}", s.handleOCPPConnection)
	s.logger.Info("🔌 OCPP 1.6J server mounted at /ocpp16 - Ready for EV charging stations")
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

// IsRunning returns whether the OCPP server is running
func (s *Server) IsRunning() bool {
	return s.running
}

// handleOCPPConnection handles WebSocket connections from charging stations
func (s *Server) handleOCPPConnection(w http.ResponseWriter, r *http.Request) {
	// Get the charger ID from the URL parameter
	chargerID := chi.URLParam(r, "id")
	if chargerID == "" {
		http.Error(w, "Charger ID is required", http.StatusBadRequest)
		return
	}

	s.logger.Info("New WebSocket connection attempt", zap.String("charger_id", chargerID))

	// Upgrade HTTP connection to WebSocket
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for now
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade connection to WebSocket", zap.Error(err))
		return
	}
	defer conn.Close()

	s.logger.Info("WebSocket connection established", zap.String("charger_id", chargerID))

	// Store the connection for this charger
	s.connections[chargerID] = conn

	// Handle WebSocket messages
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Error("WebSocket error", zap.Error(err))
			}
			break
		}

		// Process the OCPP message
		response, err := s.processOCPPMessage(chargerID, message)
		if err != nil {
			s.logger.Error("Failed to process OCPP message", zap.Error(err))
			continue
		}

		// Send response back to charger
		if response != nil {
			if err := conn.WriteMessage(websocket.TextMessage, response); err != nil {
				s.logger.Error("Failed to send response to charger", zap.Error(err))
				break
			}
		}
	}

	// Remove the connection when it's closed
	delete(s.connections, chargerID)
	s.logger.Info("WebSocket connection closed", zap.String("charger_id", chargerID))
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
	var txIDStr string
	if request.TransactionId != nil {
		txIDStr = fmt.Sprintf("%d", *request.TransactionId)
	} else {
		txIDStr = "none"
	}

	s.logger.Info("MeterValues received",
		zap.String("charge_point_id", chargePointId),
		zap.String("tx_id", txIDStr),
		zap.Int("meter_values", len(request.MeterValue)),
	)

	// Get charger ID for updating total_energy_wh
	var chargerID int64
	err := s.db.QueryRowContext(context.Background(), "SELECT id FROM chargers WHERE identity = ?", chargePointId).Scan(&chargerID)
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

				// Use the value as-is since it's already in Wh
				valueWh := value

				// If we have a transaction ID, insert meter value into the database
				if request.TransactionId != nil {
					// Get transaction ID
					var transactionID int64
					err = s.db.QueryRowContext(context.Background(), "SELECT id FROM transactions WHERE tx_id = ?", *request.TransactionId).Scan(&transactionID)
					if err != nil {
						s.logger.Error("Failed to get transaction ID", zap.Error(err))
						continue
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
				}

				// Set total_energy_wh to the current cumulative meter reading
				// Energy.Active.Import.Register is the total energy delivered by this charger since installation
				updateQuery := `
					UPDATE chargers 
					SET total_energy_wh = ?
					WHERE id = ?
				`

				result, err := s.db.ExecContext(context.Background(), updateQuery,
					int64(valueWh),
					chargerID,
				)

				if err != nil {
					s.logger.Error("Failed to update total_energy_wh", zap.Error(err))
					continue
				}

				rowsAffected, _ := result.RowsAffected()
				if rowsAffected > 0 {
					s.logger.Info("Updated total_energy_wh from charger's cumulative meter reading",
						zap.String("charge_point_id", chargePointId),
						zap.Int64("charger_id", chargerID),
						zap.Float64("meter_reading_wh", valueWh),
						zap.Float64("meter_reading_kwh", valueWh/1000.0),
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

// processOCPPMessage takes a raw message from the charger and figures out what to do with it
// OCPP messages are JSON arrays with specific formats for different types of communication
func (s *Server) processOCPPMessage(chargePointId string, message []byte) ([]byte, error) {
	s.logger.Info("Processing message from charger",
		zap.String("charge_point_id", chargePointId),
		zap.String("message", string(message)))

	// Parse the JSON message from the charger
	var ocppMessage []interface{}
	if err := json.Unmarshal(message, &ocppMessage); err != nil {
		return nil, err
	}

	// OCPP messages must have at least 3 parts: [messageType, messageId, action]
	if len(ocppMessage) < 3 {
		return nil, fmt.Errorf("invalid OCPP message format")
	}

	// Get the message type (2 = request, 3 = response, 4 = error)
	messageType, ok := ocppMessage[0].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid message type")
	}

	// Get the unique message ID (used to match requests with responses)
	messageId, ok := ocppMessage[1].(string)
	if !ok {
		return nil, fmt.Errorf("invalid message ID")
	}

	// Get the action (what the charger wants to do)
	action, ok := ocppMessage[2].(string)
	if !ok {
		return nil, fmt.Errorf("invalid action")
	}

	// Handle different types of messages from the charger
	switch int(messageType) {
	case 2: // CALL - The charger is asking us to do something
		return s.handleOCPPRequest(chargePointId, messageId, action, ocppMessage[3])
	case 3: // CALLRESULT - The charger is responding to something we asked
		return nil, nil // We don't send commands to chargers yet, so ignore responses
	case 4: // CALLERROR - The charger is telling us there was an error
		return nil, nil // We don't handle errors from chargers yet
	default:
		return nil, fmt.Errorf("unknown message type: %d", int(messageType))
	}
}

// handleOCPPRequest figures out what the charger wants and calls the right handler function
// This is like a dispatcher that routes different types of requests to the appropriate handler
func (s *Server) handleOCPPRequest(chargePointId, messageId, action string, payload interface{}) ([]byte, error) {
	s.logger.Info("Charger is requesting something",
		zap.String("charge_point_id", chargePointId),
		zap.String("action", action),
		zap.String("message_id", messageId))

	var response interface{}

	// Route the request to the appropriate handler based on what the charger wants to do
	switch action {
	case "BootNotification":
		// Charger is starting up and wants to register
		response = s.handleBootNotificationRequest(chargePointId, payload)
	case "StatusNotification":
		// Charger is sending a status update (available, charging, error, etc.)
		response = s.handleStatusNotificationRequest(chargePointId, payload)
	case "MeterValues":
		// Charger is sending energy meter readings
		response = s.handleMeterValuesRequest(chargePointId, payload)
	case "StartTransaction":
		// Someone started charging their car
		response = s.handleStartTransactionRequest(chargePointId, payload)
	case "StopTransaction":
		// Someone stopped charging their car
		response = s.handleStopTransactionRequest(chargePointId, payload)
	case "Authorize":
		// Someone is trying to use their RFID card
		response = s.handleAuthorizeRequest(chargePointId, payload)
	case "Heartbeat":
		// Charger is checking if we're still alive and updating its last seen time
		response = s.handleHeartbeatRequest(chargePointId, payload)
	default:
		// We don't know how to handle this type of request
		s.logger.Info("Unknown request type from charger", zap.String("action", action))
		// Send back an error saying we don't support this
		errorResponse := []interface{}{
			4, // CALLERROR
			messageId,
			"NotImplemented",
			"Action not implemented",
			"{}",
		}
		return json.Marshal(errorResponse)
	}

	// Send back a success response to the charger
	callResult := []interface{}{
		3, // CALLRESULT (success response)
		messageId,
		response,
	}

	return json.Marshal(callResult)
}

// handleBootNotificationRequest handles when a charging station first connects or reboots
// This registers the charger in our database and tells it how often to send status updates
func (s *Server) handleBootNotificationRequest(chargePointId string, payload interface{}) interface{} {
	s.logger.Info("Charging station booted up", zap.String("charge_point_id", chargePointId))

	// Parse the charger information
	payloadMap, ok := payload.(map[string]interface{})
	if !ok {
		s.logger.Error("Invalid charger boot data")
		return map[string]interface{}{
			"status": "Rejected",
		}
	}

	// Get charger details
	chargePointModel, _ := payloadMap["chargePointModel"].(string)
	chargePointVendor, _ := payloadMap["chargePointVendor"].(string)
	firmwareVersion, _ := payloadMap["firmwareVersion"].(string)

	// Add or update the charger in our database
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
		chargePointId, // Use the ID as the name if no name provided
		chargePointModel,
		chargePointVendor,
		firmwareVersion,
		time.Now(),
	)

	if err != nil {
		s.logger.Error("Failed to register charger", zap.Error(err))
		return map[string]interface{}{
			"status": "Rejected",
		}
	}

	s.logger.Info("Charger registered successfully",
		zap.String("charge_point_id", chargePointId),
		zap.String("model", chargePointModel),
		zap.String("vendor", chargePointVendor))

	// Tell the charger we accept it and how often to send status updates (every 5 minutes)
	return map[string]interface{}{
		"status":      "Accepted",
		"currentTime": time.Now().Format(time.RFC3339),
		"interval":    300, // Send status updates every 5 minutes
	}
}

// handleStatusNotificationRequest handles regular status updates from the charging station
// This tells us if the charger is available, charging, or has an error
func (s *Server) handleStatusNotificationRequest(chargePointId string, payload interface{}) interface{} {
	s.logger.Info("Charger status update received", zap.String("charge_point_id", chargePointId))

	// Update when we last heard from this charger
	query := `UPDATE chargers SET last_seen = ? WHERE identity = ?`
	_, err := s.db.ExecContext(context.Background(), query, time.Now(), chargePointId)
	if err != nil {
		s.logger.Error("Failed to update charger last seen time", zap.Error(err))
	}

	// Parse the status to check if charger just became available (connected)
	payloadMap, ok := payload.(map[string]interface{})
	if ok {
		status, _ := payloadMap["status"].(string)
		connectorId, _ := payloadMap["connectorId"].(float64)

		s.logger.Info("Status notification details",
			zap.String("charge_point_id", chargePointId),
			zap.String("status", status),
			zap.Float64("connector_id", connectorId))

		// When connector 1 reports Available, request meter values
		if status == "Available" && connectorId == 1 {
			s.logger.Info("Charger connector 1 available, will trigger meter values request after response",
				zap.String("charge_point_id", chargePointId))
			// Trigger meter values after a short delay to avoid concurrent writes
			go func() {
				time.Sleep(100 * time.Millisecond)
				s.sendTriggerMessage(chargePointId, "MeterValues")
			}()
		}
	}

	return map[string]interface{}{}
}

// handleMeterValuesRequest processes energy meter readings from the charging station
// This is where we get the actual energy consumption data and update our database
func (s *Server) handleMeterValuesRequest(chargePointId string, payload interface{}) interface{} {
	s.logger.Info("Energy meter reading received from charger", zap.String("charge_point_id", chargePointId))

	// Update when we last heard from this charger (for online/offline status)
	query := `UPDATE chargers SET last_seen = ? WHERE identity = ?`
	_, err := s.db.ExecContext(context.Background(), query, time.Now(), chargePointId)
	if err != nil {
		s.logger.Error("Failed to update charger last seen time", zap.Error(err))
	}

	// Parse the meter values data from the charger
	payloadMap, ok := payload.(map[string]interface{})
	if !ok {
		s.logger.Error("Invalid meter values data format")
		return map[string]interface{}{}
	}

	// Get the meter values array
	meterValues, ok := payloadMap["meterValue"].([]interface{})
	if !ok || len(meterValues) == 0 {
		s.logger.Info("No meter values in message")
		return map[string]interface{}{}
	}

	// Process each meter reading
	for _, mv := range meterValues {
		meterValue, ok := mv.(map[string]interface{})
		if !ok {
			continue
		}

		// Get the timestamp (for future use in storing meter readings)
		timestampStr, _ := meterValue["timestamp"].(string)
		_, err := time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			s.logger.Error("Invalid timestamp in meter values", zap.Error(err))
			continue
		}

		// Get the sampled values (the actual readings)
		sampledValues, ok := meterValue["sampledValue"].([]interface{})
		if !ok {
			continue
		}

		for _, sv := range sampledValues {
			sampledValue, ok := sv.(map[string]interface{})
			if !ok {
				continue
			}

			// Look for energy readings
			measurand, _ := sampledValue["measurand"].(string)
			if measurand == "Energy.Active.Import.Register" {
				valueStr, _ := sampledValue["value"].(string)
				value, err := strconv.ParseFloat(valueStr, 64)
				if err != nil {
					s.logger.Error("Invalid energy value", zap.Error(err))
					continue
				}

				// Use the value as-is since it's already in Wh
				valueWh := value

				// Find the charger in our database
				var chargerID int64
				err = s.db.QueryRowContext(context.Background(), "SELECT id FROM chargers WHERE identity = ?", chargePointId).Scan(&chargerID)
				if err != nil {
					s.logger.Error("Charger not found in database", zap.Error(err))
					continue
				}

				// Set total_energy_wh to the current cumulative meter reading
				// Energy.Active.Import.Register is the total energy delivered by this charger since installation
				updateQuery := `
					UPDATE chargers 
					SET total_energy_wh = ?
					WHERE id = ?
				`

				result, err := s.db.ExecContext(context.Background(), updateQuery,
					int64(valueWh),
					chargerID,
				)

				if err != nil {
					s.logger.Error("Failed to update energy reading", zap.Error(err))
					continue
				}

				rowsAffected, _ := result.RowsAffected()
				if rowsAffected > 0 {
					s.logger.Info("Updated total_energy_wh from charger's cumulative meter reading",
						zap.String("charge_point_id", chargePointId),
						zap.Float64("meter_reading_wh", valueWh),
						zap.Float64("meter_reading_kwh", valueWh/1000.0),
					)
				}
			}
		}
	}

	return map[string]interface{}{}
}

// handleStartTransactionRequest handles when someone starts charging their car
// This creates a new charging session and records the starting energy meter reading
func (s *Server) handleStartTransactionRequest(chargePointId string, payload interface{}) interface{} {
	s.logger.Info("Charging session started", zap.String("charge_point_id", chargePointId))

	// Parse the data from the charger
	payloadMap, ok := payload.(map[string]interface{})
	if !ok {
		s.logger.Error("Invalid charging start data")
		return map[string]interface{}{
			"idTagInfo": map[string]interface{}{
				"status": "Invalid",
			},
		}
	}

	// Get the user's RFID card ID (not used for billing yet)
	_, _ = payloadMap["idTag"].(string)
	// Get the energy meter reading when charging started
	meterStart, _ := payloadMap["meterStart"].(float64)

	// Find the charger in our database
	var chargerID int64
	err := s.db.QueryRowContext(context.Background(), "SELECT id FROM chargers WHERE identity = ?", chargePointId).Scan(&chargerID)
	if err != nil {
		s.logger.Error("Charger not found in database", zap.Error(err))
		return map[string]interface{}{
			"idTagInfo": map[string]interface{}{
				"status": "Invalid",
			},
		}
	}

	// Create a unique transaction ID (using current timestamp)
	txID := int(time.Now().Unix())

	// Record the new charging session in our database
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
		s.logger.Error("Failed to record charging session", zap.Error(err))
		return map[string]interface{}{
			"idTagInfo": map[string]interface{}{
				"status": "Invalid",
			},
		}
	}

	s.logger.Info("Charging session recorded",
		zap.String("charge_point_id", chargePointId),
		zap.Int("transaction_id", txID),
		zap.Float64("start_meter_wh", meterStart))

	return map[string]interface{}{
		"transactionId": txID,
		"idTagInfo": map[string]interface{}{
			"status": "Accepted",
		},
	}
}

// handleStopTransactionRequest handles when someone stops charging their car
// This calculates how much energy was used and records the end of the charging session
func (s *Server) handleStopTransactionRequest(chargePointId string, payload interface{}) interface{} {
	s.logger.Info("Charging session ended", zap.String("charge_point_id", chargePointId))

	// Parse the data from the charger
	payloadMap, ok := payload.(map[string]interface{})
	if !ok {
		s.logger.Error("Invalid charging stop data")
		return map[string]interface{}{
			"idTagInfo": map[string]interface{}{
				"status": "Invalid",
			},
		}
	}

	// Get the transaction ID and final energy meter reading
	transactionId, _ := payloadMap["transactionId"].(float64)
	meterStop, _ := payloadMap["meterStop"].(float64)

	// Update the charging session with end time and calculate energy used
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
		s.logger.Error("Failed to update charging session", zap.Error(err))
	} else {
		s.logger.Info("Charging session completed",
			zap.String("charge_point_id", chargePointId),
			zap.Int64("transaction_id", int64(transactionId)),
			zap.Float64("final_meter_wh", meterStop))
	}

	return map[string]interface{}{
		"idTagInfo": map[string]interface{}{
			"status": "Accepted",
		},
	}
}

// handleAuthorizeRequest handles when someone tries to use their RFID card to start charging
// For now, we accept all cards - in a real system you'd check if the card is valid and has credit
func (s *Server) handleAuthorizeRequest(chargePointId string, payload interface{}) interface{} {
	s.logger.Info("RFID card authorization request", zap.String("charge_point_id", chargePointId))

	// In a real system, you would:
	// 1. Check if the RFID card exists in your user database
	// 2. Verify the user has sufficient credit
	// 3. Check if the card is not blocked
	// For now, we just accept everyone

	return map[string]interface{}{
		"idTagInfo": map[string]interface{}{
			"status": "Accepted",
		},
	}
}

// handleHeartbeatRequest handles when the charger sends a heartbeat to check if we're still alive
// This also updates the last_seen timestamp so we know the charger is still connected
func (s *Server) handleHeartbeatRequest(chargePointId string, payload interface{}) interface{} {
	s.logger.Info("Heartbeat received from charger", zap.String("charge_point_id", chargePointId))

	// Update when we last heard from this charger
	query := `UPDATE chargers SET last_seen = ? WHERE identity = ?`
	_, err := s.db.ExecContext(context.Background(), query, time.Now(), chargePointId)
	if err != nil {
		s.logger.Error("Failed to update charger last seen time from heartbeat", zap.Error(err))
	}

	// Send back the current time to the charger
	return map[string]interface{}{
		"currentTime": time.Now().Format(time.RFC3339),
	}
}

// getClientIP extracts the client IP address from the HTTP request
func getClientIP(r *http.Request) string {
	// Check for X-Forwarded-For header (for proxies/load balancers)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if idx := len(xff); idx > 0 {
			if commaIdx := 0; commaIdx < idx {
				for i, c := range xff {
					if c == ',' {
						commaIdx = i
						break
					}
				}
				if commaIdx > 0 {
					return xff[:commaIdx]
				}
			}
			return xff
		}
	}

	// Check for X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// sendTriggerMessage sends a TriggerMessage request to the charging station
func (s *Server) sendTriggerMessage(chargePointId string, requestedMessage string) {
	// Check if we have a connection for this charger
	conn, exists := s.connections[chargePointId]
	if !exists {
		s.logger.Error("No WebSocket connection found for charger", zap.String("charge_point_id", chargePointId))
		return
	}

	// Create TriggerMessage request (OCPP 1.6)
	request := map[string]interface{}{
		"requestedMessage": requestedMessage,
		"connectorId":      0, // 0 means the whole charge point
	}

	// Convert to OCPP message format: [2, messageId, "TriggerMessage", payload]
	messageId := fmt.Sprintf("trigger_%s_%d", requestedMessage, time.Now().Unix())
	ocppMessage := []interface{}{
		2,                // CALL (request)
		messageId,        // unique message ID
		"TriggerMessage", // action
		request,          // payload
	}

	// Marshal to JSON
	messageBytes, err := json.Marshal(ocppMessage)
	if err != nil {
		s.logger.Error("Failed to marshal TriggerMessage request", zap.Error(err))
		return
	}

	s.logger.Info("Sending TriggerMessage request to charger",
		zap.String("charge_point_id", chargePointId),
		zap.String("requested_message", requestedMessage),
		zap.String("message_id", messageId))

	// Send the message to the charger via WebSocket
	if err := conn.WriteMessage(websocket.TextMessage, messageBytes); err != nil {
		s.logger.Error("Failed to send TriggerMessage request to charger",
			zap.String("charge_point_id", chargePointId),
			zap.Error(err))
		return
	}

	s.logger.Info("TriggerMessage request sent successfully",
		zap.String("charge_point_id", chargePointId),
		zap.String("requested_message", requestedMessage),
		zap.String("message_id", messageId))
}

// requestMeterValues sends a GetMeterValues request to the charging station
func (s *Server) requestMeterValues(chargePointId string) {
	// Create GetMeterValues request
	request := map[string]interface{}{
		"connectorId": 0, // 0 means main meter for the entire charge point
	}

	// Convert to OCPP message format: [2, messageId, "GetMeterValues", payload]
	messageId := fmt.Sprintf("get_meter_%d", time.Now().Unix())
	ocppMessage := []interface{}{
		2,                // CALL (request)
		messageId,        // unique message ID
		"GetMeterValues", // action
		request,          // payload
	}

	// Marshal to JSON
	messageBytes, err := json.Marshal(ocppMessage)
	if err != nil {
		s.logger.Error("Failed to marshal GetMeterValues request", zap.Error(err))
		return
	}

	s.logger.Info("Sending GetMeterValues request to charger",
		zap.String("charge_point_id", chargePointId),
		zap.String("message_id", messageId))

	// Check if we have a connection for this charger
	conn, exists := s.connections[chargePointId]
	if !exists {
		s.logger.Error("No WebSocket connection found for charger", zap.String("charge_point_id", chargePointId))
		return
	}

	// Send the message to the charger via WebSocket
	if err := conn.WriteMessage(websocket.TextMessage, messageBytes); err != nil {
		s.logger.Error("Failed to send GetMeterValues request to charger",
			zap.String("charge_point_id", chargePointId),
			zap.Error(err))
		return
	}

	s.logger.Info("GetMeterValues request sent successfully",
		zap.String("charge_point_id", chargePointId),
		zap.String("message_id", messageId))
}
