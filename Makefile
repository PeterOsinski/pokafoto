.PHONY: dev build build-web test test-cover test-web test-all lint clean docker docker-rebuild

dev:
	@echo "Starting backend..."
	@go run ./cmd/drive

dev-web:
	cd web && npm run dev

build:
	go build -ldflags="-s -w" -o bin/drive ./cmd/drive

build-web:
	cd web && npm run build

test:
	go test -count=1 ./...

test-cover:
	go test -race -count=1 -coverprofile=coverage.out ./...

test-web:
	cd web && npx vitest run

test-all: test test-web

lint:
	golangci-lint run 2>/dev/null || echo "golangci-lint not installed"

clean:
	rm -rf bin/
	rm -rf web/dist/
	rm -rf data/

docker:
	COMPOSE_BAKE=true DOCKER_BUILDKIT=1 docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-rebuild: docker docker-down docker-up
