.DEFAULT_GOAL := help

##@ Development

.PHONY: build
build: ## Build debug binary
	cargo build

.PHONY: test
test: ## Run tests
	cargo test

.PHONY: check
check: ## Run cargo check
	cargo check

.PHONY: fmt
fmt: ## Format code with rustfmt
	cargo fmt

.PHONY: clippy
clippy: ## Run clippy lints
	cargo clippy -- -D warnings

##@ Release

.PHONY: release
release: ## Build optimized release binary
	cargo build --release

##@ Maintenance

.PHONY: clean
clean: ## Remove build artifacts
	cargo clean

##@ Info

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } \
		/^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
