# Handoff: Pi Terminal World Monitor – progress and next steps

Use this document to continue development with a new agent or session. It summarizes what’s done, how to run everything, and what to do next.

---

## Current state (as of handoff)

### Done

- **Repo and layout**
  - Public GitHub repo; backend (Python/FastAPI) and two clients (Go recommended, Python alternative).
  - OpenSpec change in `openspec/changes/add-pi-terminal-world-monitor-client/`; `tasks.md` updated with completed items.

- **Backend (VPS)**
  - FastAPI app: `GET /health`, `GET /panels`, `GET /panels/world-clock`, `GET /panels/weather`, `GET /panels/global-situation-map`, **`GET /panels/crypto/top`**, **`GET /panels/crypto/stablecoins`**, **`GET /panels/crypto/news`**, **`GET /panels/crypto/btc-etf`**.
  - Panel list puts **crypto first** (top-left slot). World Clock, Weather (Open-Meteo by continent), Global Situation Map. **Crypto**: CoinGecko free API for top 1–12 / 13–24 (price, 24h%), stablecoins (status, mcap, volume, ON PEG); BTC ETF endpoint is a **stub** (see below).
  - Deployed on DigitalOcean droplet **209.38.141.129** (Ubuntu 24.10). Run: `cd /opt/pi-terminal-world-monitor-client/backend && .venv/bin/uvicorn app.main:app --host 0.0.0.0 --port 8000`. After code changes on GitHub: `git pull` on droplet then restart uvicorn.

