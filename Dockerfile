# build
FROM golang:1.22.3-alpine3.20 AS build

RUN apk add --no-cache make

COPY . /app
WORKDIR /app
RUN make build

# run
FROM alpine:3.19

EXPOSE 8080

ENV SCHEDULE="0 9-23 * * *"
ENV TODOIST_TOKEN=""
ENV TELEGRAM_BOT_ID=""
ENV TELEGRAM_CHAT_ID=""

COPY --from=build /app/bin/todoist-notifier /app/todoist-notifier

WORKDIR /app

ENTRYPOINT ["/app/todoist-notifier"]
