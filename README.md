# OCPP Power Manager

Local and cloud-ready OCPP 1.6J management server. Tracks stations, sessions, energy usage.

## Features

- **Backend**: Go + ocpp-go + chi + zap
- **Database**: SQLite (default) or Postgres
- **Frontend**: React + Tailwind dashboard
- **Real-time monitoring** of charging stations
- **Session tracking** and energy usage analytics
- **RESTful API** for station management

## Getting Started

### Prerequisites

This repository is **fully self-contained** and can run offline without external dependencies:
- **Go 1.22+** (only requirement)
- **No Node.js needed** - frontend is pre-built
- **No internet required** - all dependencies are vendored

### Quick Start (Offline Mode)

1. Clone the repository:
   ```bash
   git clone https://github.com/PavolRusnak/OCPP-Power-Manager.git
   cd OCPP-Power-Manager
   ```

2. Run the application (offline mode):
   ```bash
   # Run with migrations (first time only)
   RUN_MIGRATIONS=1 go run -mod=vendor ./cmd/OCPP-Power-Manager
   
   # Or run normally (after first setup)
   go run -mod=vendor ./cmd/OCPP-Power-Manager
   ```

3. Open your browser to `http://localhost:8080`

**That's it!** The application runs completely offline with:
- ✅ **Vendored Go dependencies** in `/vendor` folder
- ✅ **Pre-built React frontend** in `/web/dist` folder  
- ✅ **Embedded static files** using `go:embed`
- ✅ **Default SQLite database** (no Postgres required)

### Configuration

The application uses environment variables for configuration:

```bash
HTTP_ADDR=":8080"                    # Server address
DB_DRIVER="sqlite"                   # Database driver (sqlite/postgres)
DB_DSN="file:ocpppm.db?_foreign_keys=on"  # Database connection string
```

**Default Configuration:**
- **Database**: SQLite (`ocpppm.db`) - no external database required
- **Server**: Runs on `:8080` by default
- **Frontend**: Pre-built and embedded in Go binary

## Project Structure

```
├── cmd/OCPP-Power-Manager/     # Main application entry point
├── internal/
│   ├── config/                 # Configuration management
│   ├── db/                     # Database connection and utilities
│   └── httpapi/                # HTTP API handlers and routes
├── web/                        # React frontend
│   ├── src/
│   │   ├── pages/              # React pages
│   │   ├── components/         # Reusable components
│   │   └── partials/           # Layout components
│   └── dist/                   # Built frontend assets
└── migrations/                 # Database migrations (future)
```

## API Endpoints

- `GET /api/stations` - List all charging stations
- `GET /api/sessions` - List charging sessions (future)
- `GET /api/transactions` - List transactions (future)

## Development

### Backend Development
```bash
go run -mod=vendor ./cmd/OCPP-Power-Manager
```

### Frontend Development (Optional)
If you need to modify the frontend:
```bash
cd web
npm install
npm run dev
```

### Building
```bash
make build
```

**Note**: The repository includes pre-built frontend assets. Frontend development is optional unless you need to modify the UI.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

See [CONTRIBUTING_CURSOR.md](CONTRIBUTING_CURSOR.md) for development guidelines.
