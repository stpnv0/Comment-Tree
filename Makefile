.PHONY: build run test lint mocks migrate-up migrate-down docker-up docker-down clean

APP_NAME := commenttree
BUILD_DIR := ./bin


build:
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/commenttree

run: build
	CONFIG_PATH=./config/config.yaml $(BUILD_DIR)/$(APP_NAME)

test:
	go test -v -count=1 -race ./...

lint:
	golangci-lint run ./...

mocks:
	mockery

migrate-up:
	goose -dir ./migrations postgres "$(DB_DSN)" up

migrate-down:
	goose -dir ./migrations postgres "$(DB_DSN)" down

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f
