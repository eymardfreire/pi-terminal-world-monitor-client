# Pi Terminal World Monitor

Lightweight terminal-only dashboard for Raspberry Pi 3B (DietPi/Linux), inspired by [World Monitor](https://github.com/koala73/worldmonitor). Data is aggregated on a backend (VPS); the Pi runs a thin client that displays color-coded, cycling panels—no direct third-party API calls from the device.

## Layout

- **`backend/`** – FastAPI service (runs on VPS). Free APIs + RSS; cache-first; exposes panel endpoints.
- **`client/`** – Terminal client (runs on Pi). Blessed TUI; panel cycling; fetches only from backend.
- **`openspec/`** – OpenSpec change proposal and specs (source of truth for implementation).

## Install on Raspberry Pi 3B (DietPi)

To run the client on a Pi 3B and **start it at boot**, see **[docs/INSTALL-PI.md](docs/INSTALL-PI.md)**. You’ll clone the repo, install dependencies, set your backend URL, and optionally use the provided systemd user service or DietPi autostart script.

## Quick start

**Backend (VPS or local):**

```bash
cd backend && python -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
uvicorn app.main:app --host 0.0.0.0 --port 8000
```

**Client (Pi or local):**

```bash
cd client && python -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
export BACKEND_URL=http://your-backend:8000   # or leave unset for localhost
python -m client
```

## Spec and tasks

Implementation follows the active OpenSpec change:

- `openspec/changes/add-pi-terminal-world-monitor-client/`
  - `proposal.md` – Why and what
  - `design.md` – Context, goals, decisions
  - `tasks.md` – Implementation checklist (mark items `[x]` as done)
  - `specs/` – Backend, client, and Global Situation Map requirements

## Panel set (target)

Strategic Risk, Intel, Live Intelligence, Work News, continental panels, Predictions, Energy, Think Tanks, Commodities, Markets, Economy, Trade, Supply Chain, Financial News, Tech, Crypto, Fires, Market Radar, BTC ETF, Stablecoins, Armed Conflict, Global Giving, Climate, Weather Watch, World Clock, plus a **text-based Global Situation Map**. All color-coded; panel cycling over time.

