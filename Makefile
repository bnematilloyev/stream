.PHONY: help dev smoke up down stop migrate migrate-down test test-platform test-playback lint build proto clean frontend frontend-dev frontend-build deploy

help:
	@echo "Sahiy Stream — available targets:"
	@echo ""
	@echo "  Quick start (copy this ONE line only):"
	@echo "    make smoke"
	@echo "    bash run.sh"
	@echo ""
	@echo "  make smoke        Infra + migrate + services + integration tests"
	@echo "  make dev          Start infra + migrate + services"
	@echo "  make up           Start infrastructure only"
	@echo "  make down         Stop infrastructure"
	@echo "  make stop         Stop Go services"
	@echo "  make start        Start Go services only"
	@echo "  make migrate      Run database migrations"
	@echo "  make migrate-down Rollback last migration"
	@echo "  make test-platform  Run platform API smoke test"
	@echo "  make test-playback  Run playback API smoke test"
	@echo "  make proto        Generate gRPC code from protos"
	@echo "  make build        Build all services"
	@echo "  make test         Run unit tests"
	@echo "  make lint         Run golangci-lint"
	@echo "  make frontend-dev Next.js dev server (:3000)"
	@echo "  make frontend-build  Build Next.js app"
	@echo "  make clean        Remove build artifacts"

up:
	docker compose -f infra/docker/docker-compose.yml up -d

down:
	docker compose -f infra/docker/docker-compose.yml down

stop:
	@bash scripts/stop-services.sh

migrate:
	@bash scripts/migrate.sh up

migrate-down:
	@bash scripts/migrate.sh down 1

proto:
	@bash scripts/proto-gen.sh

build:
	cd services/auth-service && go build -o ../../bin/auth-service ./cmd/server
	cd services/user-service && go build -o ../../bin/user-service ./cmd/server
	cd services/stream-service && go build -o ../../bin/stream-service ./cmd/server
	cd services/media-orchestrator && go build -o ../../bin/media-orchestrator ./cmd/server
	cd services/api-gateway && go build -o ../../bin/api-gateway ./cmd/server

test:
	go work sync
	go test ./...

lint:
	golangci-lint run ./...

dev: up migrate
	@bash scripts/start-dev.sh

smoke: up migrate start
	@bash scripts/test-platform.sh
	@bash scripts/test-playback.sh

start:
	@bash scripts/start-dev.sh

test-platform:
	@bash scripts/wait-for-api.sh
	@bash scripts/test-platform.sh

test-playback:
	@bash scripts/wait-for-api.sh
	@bash scripts/test-playback.sh

frontend-dev:
	cd frontend && npm run dev:clean

frontend-clean:
	rm -rf frontend/.next

frontend-build:
	cd frontend && npm run build

deploy:
	@bash scripts/deploy.sh

frontend:
	cd frontend && npm install

clean:
	rm -rf bin/
