# Handoff: Pi Terminal World Monitor – progress and next steps

Use this document to continue development with a new agent or session. It summarizes what’s done, how to run everything, and what to do next.

---

## Current state (as of handoff)

### Done

- **Repo and layout**
  - Public GitHub repo; backend (Python/FastAPI) and two clients (Go recommended, Python alternative).
  - OpenSpec change in `openspec/changes/add-pi-terminal-world-monitor-client/`; `tasks.md` updated with completed items.

- **Backend (VPS)**
  - FastAPI app: `GET /health`, `GET /panels`, `GET /panels/world-clock`, `GET /panels/weather`, `GET /panels/global-situation-map`, **`GET /panels/crypto/top`**, **`GET /panels/crypto/stablecoins`**, **`GET /panels/crypto/btc-etf`**.
  - Panel list puts **crypto first** (top-left slot). World Clock, Weather (Open-Meteo by continent), Global Situation Map. **Crypto**: CoinGecko free API for top 1–12 / 13–24 (price, 24h%), stablecoins (status, mcap, volume, ON PEG); BTC ETF endpoint is a **stub** (see below).
  - Deployed on DigitalOcean droplet **209.38.141.129** (Ubuntu 24.10). Run: `cd /opt/pi-terminal-world-monitor-client/backend && .venv/bin/uvicorn app.main:app --host 0.0.0.0 --port 8000`. After code changes on GitHub: `git pull` on droplet then restart uvicorn.

