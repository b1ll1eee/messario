.PHONY: dev db-up db-down build test vet run init

DEV_DB_URL ?= postgres://nuntio:nuntio@localhost:5432/nuntio?sslmode=disable

db-up:
	docker compose -f deploy/docker-compose.dev.yml up -d --wait

db-down:
	docker compose -f deploy/docker-compose.dev.yml down

build:
	go build -o bin/nuntio ./cmd/nuntio

vet:
	go vet ./...

test:
	NUNTIO_TEST_DATABASE_URL=$(DEV_DB_URL) go test ./...

run: build
	NUNTIO_DATABASE_URL=$(DEV_DB_URL) NUNTIO_COOKIE_SECURE=false ./bin/nuntio serve

# make init ORG="My Company" ADMIN=admin PASSWORD=changeme
init: build
	NUNTIO_DATABASE_URL=$(DEV_DB_URL) ./bin/nuntio init --org "$(ORG)" --admin "$(ADMIN)" --password "$(PASSWORD)"

dev: db-up run
