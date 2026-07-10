.PHONY: all help info

include makefiles/Makefile.build
include makefiles/Makefile.defs
include makefiles/Makefile.lint
include makefiles/Makefile.format
include makefiles/Makefile.run
include makefiles/Makefile.testing
include makefiles/Makefile.generate

.PHONY: info
info:
	@echo "Go Arch: ${GOARCH}"
	@echo "Go OS: ${GOOS}"
	@echo "Service: ${PROJECT_NAME}"

help:
	@echo "Deeb Backend - Available Make Targets"
	@echo "======================================"
	@echo ""
	@echo "Development:"
	@echo "  run-air          - Start development server with Air (live reload)"
	@echo "  seed             - Load sample data into the dev database (seeds/seed_dev.sql)"
	@echo "  docker-stop-dev  - Stop development docker containers"
	@echo "  docker-down-dev  - Stop and remove development docker containers"
	@echo ""
	@echo "Production:"
	@echo "  run              - Start production-like server with Docker Compose"
	@echo "  docker-stop      - Stop production docker containers"
	@echo "  docker-build     - Build production Docker image"
	@echo ""
	@echo "Building:"
	@echo "  build            - Build binary for current platform"
	@echo "  build-linux      - Build binary for Linux (amd64)"
	@echo ""
	@echo "Testing:"
	@echo "  test             - Run unit tests using gotestsum"
	@echo "  test-integration - Run integration tests with testcontainers"
	@echo "  test-verbose     - Run tests with verbose output"
	@echo "  test-coverage    - Run tests with coverage report"
	@echo ""
	@echo "Code Quality:"
	@echo "  lint             - Run all linters (Go, YAML)"
	@echo "  lint-fix         - Auto-fix linting issues where possible"
	@echo "  lint-golang      - Run golangci-lint on Go code"
	@echo "  lint-golang-fix  - Auto-fix Go linting issues"
	@echo "  lint-yaml        - Run yamllint on YAML files"
	@echo "  lint-breaking    - Check for breaking API changes (buf)"
	@echo ""
	@echo "Formatting:"
	@echo "  format           - Format all code (Go, YAML)"
	@echo "  format-go        - Format Go code with gofmt"
	@echo "  format-yaml      - Format YAML files with yamlfmt"
	@echo ""
	@echo "Setup:"
	@echo "  lint-setup       - Install linting tools"
	@echo "  format-setup     - Install formatting tools"
	@echo ""
	@echo "Info:"
	@echo "  info             - Display build environment information"
	@echo "  help             - Show this help message"
