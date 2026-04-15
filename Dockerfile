# --- Build stage ---
FROM golang:1.26-alpine AS builder

ARG VERSION=dev
ARG BUILD_TIME=unknown

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/
COPY pkg/ pkg/

RUN CGO_ENABLED=0 go build \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
    -o /todoist-notifier ./cmd/daemon

# --- Runtime stage ---
FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata

RUN adduser -D -u 1000 appuser
WORKDIR /app

COPY --from=builder /todoist-notifier .

USER appuser

ENTRYPOINT ["./todoist-notifier"]
