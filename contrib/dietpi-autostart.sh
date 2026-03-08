#!/bin/sh
# Pi Terminal World Monitor – start script for DietPi autostart or manual run
# Edit BACKEND_URL below, make executable: chmod +x contrib/dietpi-autostart.sh

# Set your backend URL (replace with your VPS host and port)
export BACKEND_URL="${BACKEND_URL:-http://localhost:8000}"

# Path to repo (adjust if you cloned elsewhere)
CLIENT_DIR="${HOME}/pi-terminal-world-monitor-client/client"
VENV_PYTHON="${CLIENT_DIR}/.venv/bin/python"

if [ ! -x "$VENV_PYTHON" ]; then
  echo "Pi World Monitor: venv not found. Run: cd $CLIENT_DIR && python3 -m venv .venv && .venv/bin/pip install -r requirements.txt"
  exit 1
fi

exec "$VENV_PYTHON" -m client
