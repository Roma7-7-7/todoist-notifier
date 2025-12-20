# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based AWS Lambda function that sends Telegram notifications for Todoist tasks. It filters tasks due today based on time labels (12pm, 3pm, 6pm, 9pm) and sends formatted notifications through Telegram.

## Build and Development Commands

### Build
```bash
make build
```
Builds the Lambda ARM64 binary and creates a deployment zip at `./bin/todoist-notifier-lambda-arm.zip`.

### Lint
```bash
golangci-lint run
```
Runs comprehensive linting with the extensive configuration in `.golangci.yaml`. The project uses 70+ linters including security checks (gosec, errchkjson, wrapcheck).

### Test
```bash
go test -v ./...
```
Runs all tests in the project.

### Local Development
```bash
ENV=dev TODOIST_TOKEN=<token> TELEGRAM_BOT_ID=<bot_id> TELEGRAM_CHAT_ID=<chat_id> go run cmd/lambda/main.go
```
Runs the lambda handler locally in dev mode (doesn't start lambda server, runs once).

For testing the Todoist client:
```bash
ENV=dev TOKEN=<token> go run cmd/app/main.go
```

## Architecture

### Entry Points
- `cmd/lambda/main.go` - AWS Lambda entry point for production deployment
- `cmd/app/main.go` - Development/testing tool for direct Todoist API interaction

### Core Components

**Configuration Management** (`internal/config.go`)
- Supports dual configuration sources: environment variables (dev) or AWS SSM Parameter Store (prod)
- Uses `FORCE_SSM=true` to override dev mode and force SSM usage
- Timezone configuration via `LOCATION` env var (defaults to "Europe/Kyiv")

**Task Filtering Logic** (`internal/tasks.go`)
- `FilterAndSortTasks()` implements time-based filtering using task labels
- Tasks with time labels (12pm, 3pm, 6pm, 9pm) are filtered based on current hour
- Tasks are sorted by priority (descending) then project ID
- Template-based message rendering with priority-based emoji circles (🔴🟠🔵⚪)

**Lambda Handler** (`internal/lambda.go`)
- Only processes requests during working hours (9:00-23:00 in configured timezone)
- Fetches uncompleted tasks, filters for today, renders message, sends via Telegram
- Interfaces `TodoistClient` and `HTTPMessagePublisher` enable testing/mocking

### External Packages

**pkg/todoist** - Todoist REST API client
- `GetTasks(ctx, isCompleted)` - Fetch tasks with completion status filter
- `UpdateTask(ctx, taskID, updateReq)` - Update task priority and labels
- Built-in retry logic with configurable retries and delay
- Returns structured `Task` objects with ID, Content, Priority, Due date, Labels

**pkg/ssm** - AWS Systems Manager Parameter Store wrapper
- `FetchParameters()` helper for batch parameter retrieval with decryption support

### Dependencies
- AWS Lambda Go SDK for Lambda execution
- AWS SDK v2 for SSM parameter access
- Custom Telegram client: `github.com/Roma7-7-7/telegram`
- Environment config: `github.com/kelseyhightower/envconfig`

## Development Notes

### Linting Configuration
The `.golangci.yaml` is extremely comprehensive with strict settings:
- Function length limit: 100 lines, 50 statements
- Line length: 175 characters
- Cognitive complexity: max 30
- All security linters enabled (gosec, errchkjson, wrapcheck)
- Test files exempt from: bodyclose, dupl, errcheck, funlen, goconst, gosec, noctx, wrapcheck

### Magic Number Linting
Use `//nolint:mnd // reason` for justified magic numbers, as the project enables the `mnd` (magic number detector) linter.

### Error Handling
All errors must be wrapped (wrapcheck linter enabled). Use `fmt.Errorf("context: %w", err)` pattern consistently.

### Time Label System
The notification system uses Todoist labels for time-based filtering:
- `12pm` - task shows after noon
- `3pm` - task shows after 15:00
- `6pm` - task shows after 18:00  
- `9pm` - task shows after 21:00
Tasks without time labels or with passed time labels appear in notifications.

### Deployment
The Lambda function expects:
- ARM64 architecture (cross-compiled in Makefile)
- SSM parameters at `/todoist-notifier-bot/prod/{todoist-token,telegram-token,telegram-chat-id}`
- Bootstrap binary naming for custom Lambda runtime
