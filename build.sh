#!/bin/bash

set -e

APP_NAME="pilketos"
SETUP_NAME="pilketos-setup"
BUILD_DIR="dist"

echo "=========================================="
echo "  Building Pilketos E-Voting System"
echo "=========================================="
echo ""

rm -rf $BUILD_DIR
mkdir -p $BUILD_DIR

echo "[1/3] Building main application (Linux AMD64)..."
GOOS=linux GOARCH=amd64 go build -o $BUILD_DIR/$APP_NAME .
echo "      Done: $BUILD_DIR/$APP_NAME"

echo ""
echo "[2/3] Building installer (Linux AMD64)..."
GOOS=linux GOARCH=amd64 go build -o $BUILD_DIR/$SETUP_NAME ./installer
echo "      Done: $BUILD_DIR/$SETUP_NAME"

echo ""
echo "[3/3] Creating documentation..."
cat > $BUILD_DIR/README.md << 'EOF'
# Pilketos E-Voting System

## Quick Start

1. Upload this folder to your VPS
2. SSH into your VPS
3. Run the installer:
   ```bash
   chmod +x pilketos-setup
   sudo ./pilketos-setup
   ```
4. Follow the wizard instructions
5. Access your app at https://your-domain.com

## What the Installer Does

- Creates installation directory (/opt/pilketos)
- Copies the application binary
- Generates config.yaml with your settings
- Sets up systemd service
- Configures Nginx reverse proxy
- Starts the application

## Requirements

- Ubuntu/Debian VPS (fresh install recommended)
- Root/sudo access
- Domain pointing to server IP
- Ports 80 and 443 open

## After Installation

```bash
# Check status
sudo systemctl status pilketos

# View logs
sudo journalctl -u pilketos -f

# Restart
sudo systemctl restart pilketos
```

## Troubleshooting

### Service won't start
```bash
sudo journalctl -u pilketos -n 50
```

### Nginx 502 error
```bash
sudo systemctl status pilketos
curl http://localhost:8024
```

### SSL certificate issues
```bash
sudo certbot --nginx -d your-domain.com
```
EOF
echo "      Done: README.md"

echo ""
echo "=========================================="
echo "  Build Complete!"
echo "=========================================="
echo ""
echo "Contents of ./$BUILD_DIR/:"
ls -la $BUILD_DIR/
echo ""
echo "Upload the entire '$BUILD_DIR' folder to your VPS, then run:"
echo "  chmod +x pilketos-setup"
echo "  sudo ./pilketos-setup"
