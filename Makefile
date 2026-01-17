.PHONY: help build release clean check fmt clippy test

help:
	@echo "Available commands:"
	@echo "  make build    - Build debug binary"
	@echo "  make release  - Build optimized release binary"
	@echo "  make test     - Run tests"
	@echo "  make check    - Run cargo check"
	@echo "  make fmt      - Format code with rustfmt"
	@echo "  make clippy   - Run clippy lints"
	@echo "  make clean    - Remove build artifacts"

build:
	cargo build

release:
	cargo build --release

test:
	cargo test

check:
	cargo check

fmt:
	cargo fmt

clippy:
	cargo clippy -- -D warnings

clean:
	cargo clean
