## 1. Repo and project setup
- [x] 1.1 Create new repository for the project (name TBD by implementer, e.g. `pi-world-monitor` or `terminal-world-dashboard`).
- [x] 1.2 Initialize backend project on VPS codebase (e.g. FastAPI/Node/other) with dependency list and minimal health endpoint.
- [x] 1.3 Initialize terminal client project (e.g. Node/Python/Rust) with dependency list; ensure it runs in a Linux terminal (target: Pi 3B).

## 2. Backend: API contract and data shape
- [ ] 2.1 Define REST (or polling) API surface: panel-based or grouped endpoints, plus optional bootstrap "all panels" endpoint.
- [ ] 2.2 Define response schemas per panel category (strategic risk, intel, news by region, predictions, energy, think tanks, commodities, markets, economy, trade, supply chain, financial news, tech, crypto, fires, market radar, BTC ETF, stablecoins, conflicts, global giving, climate, weather, world clock).
- [ ] 2.3 Define schema for the text-based Global Situation Map (regions/countries, severity, event-type labels).

## 3. Backend: Data connectors (free APIs + RSS)
- [ ] 3.1 Implement cache-first fetchers for each data category; use free APIs and RSS only; respect rate limits and backoff.
- [ ] 3.2 Wire at least one source per panel (or document "no free source" and return placeholder/empty with clear status).
- [ ] 3.3 Add strategic risk / CII-style summary (e.g. from conflict + news feeds or public indices) for Strategic Risk Overview panel.
- [ ] 3.4 Add Weather Watch data (e.g. Open-Meteo or same pattern as PatchNotesLive weather) and World Clock (server time + optional timezone list).
- [ ] 3.5 Implement Global Situation Map data pipeline (aggregate hotspots, conflict zones, severity by region/country into text-oriented structure).

## 4. Terminal client: layout and panels
- [ ] 4.1 Implement terminal UI framework (ncurses/blessed/notcurses/raw ANSI) with panel containers and a clear layout.
- [ ] 4.2 Implement panel cycling: ordered list of panels (or groups), configurable duration per step, loop forever.
- [ ] 4.3 For each panel type, implement fetcher (call backend) and renderer with color coding (severity, trend, category).
- [ ] 4.4 Implement World Clock panel (and optionally a persistent clock in header/footer if layout allows).
- [ ] 4.5 Implement Weather Watch panel (similar to PatchNotesLive: cities/conditions, color-coded).

## 5. Terminal client: Global Situation Map (text)
- [ ] 5.1 Consume backend Global Situation Map endpoint and render as readable text (e.g. by region, with severity and event-type labels).
- [ ] 5.2 Apply color coding to severity levels (e.g. critical=elevated=monitoring=normal).

## 6. Deployment and ops
- [ ] 6.1 Deploy backend to operator's VPS; document env vars and run instructions.
- [ ] 6.2 Document how to run the terminal client on the Pi (e.g. at boot or in a tmux/screen session).
- [ ] 6.3 Optional: add a simple "auto-start on boot" script for DietPi that launches the terminal dashboard (e.g. in a virtual terminal or framebuffer).

## 7. Validation
- [ ] 7.1 Backend: all panel endpoints return valid schema; cache and backoff behave under load.
- [ ] 7.2 Client: all panels render without crash; cycling advances correctly; colors are consistent.
- [ ] 7.3 Global Situation Map: text output is readable and matches backend severity/event structure.
- [ ] 7.4 Run client on Pi 3B (or equivalent) and confirm acceptable CPU/memory and readability.
