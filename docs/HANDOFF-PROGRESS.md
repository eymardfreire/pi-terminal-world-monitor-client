# Handoff: Pi Terminal World Monitor – progress and next steps

Use this document to continue development with a new agent or session. It summarizes what’s done, how to run everything, and what was changed so the next agent has full context.

---

## Session summary (for next agent)

**Context:** This session **replaced the Global Situation panel with an 8 News panel** (4 top, 4 bottom): World, US, Europe, Middle East, Africa, Asia-Pacific, Energy & Resources, Government. Each sub-panel shows **X NEW** (backlog count), **10s** timer, and cycles **headline + article/blurb** (like crypto news). GSM is scrapped from the UI; backend and docs kept for future. Crypto dashboard (56 coins, stablecoins, gainers/losers, crypto news, BTC ETF) is unchanged—do not re-implement.

### Backend changes (all in `backend/app/panels.py`)

- **Crypto top (`GET /panels/crypto/top`)**  
  - **Single fetch for 56 coins:** `_fetch_top56_coins()` requests page 1, per_page 56 from CoinGecko; client cycles pages by `range_start` + `per_page` (5–25).  
  - **Query params:** `range_start` (1-based), `per_page` (optional, default 11, clamped 5–25).  
  - **Price changes:** `price_1h_pct`, `price_24h_pct`, `price_7d_pct`. Cache key `_TOP56_CACHE_KEY`, `TOP_COINS_COUNT = 56`.

- **Crypto news (`GET /panels/crypto/news`)**  
  - CoinDesk RSS; `text_of()` uses `itertext()` so CDATA is included. **Keep first non-empty `description`** (do not overwrite by empty `dc:description`). Returns `title`, `link`, `pub_date`, `description` (blurb). Cached 5 min.

- **Crypto stablecoins (`GET /panels/crypto/stablecoins`)**  
  - Fetches with `price_change="24h"`. Per-coin: `symbol`, `name`, `price`, `peg_status`, `deviation_pct`, `market_cap_b`, `volume_b`, `price_change_24h_pct`. `STABLECOIN_IDS` includes FDUSD (first-digital-usd). Response used for **tile** (status + MCap | Vol) and **ticker list only** (no per-ticker mcap/vol in client).

- **Crypto gainers-losers (`GET /panels/crypto/gainers-losers`)**  
  - Top 28 gainers and 28 losers by 24h price change (from top 100 mcap, excluding stablecoins). Each entry: `symbol`, `price`, **`change_24h_pct`**. Cached in `_crypto_cache`.

- **BTC ETF (`GET /panels/crypto/btc-etf`)**  
  - Stub data; same response shape. Real source to be wired later.

### Go client changes (all in `client-go/main.go`)

- **Crypto panel layout**  
  - **5 sub-panels:** Top row = Top Cryptos | (Stablecoins | Gainers/Losers); bottom row = Crypto News | BTC ETF. `cryptoSubpanelViews[0..4]`.

- **Top Cryptos**  
  - **56 coins;** 8s cycle; resolution-aware `perPage` from `vTop.GetRect()` (clamped 5–25). `rangeStarts` built from `topCoinsCount = 56`. `renderCryptoTopWithRange(baseURL, rangeStart, perPage)`.

- **Stablecoins**  
  - **Timer 8s** in title. **Tile:** line 1 = status (Healthy/Caution), line 2 = `MCap: $x.xB | Vol: $x.xB`. **Tickers only:** one line per coin = ticker, $price (2 decimals), ON PEG/OFF PEG, deviation %. No per-ticker MCap/Vol/change. **Paging:** like Top Cryptos; `perPage = (height - 2 - 3)` (tile 2 + blank); cycle pages every 8s. `renderCryptoStablecoinsPageFromData()`, `renderCryptoStablecoinsPage()`.

- **Crypto gainers / losers**  
  - **12 per page.** Cycle every 10s: gainers 0–11 → gainers 12–23 → losers 0–11 → losers 12–23 (phase 0–3). **Change %** next to price (green gainers, red losers). `gainersLosersPerPage = 12`. `renderCryptoGainersLosers(baseURL, showGainers, pageStart, perPage)`; struct `Change24hPct *float64`.

