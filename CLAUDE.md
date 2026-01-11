# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based notification system that sends Telegram notifications for Todoist tasks. It supports two deployment modes:
- **Lambda Mode**: Event-driven AWS Lambda function triggered by EventBridge
- **Daemon Mode**: Long-running daemon for EC2 instances with configurable scheduling

Both modes share the same core notification logic and filter tasks due today based on time labels (12pm, 3pm, 6pm, 9pm).

## Build and Development Commands

### Build All
```bash
make build
```
Builds both Lambda (ARM64) and daemon (AMD64) binaries.

### Build Lambda Only
```bash
make build-lambda
```
Builds the Lambda ARM64 binary and creates a deployment zip at `./bin/todoist-notifier-lambda-arm.zip`.

### Build Daemon (AMD64)
```bash
make build-daemon
```
Builds the daemon binary for x86_64 EC2 instances at `./bin/todoist-notifier-daemon`.

### Build Daemon (ARM64)
```bash
make build-daemon-arm
```
Builds the daemon binary for ARM64/Graviton EC2 instances at `./bin/todoist-notifier-daemon-arm`.

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

**Lambda mode (run once):**
```bash
ENV=dev TODOIST_TOKEN=<token> TELEGRAM_BOT_ID=<bot_id> TELEGRAM_CHAT_ID=<chat_id> go run cmd/lambda/main.go
```

**Daemon mode (runs continuously with scheduler + interactive bot):**
```bash
ENV=dev TODOIST_TOKEN=<token> TELEGRAM_BOT_ID=<bot_id> TELEGRAM_CHAT_ID=<chat_id> SCHEDULE="*/5 * * * *" go run cmd/daemon/main.go
```

The daemon mode includes:
- Cron scheduler for automated notifications (configured via `SCHEDULE`)
- Interactive Telegram bot responding to `/tasks` command
- Both components run in the same process and share notification logic

**Testing the Todoist client:**
```bash
ENV=dev TOKEN=<token> go run cmd/app/main.go
```

## Architecture

### Entry Points
- `cmd/lambda/main.go` - AWS Lambda entry point for event-driven deployment
- `cmd/daemon/main.go` - Long-running daemon for EC2 instances with scheduler
- `cmd/app/main.go` - Development/testing tool for direct Todoist API interaction

### Core Components

**Notification Engine** (`internal/notifier.go`)
- `Notifier` struct - Core notification logic shared by both Lambda and daemon modes
- `SendNotification(ctx)` - Fetches tasks, filters for today, renders message, sends via Telegram
- Interfaces `TodoistClient` and `HTTPMessagePublisher` enable testing/mocking

**Configuration Management** (`internal/config.go`)
- Supports dual configuration sources: environment variables or AWS SSM Parameter Store
- **Simple deployment**: If all required env vars are set (`TODOIST_TOKEN`, `TELEGRAM_BOT_ID`, `TELEGRAM_CHAT_ID`), SSM is automatically skipped
- **AWS deployment**: If required env vars are missing, SSM Parameter Store is used (requires AWS credentials)
- Timezone configuration via `LOCATION` env var (defaults to "Europe/Kyiv")
- `SCHEDULE` env var for daemon mode cron expression (defaults to "0 * 9-23 * * *" - every hour from 9am to 11pm)

**Task Filtering Logic** (`internal/tasks.go`)
- `FilterAndSortTasks()` implements time-based filtering using task labels
- Tasks with time labels (12pm, 3pm, 6pm, 9pm) are filtered based on current hour
- Tasks are sorted by priority (descending) then project ID
- Template-based message rendering with priority-based emoji circles (🔴🟠🔵⚪)

**Lambda Handler** (`internal/lambda.go`)
- Thin wrapper around `Notifier` for AWS Lambda integration
- Delegates all notification logic to shared `Notifier.SendNotification()`

**Daemon Scheduler** (`cmd/daemon/main.go`)
- Cron-based scheduling using gocron v2 library
- Configurable via `SCHEDULE` env var with standard cron expressions (minute hour day month weekday)
- Graceful shutdown on SIGINT/SIGTERM
- **Does NOT run notification on startup** - waits for first scheduled time to avoid unnecessary notifications during deployments
- Default schedule: "0 * 9-23 * * *" (every hour from 9am to 11pm)
- Timezone-aware scheduling using `LOCATION` env var

**Bot Integration** (`internal/bot.go`)
- Interactive Telegram bot using `gopkg.in/telebot.v3`
- Runs alongside scheduler in daemon mode (both share same `Notifier` instance)
- Chat ID validation middleware blocks unauthorized access
- `/tasks` command - returns filtered tasks for current time
- Reuses existing task filtering and message rendering logic
- Graceful shutdown on context cancellation

