# Todoist Notifier Deployment Guide

This guide covers deploying the Todoist Telegram Notifier in both daemon mode (EC2/VPS) and Lambda mode (AWS).

## Table of Contents

- [Daemon Mode (EC2/VPS)](#daemon-mode-ecvps)
  - [Automated Setup](#automated-setup)
  - [Manual Setup](#manual-setup)
  - [Configuration](#configuration)
  - [Updating](#updating)
  - [Management](#management)
- [Lambda Mode (AWS)](#lambda-mode-aws)
- [Troubleshooting](#troubleshooting)

---

## Daemon Mode (EC2/VPS)

The daemon runs as a long-running process with:
- Cron-based scheduling for notifications
- Interactive Telegram bot responding to `/tasks` command
- Auto-restart on failure
- Automatic updates from GitHub releases

### Automated Setup

**One-time setup** (recommended):

```bash
curl -fsSL https://raw.githubusercontent.com/Roma7-7-7/todoist-notifier/main/deployment/setup.sh | sudo bash
```

This will:
1. Create directory structure at `/opt/todoist-notifier`
2. Download deployment scripts
3. Install systemd service
4. Configure passwordless sudo for service management
5. Prompt for configuration (tokens, schedule, timezone)
6. Deploy the latest binary
7. Enable auto-start on boot

**What you'll need:**
- Todoist API token (from https://todoist.com/app/settings/integrations/developer)
- Telegram bot token (from @BotFather)
- Your Telegram chat ID (send a message to @userinfobot)

### Manual Setup

If you prefer manual control:

1. **Create directory structure:**
```bash
sudo mkdir -p /opt/todoist-notifier/{bin,backups}
```

2. **Download latest binary:**
```bash
# For AMD64/x86_64 (t3, m5, c5 instances)
curl -L -o todoist-notifier-daemon \
  https://github.com/Roma7-7-7/todoist-notifier/releases/latest/download/todoist-notifier-daemon-amd64

# For ARM64 (t4g, m6g, c6g Graviton instances)
curl -L -o todoist-notifier-daemon \
  https://github.com/Roma7-7-7/todoist-notifier/releases/latest/download/todoist-notifier-daemon-arm64

chmod +x todoist-notifier-daemon
sudo mv todoist-notifier-daemon /opt/todoist-notifier/bin/
```

3. **Create environment file:**
```bash
sudo nano /opt/todoist-notifier/.env
```

```ini
# Environment
ENV=prod

# Todoist Configuration
TODOIST_TOKEN=your_todoist_token_here

# Telegram Configuration
TELEGRAM_BOT_ID=your_telegram_bot_token_here
TELEGRAM_CHAT_ID=your_telegram_chat_id_here

# Scheduler Configuration
SCHEDULE=0 * 9-23 * * *
LOCATION=Europe/Kyiv

# Optional: Force SSM Parameter Store usage (requires AWS configuration)
# FORCE_SSM=true
```

```bash
sudo chmod 600 /opt/todoist-notifier/.env
```

4. **Create systemd service:**
```bash
sudo curl -L -o /etc/systemd/system/todoist-notifier.service \
  https://raw.githubusercontent.com/Roma7-7-7/todoist-notifier/main/deployment/systemd/todoist-notifier.service

# Replace {{SERVICE_USER}} with your username
sudo sed -i "s/{{SERVICE_USER}}/$USER/g" /etc/systemd/system/todoist-notifier.service

sudo systemctl daemon-reload
sudo systemctl enable todoist-notifier.service
sudo systemctl start todoist-notifier.service
```

### Configuration

Edit `/opt/todoist-notifier/.env`:

**Required:**
- `TODOIST_TOKEN` - Your Todoist API token
- `TELEGRAM_BOT_ID` - Your Telegram bot token
- `TELEGRAM_CHAT_ID` - Your Telegram chat ID

**Optional:**
- `SCHEDULE` - Cron expression for notification schedule
  - Default: `0 * 9-23 * * *` (every hour from 9am to 11pm)
  - Format: `minute hour day month weekday`
  - Examples:
    - `0,30 * 9-23 * * *` - Every 30 minutes from 9am to 11pm
    - `0 9,12,15,18,21 * * *` - At 9am, 12pm, 3pm, 6pm, and 9pm
    - `*/15 * * * *` - Every 15 minutes (all day)
- `LOCATION` - Timezone (default: `Europe/Kyiv`)
- `FORCE_SSM` - Set to `true` to use AWS SSM Parameter Store for secrets

**After changing configuration:**
```bash
sudo systemctl restart todoist-notifier.service
```

### Updating

**Automatic update:**
```bash
/opt/todoist-notifier/deploy.sh
```

This script:
- Checks GitHub for the latest release
- Downloads the appropriate binary for your architecture
- Backs up the old binary (keeps last 5 backups)
- Restarts the service with the new version

**Optional: Set up automatic updates via cron:**
```bash
# Check for updates every 6 hours
crontab -e
```
Add:
```
0 */6 * * * /opt/todoist-notifier/deploy.sh
```

### Management

**Service status:**
```bash
sudo systemctl status todoist-notifier.service
```

**View logs:**
```bash
# Real-time logs
sudo journalctl -u todoist-notifier.service -f

# Last 100 lines
sudo journalctl -u todoist-notifier.service -n 100

# Logs from last hour
sudo journalctl -u todoist-notifier.service --since "1 hour ago"
```

**Restart service:**
```bash
sudo systemctl restart todoist-notifier.service
```

**Stop service:**
```bash
sudo systemctl stop todoist-notifier.service
```

**Start service:**
```bash
sudo systemctl start todoist-notifier.service
```

**Rollback to previous version:**
```bash
# List available backups
ls -lh /opt/todoist-notifier/backups/

# Copy backup binary
sudo cp /opt/todoist-notifier/backups/backup-YYYYMMDD-HHMMSS/todoist-notifier-daemon \
  /opt/todoist-notifier/bin/todoist-notifier-daemon

sudo systemctl restart todoist-notifier.service
```

---

## Lambda Mode (AWS)

Lambda mode provides event-driven execution via EventBridge triggers.

### Prerequisites

- AWS account with Lambda permissions
- AWS CLI configured (optional, can use AWS Console)

### Deployment

1. **Download Lambda package:**
```bash
curl -L -O https://github.com/Roma7-7-7/todoist-notifier/releases/latest/download/todoist-notifier-lambda-arm.zip
```

2. **Create Lambda function:**
   - Runtime: `provided.al2` (custom runtime)
   - Architecture: ARM64
   - Handler: `bootstrap`
   - Memory: 128 MB
   - Timeout: 30 seconds

3. **Upload deployment package:**
```bash
aws lambda update-function-code \
  --function-name todoist-notifier \
  --zip-file fileb://todoist-notifier-lambda-arm.zip
```

Or upload via AWS Console.

4. **Configure environment variables** (if not using SSM):
   - `ENV=dev`
   - `TODOIST_TOKEN=your_token`
   - `TELEGRAM_BOT_ID=your_bot_token`
   - `TELEGRAM_CHAT_ID=your_chat_id`
   - `LOCATION=Europe/Kyiv`

5. **Or configure SSM Parameter Store** (recommended):

```bash
aws ssm put-parameter \
  --name /todoist-notifier-bot/prod/todoist-token \
  --value "your_todoist_token" \
  --type SecureString

aws ssm put-parameter \
  --name /todoist-notifier-bot/prod/telegram-token \
  --value "your_telegram_bot_token" \
  --type SecureString

aws ssm put-parameter \
  --name /todoist-notifier-bot/prod/telegram-chat-id \
  --value "your_chat_id" \
  --type String
```

Add SSM permissions to Lambda execution role:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ssm:GetParameters"
      ],
      "Resource": "arn:aws:ssm:*:*:parameter/todoist-notifier-bot/prod/*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "kms:Decrypt"
      ],
      "Resource": "*"
    }
  ]
}
```

6. **Create EventBridge rule:**

Schedule expression examples:
- `cron(0 * * * ? *)` - Every hour
- `cron(0/30 9-23 * * ? *)` - Every 30 minutes from 9am to 11pm
- `cron(0 9,12,15,18,21 * * ? *)` - At 9am, 12pm, 3pm, 6pm, 9pm

Target: Your Lambda function

### Updating Lambda

```bash
# Download latest release
curl -L -O https://github.com/Roma7-7-7/todoist-notifier/releases/latest/download/todoist-notifier-lambda-arm.zip

# Update function
aws lambda update-function-code \
  --function-name todoist-notifier \
  --zip-file fileb://todoist-notifier-lambda-arm.zip
```

---

## Troubleshooting

### Daemon Mode

**Service won't start:**
```bash
# Check service status
sudo systemctl status todoist-notifier.service

# Check logs for errors
sudo journalctl -u todoist-notifier.service -n 50

# Verify binary permissions
ls -lh /opt/todoist-notifier/bin/todoist-notifier-daemon

# Verify environment file
sudo cat /opt/todoist-notifier/.env
```

**No notifications received:**
1. Check service is running: `sudo systemctl status todoist-notifier.service`
2. Check logs for errors: `sudo journalctl -u todoist-notifier.service -f`
3. Verify Telegram bot token and chat ID are correct
4. Test Telegram bot: Send `/tasks` command to your bot
5. Check timezone configuration matches your location
6. Verify schedule expression is correct

**Binary architecture mismatch:**
```bash
# Check your system architecture
uname -m

# x86_64 = use todoist-notifier-daemon-amd64
# aarch64 = use todoist-notifier-daemon-arm64
```

### Lambda Mode

**Function timeout:**
- Increase timeout in Lambda configuration (recommended: 30 seconds)

**SSM parameter not found:**
- Verify parameter names match exactly
- Check IAM role has SSM permissions
- Ensure KMS decrypt permission is granted

**No notifications:**
- Check CloudWatch Logs for errors
- Verify environment variables or SSM parameters
- Test Telegram bot connectivity

### Common Issues

**Rate limiting:**
- Todoist API: Max 450 requests per 15 minutes
- Telegram Bot API: 30 messages per second

**Time-based filtering:**
The notifier filters tasks based on time labels (12pm, 3pm, 6pm, 9pm). Tasks without time labels or with passed times will appear in notifications.

**Telegram bot not responding:**
1. Verify bot token is correct
2. Check chat ID matches your Telegram user
3. Ensure bot is not blocked
4. Send `/start` to the bot first

---

## Directory Structure (Daemon Mode)

```
/opt/todoist-notifier/
├── bin/
│   ├── todoist-notifier-daemon    # Binary
│   └── VERSION                     # Build metadata
├── backups/
│   ├── backup-20260111-120000/    # Automatic backups
│   ├── backup-20260111-130000/
│   └── ...                         # (keeps last 5)
├── .env                            # Configuration
├── deploy.sh                       # Deployment script
├── current_version                 # Current release tag
└── deployment.log                  # Deployment history
```

---

## Support

- **Issues**: https://github.com/Roma7-7-7/todoist-notifier/issues
- **Documentation**: See `CLAUDE.md` for development guidelines
