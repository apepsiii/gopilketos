#!/bin/bash

echo "========================================"
echo "   Pilketos Build for ARM64 (Armbian) 🐧"
echo "========================================"
echo ""

VERSION=$(grep 'AppVersion.*=' main.go | grep -oP 'v[0-9]+\.[0-9]+\.[0-9]+' | tr '.' '_')
DATE=$(date +"%d%m%y")

echo "[*] Version: $VERSION"
echo ""

echo "[1/2] Building pilketos binary (ARM64)..."
OUT_APP="pilketos_${VERSION}_${DATE}_arm64"
# CGO_ENABLED=1 needed for mattn/go-sqlite3
CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o "$OUT_APP" .

if [ $? -eq 0 ]; then
    echo "    ✅ $OUT_APP"
else
    echo "    ❌ Failed! Make sure GCC is installed (apt install gcc)"
    exit 1
fi

echo ""
echo "[2/2] Building pilketos-setup wizard (ARM64)..."
OUT_SETUP="pilketos-setup_${VERSION}_${DATE}_arm64"
cd installer && CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o "../$OUT_SETUP" . && cd ..

if [ $? -eq 0 ]; then
    echo "    ✅ $OUT_SETUP"
else
    echo "    ❌ Failed!"
    exit 1
fi

echo ""
echo "========================================"
echo "         BUILD SUCCESS! ✅"
echo "========================================"
echo ""
echo "Output files:"
echo "  • $OUT_APP   (application)"
echo "  • $OUT_SETUP (installer wizard)"
echo ""
echo "Deploy to ARM64 VPS:"
echo "  scp $OUT_APP $OUT_SETUP root@IP_VPS:/root/"
echo ""
echo "On VPS:"
echo "  chmod +x $OUT_SETUP"
echo "  ./$OUT_SETUP"
echo ""