### External Packages

**pkg/todoist** - Todoist REST API client
- `GetTasks(ctx, isCompleted)` - Fetch tasks with completion status filter
- `UpdateTask(ctx, taskID, updateReq)` - Update task priority and labels
- Built-in retry logic with configurable retries and delay
- Returns structured `Task` objects with ID, Content, Priority, Due date, Labels

**pkg/ssm** - AWS Systems Manager Parameter Store wrapper
- `FetchParameters()` helper for batch parameter retrieval with decryption support

### Dependencies
- AWS Lambda Go SDK for Lambda execution (Lambda mode only)
- AWS SDK v2 for SSM parameter access
- Custom Telegram client: `github.com/Roma7-7-7/telegram`
- Telegram bot library: `gopkg.in/telebot.v3` (daemon mode only)
- Cron scheduler: `github.com/go-co-op/gocron/v2` (daemon mode only)
- Environment config: `github.com/kelseyhightower/envconfig`

## Deployment

### Lambda Deployment
The Lambda function expects:
- ARM64 architecture (cross-compiled in Makefile)
- SSM parameters at `/todoist-notifier-bot/prod/{todoist-token,telegram-token,telegram-chat-id}`
- Bootstrap binary naming for custom Lambda runtime
- EventBridge rule for scheduling (e.g., every 30 minutes during 9am-11pm)

**Environment Variables:**
- `ENV`: Set to "prod" (default) or "dev"
- `LOCATION`: Timezone (default: "Europe/Kyiv")

**Note**: Lambda deployments typically use SSM for secrets. If you want to test Lambda locally with env vars, set `TODOIST_TOKEN`, `TELEGRAM_BOT_ID`, and `TELEGRAM_CHAT_ID` to skip SSM.

### Daemon Deployment (EC2)
The daemon binary can run on any Linux EC2 instance:

**Environment Variables:**
- `ENV`: Set to "prod" (default) or "dev"
- `TODOIST_TOKEN`: Todoist API token (required for simple deployment, optional if using SSM)
- `TELEGRAM_BOT_ID`: Telegram bot token (required for simple deployment, optional if using SSM)
- `TELEGRAM_CHAT_ID`: Telegram chat ID (required for simple deployment, optional if using SSM)
- `SCHEDULE`: Cron expression for notification schedule - defaults to "0 * 9-23 * * *" (every hour from 9am to 11pm)
  - Format: `minute hour day month weekday` (standard cron without seconds)
  - Examples:
    - `"0 * 9-23 * * *"` - Every hour from 9am to 11pm (default)
    - `"0,30 * 9-23 * * *"` - Every 30 minutes from 9am to 11pm
    - `"0 9,12,15,18,21 * * *"` - At 9am, 12pm, 3pm, 6pm, and 9pm
    - `"*/15 * * * *"` - Every 15 minutes (all day)
- `LOCATION`: Timezone (default: "Europe/Kyiv")

**Deployment Options:**

**Option 1: Simple deployment with environment variables (recommended for non-AWS)**
```bash
ENV=prod \
SCHEDULE="0 * 9-23 * * *" \
TODOIST_TOKEN=<token> \
TELEGRAM_BOT_ID=<bot_id> \
TELEGRAM_CHAT_ID=<chat_id> \
./todoist-notifier-daemon
```

**Option 2: AWS deployment with SSM Parameter Store**
- Attach IAM instance profile with `ssm:GetParameters` permission
- Store secrets in SSM at `/todoist-notifier-bot/prod/{todoist-token,telegram-token,telegram-chat-id}`
- Don't set `TODOIST_TOKEN`, `TELEGRAM_BOT_ID`, or `TELEGRAM_CHAT_ID` env vars (SSM will be used automatically)

**Running as systemd service (simple deployment):**
```ini
[Unit]
Description=Todoist Telegram Notifier
After=network.target

[Service]
Type=simple
User=your-user
Environment="ENV=prod"
Environment="SCHEDULE=0 * 9-23 * * *"
Environment="LOCATION=Europe/Kyiv"
Environment="TODOIST_TOKEN=your-token"
Environment="TELEGRAM_BOT_ID=your-bot-id"
Environment="TELEGRAM_CHAT_ID=your-chat-id"
ExecStart=/usr/local/bin/todoist-notifier-daemon
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

**Running as systemd service (SSM deployment):**
```ini
[Unit]
Description=Todoist Telegram Notifier
After=network.target

[Service]
Type=simple
User=ec2-user
Environment="ENV=prod"
Environment="SCHEDULE=0 * 9-23 * * *"
Environment="LOCATION=Europe/Kyiv"
ExecStart=/usr/local/bin/todoist-notifier-daemon
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

