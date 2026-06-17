# Makefile for Bookmark Worker

# =============================================================================
# APPLICATION METADATA
# =============================================================================

APP_NAME    := bookmark-worker
CMD_PATH    := ./cmd/worker/main.go

BIN_DIR     := ./bin

# =============================================================================
# COVERAGE & QUALITY GATES
# =============================================================================

COVERAGE_DIR       ?= coverage_report
COVERAGE_THRESHOLD ?= 80

# Infrastructure dirs: exclude from coverage % but SCAN for security
INFRA_DIRS := \
	cmd \
	internal/bootstrap

# System artifacts: auto-generated, vendored, test infrastructure (NO SCAN)
SYSTEM_DIRS := vendor bin internal/test mocks
SYSTEM_FILES := _test.go test_helper.go mock.go

comma := ,
space := $(subst ,, )

ALL_EXCLUDES := $(INFRA_DIRS) $(SYSTEM_DIRS) $(SYSTEM_FILES)
COVERAGE_EXCLUDE := $(subst $(space),|,$(strip $(ALL_EXCLUDES)))
COVERPKG := ./...

# =============================================================================
# GO COMPILER
# =============================================================================

GO      := go
GOLINT  := golangci-lint
CGO     := 0

LDFLAGS := -ldflags "-s -w"

# =============================================================================
# DEVELOPMENT
# =============================================================================

.DEFAULT_GOAL := help

.PHONY: help run fmt vet lint tidy

help:
	@echo "Development:"
	@echo "  make run             Run locally"
	@echo "  make fmt             Format code"
	@echo "  make vet             Static analysis"
	@echo "  make lint            Linter"
	@echo "  make tidy            Dependencies"
	@echo ""
	@echo "Testing:"
	@echo "  make test            Local tests + coverage report"
	@echo ""
	@echo "Build:"
	@echo "  make build           Build binary"

run:
	@echo "Starting $(APP_NAME)..."
	SERVICE_NAME=$(APP_NAME) $(GO) run $(CMD_PATH)

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

lint:
	@which $(GOLINT) > /dev/null || (echo "Error: golangci-lint not found"; exit 1)
	$(GOLINT) run ./...

tidy:
	$(GO) mod tidy

# =============================================================================
# TESTING
# =============================================================================

.PHONY: test

test:
	@$(GO) clean -testcache
	@mkdir -p $(COVERAGE_DIR)
	@$(GO) test ./... -coverprofile=$(COVERAGE_DIR)/coverage.tmp -covermode=atomic -coverpkg=$(COVERPKG) -p 1
	@head -1 $(COVERAGE_DIR)/coverage.tmp > $(COVERAGE_DIR)/coverage.out
	@grep -vE "$(COVERAGE_EXCLUDE)" $(COVERAGE_DIR)/coverage.tmp | tail -n +2 >> $(COVERAGE_DIR)/coverage.out || true
	@$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@total=$$($(GO) tool cover -func=$(COVERAGE_DIR)/coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Coverage: $$total%"

# =============================================================================
# BUILD
# =============================================================================

.PHONY: build clean

build:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=$(CGO) $(GO) build $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME) $(CMD_PATH)

clean:
	rm -rf $(BIN_DIR) $(COVERAGE_DIR)
