# Install on Raspberry Pi 3B (DietPi)

Run the terminal dashboard on a Pi 3B with DietPi. You can run it manually or **start it automatically at boot**.

## Prerequisites

- Raspberry Pi 3B with [DietPi](https://dietpi.com/) installed (fresh install is fine).
- Backend running somewhere reachable from the Pi (e.g. your VPS). The Pi only talks to this backend.

---

## Option A: Go client (recommended)

Single binary, [tview](https://github.com/rivo/tview)-based UI with proper panel borders and layout.

### 1. Install Go on the Pi (if not present)

```bash
# On DietPi: install from package or download
sudo apt install -y golang
# or use the official binary: https://go.dev/dl/
```

### 2. Clone and build

```bash
cd ~
git clone https://github.com/YOUR_USERNAME/pi-terminal-world-monitor-client.git
cd pi-terminal-world-monitor-client/client-go
go mod tidy
go build -o pi-world-monitor-client .
```

### 3. Run manually

```bash
export BACKEND_URL=http://YOUR_VPS:8000
./pi-world-monitor-client
```

Press **Q** to quit.

### 4. Run at startup (systemd)

Use the same systemd steps below, but set `ExecStart` to the Go binary, e.g.:

```bash
ExecStart=%h/pi-terminal-world-monitor-client/client-go/pi-world-monitor-client
WorkingDirectory=%h/pi-terminal-world-monitor-client/client-go
```

---

## Option B: Python client

### 1. Install from GitHub

```bash
cd ~
git clone https://github.com/YOUR_USERNAME/pi-terminal-world-monitor-client.git
cd pi-terminal-world-monitor-client/client
python3 -m venv .venv
.venv/bin/pip install -r requirements.txt
```

### 2. Configure backend URL

```bash
export BACKEND_URL=http://YOUR_VPS_IP_OR_HOST:8000
```

### 3. Run manually

```bash
cd ~/pi-terminal-world-monitor-client/client
source .venv/bin/activate
export BACKEND_URL=http://YOUR_VPS:8000
python -m client
```

Press **Ctrl+C** to exit.

---

## Run at startup (DietPi) – either client

Two options: **systemd user service** (recommended) or **DietPi autostart**.

### Systemd user service (after login)

Runs the client when you log in (or when the default user auto-logs in to console). Good for “dashboard on the console I see when I sit at the Pi.”

1. Copy the service file and env template:

```bash
mkdir -p ~/.config/systemd/user
cp ~/pi-terminal-world-monitor-client/contrib/systemd/pi-world-monitor.service ~/.config/systemd/user/

# Optional: set backend URL in an env file
mkdir -p ~/.config/pi-world-monitor
echo 'BACKEND_URL=http://YOUR_VPS:8000' > ~/.config/pi-world-monitor/env
```

2. Edit the service file so paths and user match your setup:

```bash
nano ~/.config/systemd/user/pi-world-monitor.service
```

Set `BACKEND_URL` in the `Environment=` line, or use `EnvironmentFile=` pointing to `~/.config/pi-world-monitor/env`. For the **Go client**, set `ExecStart=%h/pi-terminal-world-monitor-client/client-go/pi-world-monitor-client` and `WorkingDirectory=%h/pi-terminal-world-monitor-client/client-go`. For the **Python client**, use `client/` and `python -m client` as in the contrib service file.

3. Enable and start (user session):

```bash
systemctl --user daemon-reload
systemctl --user enable pi-world-monitor
systemctl --user start pi-world-monitor
```

To have it start at boot with **console auto-login**, enable automatic login for the default user (DietPi: **dietpi-config** → Auto-start options / login). The user session will start and the service will run.

To see output:

```bash
journalctl --user -u pi-world-monitor -f
```

### DietPi autostart script

If you prefer to start the dashboard from DietPi’s autostart (e.g. a custom command at boot):

1. Make the script executable:

```bash
chmod +x ~/pi-terminal-world-monitor-client/contrib/dietpi-autostart.sh
```

2. Edit the script and set `BACKEND_URL` at the top.

3. Add to DietPi autostart:

- **dietpi-config** → **Autostart** → add a custom command, e.g.:
  - `/home/dietpi/pi-terminal-world-monitor-client/contrib/dietpi-autostart.sh`

Or run it from your shell profile (e.g. `.profile`) so it starts when you log in:

```bash
# At the end of ~/.profile (optional)
# /home/dietpi/pi-terminal-world-monitor-client/contrib/dietpi-autostart.sh
```

## 5. Updating

```bash
cd ~/pi-terminal-world-monitor-client
git pull
cd client
.venv/bin/pip install -r requirements.txt
# Restart the service or run manually again
```

## Troubleshooting

- **Blank or no TUI:** Ensure you’re on a real terminal (SSH or physical console). The client needs a TTY.
- **Connection errors:** Check `BACKEND_URL`, firewall, and that the backend is listening on the right interface/port.
- **High CPU:** Panel cycling and redraws are tuned for Pi 3B; if you add heavy rendering later, reduce cycle frequency.