**With environment variables (deprecated, use Option 1 or 2 above):**
```bash
ENV=prod \
SCHEDULE="0 * 9-23 * * *" \
TODOIST_TOKEN=<token> \
TELEGRAM_BOT_ID=<bot_id> \
TELEGRAM_CHAT_ID=<chat_id> \
./todoist-notifier-daemon
```

## Development Notes

### Configuration Management Convention

**CRITICAL RULE**: All environment variables and their default values MUST be defined in `internal/config.go`.

**Never read environment variables directly with `os.Getenv()` outside of `GetConfig()`** - always use the Config struct.

**Why this matters:**
- Single source of truth for all configuration
- Centralized default values
- Easier testing (can pass mock Config instead of setting env vars)
- Clear visibility of all configuration options
- Prevents reading ENV multiple times (DRY principle)

**Current environment variables (all defined in Config struct):**
- `ENV` → `Config.Dev` (bool) - defaults to false (prod mode)
- `TODOIST_TOKEN` → `Config.TodoistToken` - required (via env or SSM)
- `TELEGRAM_BOT_ID` → `Config.TelegramToken` - required (via env or SSM)
- `TELEGRAM_CHAT_ID` → `Config.TelegramChatID` - required (via env or SSM)
- `SCHEDULE` → `Config.Schedule` - defaults to "0 * 9-23 * * *" (every hour from 9am to 11pm)
- `LOCATION` → `Config.Location` - defaults to "Europe/Kyiv"

**Example:**
```go
// ✅ GOOD - use config struct
conf, err := internal.GetConfig(ctx)
log := internal.NewLogger(conf.Dev)
notifier, err := internal.NewNotifier(client, publisher, conf.TelegramChatID, conf.Location, log)

// ❌ BAD - don't read env vars directly
location := os.Getenv("LOCATION")
if location == "" {
    location = "Europe/Kyiv"
}
```

**Adding new configuration:**
1. Add field to `Config` struct in `internal/config.go`
2. Read env var in `GetConfig()` with appropriate default value
3. If the field is required (no default), add validation in `validate()` method
4. Update this documentation with the new env var
5. Use `conf.FieldName` throughout the codebase

**Validation:**
All required environment variables are validated in `GetConfig()`:
- Missing required variables cause immediate failure with clear error message
- Error lists all missing variables at once (not one at a time)
- Validation happens for both env var mode and SSM mode
- The `validate()` method is called after all config is loaded

### Linting Configuration
The `.golangci.yaml` is extremely comprehensive with strict settings:
- Function length limit: 100 lines, 50 statements
- Line length: 175 characters
- Cognitive complexity: max 30
- All security linters enabled (gosec, errchkjson, wrapcheck)
- Test files exempt from: bodyclose, dupl, errcheck, funlen, goconst, gosec, noctx, wrapcheck

### Magic Number Linting
Use `//nolint:mnd // reason` for justified magic numbers, as the project enables the `mnd` (magic number detector) linter.

### Code Comments

**DO NOT add comments that simply describe what the code does** - the code itself should be self-documenting.

**Only add comments when:**
- Explaining WHY something is done (not what is done)
- Documenting non-obvious behavior or edge cases
- Providing context that cannot be expressed in code
- Package-level documentation (godoc)
- Public API documentation

**Examples:**
```go
// ❌ BAD - comment just repeats what code does
// Get config first to determine environment
conf, err := internal.GetConfig(ctx)

// ❌ BAD - obvious from the code
// Create logger
log := internal.NewLogger(conf.Dev)

// ❌ BAD - states the obvious
// Start scheduler
scheduler.Start()

// ✅ GOOD - explains a constraint/requirement
const maxRetries = 5 // API rate limit requires exponential backoff after 3 failures

// ✅ GOOD - explains non-obvious behavior
// NewLogger must be called after GetConfig to avoid reading ENV twice
func NewLogger(isDev bool) *slog.Logger

// ✅ GOOD - godoc for public API
// FetchParameters retrieves multiple SSM parameters and populates their destination pointers.
// Returns ErrParamsNotFound if any requested parameter does not exist.
func FetchParameters(ctx context.Context, client Client, params map[string]*string, opts ...OptionsF) error
```

**General rule**: If the comment can be removed and the code is still clear, remove it.

### Logging Configuration

**ALWAYS use `internal.NewLogger(isDev)` for logger initialization** - never create loggers directly.

The `NewLogger()` function configures logging based on environment:

**DEV environment (`isDev=true`):**
- Text handler for human-readable output
- DEBUG level for detailed logging
- Suitable for local development and debugging

**PROD environment (`isDev=false`):**
- JSON handler for structured logging
- INFO level for production use
- Suitable for AWS CloudWatch and log aggregation systems