- **Crypto News**  
  - 20s cycle; **two articles per page**; headline + blurb each; **em dash** separator between articles. Pool refresh every 6s. `renderCryptoNewsTwoItems()`, `renderCryptoNewsOneItem()`, `fetchCryptoNewsItems()`.

- **BTC ETF Tracker**  
  - 6s refresh, countdown in title; all ETFs at once; fixed-width columns. Unchanged.

### Docs and deploy

- **AGENTS.md:** When changes require deploy, give (1) local commit/push with relevant message, (2) VPS pull-and-restart commands.  
- **Deploy workflow** in this doc: port 8000 kill, pull, uvicorn restart.

---

## Current state (as of handoff)

### Done

- **Repo and layout**  
  - Backend (Python/FastAPI), Go client (recommended), Python client (alternative). OpenSpec in `openspec/changes/add-pi-terminal-world-monitor-client/`.

- **Backend (VPS 209.38.141.129)**  
  - **Endpoints:** `GET /health`, `GET /panels`, `GET /panels/world-clock`, `GET /panels/weather`, `GET /panels/news`, `GET /panels/global-situation-map` (kept for future), `GET /panels/crypto/*`, …  
  - **News:** `GET /panels/news` — 8 feeds: World News, United States (BBC US/Canada), Europe, Middle East, Africa, **Latin America**, Asia-Pacific, Government. Each feed: `id`, `name`, `new_count` (items in last 6h), `items[]` (title, link, pub_date, description, source). RSS, 5 min cache.  
  - **Crypto top:** 56 coins, slice by `range_start` + `per_page` (5–25), 1h/24h/7d.  
  - **Crypto news:** CoinDesk RSS, description (blurb), 5 min cache.  
  - **Stablecoins:** status_label, market_cap_b, volume_b, per-coin peg + optional mcap/vol/24h (client uses tile + tickers only).  
  - **Gainers-losers:** 28 gainers, 28 losers, `symbol`, `price`, `change_24h_pct`.  
  - **BTC ETF:** Stub; ready for real source.

- **Go client**  
  - **Build:** `cd client-go && go build -o pi-world-monitor-client .`  
  - **Env:** `BACKEND_URL`, `CYCLE_SECONDS`, `GRID_COLS` / `GRID_ROWS`. Press **Q** to quit.  
  - **Crypto panel:** 5 sub-panels (Top Cryptos 56/8s | Stablecoins 8s + Gainers/Losers 10s; News 20s | BTC ETF 6s).  
  - **News panel:** Replaces Global Situation in grid. 8 sub-panels (4 top, 4 bottom): World News, United States, Europe, Middle East, Africa, **Latin America**, Asia-Pacific, Government. Each shows **X NEW** (backlog), **25s** timer with **5s offset** (staggered top-left→right), **headline + article/blurb** (description in italic). Feed data refreshed every 30s.  
  - **Weather, World Clock:** use `panelContent()` and grid refresh on main cycle.

- **Deployment and docs**  
  - docs/DEPLOY-BACKEND.md, INSTALL-PI.md, contrib/systemd/, AGENTS.md.

### Not done

- **Global Situation Map** – Scrapped from UI; backend and docs kept for possible future use.  
- **BTC ETF real data** – Stub in place; wire Farside/Blockworks or similar.  
- **Crypto sparklines** – Optional.  
- **Pi 3B validation** – ARM build and INSTALL-PI.md.  
- **OpenSpec** – Mark completed items; archive when feature-complete.

---

## How to run (quick reference)

**Backend (droplet or local):**
```bash
cd backend && .venv/bin/uvicorn app.main:app --host 0.0.0.0 --port 8000
```

**Go client (local or Pi):**
```bash
cd client-go && go build -o pi-world-monitor-client .
export BACKEND_URL=http://209.38.141.129:8000
./pi-world-monitor-client
```

**Smoke test backend:**
```bash
curl -s http://209.38.141.129:8000/health
curl -s http://209.38.141.129:8000/panels
curl -s http://209.38.141.129:8000/panels/crypto/top?range_start=1&per_page=11
curl -s http://209.38.141.129:8000/panels/news
```

---

