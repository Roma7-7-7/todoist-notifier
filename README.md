# Todoist Notifier

Sends Telegram notifications for uncompleted Todoist tasks due today.

## What It Does

Fetches uncompleted tasks from Todoist and sends them to Telegram. Tasks are filtered by:
- **Due date** - only tasks due today
- **Time labels** - tasks with `12pm`, `3pm`, `6pm`, or `9pm` labels are hidden until that hour passes
- **Priority** - sorted by priority (🔴 P1, 🟠 P2, 🔵 P3, ⚪ P4)

Example: A task labeled `3pm` won't appear in notifications until 3 PM, even if it's due today.

## Deployment Modes

**Lambda** - Event-driven function triggered by AWS EventBridge (e.g., every 30 minutes)

**Daemon** - Long-running process with cron scheduler (e.g., hourly from 9am-11pm)

**For deployment instructions**, see [deployment/README.md](deployment/README.md)

## Quick Start

**Build:**
```bash
make build          # Both Lambda and daemon
make build-lambda   # Lambda only (ARM64)
make build-daemon   # Daemon only (AMD64)
```

See `Makefile` for all build targets.

**Run locally:**
```bash
# Lambda mode (runs once)
ENV=dev TODOIST_TOKEN=<token> TELEGRAM_BOT_ID=<bot_id> TELEGRAM_CHAT_ID=<chat_id> \
  go run cmd/lambda/main.go

# Daemon mode (runs with scheduler)
ENV=dev TODOIST_TOKEN=<token> TELEGRAM_BOT_ID=<bot_id> TELEGRAM_CHAT_ID=<chat_id> \
  SCHEDULE="0 * 9-23 * * *" go run cmd/daemon/main.go
```

## Configuration

**Environment Variables:**
- `TODOIST_TOKEN` - Todoist API token (required)
- `TELEGRAM_BOT_ID` - Telegram bot token (required)
- `TELEGRAM_CHAT_ID` - Telegram chat ID (required)
- `SCHEDULE` - Cron expression for daemon mode (default: `0 * 9-23 * * *`)
- `LOCATION` - Timezone (default: `Europe/Kyiv`)
- `ENV` - Set to `dev` for development mode
- `FORCE_SSM` - Set to `true` to use AWS SSM Parameter Store

**Production (AWS SSM):**

Set `FORCE_SSM=true` and store secrets in SSM at:
- `/todoist-notifier-bot/prod/todoist-token`
- `/todoist-notifier-bot/prod/telegram-token`
- `/todoist-notifier-bot/prod/telegram-chat-id`

## Architecture

```
cmd/
  lambda/     - Lambda entry point
  daemon/     - Daemon entry point with cron scheduler
internal/
  notifier.go - Shared notification logic
  tasks.go    - Task filtering and rendering
  config.go   - Configuration management
pkg/
  todoist/    - Todoist API client
  ssm/        - AWS SSM wrapper
```

Both modes use the same core `Notifier` - Lambda wraps it for AWS Lambda, daemon wraps it with a scheduler.

## Development

See `CLAUDE.md` for comprehensive development guidelines including:
- Code style and linting rules
- Error handling conventions
- Configuration management patterns
- Testing and debugging
