.PHONY: all up down logs status test test-e2e complexity security lint format help

# Default target
all: help

help:
	@echo "=================================================================="
	@echo "   Haulmer Payment Processing Gateway - Automation CLI            "
	@echo "=================================================================="
	@echo "Lifecycle & Orchestration:"
	@echo "  make up             Start all services in background via Docker Compose"
	@echo "  make down           Stop services and tear down containers/volumes"
	@echo "  make logs           Follow unified logs across all containers"
	@echo "  make status         Inspect health and status of running services"
	@echo ""
	@echo "Testing & Quality Assurance:"
	@echo "  make test           Run Go unit/integration tests with race detector"
	@echo "  make test-e2e       Execute automated E2E test suite against running stack"
	@echo "  make complexity     Check cyclomatic (<=15) and cognitive (<=20) complexity"
	@echo "  make security       Scan code and dependencies for CVEs (govulncheck + audit)"
	@echo "  make lint           Run static analysis (go vet and ESLint)"
	@echo "  make format         Format Go source code (gofmt)"
	@echo "=================================================================="

up:
	docker compose up --build -d

down:
	docker compose down -v

logs:
	docker compose logs -f

status:
	docker compose ps

test:
	@echo "==> Running tests in acquirer-mock..."
	cd acquirer-mock && go test -v -race ./...
	@echo "==> Running tests in payments-api..."
	cd payments-api && go test -v -race ./...

test-e2e:
	@echo "==> Running E2E test suite..."
	./scripts/test-e2e.sh

complexity:
	@echo "==> Checking Cyclomatic Complexity (threshold <= 15)..."
	go run github.com/fzipp/gocyclo/cmd/gocyclo@latest -over 15 payments-api acquirer-mock
	@echo "==> Checking Cognitive Complexity (threshold <= 20)..."
	go run github.com/uudashr/gocognit/cmd/gocognit@latest -over 20 payments-api acquirer-mock
	@echo "✓ Complexity checks passed!"

security:
	@echo "==> Scanning Go packages for CVEs (govulncheck)..."
	cd payments-api && go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	cd acquirer-mock && go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	@echo "==> Auditing frontend dependencies (npm audit)..."
	cd web-dashboard && npm audit --audit-level=high
	@echo "✓ Zero vulnerabilities found!"

lint:
	@echo "==> Running Go static analysis (go vet)..."
	cd payments-api && go vet ./...
	cd acquirer-mock && go vet ./...
	@echo "==> Running ESLint in web-dashboard..."
	cd web-dashboard && npm run lint
	@echo "✓ Lint checks passed!"

format:
	@echo "==> Formatting Go source files..."
	gofmt -s -w payments-api/ acquirer-mock/
