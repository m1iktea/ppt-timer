#!/usr/bin/env bash
# 在 macOS/Linux 上交叉编译 Windows 版本，产物在 dist/
set -euo pipefail
cd "$(dirname "$0")"
go test ./...
rm -rf dist && mkdir -p dist
for arch in amd64 arm64; do
  GOOS=windows GOARCH=$arch go build -trimpath -ldflags "-H windowsgui -s -w" -o "dist/ppt-timer-$arch.exe" .
done
cp README.md dist/
ls -lh dist
