#!/bin/zsh
set -euo pipefail
cd "$(dirname "$0")/.."
go vet ./...
echo "ok: go vet ./..."
