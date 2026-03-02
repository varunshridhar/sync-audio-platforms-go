SHELL := /bin/sh
# Local Docker env file used by docker-up-dev/down-dev targets.
ENV_FILE ?= /home/varun/personal/sync-audio-platforms-go/.env.docker
# Backend env file loaded by backend-run when present.
BACKEND_ENV_FILE ?= /home/varun/personal/sync-audio-platforms-go/backend/env
# Compose file path centralized so all Docker targets stay consistent.
COMPOSE_FILE := /home/varun/personal/sync-audio-platforms-go/docker-compose.yml
# Docker command can be overridden, e.g. DOCKER="sudo docker" make docker-down-dev
DOCKER ?= docker

.PHONY: help backend-run backend-run-sqlite backend-build backend-test frontend-install frontend-dev frontend-build frontend-lint e2e-local-sqlite docker-build docker-up docker-down docker-logs docker-up-dev docker-down-dev

help:
	@echo "This Makefile is the main developer entrypoint for common tasks."
	@echo "Available targets:"
	@echo "  backend-run       Run backend API locally"
	@echo "  backend-run-sqlite Run backend API locally with SQLite (no GCP)"
	@echo "  backend-build     Build backend binary"
	@echo "  backend-test      Run backend tests"
	@echo "  frontend-install  Install frontend dependencies"
	@echo "  frontend-dev      Run frontend dev server"
	@echo "  frontend-build    Build frontend app"
	@echo "  frontend-lint     Run frontend lint"
	@echo "  e2e-local-sqlite  Boot local backend+frontend and verify API connectivity"
	@echo "  docker-build      Build all Docker images"
	@echo "  docker-up         Start full stack with Docker Compose"
	@echo "  docker-down       Stop Docker Compose stack"
	@echo "  docker-logs       Tail Docker Compose logs"
	@echo "  docker-up-dev     Start stack using local .env.docker"
	@echo "  docker-down-dev   Stop stack using local .env.docker"

backend-run:
	cd /home/varun/personal/sync-audio-platforms-go/backend && \
	if [ -f "$(BACKEND_ENV_FILE)" ]; then set -a; . "$(BACKEND_ENV_FILE)"; set +a; fi && \
	go run ./cmd/api

backend-run-sqlite:
	cd /home/varun/personal/sync-audio-platforms-go/backend && \
	STORE_PROVIDER=sqlite \
	SQLITE_PATH=/tmp/sync-audio-platforms.db \
	go run ./cmd/api

backend-build:
	cd /home/varun/personal/sync-audio-platforms-go/backend && go build -o bin/api ./cmd/api

backend-test:
	cd /home/varun/personal/sync-audio-platforms-go/backend && go test ./...

frontend-install:
	cd /home/varun/personal/sync-audio-platforms-go/frontend && npm install

frontend-dev:
	cd /home/varun/personal/sync-audio-platforms-go/frontend && npm run dev

frontend-build:
	cd /home/varun/personal/sync-audio-platforms-go/frontend && npm run build

frontend-lint:
	cd /home/varun/personal/sync-audio-platforms-go/frontend && npm run lint

