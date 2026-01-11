#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
GITHUB_REPO="Roma7-7-7/todoist-notifier"
INSTALL_DIR="/opt/todoist-notifier"
BIN_DIR="${INSTALL_DIR}/bin"
BACKUP_DIR="${INSTALL_DIR}/backups"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Todoist Telegram Notifier Setup${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if running as root
if [[ "$EUID" -ne 0 ]]; then
    echo -e "${RED}Please run as root or with sudo${NC}"
    exit 1
fi

# Determine which user to run the service as
if [[ -n "$SUDO_USER" ]]; then
    SERVICE_USER="$SUDO_USER"
else
    echo -e "${YELLOW}Enter the username to run the service as (default: current user):${NC}"
    read -r SERVICE_USER
    if [[ -z "$SERVICE_USER" ]]; then
        SERVICE_USER=$(whoami)
    fi
fi

echo -e "${GREEN}Service will run as user: ${SERVICE_USER}${NC}"
echo ""

# Verify user exists
if ! id "$SERVICE_USER" &>/dev/null; then
    echo -e "${RED}User $SERVICE_USER does not exist${NC}"
    exit 1
fi

# Check if already installed
if [[ -d "${INSTALL_DIR}" ]]; then
    echo -e "${YELLOW}Warning: Installation directory ${INSTALL_DIR} already exists${NC}"
    echo -e "${YELLOW}This will update the installation (binary and scripts only).${NC}"
    read -p "Do you want to continue? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Setup cancelled."
        exit 1
    fi
fi

echo -e "${GREEN}[1/7] Creating directory structure...${NC}"
mkdir -p "${BIN_DIR}"
mkdir -p "${BACKUP_DIR}"
chown -R "${SERVICE_USER}:${SERVICE_USER}" "${INSTALL_DIR}"
echo "✓ Directories created"

echo ""
echo -e "${GREEN}[2/7] Downloading deploy.sh script...${NC}"
curl -L -o "${INSTALL_DIR}/deploy.sh" \
    "https://raw.githubusercontent.com/${GITHUB_REPO}/main/deployment/deploy.sh"
chmod +x "${INSTALL_DIR}/deploy.sh"
chown "${SERVICE_USER}:${SERVICE_USER}" "${INSTALL_DIR}/deploy.sh"
echo "✓ Deploy script installed"

echo ""
echo -e "${GREEN}[3/7] Installing systemd service...${NC}"
# Download and install service
curl -L -s "https://raw.githubusercontent.com/${GITHUB_REPO}/main/deployment/systemd/todoist-notifier.service" | \
    sed "s/{{SERVICE_USER}}/${SERVICE_USER}/g" > /etc/systemd/system/todoist-notifier.service

systemctl daemon-reload
echo "✓ Systemd service installed"

echo ""
echo -e "${GREEN}[4/7] Configuring sudoers for passwordless service management...${NC}"
# Create sudoers file for the service user
cat > /etc/sudoers.d/todoist-notifier <<EOF
${SERVICE_USER} ALL=(ALL) NOPASSWD: /bin/systemctl start todoist-notifier.service
${SERVICE_USER} ALL=(ALL) NOPASSWD: /bin/systemctl stop todoist-notifier.service
${SERVICE_USER} ALL=(ALL) NOPASSWD: /bin/systemctl restart todoist-notifier.service
${SERVICE_USER} ALL=(ALL) NOPASSWD: /bin/systemctl status todoist-notifier.service
${SERVICE_USER} ALL=(ALL) NOPASSWD: /bin/systemctl is-active todoist-notifier.service
EOF
chmod 0440 /etc/sudoers.d/todoist-notifier
echo "✓ Sudoers configuration installed"

echo ""
echo -e "${GREEN}[5/7] Setting up environment file...${NC}"
ENV_FILE="${INSTALL_DIR}/.env"

if [[ -f "$ENV_FILE" ]]; then
    echo -e "${YELLOW}Environment file already exists at: ${ENV_FILE}${NC}"
    echo -e "${YELLOW}Skipping environment file creation to preserve existing configuration${NC}"
else
    echo ""
    echo -e "${BLUE}Configuration Setup:${NC}"
    echo ""

    # Bot configuration
    echo -e "${YELLOW}--- Configuration ---${NC}"
    echo -e "${BLUE}Enter your Todoist API token:${NC}"
    read -r TODOIST_TOKEN

    echo -e "${BLUE}Enter your Telegram bot token (from @BotFather):${NC}"
    read -r TELEGRAM_BOT_ID

    echo -e "${BLUE}Enter your Telegram chat ID:${NC}"
    read -r TELEGRAM_CHAT_ID

    echo ""
    echo -e "${YELLOW}--- Optional Configuration (press Enter for defaults) ---${NC}"

    echo -e "${BLUE}Enter cron schedule (default: 0 * 9-23 * * * = hourly from 9am-11pm):${NC}"
    read -r SCHEDULE
    if [[ -z "$SCHEDULE" ]]; then
        SCHEDULE="0 * 9-23 * * *"
    fi

    echo -e "${BLUE}Enter timezone location (default: Europe/Kyiv):${NC}"
    read -r LOCATION
    if [[ -z "$LOCATION" ]]; then
        LOCATION="Europe/Kyiv"
    fi

    cat > "$ENV_FILE" <<EOF
# Environment
ENV=prod

# Todoist Configuration
TODOIST_TOKEN=${TODOIST_TOKEN}

# Telegram Configuration
TELEGRAM_BOT_ID=${TELEGRAM_BOT_ID}
TELEGRAM_CHAT_ID=${TELEGRAM_CHAT_ID}

# Scheduler Configuration
SCHEDULE=${SCHEDULE}
LOCATION=${LOCATION}

# Optional: Force SSM Parameter Store usage (requires AWS configuration)
# FORCE_SSM=true
EOF

    chmod 600 "$ENV_FILE"
    chown "${SERVICE_USER}:${SERVICE_USER}" "$ENV_FILE"
    echo "✓ Environment file created at: ${ENV_FILE}"
fi

echo ""
echo -e "${GREEN}[6/7] Running initial deployment...${NC}"
sudo -u "${SERVICE_USER}" "${INSTALL_DIR}/deploy.sh"
echo "✓ Initial deployment completed"

echo ""
echo -e "${GREEN}[7/7] Enabling service auto-start...${NC}"
systemctl enable todoist-notifier.service
echo "✓ Service will start automatically on boot"

echo ""
echo -e "${GREEN}[8/8] Verifying installation...${NC}"

# Check service status
SERVICE_ACTIVE=$(systemctl is-active todoist-notifier.service || echo "inactive")

if [[ "$SERVICE_ACTIVE" = "active" ]]; then
    echo "✓ Service is running"
else
    echo -e "${YELLOW}⚠ Service may not be running: $SERVICE_ACTIVE${NC}"
    echo -e "${YELLOW}Check configuration and logs${NC}"
fi

# Check version
if [[ -f "${INSTALL_DIR}/current_version" ]]; then
    VERSION=$(cat "${INSTALL_DIR}/current_version")
    echo "✓ Installed version: ${VERSION}"
fi

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Setup completed successfully!${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

echo -e "${YELLOW}Important Information:${NC}"
echo ""
echo "Configuration file: ${ENV_FILE}"
echo "  Edit this file to configure the notifier (requires service restart)"
echo ""
echo -e "${YELLOW}Useful commands:${NC}"
echo "  Service Status:  sudo systemctl status todoist-notifier.service"
echo "  Service Logs:    sudo journalctl -u todoist-notifier.service -f"
echo "  Deploy Updates:  ${INSTALL_DIR}/deploy.sh"
echo "  Stop Service:    sudo systemctl stop todoist-notifier.service"
echo "  Start Service:   sudo systemctl start todoist-notifier.service"
echo "  Restart Service: sudo systemctl restart todoist-notifier.service"
echo ""
echo -e "${YELLOW}Configuration:${NC}"
echo "  - Edit ${ENV_FILE} to change settings"
echo "  - After editing, restart service: sudo systemctl restart todoist-notifier.service"
echo ""
echo -e "${YELLOW}Automatic Updates:${NC}"
echo "  - Run ${INSTALL_DIR}/deploy.sh manually"
echo "  - Or set up a cron job: 0 */6 * * * ${INSTALL_DIR}/deploy.sh"
echo ""
