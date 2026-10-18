.PHONY: dev db-up db-down build test vet run init web web-dev release

DEV_DB_URL ?= postgres://messario:messario@localhost:5432/messario?sslmode=disable

db-up:
	docker compose -f deploy/docker-compose.dev.yml up -d --wait

db-down:
	docker compose -f deploy/docker-compose.dev.yml down

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/messario ./cmd/messario

# Build the Vue client into internal/web/dist so `make build` embeds it.
web:
	cd web && npm ci --no-audit --no-fund && npm run build

# Vite dev server on :5173 proxying to the Go API on :8080.
web-dev:
	cd web && npm run dev

release: web build

vet:
	go vet ./...

test:
	MESSARIO_TEST_DATABASE_URL=$(DEV_DB_URL) go test ./...

run: build
	MESSARIO_DATABASE_URL=$(DEV_DB_URL) MESSARIO_COOKIE_SECURE=false ./bin/messario serve

# make init ORG="My Company" ADMIN=admin PASSWORD=changeme
init: build
	MESSARIO_DATABASE_URL=$(DEV_DB_URL) ./bin/messario init --org "$(ORG)" --admin "$(ADMIN)" --password "$(PASSWORD)"

dev: db-up run
