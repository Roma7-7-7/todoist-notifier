VERSION ?= dev
BUILD_TIME ?= $(shell date -u +%Y%m%d-%H%M%S)
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS = -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)
LDFLAGS_RELEASE = $(LDFLAGS) -w -s

clean:
	rm -rf ./bin

build-lambda: clean
	go mod download
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ./bin/todoist-notifier-lambda-arm ./cmd/lambda/main.go
	cp ./bin/todoist-notifier-lambda-arm ./bin/bootstrap
	zip -j ./bin/todoist-notifier-lambda-arm.zip ./bin/bootstrap

build-daemon: clean
	go mod download
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/todoist-notifier-daemon ./cmd/daemon/main.go

build-daemon-arm: clean
	go mod download
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ./bin/todoist-notifier-daemon-arm ./cmd/daemon/main.go

build: build-lambda build-daemon

build-lambda-release:
	@mkdir -p ./bin
	go mod download
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS_RELEASE)" -o ./bin/todoist-notifier-lambda-arm ./cmd/lambda/main.go
	cp ./bin/todoist-notifier-lambda-arm ./bin/bootstrap
	cd ./bin && zip -j todoist-notifier-lambda-arm.zip bootstrap
	rm ./bin/bootstrap

build-daemon-release:
	@mkdir -p ./bin
	go mod download
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS_RELEASE)" -o ./bin/todoist-notifier-daemon-amd64 ./cmd/daemon/main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS_RELEASE)" -o ./bin/todoist-notifier-daemon-arm64 ./cmd/daemon/main.go

build-release: build-lambda-release build-daemon-release
	@echo "$(GIT_COMMIT)" > ./bin/VERSION
	@echo "Build complete: $(VERSION) ($(BUILD_TIME))"
