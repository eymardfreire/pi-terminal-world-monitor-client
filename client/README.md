# Pi Terminal World Monitor – Terminal Client

Runs on the Pi 3B (DietPi/Linux). Fetches panel data from the backend only; displays a color-coded, cycling TUI. No direct third-party API calls.

## Setup

```bash
cd client
python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

## Run

Set the backend URL (default: http://localhost:8000), then run the module from the `client` directory:

```bash
export BACKEND_URL=http://your-vps:8000
python -m client
```

**From the repo root** (same effect):

```bash
cd client
export BACKEND_URL=http://your-vps:8000
.venv/bin/python -m client
```

Using `.venv/bin/python -m client` avoids needing to `source activate`—handy in **fish** and other shells where `source .venv/bin/activate` may fail.

**Configuration (env):**

- `BACKEND_URL` – API base URL (default: `http://localhost:8000`).
- `CYCLE_SECONDS` – Seconds per panel before switching (default: `8`, clamped 1–120).

## Panel cycling

Panels cycle on a configurable timer. All data comes from the backend API.
