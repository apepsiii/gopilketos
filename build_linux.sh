#!/bin/bash

echo "========================================"
echo "   Pilketos Build for Linux (VPS) 🐧"
echo "========================================"
echo ""

echo "[*] Compiling binary for Linux..."

# Extract version from main.go (e.g., "v1.0.0" -> "v1_0_0")
VERSION=$(grep 'AppVersion.*=' main.go | grep -oP 'v[0-9]+\.[0-9]+\.[0-9]+' | tr '.' '_')
DATE=$(date +"%d%m%y")
OUT_FILE="pilketos_${VERSION}_${DATE}"

echo "[*] Version: $VERSION"
echo "[*] Output: $OUT_FILE"

# SQLite requires CGO for modernc.org/sqlite (pure Go, no CGO needed)
# If using mattn/go-sqlite3, need CGO_ENABLED=1
# This project uses modernc.org/sqlite which is pure Go

GOOS=linux GOARCH=amd64 go build -o "$OUT_FILE" .

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ BUILD SUCCESS!"
    echo "File '$OUT_FILE' siap diupload ke VPS."
    echo ""
    echo "Cara deploy:"
    echo "1. Upload file ke VPS: scp $OUT_FILE root@IP_VPS:/opt/pilketos/"
    echo "2. Di VPS: chmod +x $OUT_FILE && ./$OUT_FILE"
else
    echo ""
    echo "❌ BUILD FAILED!"
fi