- **Go client (recommended)**
  - `client-go/`: Go 1.21 + [tview](https://github.com/rivo/tview). Build: `go mod tidy && go build -o pi-world-monitor-client .`
  - Env: `BACKEND_URL`, `CYCLE_SECONDS` (default 8), `GRID_COLS` / `GRID_ROWS` (default 2×2; use e.g. 3×3 for more, smaller panels). Press **Q** to quit.
  - **Crypto** (top-left): **4 sub-panels in 2×2**, each with its own border and **individual timer**: (1) **Top Cryptos** (top-left) – cycles **1–12 / 13–24 every 16s**; (2) **Stablecoins** (top-right) – status, mcap, volume, peg health, refresh 6s; (3) **Crypto News** (bottom-left) – headlines from backend RSS, refresh 6s; (4) **BTC ETF Tracker** (bottom-right) – stub for now, refresh 6s. Price changes green/red. Weather Watch: continent cycling, heat map, icons. Global Situation Map: severity color-coded. All use **QueueUpdateDraw** so panels refresh without keypress.

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

### Recently done (this session)

- **Crypto News** – Backend `GET /panels/crypto/news` (CoinDesk RSS, 5 min cache). Go client **Crypto News** sub-panel.
- **Crypto panel layout** – Re-laid to **4 sub-panels in 2×2**: Top Cryptos (top-left), Stable Coins (top-right), Crypto News (bottom-left), BTC ETF Tracker (bottom-right). Each has its own refresh timer; **Top Cryptos** cycles 1–12 / 13–24 every **16 seconds** (per user sketch).

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

## Deploy workflow: push and restart backend

Whenever backend or client code changes and the user needs to deploy:

**1. Local (your machine): commit and push with a relevant commit message**

Use a **specific** message that describes what changed (e.g. "Add 1h/7d price change to crypto top panel", "Crypto panel: 11 per page, 3 panels, live timer"), not generic text like "Updates" or "Fix".

```bash
cd /path/to/pi-terminal-world-monitor-client
git add -A
git status   # optional: review what will be committed
git commit -m "Your specific commit message here"
git push origin main
```

Use your actual branch name if not `main` (e.g. `master`).

**2. VPS: pull and restart the backend**

SSH in, then from the repo root:

```bash
cd /opt/pi-terminal-world-monitor-client && git pull && cd backend && .venv/bin/uvicorn app.main:app --host 0.0.0.0 --port 8000
```

If uvicorn is already running in the foreground, stop it with **Ctrl+C**, then run the line above again.

If you get **"address already in use" (Errno 98)** on port 8000, another process is using it. Stop it first, then start uvicorn:

```bash
# Find what is using port 8000 (try both; use root if needed)
sudo lsof -i :8000
# Or: sudo ss -tlnp | grep 8000
# Or: sudo fuser -v 8000/tcp

# Kill that process (use the PID; -9 forces if normal kill doesn't work)
sudo kill -9 <PID>

# Wait a second for the port to release, then start the backend
sleep 2
cd /opt/pi-terminal-world-monitor-client && git pull && cd backend && .venv/bin/uvicorn app.main:app --host 0.0.0.0 --port 8000
```

If **systemd** is managing the backend, it may be restarting the process. Then use:

```bash
sudo systemctl stop pi-world-monitor   # or your service name
# or: sudo systemctl restart pi-world-monitor
```

One-liner (force kill, wait, then start):

```bash
sudo kill -9 $(sudo lsof -t -i :8000) 2>/dev/null; sleep 2; cd /opt/pi-terminal-world-monitor-client && git pull && cd backend && .venv/bin/uvicorn app.main:app --host 0.0.0.0 --port 8000
```

If the backend runs under systemd, restart the service instead:

```bash
cd /opt/pi-terminal-world-monitor-client && git pull && systemctl restart pi-world-monitor
```

(Replace `pi-world-monitor` with the actual service name from `contrib/systemd/` if different.)

**Rule for agents:** When you make changes that require the user to push and restart the backend, always provide these two command blocks and **suggest a concrete commit message** based on the change.

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

**Crypto panel (done):** The crypto slot now shows **3 sub-panels at once**, each with borders: Top cryptos (cycles 1–12 / 13–24 every 6s), Stablecoins, BTC ETF Tracker. Only the top-cryptos content and title cycle; all three refresh every 6s.

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

The **Crypto** panel occupies the **top-left** slot and shows **4 bordered sub-panels in a 2×2 grid**:

1. **Top Cryptos** (top-left) – Border title toggles "Top cryptos (1-12)" / "Top cryptos (13-24)". **Own timer: 16s per range**; cycles between `GET /panels/crypto/top?range_start=1` and `range_start=13` (CoinGecko). Content: rank, symbol, price, 24h% (green/red), 7d% when available.
2. **Stablecoins** (top-right) – `GET /panels/crypto/stablecoins`. Status (Healthy/Caution), market cap, volume, per-coin PEG HEALTH. Refreshes every 6s (own timer).
3. **Crypto News** (bottom-left) – `GET /panels/crypto/news`. Headlines from CoinDesk RSS; backend caches 5 min. Refreshes every 6s (own timer).
4. **BTC ETF Tracker** (bottom-right) – `GET /panels/crypto/btc-etf`. **Currently a stub.** Backend returns `status`, `message`, empty `etfs`, null `total_flows_24h`, `total_aum`.

Implementation: `client-go/main.go` builds a 2×2 `tview.Flex` (two rows of two columns); each sub-panel has its **own goroutine/timer** (Top Cryptos 16s, others 6s).

### What to implement for BTC ETF Tracker (World Monitor parity)

- **Backend** (`backend/app/panels.py`): Replace the stub in `crypto_btc_etf()` with a real data pipeline. World Monitor ([github.com/koala73/worldmonitor](https://github.com/koala73/worldmonitor)) pulls BTC ETF stats – flows (in/out), AUM, per-ETF breakdown. Use a **free** source and cache with TTL. Expose: list of ETFs with name, flows 24h, AUM; aggregates (total flows 24h, total AUM).
- **Client** (`client-go/main.go`): In `renderCryptoBtcEtf()`, parse the real response and render total flows (green/red by sign), total AUM, and per-ETF table. The BTC ETF sub-panel is the third of the 3 stacked sub-panels; it already refreshes every 6s with the others.
- **Terminal graphs for crypto**: For top-cryptos sub-panels, consider ASCII/Unicode sparklines (e.g. `▁▂▃▄▅▆▇█`). CoinGecko `market_chart` can provide history; backend could return a short series; client renders one-line sparkline per coin.

### Backend response shapes (reference)

- **`/panels/crypto/top`**: `{ "status", "range", "coins": [ { "rank", "symbol", "name", "price", "price_24h_pct", "price_7d_pct" } ] }`
- **`/panels/crypto/stablecoins`**: `{ "status", "status_label", "market_cap_b", "volume_b", "coins": [ { "symbol", "name", "price", "peg_status", "deviation_pct" } ] }`
- **`/panels/crypto/news`**: `{ "status", "source": "rss", "items": [ { "title", "link", "pub_date" } ] }`
- **`/panels/crypto/btc-etf`** (target): `{ "status", "etfs": [ { "name", "flows_24h", "aum" } ], "total_flows_24h", "total_aum" }`

---

## User layout sketches (target design for next agent)

The user provided **hand-drawn sketches** of the desired dashboard layout. Use these as the source of truth for the final UI.

**Screenshot locations (in this repo):**
- `assets/image-c7817cce-3fb5-45da-87c1-81a319f5cbc3.png` – Overall 2×2 grid layout and Crypto sub-panel arrangement
- `assets/image-3008fbb0-d000-4718-9a1b-5cc6cb292525.png` – Timing/rotation notes and “individual timer” requirement

**Main grid (from sketch):**
- **Top row:** Crypto (left) | Weather Watch (right)
- **Bottom row:** Global Situation Map (left) | World Clock (right)
- All four panels have clear borders.

**Crypto panel – target layout (from sketch):**
Inside the Crypto panel there are **4 bordered sub-panels** (not 3):
1. **Top Cryptos** – top-left; list of items; **rotates individually**
2. **Stable Coins** – top-middle; list content
3. **Crypto News** – top-right; list content *(new – not yet implemented; needs backend + client)*
4. **BTC ETF Tracker** – below Stable Coins (or similar position); list content

So the Crypto area should be a 2×2 or similar grid of four sub-panels, each with borders, not a single column of three.

**Timing (from sketch):**
- **Top Cryptos:** Show “Top 1–12” for **16 seconds**, then “Top 13–24” for **16 seconds**, then repeat. This sub-panel **rotates individually** (its own timer).
- **Rule from sketch:** “Each panel subpanel in rotation needs an individual timer, and data should refresh constantly.”
- So: each sub-panel that has rotating or updating content should have its **own timer**; data across panels should **refresh constantly** (e.g. periodic refetch), not only when the user interacts.

**Next agent tasks implied by sketches (done this session):**
1. ~~Re-layout the Crypto panel into **4 bordered sub-panels** (Top Cryptos, Stable Coins, Crypto News, BTC ETF Tracker) in 2×2.~~
2. ~~Change Top Cryptos timing to **16s per range** (1–12 then 13–24), with its **own dedicated timer**.~~
3. ~~Give each rotating/updating sub-panel its **own timer**; ensure data **refreshes constantly**.~~
4. ~~Add **Crypto News** sub-panel: backend `GET /panels/crypto/news` (RSS) + client renderer; include in the Crypto 4-subpanel layout.~~
5. Keep Weather Watch, Global Situation Map, and World Clock as in the 2×2 main grid; no change to their content from the sketches.

---

## Reference

- **OpenSpec change:** `openspec/changes/add-pi-terminal-world-monitor-client/` (proposal.md, design.md, tasks.md, specs/).
- **World Monitor (inspiration):** https://github.com/koala73/worldmonitor
- **tview:** https://github.com/rivo/tview
- **Backend droplet:** 209.38.141.129 (Ubuntu 24.10); ensure `git pull` and restart uvicorn after pushing backend changes.