**Important**: Always initialize config first, then create the logger based on `conf.Dev`:

```go
// ✅ GOOD - get config first, then create logger
conf, err := internal.GetConfig(ctx)
if err != nil {
    slog.ErrorContext(ctx, "failed to get config", "error", err)
    return 1
}
log := internal.NewLogger(conf.Dev)

// ❌ BAD - don't create loggers directly
log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))

// ❌ BAD - don't read ENV multiple times
isDev := os.Getenv("ENV") == "dev"
log := internal.NewLogger(isDev)
```

**Why this order?**
- Avoids reading `ENV` variable multiple times (DRY principle)
- Config is the single source of truth for environment determination
- For config errors, use default `slog` package directly since we don't have config yet

This ensures consistent logging configuration across all entry points (Lambda, daemon, dev tools).

### Context-Aware Logging

**CRITICAL RULE**: Always use context-aware logging methods when a context is available.

**Always use the `Context` variants of logging methods:**
```go
// ✅ GOOD - use context-aware logging
log.InfoContext(ctx, "message sent")
log.WarnContext(ctx, "failed to get next run time", "error", err)
log.ErrorContext(ctx, "failed to create notifier", "error", err)
log.DebugContext(ctx, "no tasks for today")

// ❌ BAD - don't use non-context logging when context is available
log.Info("message sent")
log.Warn("failed to get next run time", "error", err)
log.Error("failed to create notifier", "error", err)
log.Debug("no tasks for today")
```

**Why use context-aware logging?**
- Enables distributed tracing and request correlation
- Automatically includes context metadata in logs
- Supports timeout and cancellation tracking
- Better integration with observability tools
- Consistent structured logging with context propagation

**When NOT to use context logging:**
- Only use non-context methods (`Info`, `Warn`, etc.) when no context is available
- This should be rare - most functions should accept and propagate context

### Error Handling

**CRITICAL RULE**: All errors returned from public functions/methods MUST be wrapped with context.

**Always wrap errors when returning them:**
```go
err := DoSomething()
if err != nil {
    return fmt.Errorf("do something: %w", err)
}
```

**Key principles:**
1. **Wrap errors from external packages/libraries** - they need context in your domain
2. **Don't wrap errors from private methods that already wrap** - avoid double-wrapping and redundant messages
3. **Never return bare `err`** from public functions/methods unless it's already wrapped by a private method
4. Use lowercase context messages without ending punctuation: `"get tasks: %w"` not `"Get tasks: %w."`
5. Use the `%w` verb to wrap errors (enables error unwrapping with `errors.Is` and `errors.As`)
6. Context should describe the operation that failed, not repeat error details
7. The wrapcheck linter enforces this rule and will fail the build if violated

**Examples from the codebase:**
```go
// ✅ GOOD - wrapped with context
tasks, err := n.todoistClient.GetTasks(ctx, false)
if err != nil {
    return fmt.Errorf("get tasks: %w", err)
}

// ✅ GOOD - wrapped with parameter context
loc, err := time.LoadLocation(location)
if err != nil {
    return nil, fmt.Errorf("error loading timezone %q: %w", location, err)
}

// ❌ BAD - bare error return from external library
err := externalLib.DoSomething()
if err != nil {
    return err  // Missing context - what failed?
}

// ✅ GOOD - don't double-wrap private method errors
resp, err := c.doWithRetry(ctx, req)  // doWithRetry already wraps with "do request with %d retries: %w"
if err != nil {
    return nil, err  // Already has context, don't add more
}

// ❌ BAD - double-wrapping creates redundant messages
resp, err := c.doWithRetry(ctx, req)
if err != nil {
    return nil, fmt.Errorf("do request with retry: %w", err)  // Results in "do request with retry: do request with 5 retries: ..."
}
```

**When NOT to wrap:**
- Errors from private methods that already provide wrapped errors with good context
- In `main()` functions and other entry points where errors are only logged (terminal handlers)
- When re-returning an error that's already wrapped with sufficient context

**Decision rule**: Ask "does this error already have enough context?" If the error comes from a private helper that wraps it, don't wrap again. If it comes from an external library or the standard library, wrap it.

### Time Label System
The notification system uses Todoist labels for time-based filtering:
- `12pm` - task shows after noon
- `3pm` - task shows after 15:00
- `6pm` - task shows after 18:00  
- `9pm` - task shows after 21:00
Tasks without time labels or with passed time labels appear in notifications.

### Code Reusability
Both Lambda and daemon modes share the same core logic:
- `internal/notifier.go` contains the shared `Notifier` type
- `internal/lambda.go` is a thin wrapper for Lambda integration
- `cmd/daemon/main.go` adds scheduler logic around the shared notifier
- All notification logic (fetch, filter, render, send) is in one place
