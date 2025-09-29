.PHONY: build run migrate-up migrate-down clean

# Build the application
build:
	go build -tags "sqlite_omit_load_extension" -o OCPP-Power-Manager.exe ./cmd/OCPP-Power-Manager

# Run the application (load .env if present)
run:
	@if [ -f .env ]; then \
		set -a; \
		. ./.env; \
		set +a; \
	fi; \
	go run ./cmd/OCPP-Power-Manager

# Run database migrations up
migrate-up:
	@if [ -f .env ]; then \
		set -a; \
		. ./.env; \
		set +a; \
	fi; \
	goose -dir migrations $(DB_DRIVER) $(DB_DSN) up

# Run database migrations down
migrate-down:
	@if [ -f .env ]; then \
		set -a; \
		. ./.env; \
		set +a; \
	fi; \
	goose -dir migrations $(DB_DRIVER) $(DB_DSN) down

# Clean build artifacts
clean:
	rm -f OCPP-Power-Manager.exe
