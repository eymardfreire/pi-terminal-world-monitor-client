# Handoff: Pi Terminal World Monitor – progress and next steps

Use this document to continue development with a new agent or session. It summarizes what’s done, how to run everything, and what was changed in the last session so the next agent has full context.

---

## Session summary (for next agent)

**Context:** The previous session implemented the crypto dashboard (top cryptos, stablecoins, crypto news, BTC ETF Tracker), made the top-cryptos panel resolution-aware with a static title and 8s cycle, and finished the BTC ETF Tracker UI with stub data, aligned columns, all ETFs in one view, and a 6s refresh timer. Below is the full list of changes; **do not re-implement these**—they are done.

### Backend changes (all in `backend/app/panels.py`)

- **Crypto top (`GET /panels/crypto/top`)**  
  - **Single fetch for 33 coins:** `_fetch_top33_coins()` requests page 1, per_page 33 from CoinGecko once; all three “pages” (1–11, 12–22, 23–33) are sliced from this list to avoid rate limits.  
  - **Query params:** `range_start` (1-based, e.g. 1, 12, 23) and **`per_page`** (optional, default 11, clamped 5–25). Client uses `per_page` for resolution-aware line count.  
  - **Price changes:** Request uses `price_change_percentage=1h,24h,7d`; response includes `price_1h_pct`, `price_24h_pct`, `price_7d_pct`.  
  - **Caching:** Only non-empty results are cached; empty/rate-limited responses are not cached so the next request retries. One retry with 1s delay on failure.

- **Crypto news (`GET /panels/crypto/news`)**  
  - Fetches CoinDesk RSS (URL without trailing slash to avoid 308). Parses with `xml.etree.ElementTree`; `text_of()` handles CDATA. Returns `{ "status", "source": "rss", "items": [ { "title", "link", "pub_date", "description" } ] }`. `description` is the article blurb (from RSS `<description>`). Cached 5 min (`_crypto_news_cache`).

- **BTC ETF (`GET /panels/crypto/btc-etf`)**  
  - **Stub data** in `BTC_ETF_STUB`: 10 ETFs (IBIT, FBTC, ARKB, BITB, GBTC, HODL, BRRR, EZBC, BTCO, BTCW) with `ticker`, `issuer`, `est_flow_m`, `volume_m`, `change_pct`.  
  - Response: `net_flow_label` ("NET OUTFLOW" / "NET INFLOW"), `est_flow_m`, `total_vol_m`, `etfs_up`, `etfs_down`, `etfs[]`.  
  - Real data source (e.g. Farside/Blockworks) to be wired later; replace `_btc_etf_stub()`.

### Go client changes (all in `client-go/main.go`)

- **Top Cryptos sub-panel**  
  - **Static title:** Always `" Top cryptos by mcap (8s) "` (no range in title).  
  - **8s cycle:** Page advances every 8 seconds; countdown in title 8→7→…→1 then flip.  
  - **Resolution-aware lines:** On refresh, client gets panel height via `vTop.GetRect()` inside `app.QueueUpdateDraw()`, computes `linesPerPage = height - 2` (clamped 5–25), builds `rangeStarts` (e.g. for 11: [1,12,23]; for 14: [1,15,29]), and calls `/panels/crypto/top?range_start=%d&per_page=%d`. So the number of rows shown matches the panel height on different monitors (16" 1600p vs 27" 1440p).  
  - **API:** `renderCryptoTopWithRange(baseURL, rangeStart, perPage)`; backend returns `range` string (e.g. "1-11") but title no longer uses it.

- **Stablecoins**  
  - Unchanged: 6s refresh; no paging.

- **Crypto News**  
  - **Real data:** Backend returns `description` (blurb) per item from RSS. Panel shows **headline**, then a blank line, then **description/blurb** word-wrapped to panel width. Layout fills the panel: `renderCryptoNewsWithSize(baseURL, width, contentHeight)` uses panel rect (from `vNews.GetRect()` on refresh); multiple articles shown with as much blurb as fits. 6s refresh; initial draw uses actual panel size.

