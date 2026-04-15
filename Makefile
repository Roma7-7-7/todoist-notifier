VERSION ?= dev
BUILD_TIME ?= $(shell date -u +%Y%m%d-%H%M%S)

LDFLAGS = -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)

build:
	go mod download
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o ./bin/todoist-notifier ./cmd/daemon

test:
	go test -v ./...

# ==========================================
# Docker
# ==========================================

docker-build: ## Build Docker image
	VERSION=$(VERSION) BUILD_TIME=$(BUILD_TIME) docker-compose build

docker-up: ## Build and run with Docker Compose
	VERSION=$(VERSION) BUILD_TIME=$(BUILD_TIME) docker-compose up --build -d

docker-down: ## Stop Docker Compose services
	docker-compose down
