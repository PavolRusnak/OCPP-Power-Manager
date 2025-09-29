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

Install the required tools (see [PREREQUISITES.md](PREREQUISITES.md)):
- Go 1.22+
- Node.js 18+
- Git

### Local Development

1. Clone the repository:
   ```bash
   git clone https://github.com/PavolRusnak/OCPP-Power-Manager.git
   cd OCPP-Power-Manager
   ```

2. Run the application:
   ```bash
   make run
   ```

3. Open your browser to `http://localhost:8080`

### Configuration

The application uses environment variables for configuration:

```bash
HTTP_ADDR=":8080"                    # Server address
DB_DRIVER="sqlite"                   # Database driver (sqlite/postgres)
DB_DSN="file:ocpppm.db?_foreign_keys=on"  # Database connection string
```

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
go run ./cmd/OCPP-Power-Manager
```

### Frontend Development
```bash
cd web
npm install
npm run dev
```

### Building
```bash
make build
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

See [CONTRIBUTING_CURSOR.md](CONTRIBUTING_CURSOR.md) for development guidelines.