- **Go client (recommended)**
  - `client-go/`: Go 1.21 + [tview](https://github.com/rivo/tview). Build: `go mod tidy && go build -o pi-world-monitor-client .`
  - Env: `BACKEND_URL`, `CYCLE_SECONDS` (default 8), `GRID_COLS` / `GRID_ROWS` (default 2×2; use e.g. 3×3 for more, smaller panels). Press **Q** to quit.
  - **Crypto** (top-left): 3 sub-panels cycling every 6s – (1) Top 12 cryptos by mcap, (2) Top 13–24, (3) Stablecoins (status, mcap, volume, peg health), (4) BTC ETF Tracker (stub message). Price change green/red. Weather Watch: continent cycling, heat map, icons. Global Situation Map: severity color-coded. All use **QueueUpdateDraw** so panels refresh without keypress.

- **Python client**
  - `client/`: Python + blessed. Multi-panel grid; use if Go isn’t available. Run: `python -m client` from `client/` with venv and `BACKEND_URL` set.

- **Deployment and docs**
  - **docs/DEPLOY-BACKEND.md** – Deploy backend on droplet (clone, venv or virtualenv fallback, ufw, optional systemd).
  - **docs/INSTALL-PI.md** – Install on Pi 3B (DietPi): Go client (recommended) or Python client; systemd and DietPi autostart.
  - **contrib/systemd/** – `pi-world-monitor.service` (Python), `pi-world-monitor-go.service` (Go).

### Not done (see Next steps)

- **BTC ETF Tracker** – backend and client stub in place; needs real data source (see Crypto + BTC ETF section below).
- Terminal **sparklines/graphs** for crypto (e.g. ASCII/Unicode mini-charts) – not yet implemented.
- Remaining panel categories; Global Situation Map stub; formal OpenAPI; Pi 3B validation.

---

## How to run (quick reference)

**Backend (droplet or local):**
```bash
cd backend && .venv/bin/uvicorn app.main:app --host 0.0.0.0 --port 8000
```

**Go client (local or Pi):**
```bash
cd client-go
go mod tidy && go build -o pi-world-monitor-client .
export BACKEND_URL=http://209.38.141.129:8000   # or your backend URL
./pi-world-monitor-client
```

**Smoke test backend:**
```bash
curl -s http://209.38.141.129:8000/health
curl -s http://209.38.141.129:8000/panels
curl -s http://209.38.141.129:8000/panels/world-clock
```

---

## Repo layout

| Path | Purpose |
|------|--------|
| `backend/` | FastAPI app; `app/main.py`, `app/panels.py`; runs on VPS |
| `client-go/` | Go + tview client; **recommended** for Pi and desktop |
| `client/` | Python + blessed client; alternative |
| `docs/` | DEPLOY-BACKEND.md, INSTALL-PI.md, GITHUB-SETUP.md, **HANDOFF-PROGRESS.md** (this file) |
| `contrib/` | systemd units, dietpi-autostart.sh |
| `openspec/changes/add-pi-terminal-world-monitor-client/` | Proposal, design, **tasks.md**, specs |

---

## Next steps (for the next agent)

1. ~~**Backend: real Weather data (task 3.4)**~~ **Done.** Open-Meteo wired in `backend/app/panels.py`; 10 min cache; London, New York, Tokyo, Berlin.

2. ~~**Backend: Global Situation Map (tasks 2.3, 3.5)**~~ **Done.** Schema: `regions[]` with `name`, `severity`, `events[]`. Endpoint `GET /panels/global-situation-map`; stub data (pipeline ready for real feeds). Go client renders with severity colors.

3. **Backend: cache-first and more panels (tasks 3.1, 3.2, 2.2)**  
   Add cache (e.g. in-memory with TTL) for external calls; respect rate limits and backoff. Add at least one more panel (e.g. strategic risk or a news feed) with a free API or RSS; document response schemas in code or OpenAPI.

4. ~~**Go client: new panels and Global Situation Map (tasks 4.3, 5.1, 5.2)**~~ **Done.** GSM panel added; severity colors (red/yellow/cyan/green); panel list comes from backend.

5. **Pi 3B validation (task 7.4)**  
   Build Go client for ARM (e.g. `GOOS=linux GOARCH=arm GOARM=7`), run on DietPi, confirm acceptable CPU/memory and readability. Update INSTALL-PI.md if needed.

6. **OpenSpec**  
   As you complete items, mark them `[x]` in `openspec/changes/add-pi-terminal-world-monitor-client/tasks.md`. When the change is feature-complete, use `/opsx:archive` (or the project’s OpenSpec workflow) to archive the change and update main specs.

---

## Crypto panel and BTC ETF (for next agent)

The **Crypto** panel occupies the **top-left** slot and cycles through four sub-views every 6 seconds:

1. **Top cryptos 1–12** – `GET /panels/crypto/top?range_start=1` (CoinGecko). Shows rank, symbol, price, 24h% (green/red), 7d% when available.
2. **Top cryptos 13–24** – `GET /panels/crypto/top?range_start=13`.
3. **Stablecoins** – `GET /panels/crypto/stablecoins`. Status (Healthy/Caution), market cap, volume, per-coin PEG HEALTH (ON PEG / OFF PEG + deviation %).
4. **BTC ETF Tracker** – `GET /panels/crypto/btc-etf`. **Currently a stub.** Backend returns `status`, `message`, empty `etfs`, null `total_flows_24h`, `total_aum`.

### What to implement for BTC ETF Tracker (World Monitor parity)

- **Backend** (`backend/app/panels.py`): Replace the stub in `crypto_btc_etf()` with a real data pipeline. World Monitor ([github.com/koala73/worldmonitor](https://github.com/koala73/worldmonitor)) pulls BTC ETF stats – flows (in/out), AUM, per-ETF breakdown. Use a **free** source and cache with TTL. Expose: list of ETFs with name, flows 24h, AUM; aggregates (total flows 24h, total AUM).
- **Client** (`client-go/main.go`): In `renderCryptoBtcEtf()`, parse the real response and render total flows (green/red by sign), total AUM, and per-ETF table. Keep 6s cycle.
- **Terminal graphs for crypto**: For top-cryptos sub-panels, consider ASCII/Unicode sparklines (e.g. `▁▂▃▄▅▆▇█`). CoinGecko `market_chart` can provide history; backend could return a short series; client renders one-line sparkline per coin.

### Backend response shapes (reference)

- **`/panels/crypto/top`**: `{ "status", "range", "coins": [ { "rank", "symbol", "name", "price", "price_24h_pct", "price_7d_pct" } ] }`
- **`/panels/crypto/stablecoins`**: `{ "status", "status_label", "market_cap_b", "volume_b", "coins": [ { "symbol", "name", "price", "peg_status", "deviation_pct" } ] }`
- **`/panels/crypto/btc-etf`** (target): `{ "status", "etfs": [ { "name", "flows_24h", "aum" } ], "total_flows_24h", "total_aum" }`

---

## Reference

- **OpenSpec change:** `openspec/changes/add-pi-terminal-world-monitor-client/` (proposal.md, design.md, tasks.md, specs/).
- **World Monitor (inspiration):** https://github.com/koala73/worldmonitor
- **tview:** https://github.com/rivo/tview
- **Backend droplet:** 209.38.141.129 (Ubuntu 24.10); ensure `git pull` and restart uvicorn after pushing backend changes.
