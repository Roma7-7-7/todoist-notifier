
build:
	go mod download
	CGO_ENABLED=0 go build -o ./bin/todoist-notifier ./cmd/telegram/main.go

docker-build:
	docker build -t todoist-notifier .

docker-compose:
	docker-compose up -d