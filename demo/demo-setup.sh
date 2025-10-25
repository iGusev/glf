#!/bin/bash
# Setup script for VHS demo recording
# Creates all necessary fake data and git repositories

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "🎬 Setting up demo environment..."

# 1. Generate fake data
echo "📊 Generating fake project data..."
cd "$PROJECT_ROOT"
go run demo/generate-fake-data.go

# 2. Create fake git repository for 'glf .' demo
echo "🗂️  Creating fake git repository..."
mkdir -p ~/projects/backend/api/user-service
cd ~/projects/backend/api/user-service
git init > /dev/null 2>&1 || true
git remote remove origin > /dev/null 2>&1 || true
git remote add origin git@gitlab.company.com:backend/api/user-service.git

echo "✅ Demo environment ready!"
echo ""
echo "To record demo:"
echo "  cd $PROJECT_ROOT"
echo "  vhs demo/demo.tape"
echo ""
echo "To cleanup after demo:"
echo "  ./demo/demo-cleanup.sh"