- **BTC ETF Tracker sub-panel**  
  - **All ETFs at once:** No paging; `renderCryptoBtcEtfAll(baseURL)` renders header + full `etfs[]` list.  
  - **Refresh and timer:** Data refreshes every **6 seconds**; title is `" BTC ETF Tracker (6s) "` with live countdown 6→5→…→1, then refresh and reset to 6.  
  - **Alignment:** Fixed-width columns so header and data align: TICKER 6, ISSUER 15, EST. FLOW 11, VOLUME 8, CHANGE 8. Constants `btcColTicker`, `btcColIssuer`, etc. First header row uses a first block width of `btcColTicker+1+btcColIssuer` so "Est. Flow", "Total Vol", "ETFs" align with columns.  
  - **Response struct:** `cryptoBtcEtfResp` has `NetFlowLabel`, `EstFlowM`, `TotalVolM`, `EtfsUp`, `EtfsDown`, `Etfs []cryptoBtcEtfEntry`; each entry has `Ticker`, `Issuer`, `EstFlowM`, `VolumeM`, `ChangePct`. Format helpers: `formatBtcEtfFlow`, `formatBtcEtfVol`.

- **Price formatting (top cryptos)**  
  - `fmtPrice()`: commas and two decimals for prices ≥ 1 (e.g. $66,825.00); smaller values as before.

### Docs and deploy

- **AGENTS.md:** Rule added: when changes require deploy, give user (1) local commit/push with a **relevant commit message**, (2) VPS pull-and-restart commands.  
- **HANDOFF-PROGRESS.md:** “Deploy workflow” section with push/pull/restart steps and **port 8000 in use** handling: `sudo lsof -i :8000`, `sudo kill -9 <PID>`, optional one-liner with `sleep 2`.

---

## Current state (as of handoff)

### Done

- **Repo and layout**  
  - Public GitHub repo; backend (Python/FastAPI) and two clients (Go recommended, Python alternative).  
  - OpenSpec change in `openspec/changes/add-pi-terminal-world-monitor-client/`; `tasks.md` updated with completed items.

- **Backend (VPS)**  
  - **Endpoints:** `GET /health`, `GET /panels`, `GET /panels/world-clock`, `GET /panels/weather`, `GET /panels/global-situation-map`, `GET /panels/crypto/top`, `GET /panels/crypto/stablecoins`, `GET /panels/crypto/news`, `GET /panels/crypto/btc-etf`.  
  - **Crypto top:** One CoinGecko request for top 33; slice by `range_start` + `per_page` (5–25). 1h/24h/7d price change; cache only non-empty; retry once.  
  - **Crypto news:** CoinDesk RSS, 5 min cache.  
  - **BTC ETF:** Stub returning 10 ETFs with header (net flow, est flow, total vol, etfs up/down); ready for real source.  
  - **Deployed:** DigitalOcean **209.38.141.129** (Ubuntu 24.10). Run: `cd /opt/pi-terminal-world-monitor-client/backend && .venv/bin/uvicorn app.main:app --host 0.0.0.0 --port 8000`. After backend code changes: `git pull` on droplet then restart uvicorn (see “Deploy workflow” below).

- **Go client (recommended)**  
  - **Build:** `cd client-go && go mod tidy && go build -o pi-world-monitor-client .`  
  - **Env:** `BACKEND_URL`, `CYCLE_SECONDS` (default 8), `GRID_COLS` / `GRID_ROWS` (default 2×2). Press **Q** to quit.  
  - **Crypto panel (top-left slot):** 4 sub-panels in **2×2**:  
    1. **Top Cryptos** – Static title “Top cryptos by mcap (8s)”, 8s cycle, **resolution-aware** (lines per page from panel height), 33 coins in rotation.  
    2. **Stablecoins** – 6s refresh.  
    3. **Crypto News** – 6s refresh.  
    4. **BTC ETF Tracker** – All ETFs at once, 6s refresh, title “BTC ETF Tracker (6s)” with countdown; fixed-width columns.  
  - Weather Watch, Global Situation Map, World Clock unchanged; all use QueueUpdateDraw.

- **Python client**  
  - `client/`: alternative; run with venv and `BACKEND_URL`.

- **Deployment and docs**  
  - docs/DEPLOY-BACKEND.md, INSTALL-PI.md, contrib/systemd/, AGENTS.md (deploy rule).

### Not done

