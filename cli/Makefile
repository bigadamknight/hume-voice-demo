.PHONY: help install dev test lint format verify clean

help:
	@echo "Available commands:"
	@echo "  make install    - Install dependencies"
	@echo "  make dev        - Run development setup"
	@echo "  make verify     - Verify setup and configuration"
	@echo "  make lint       - Run linters"
	@echo "  make format     - Format code"
	@echo "  make clean      - Clean generated files"

install:
	pip install -r requirements.txt

dev: install
	@echo "Development environment ready"
	@echo "Run: python cli.py start"

verify:
	python verify_setup.py

lint:
	@command -v ruff >/dev/null 2>&1 && ruff check . || echo "ruff not installed, skipping"
	@command -v mypy >/dev/null 2>&1 && mypy . --ignore-missing-imports || echo "mypy not installed, skipping"

format:
	@command -v ruff >/dev/null 2>&1 && ruff format . || echo "ruff not installed, skipping"
	@command -v black >/dev/null 2>&1 && black . || echo "black not installed, skipping"

clean:
	find . -type d -name "__pycache__" -exec rm -r {} + 2>/dev/null || true
	find . -type f -name "*.pyc" -delete
	find . -type f -name "*.pyo" -delete
	find . -type f -name "*.pyd" -delete
	rm -rf .mypy_cache
	rm -rf .ruff_cache

