ENGINE_DIR := engine
SHELL_DIR := shell
GO ?= go
BUN ?= bun

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: deps
deps: ## Install engine and shell dependencies
	cd $(ENGINE_DIR) && $(GO) mod download
	cd $(SHELL_DIR) && $(BUN) install

.PHONY: build
build: ## Build the engine binary
	@# The shell is plain JavaScript with no bundler, so there is nothing
	@# to build for it. Electron loads src/ directly.
	cd $(ENGINE_DIR) && $(GO) build -o ../dist/tama-engine ./cmd/engine

.PHONY: test
test: ## Run the test suite
	cd $(ENGINE_DIR) && $(GO) test ./...

.PHONY: fmt
fmt: ## Format the Go sources
	cd $(ENGINE_DIR) && $(GO) fmt ./...

.PHONY: vet
vet: ## Vet the Go sources
	cd $(ENGINE_DIR) && $(GO) vet ./...

.PHONY: check
check: fmt vet test ## Format, vet and test in one go

.PHONY: run
run: build ## Run the app
	@# The shell starts the engine itself, so this is the whole app.
	cd $(SHELL_DIR) && $(BUN) run start

.PHONY: debug
debug: build ## Run the app with the DevTools port open on 9222
	cd $(SHELL_DIR) && $(BUN) run debug

.PHONY: engine
engine: build ## Run the engine alone, without a window
	./dist/tama-engine

.PHONY: rom
rom: ## Fetch and verify the Tamagotchi P1 ROM (set ROM_URL first)
	@./scripts/fetch-rom.sh

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf dist $(SHELL_DIR)/out
