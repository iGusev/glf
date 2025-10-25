#!/bin/bash
# Wrapper script to run GLF with demo data for VHS recording

# Get absolute path to project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Setup fake HOME with demo config and cache
DEMO_HOME="$SCRIPT_DIR/home"
mkdir -p "$DEMO_HOME/.config/glf"
mkdir -p "$DEMO_HOME/.cache"

# Create demo config
cat > "$DEMO_HOME/.config/glf/config.yaml" << EOF
gitlab:
  url: https://gitlab.company.com
  token: demo-token-for-vhs-recording
  timeout: 30

cache:
  dir: ~/.cache/glf
EOF

# Link demo cache data
rm -rf "$DEMO_HOME/.cache/glf"
ln -sf "$SCRIPT_DIR/data/glf" "$DEMO_HOME/.cache/glf"

# Run GLF with fake HOME
HOME="$DEMO_HOME" "$PROJECT_ROOT/glf" "$@"