- **BTC ETF real data** – Stub in place; wire Farside/Blockworks or other free source in `_btc_etf_stub()` / `crypto_btc_etf()`.  
- **Terminal sparklines** for crypto (ASCII/Unicode mini-charts).  
- Other panel categories; formal OpenAPI; Pi 3B validation (task 7.4).

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
export BACKEND_URL=http://209.38.141.129:8000
./pi-world-monitor-client
```

**Smoke test backend:**
```bash
curl -s http://209.38.141.129:8000/health
curl -s http://209.38.141.129:8000/panels
curl -s http://209.38.141.129:8000/panels/crypto/top?range_start=1&per_page=11
curl -s http://209.38.141.129:8000/panels/crypto/btc-etf
```

---

## Deploy workflow: push and restart backend

When **backend** code changes and the user needs to deploy:

**1. Local: commit and push with a relevant commit message** (not generic “Updates”).  
**2. VPS: pull and restart.** If port 8000 is in use: `sudo lsof -i :8000`, then `sudo kill -9 <PID>`, then:

```bash
sudo kill -9 $(sudo lsof -t -i :8000) 2>/dev/null; sleep 2; cd /opt/pi-terminal-world-monitor-client && git pull && cd backend && .venv/bin/uvicorn app.main:app --host 0.0.0.0 --port 8000
```

If the backend is managed by systemd: `systemctl restart pi-world-monitor` (or the actual service name).

**Rule for agents:** When your changes require the user to push and restart the backend, always give (1) the exact commit message suggestion and (2) the VPS commands. See AGENTS.md.

**Note:** Client-only changes (e.g. BTC ETF “all at once” + timer) do **not** require backend push/restart; user only rebuilds and runs the Go client.

---

## Repo layout

| Path | Purpose |
|------|--------|
| `backend/` | FastAPI app; `app/main.py`, `app/panels.py`; runs on VPS |
| `client-go/` | Go + tview client; **recommended** |
| `client/` | Python + blessed client; alternative |
| `docs/` | DEPLOY-BACKEND.md, INSTALL-PI.md, **HANDOFF-PROGRESS.md** (this file) |
| `contrib/` | systemd units, dietpi-autostart.sh |
| `openspec/changes/add-pi-terminal-world-monitor-client/` | Proposal, design, tasks.md, specs |

---

## Backend API reference (crypto)

- **`GET /panels/crypto/top?range_start=1&per_page=11`**  
  Response: `{ "status", "source", "range": "1-11", "coins": [ { "rank", "symbol", "name", "price", "price_1h_pct", "price_24h_pct", "price_7d_pct" } ] }`.  
  Single internal fetch of 33; slice by `range_start` and `per_page` (5–25).

- **`GET /panels/crypto/stablecoins`**  
  `{ "status", "status_label", "market_cap_b", "volume_b", "coins": [ { "symbol", "name", "price", "peg_status", "deviation_pct" } ] }`

- **`GET /panels/crypto/news`**  
  `{ "status", "source": "rss", "items": [ { "title", "link", "pub_date", "description" } ] }` — `description` is the article blurb from RSS.

- **`GET /panels/crypto/btc-etf`**  
  `{ "status", "source": "stub", "net_flow_label", "est_flow_m", "total_vol_m", "etfs_up", "etfs_down", "etfs": [ { "ticker", "issuer", "est_flow_m", "volume_m", "change_pct" } ] }`

---

## Next steps (for the next agent)

1. **BTC ETF real data** – Replace `_btc_etf_stub()` with a free source (e.g. Farside Investors data via Blockworks or similar); keep same response shape.  
2. **Crypto sparklines** – Optional ASCII/Unicode sparklines for top cryptos (e.g. CoinGecko `market_chart` + backend series + client render).  
3. **Pi 3B validation (task 7.4)** – Build Go client for ARM (`GOOS=linux GOARCH=arm GOARM=7`), test on DietPi; update INSTALL-PI.md if needed.  
4. **OpenSpec** – Mark completed items in `openspec/changes/.../tasks.md`; when feature-complete, use `/opsx:archive` (or project workflow).

---

## Reference

- **OpenSpec change:** `openspec/changes/add-pi-terminal-world-monitor-client/` (proposal.md, design.md, tasks.md, specs/).  
- **World Monitor (inspiration):** https://github.com/koala73/worldmonitor  
- **tview:** https://github.com/rivo/tview  
- **Backend droplet:** 209.38.141.129 (Ubuntu 24.10).
