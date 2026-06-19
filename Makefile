# USG-ITSM developer tasks. POSIX make (use Git Bash / WSL on Windows).
.DEFAULT_GOAL := help

SHELL        := /bin/sh
COMPOSE      := docker compose -f compose/docker-compose.yml
GO_MODULES   := pkg services/gateway services/identity services/ticketing services/notification

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

.PHONY: hooks
hooks: ## Install the Conventional Commits git hook (run once)
	git config core.hooksPath .githooks
	@chmod +x .githooks/* 2>/dev/null || true
	@echo "Git hooks installed (core.hooksPath=.githooks)."

.PHONY: up
up: ## Start the local dependency stack
	$(COMPOSE) up -d

.PHONY: down
down: ## Stop the local dependency stack
	$(COMPOSE) down

.PHONY: logs
logs: ## Tail the local stack logs
	$(COMPOSE) logs -f

.PHONY: build
build: ## Build all Go modules
	@for m in $(GO_MODULES); do echo "== build $$m =="; (cd $$m && go build ./...); done

.PHONY: test
test: ## Run unit tests for all Go modules
	@for m in $(GO_MODULES); do echo "== test $$m =="; (cd $$m && go test ./... -race -count=1); done

.PHONY: lint
lint: ## go vet + gofmt check
	@for m in $(GO_MODULES); do echo "== vet $$m =="; (cd $$m && go vet ./...); done
	@unformatted="$$(gofmt -l $$(git ls-files '*.go'))"; \
		if [ -n "$$unformatted" ]; then echo "Not gofmt-formatted:"; echo "$$unformatted"; exit 1; fi
	@echo "lint ok"

.PHONY: tidy
tidy: ## go mod tidy across modules
	@for m in $(GO_MODULES); do echo "== tidy $$m =="; (cd $$m && go mod tidy); done

.PHONY: run-gateway
run-gateway: ## Run the gateway service locally
	cd services/gateway && go run ./cmd/gateway

.PHONY: run-identity
run-identity: ## Run the identity service locally
	cd services/identity && go run ./cmd/identity

.PHONY: run-ticketing
run-ticketing: ## Run the ticketing service locally (needs DATABASE_URL + OIDC_ISSUER)
	cd services/ticketing && go run ./cmd/ticketing

.PHONY: run-notification
run-notification: ## Run the notification service locally (needs NATS_URL)
	cd services/notification && go run ./cmd/notification
