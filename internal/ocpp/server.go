package ocpp

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/lorenzodonini/ocpp-go/ws"
	"go.uber.org/zap"
)

// Server handles OCPP 1.6J connections
type Server struct {
	db     *sql.DB
	logger *zap.Logger
	cs     *core.CentralSystem
}

// New creates a new OCPP server
func New(db *sql.DB, logger *zap.Logger) *Server {
	cs := core.NewCentralSystem(nil, nil)
	s := &Server{
		db:     db,
		logger: logger,
		cs:     cs,
	}

	// Register handlers
	cs.SetCoreHandler(s)

	return s
}

// Start starts the OCPP WebSocket server
func (s *Server) Start(ctx context.Context, addr string) error {
	s.logger.Info("Starting OCPP 1.6J server", zap.String("addr", addr))

	// Create WebSocket server
	wsServer := ws.NewServer()
	wsServer.SetNewClientHandler(func(chargePointId string) {
		s.logger.Info("New OCPP client connected", zap.String("charge_point_id", chargePointId))
	})

	// Start the server
	go func() {
		if err := wsServer.Start(8081, s.cs); err != nil {
			s.logger.Error("OCPP server failed", zap.Error(err))
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	s.logger.Info("Stopping OCPP server")
	return nil
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
			Status:      types.RegistrationStatusRejected,
			CurrentTime: types.NewDateTime(time.Now()),
			Interval:    300,
		}, nil
	}

	return &core.BootNotificationConfirmation{
		Status:      types.RegistrationStatusAccepted,
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
	txID := fmt.Sprintf("%d_%d", chargerID, time.Now().Unix())

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
		zap.String("tx_id", request.TransactionId),
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
		zap.String("tx_id", request.TransactionId),
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

	// Insert meter values
	for _, mv := range request.MeterValue {
		for _, sample := range mv.SampledValue {
			if sample.Measurand != nil && *sample.Measurand == "Energy.Active.Import.Register" {
				value, err := strconv.ParseFloat(sample.Value, 64)
				if err != nil {
					s.logger.Error("Failed to parse meter value", zap.Error(err))
					continue
				}

				// Convert to Wh if needed (assume kWh if value > 1000)
				if value > 1000 {
					value = value * 1000
				}

				query := `
					INSERT INTO meter_values (transaction_id, ts, measurand, value)
					VALUES (?, ?, ?, ?)
				`

				_, err = s.db.ExecContext(context.Background(), query,
					transactionID,
					mv.Timestamp.Time,
					*sample.Measurand,
					value,
				)

				if err != nil {
					s.logger.Error("Failed to insert meter value", zap.Error(err))
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
		Status: types.DataTransferStatusAccepted,
	}, nil
}

// OnChangeAvailability handles ChangeAvailability requests
func (s *Server) OnChangeAvailability(chargePointId string, request *core.ChangeAvailabilityRequest) (*core.ChangeAvailabilityConfirmation, error) {
	s.logger.Info("ChangeAvailability received",
		zap.String("charge_point_id", chargePointId),
		zap.String("type", string(request.Type)),
	)

	return &core.ChangeAvailabilityConfirmation{
		Status: types.AvailabilityStatusAccepted,
	}, nil
}

// OnChangeConfiguration handles ChangeConfiguration requests
func (s *Server) OnChangeConfiguration(chargePointId string, request *core.ChangeConfigurationRequest) (*core.ChangeConfigurationConfirmation, error) {
	s.logger.Info("ChangeConfiguration received",
		zap.String("charge_point_id", chargePointId),
		zap.String("key", request.Key),
	)

	return &core.ChangeConfigurationConfirmation{
		Status: types.ConfigurationStatusAccepted,
	}, nil
}

// OnClearCache handles ClearCache requests
func (s *Server) OnClearCache(chargePointId string, request *core.ClearCacheRequest) (*core.ClearCacheConfirmation, error) {
	s.logger.Info("ClearCache received", zap.String("charge_point_id", chargePointId))

	return &core.ClearCacheConfirmation{
		Status: types.ClearCacheStatusAccepted,
	}, nil
}

// OnGetConfiguration handles GetConfiguration requests
func (s *Server) OnGetConfiguration(chargePointId string, request *core.GetConfigurationRequest) (*core.GetConfigurationConfirmation, error) {
	s.logger.Info("GetConfiguration received", zap.String("charge_point_id", chargePointId))

	return &core.GetConfigurationConfirmation{
		ConfigurationKey: []types.ConfigurationKey{},
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
		zap.String("tx_id", request.TransactionId),
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
		Status: types.ResetStatusAccepted,
	}, nil
}

// OnUnlockConnector handles UnlockConnector requests
func (s *Server) OnUnlockConnector(chargePointId string, request *core.UnlockConnectorRequest) (*core.UnlockConnectorConfirmation, error) {
	s.logger.Info("UnlockConnector received",
		zap.String("charge_point_id", chargePointId),
		zap.Int("connector_id", request.ConnectorId),
	)

	return &core.UnlockConnectorConfirmation{
		Status: types.UnlockStatusUnlocked,
	}, nil
}