e2e-local-sqlite:
	@set -eu; \
	BACKEND_PORT=8081; \
	FRONTEND_PORT=3002; \
	BACKEND_LOG=/tmp/sync-audio-backend-e2e.log; \
	FRONTEND_LOG=/tmp/sync-audio-frontend-e2e.log; \
	COOKIE_JAR=/tmp/sync-audio-e2e-cookie.txt; \
	EMAIL="make_e2e_$$(date +%s)@example.com"; \
	rm -f $$BACKEND_LOG $$FRONTEND_LOG $$COOKIE_JAR /tmp/sync-audio-e2e.db /home/varun/personal/sync-audio-platforms-go/frontend/.next/dev/lock; \
	cleanup() { \
		test -n "$${BACKEND_PID:-}" && kill $$BACKEND_PID 2>/dev/null || true; \
		test -n "$${FRONTEND_PID:-}" && kill $$FRONTEND_PID 2>/dev/null || true; \
	}; \
	trap cleanup EXIT INT TERM; \
	cd /home/varun/personal/sync-audio-platforms-go/backend && \
	APP_ENV=development \
	PORT=$$BACKEND_PORT \
	ALLOWED_ORIGIN=http://127.0.0.1:$$FRONTEND_PORT \
	STORE_PROVIDER=sqlite \
	SQLITE_PATH=/tmp/sync-audio-e2e.db \
	SESSION_HMAC_KEY=afa47ce16f2d262225a23d1ad15f1ad050f3622415631ff62bca2acda3601d21 \
	TOKEN_ENCRYPTION_KEY=MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY= \
	DEFAULT_RATE_LIMIT_PER_MIN=120 \
	SIGNUP_RATE_LIMIT_PER_HOUR=20 \
	ADMIN_EMAILS=dev@example.com \
	TURNSTILE_SECRET_KEY= \
	SIGNUP_ACCESS_CODES=Ryanthisisforoouuuuu \
	ACCESS_CODE_MAX_USES=5 \
	ACCESS_CODE_MAX_FAILURES=5 \
	ACCESS_CODE_LOCKOUT_MINUTES=15 \
	go run ./cmd/api >$$BACKEND_LOG 2>&1 & \
	BACKEND_PID=$$!; \
	for i in 1 2 3 4 5 6 7 8 9 10; do \
		curl -fsS http://127.0.0.1:$$BACKEND_PORT/v1/health >/dev/null 2>&1 && break; \
		sleep 1; \
	done; \
	curl -fsS http://127.0.0.1:$$BACKEND_PORT/v1/health >/dev/null; \
	cd /home/varun/personal/sync-audio-platforms-go/frontend && \
	NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:$$BACKEND_PORT \
	NEXT_PUBLIC_TURNSTILE_SITE_KEY=test \
	npm run dev -- --hostname 127.0.0.1 --port $$FRONTEND_PORT >$$FRONTEND_LOG 2>&1 & \
	FRONTEND_PID=$$!; \
	for i in 1 2 3 4 5 6 7 8 9 10 11 12; do \
		curl -fsS http://127.0.0.1:$$FRONTEND_PORT/login >/dev/null 2>&1 && break; \
		sleep 1; \
	done; \
	curl -fsS http://127.0.0.1:$$FRONTEND_PORT/login >/dev/null; \
	curl -fsS -c $$COOKIE_JAR -b $$COOKIE_JAR -H "Origin: http://127.0.0.1:$$FRONTEND_PORT" -H "Content-Type: application/json" \
		-d "$$(printf '{"email":"%s","captchaToken":"test-token","accessCode":"Ryanthisisforoouuuuu","website":""}' "$$EMAIL")" \
		http://127.0.0.1:$$BACKEND_PORT/v1/auth/login >/tmp/sync-audio-e2e-login.json; \
	curl -fsS -b $$COOKIE_JAR -H "Origin: http://127.0.0.1:$$FRONTEND_PORT" http://127.0.0.1:$$BACKEND_PORT/v1/me >/tmp/sync-audio-e2e-me.json; \
	grep -q "$$EMAIL" /tmp/sync-audio-e2e-me.json; \
	grep -q '"status":"approved"' /tmp/sync-audio-e2e-me.json; \
	echo "e2e-local-sqlite: PASS"

docker-build:
	$(DOCKER) compose -f $(COMPOSE_FILE) build

docker-up:
	$(DOCKER) compose -f $(COMPOSE_FILE) up -d

docker-down:
	$(DOCKER) compose -f $(COMPOSE_FILE) down

docker-logs:
	$(DOCKER) compose -f $(COMPOSE_FILE) logs -f --tail=100

docker-up-dev:
	@test -f $(ENV_FILE) || (echo "Missing $(ENV_FILE). Copy env.docker.example to .env.docker and fill values."; exit 1)
	$(DOCKER) compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) up -d --build

docker-down-dev:
	@test -f $(ENV_FILE) || (echo "Missing $(ENV_FILE)."; exit 1)
	$(DOCKER) compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) down