## Deploy workflow: push and restart backend

When **backend** code changes:

**1. Local:** commit and push with a **relevant commit message**.  
**2. VPS:** pull and restart. If port 8000 is in use:

```bash
sudo kill -9 $(sudo lsof -t -i :8000) 2>/dev/null; sleep 2; cd /opt/pi-terminal-world-monitor-client && git pull && cd backend && .venv/bin/uvicorn app.main:app --host 0.0.0.0 --port 8000
```

Client-only changes: just rebuild and run the Go client.

---

## Repo layout

| Path | Purpose |
|------|--------|
| `backend/` | FastAPI app; `app/main.py`, `app/panels.py` |
| `client-go/` | Go + tview client; **recommended** |
| `client/` | Python + blessed client; alternative |
| `docs/` | DEPLOY-BACKEND.md, INSTALL-PI.md, **HANDOFF-PROGRESS.md** (this file) |
| `contrib/` | systemd units, dietpi-autostart.sh |
| `openspec/changes/add-pi-terminal-world-monitor-client/` | Proposal, design, tasks.md, specs |

---

## Backend API reference (crypto + news + GSM)

- **`GET /panels/news`**  
  `{ "status", "source", "feeds": [ { "id", "name", "new_count", "items": [ ... ] } ] }` — 8 feeds: World News, United States (BBC US/Canada), Europe, Middle East, Africa, **Latin America**, Asia-Pacific, Government. Client: 4+4 layout, **25s** per panel, **5s offset** (staggered refresh top-left→right), X NEW, headline + blurb (description in italic).

- **`GET /panels/crypto/top?range_start=1&per_page=11`**  
  `{ "status", "source", "range", "coins": [ { "rank", "symbol", "name", "price", "price_1h_pct", "price_24h_pct", "price_7d_pct" } ] }` — 56 coins total; slice by params.

- **`GET /panels/crypto/stablecoins`**  
  `{ "status", "status_label", "market_cap_b", "volume_b", "coins": [ { "symbol", "name", "price", "peg_status", "deviation_pct", "market_cap_b", "volume_b", "price_change_24h_pct" } ] }` — Client uses tile (status + MCap\|Vol) and ticker list only.

- **`GET /panels/crypto/gainers-losers`**  
  `{ "status", "gainers": [ { "symbol", "price", "change_24h_pct" } ], "losers": [ ... ] }` — 28 each.

- **`GET /panels/crypto/news`**  
  `{ "status", "source": "rss", "items": [ { "title", "link", "pub_date", "description" } ] }`

- **`GET /panels/crypto/btc-etf`**  
  Stub: `net_flow_label`, `est_flow_m`, `total_vol_m`, `etfs_up`, `etfs_down`, `etfs[]`.

- **`GET /panels/global-situation-map`**  
  (Kept for future.) `{ "status", "regions", "summary", "layers", … }` — Not shown in default grid; see “Global Situation Map” below.

---

## Global Situation Map (scrapped from UI; kept for future)

- **Backend:** `GET /panels/global-situation-map` still available; returns regions, summary, layers (stub).  
- **Client:** Panel no longer in default grid; `panelContent("global-situation-map")` and render helpers still present so it can be re-enabled.  
- **Intent:** May be re-implemented later; docs and API kept.

---

## Next steps (for the next agent)

1. **News panel** – Done (8 feeds, 4+4 layout, 10s cycle, X NEW backlog, headline+blurb per panel). **Global Situation Map** – Removed from UI; backend/docs kept for future.  
2. **BTC ETF real data** – Replace stub with Farside/Blockworks or similar; keep response shape.  
3. **Crypto sparklines** – Optional ASCII/Unicode for top cryptos.  
4. **Pi 3B validation** – ARM build, test on DietPi, update INSTALL-PI.md.  
5. **OpenSpec** – Update tasks.md; archive when complete.

---

## Reference

- **OpenSpec change:** `openspec/changes/add-pi-terminal-world-monitor-client/`  
- **World Monitor (inspiration):** https://github.com/koala73/worldmonitor  
- **tview:** https://github.com/rivo/tview  
- **Backend droplet:** 209.38.141.129 (Ubuntu 24.10).
