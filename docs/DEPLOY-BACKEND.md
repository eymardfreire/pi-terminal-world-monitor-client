# Deploy backend on the VPS (DigitalOcean droplet)

Use this to run the Pi Terminal World Monitor backend on your droplet so the Pi client can reach it.

**Your backend droplet:** `209.38.141.129` (Ubuntu 24.10, SFO3). The Pi will use `BACKEND_URL=http://209.38.141.129:8000` (or with a domain if you add one later).

## 1. SSH into the droplet

```bash
ssh root@209.38.141.129
# or: ssh your_user@209.38.141.129
```

## 2. Clone and run the backend

On Ubuntu/Debian, install the venv package first (required for `python3 -m venv`):

```bash
sudo apt update
sudo apt install -y python3-venv
```

Then clone and run:

```bash
cd /opt   # or ~
git clone https://github.com/eymardfreire/pi-terminal-world-monitor-client.git
cd pi-terminal-world-monitor-client/backend

python3 -m venv .venv
.venv/bin/pip install -r requirements.txt

# Run (listens on all interfaces so the Pi can connect)
.venv/bin/uvicorn app.main:app --host 0.0.0.0 --port 8000
```

If you already ran `python3 -m venv .venv` and it failed (e.g. "ensurepip is not available"), remove the broken venv and install the package:

```bash
rm -rf .venv
sudo apt install -y python3-venv
python3 -m venv .venv
.venv/bin/pip install -r requirements.txt
.venv/bin/uvicorn app.main:app --host 0.0.0.0 --port 8000
```

**If apt gives 404 errors** or **"externally-managed-environment"** blocks pip:

1. Try venv with full Python (sometimes works when python3-venv 404s):
   ```bash
   sudo apt install -y python3-full
   python3 -m venv .venv
   .venv/bin/pip install -r requirements.txt
   .venv/bin/uvicorn app.main:app --host 0.0.0.0 --port 8000
   ```

2. If that still fails, bootstrap pip with the PEP 668 override and install into a **virtualenv created by pip** (no system venv):
   ```bash
   curl -sS https://bootstrap.pypa.io/get-pip.py -o /tmp/get-pip.py
   PIP_BREAK_SYSTEM_PACKAGES=1 python3 /tmp/get-pip.py --user
   export PATH="$HOME/.local/bin:$PATH"
   python3 -m pip install --user --break-system-packages virtualenv
   python3 -m virtualenv .venv
   .venv/bin/pip install -r requirements.txt
   .venv/bin/uvicorn app.main:app --host 0.0.0.0 --port 8000
   ```
   (This uses the standalone `virtualenv` package instead of the system `venv` module.)

Test from your machine:

```bash
curl http://209.38.141.129:8000/health
# → {"status":"ok","service":"pi-terminal-world-monitor-backend"}

curl http://209.38.141.129:8000/panels
# → {"panels":["world-clock","weather"],"status":"ok"}

curl http://209.38.141.129:8000/panels/world-clock
# → UTC time and zones
```

Then run the **client on your laptop** to test panels against the droplet:

```bash
cd /path/to/pi-terminal-world-monitor-client/client
source .venv/bin/activate
export BACKEND_URL=http://209.38.141.129:8000
python -m client
```

You should see World Clock and Weather Watch panels cycling every 8 seconds.

## 3. Open port 8000 (firewall)

On the droplet, if `ufw` is enabled:

```bash
sudo ufw allow 8000/tcp
sudo ufw status
sudo ufw reload
```

In the DigitalOcean dashboard: **Networking** → **Firewall** (if you use a DO firewall), add an inbound rule for **TCP 8000**.

## 4. Run in the background (optional)

**Option A – systemd (survives reboot)**

```bash
sudo tee /etc/systemd/system/pi-world-monitor-backend.service << 'EOF'
[Unit]
Description=Pi Terminal World Monitor API
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/pi-terminal-world-monitor-client/backend
ExecStart=/opt/pi-terminal-world-monitor-client/backend/.venv/bin/uvicorn app.main:app --host 0.0.0.0 --port 8000
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable pi-world-monitor-backend
sudo systemctl start pi-world-monitor-backend
sudo systemctl status pi-world-monitor-backend
```

Adjust `User`, `WorkingDirectory`, and `ExecStart` if you cloned elsewhere or use a different user.

**Option B – tmux/screen**

```bash
tmux new -s backend
cd /opt/pi-terminal-world-monitor-client/backend
.venv/bin/uvicorn app.main:app --host 0.0.0.0 --port 8000
# Detach: Ctrl+B, then D
```

## 5. Pi client configuration

On the Pi (or in the systemd/env config), set:

```bash
export BACKEND_URL=http://209.38.141.129:8000
```

Then run the client; it will use this backend.

## 6. Updating the backend

```bash
cd /opt/pi-terminal-world-monitor-client
git pull
cd backend
.venv/bin/pip install -r requirements.txt
sudo systemctl restart pi-world-monitor-backend   # if using systemd
```
