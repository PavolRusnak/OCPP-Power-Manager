.PHONY: build run migrate-up migrate-down clean

# Build the application
build:
	go build -tags "sqlite_omit_load_extension" -o OCPP-Power-Manager.exe ./cmd/OCPP-Power-Manager

# Run the application (load .env if present)
run:
	@if exist .env (set -a && . ./.env && set +a;) || true
	go run -mod=vendor ./cmd/OCPP-Power-Manager

# Run database migrations up
migrate-up:
	@if exist .env (set -a && . ./.env && set +a;) || true
	goose -dir migrations sqlite3 "file:ocpppm.db?_foreign_keys=on" up

# Run database migrations down
migrate-down:
	@if exist .env (set -a && . ./.env && set +a;) || true
	goose -dir migrations sqlite3 "file:ocpppm.db?_foreign_keys=on" down

# Clean build artifacts
clean:
	rm -f OCPP-Power-Manager.exe
