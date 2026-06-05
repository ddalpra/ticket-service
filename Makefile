# ─────────────────────────────────────────────────────────────────────────────
#  Ticket Service — Makefile
# ─────────────────────────────────────────────────────────────────────────────
BINARY      := ticket-service
MAIN        := ./cmd/server
MODULE      := $(shell go list -m)
COVERAGE    := coverage.out

.DEFAULT_GOAL := help

# ── Build ─────────────────────────────────────────────────────────────────────

.PHONY: build
build: ent-gen ## Compila il binario
	go build -ldflags="-w -s" -o $(BINARY) $(MAIN)

.PHONY: run
run: ## Avvia il server (richiede .env compilato)
	@export $$(grep -v '^#' .env | xargs) && go run $(MAIN)

.PHONY: run-air
run-air: ## Avvia con hot-reload (richiede: go install github.com/air-verse/air@latest)
	air

# ── Ent ───────────────────────────────────────────────────────────────────────

.PHONY: ent-gen
ent-gen: ## Genera il codice Ent dagli schema
	go generate ./ent/...

.PHONY: ent-new
ent-new: ## Crea un nuovo schema Ent (uso: make ent-new NAME=MyEntity)
	go run -mod=mod entgo.io/ent/cmd/ent new --target ./ent/schema $(NAME)

# ── Test ──────────────────────────────────────────────────────────────────────

.PHONY: test
test: ## Esegui tutti i test
	go test ./... -v -count=1

.PHONY: test-short
test-short: ## Esegui i test escludendo quelli lenti (integration)
	go test ./... -short -count=1

.PHONY: coverage
coverage: ## Genera report di coverage HTML
	go test ./... -coverprofile=$(COVERAGE) -covermode=atomic
	go tool cover -html=$(COVERAGE) -o coverage.html
	@echo "Report: coverage.html"

.PHONY: coverage-pct
coverage-pct: ## Mostra la percentuale di coverage
	go test ./... -coverprofile=$(COVERAGE) -covermode=atomic
	go tool cover -func=$(COVERAGE) | grep total

# ── Quality ───────────────────────────────────────────────────────────────────

.PHONY: lint
lint: ## Lint con golangci-lint
	golangci-lint run ./...

.PHONY: fmt
fmt: ## Formatta il codice
	gofmt -w .
	goimports -w .

.PHONY: vet
vet: ## go vet
	go vet ./...

.PHONY: tidy
tidy: ## go mod tidy
	go mod tidy

.PHONY: check
check: fmt vet lint test ## Esegui tutti i controlli qualità

# ── Dev tools ─────────────────────────────────────────────────────────────────

.PHONY: tools
tools: ## Installa i tool di sviluppo
	go install github.com/air-verse/air@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# ── Utilità ───────────────────────────────────────────────────────────────────

.PHONY: clean
clean: ## Rimuovi artefatti di build
	rm -f $(BINARY) $(COVERAGE) coverage.html

.PHONY: help
help: ## Mostra questo help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' | sort

# ── Mail Worker ───────────────────────────────────────────────────────────────

.PHONY: mailworker
mailworker: ## Avvia il mail worker (polling Gmail)
	@export $$(grep -v '^#' .env | xargs) && go run ./cmd/mailworker

.PHONY: mailworker-auth
mailworker-auth: ## Prima autenticazione OAuth2 Gmail (apre il browser)
	@export $$(grep -v '^#' .env | xargs) && MAIL_POLL_INTERVAL=999h go run ./cmd/mailworker
