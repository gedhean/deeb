.PHONY: all lint test format dev

FRONTEND_DIR := frontend
BACKEND_DIR := backend

.PHONY: all
all: lint test format

.PHONY: lint
lint:
	echo "Running checks and fixes on $(FRONTEND_DIR)..."
	pnpm --dir $(FRONTEND_DIR) run lint-fix

	echo "Running checks and fixes on $(BACKEND_DIR)..."
	make -C $(BACKEND_DIR) lint-fix
	make -C $(BACKEND_DIR) format

.PHONY: test
test:
	echo "Running tests on $(FRONTEND_DIR)..."
	pnpm --dir $(FRONTEND_DIR) run test

	echo "Running tests on $(BACKEND_DIR)..."
	make -C $(BACKEND_DIR) test

.PHONY: format
format:
	echo "Formatting $(FRONTEND_DIR)..."
	pnpm --dir $(FRONTEND_DIR) run format

	echo "Formatting $(BACKEND_DIR)..."
	make -C $(BACKEND_DIR) format

.PHONY: build
build:
	echo "Building $(FRONTEND_DIR)..."
	pnpm --dir $(FRONTEND_DIR) run build

	echo "Building $(BACKEND_DIR)..."
	make -C $(BACKEND_DIR) build

.PHONY: dev
dev: ensure-concurrently
	@echo "Starting frontend and backend..."
	@pnpm dlx concurrently \
		--names "FRONTEND,BACKEND" \
		--prefix-colors "cyan,magenta" \
		--kill-others \
		"pnpm --dir $(FRONTEND_DIR) install && pnpm --dir $(FRONTEND_DIR) run dev" \
		"make -C $(BACKEND_DIR) run-air"

# alias cmd for dev
run: dev

ensure-concurrently:
	@which concurrently > /dev/null || (echo "Installing concurrently..." && pnpm install -g concurrently)
