.PHONY: test test-integration clean build

# Run unit tests
test:
	go test -v ./...

# Run integration tests
test-integration:
	pwsh ./Test-Integration.ps1

# Clean up Docker resources
clean:
	docker compose down --volumes --remove-orphans

# Build and start services
build:
	docker compose up --build -d

# Run all tests
test-all: test test-integration
