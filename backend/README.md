# Pi Terminal World Monitor – Backend

Runs on the operator's VPS. Aggregates data from free APIs and RSS; exposes a REST API for the terminal client. No direct third-party calls from the Pi.

## Setup

```bash
cd backend
python -m venv .venv
source .venv/bin/activate   # or .venv\Scripts\activate on Windows
pip install -r requirements.txt
```

## Run

```bash
uvicorn app.main:app --host 0.0.0.0 --port 8000
```

## API

- `GET /health` – Health check (no auth).

Panel endpoints will be added as per the OpenSpec change (panel-based or bootstrap).
