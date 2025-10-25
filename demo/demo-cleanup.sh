#!/bin/bash
# Cleanup script for VHS demo recording
# Removes all fake data and temporary files created by demo-setup.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "🧹 Cleaning up demo environment..."

# 1. Remove fake git repository
if [ -d ~/projects/backend ]; then
    echo "🗑️  Removing fake git repository..."
    rm -rf ~/projects/backend
fi

# 2. Remove demo-home directory
if [ -d "$SCRIPT_DIR/home" ]; then
    echo "🗑️  Removing demo home directory..."
    rm -rf "$SCRIPT_DIR/home"
fi

# 3. Remove generated demo data (optional - keep for reuse)
# Uncomment if you want to clean everything
# if [ -d "$SCRIPT_DIR/data" ]; then
#     echo "🗑️  Removing fake data..."
#     rm -rf "$SCRIPT_DIR/data"
# fi

echo "✅ Cleanup complete!"
echo ""
echo "Note: demo/data/ was preserved for reuse."
echo "To remove it manually: rm -rf demo/data/"
