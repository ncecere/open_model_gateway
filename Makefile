CONFIG ?= $(PWD)/deploy/router.local.yaml
DB_URL ?= postgres://open_gateway:open_gateway@localhost:5432/open_gateway?sslmode=disable
REDIS_URL ?= redis://localhost:6379/0
COMPOSE ?= docker compose -f deploy/docker-compose.yml

.PHONY: help compose-up compose-down compose-logs run-backend test-backend build-ui

help:
	@echo "Useful targets:"
	@echo "  compose-up        Start Postgres and Redis containers"
	@echo "  compose-down      Stop containers and remove volumes"
	@echo "  compose-logs      Tail logs from Compose services"
	@echo "  run-backend       Build UI + run router with deploy/router.local.yaml"
	@echo "  test-backend      go test ./... inside backend/"

compose-up:
	$(COMPOSE) up -d

compose-down:
	$(COMPOSE) down -v

compose-logs:
	$(COMPOSE) logs -f

build-ui:
	cd backend/frontend && bun run build
	rm -rf backend/internal/httpserver/ui/dist
	mkdir -p backend/internal/httpserver/ui/dist
	cp -R backend/frontend/dist/. backend/internal/httpserver/ui/dist/

run-backend: build-ui
	cd backend && \
		ROUTER_CONFIG_FILE=$(CONFIG) \
		ROUTER_DB_URL=$(DB_URL) \
		ROUTER_DATABASE_URL=$(DB_URL) \
		ROUTER_REDIS_URL=$(REDIS_URL) \
		go run ./cmd/routerd

test-backend:
	cd backend && go test ./...
