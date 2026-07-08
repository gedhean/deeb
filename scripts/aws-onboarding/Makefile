# Local testing for the multi-account RDS/Aurora onboarder.
# See tests/README.md for the full walkthrough.

COMPOSE := docker compose -f tests/docker-compose.test.yaml
PY := python3

.PHONY: help test-unit test-up test-down test-e2e

help:
	@echo "Targets:"
	@echo "  test-unit   Fast logic tests (no AWS/DB/network)."
	@echo "  test-up     Start the local test stack (moto + MySQL + Postgres)."
	@echo "  test-down   Stop and remove the local test stack (and volumes)."
	@echo "  test-e2e    Run the end-to-end test (requires test-up + a local Deeb)."
	@echo ""
	@echo "  test-e2e needs DEEB_API_URL and DEEB_API_TOKEN pointing at a running Deeb."

test-unit:
	$(PY) tests/test_sweep.py

test-up:
	$(COMPOSE) up -d --wait

test-down:
	$(COMPOSE) down -v

# Assumes `make test-up` has run and a local Deeb is reachable.
test-e2e:
	@test -n "$$DEEB_API_URL" || (echo "set DEEB_API_URL" && exit 1)
	@test -n "$$DEEB_API_TOKEN" || (echo "set DEEB_API_TOKEN" && exit 1)
	$(PY) tests/e2e_local.py
