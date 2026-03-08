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

Set the backend URL (default: http://localhost:8000):

```bash
export BACKEND_URL=http://your-vps:8000
python -m client
```

Or with default localhost (for development):

```bash
python -m client
```

## Panel cycling

Panels cycle on a configurable timer. All data comes from the backend API.
