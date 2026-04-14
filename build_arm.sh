#!/bin/bash

echo "========================================"
echo "   Pilketos Build for ARM64 (Armbian) 🐧"
echo "========================================"
echo ""

echo "[*] Compiling binary for ARM64..."

# Extract version from main.go (e.g., "v1.0.0" -> "v1_0_0")
VERSION=$(grep 'AppVersion.*=' main.go | grep -oP 'v[0-9]+\.[0-9]+\.[0-9]+' | tr '.' '_')
DATE=$(date +"%d%m%y")
OUT_FILE="pilketos_${VERSION}_${DATE}_arm64"

echo "[*] Version: $VERSION"
echo "[*] Output: $OUT_FILE"

# ARM64 for modern Armbian (Orange Pi, Rock Pi, Raspberry Pi 4, etc.)
# modernc.org/sqlite is pure Go, no CGO needed

GOOS=linux GOARCH=arm64 go build -o "$OUT_FILE" .

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ BUILD SUCCESS!"
    echo "File '$OUT_FILE' siap diupload ke VPS Armbian."
    echo ""
    echo "Cara deploy:"
    echo "1. Upload file ke VPS: scp $OUT_FILE root@IP_VPS:/opt/pilketos/"
    echo "2. Di VPS: chmod +x $OUT_FILE && ./$OUT_FILE"
else
    echo ""
    echo "❌ BUILD FAILED!"
fi