clean:
	rm -rf ./bin

build: clean
	go mod download
	CGO_ENABLED=0 go build -o ./bin/todoist-notifier ./cmd/telegram/main.go

build-lambda-arm: clean
	go mod download
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ./bin/todoist-notifier-lambda-arm ./cmd/lambda/main.go
	cp ./bin/todoist-notifier-lambda-arm ./bin/bootstrap
	zip -j ./bin/todoist-notifier-lambda-arm.zip ./bin/bootstrap
