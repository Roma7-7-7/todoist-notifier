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
