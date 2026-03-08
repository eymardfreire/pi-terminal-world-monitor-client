# Change: Add Pi terminal World Monitor–style dashboard (new repo + VPS backend)

## Why
World Monitor ([github.com/koala73/worldmonitor](https://github.com/koala73/worldmonitor)) is a full-featured global intelligence dashboard that is too heavy to run in a browser on a Raspberry Pi 3B. A lightweight, terminal-only client fed by a dedicated backend on the operator's VPS will deliver the same categories of data (strategic risk, intel, news by region, markets, crypto, conflicts, climate, etc.) in a clean, color-coded, panel-based TUI suitable for low-resource hardware and aligned with the goal of getting the most out of older hardware.

## What Changes
- **New repository** for the project (separate from PatchNotesLive).
- **New backend service on the operator's VPS** that aggregates data from free APIs and RSS feeds and exposes a simple API for the terminal client. No direct third-party calls from the Pi.
- **Terminal dashboard client** (runs on Pi 3B under DietPi/Linux):
  - Panels as containers for organized data; layout decided by the implementing agent.
  - **Panel set:** Strategic Risk Overview, Intel Feed, Live Intelligence, Work News, one panel per continent (Africa, Americas, Asia-Pacific, Europe, Latin America, Middle East, etc.), Predictions, Energy and Resources, Think Tanks, Commodities, Markets, Economy Indicators, Trade Policy, Supply Chain, Financial News, Technology, Crypto, Fires, Market Radar, BTC ETF Tracker, Stablecoins, Armed Conflict Events, Global Giving, Climate Anomalies, Weather Watch (similar to PatchNotesLive weather panel), World Clock.
  - **Color coding** for severity, trend, and category across all panels.
  - **Panel cycling** so that over time all data categories are shown (screen real estate is limited).
  - **Text-based Global Situation Map** — readable text summary of global situation (e.g. by region/country with alert level and key event types), no graphical map.
- **Data sources:** free APIs and RSS only; backend handles caching, rate limits, and graceful degradation. World Monitor's public API (`api.worldmonitor.app`) and docs can be used as reference for data categories and, where allowed, endpoints.

## Impact
- **Affected specs (new capabilities):** `terminal-dashboard-backend`, `terminal-dashboard-client`, `global-situation-map` (see `changes/add-pi-terminal-world-monitor-client/specs/`).
- **Affected code:** None in PatchNotesLive. Delivered in a **new repo** and a **separate backend** on the operator's VPS.
- **Execution context:** Proposal is authored here for handoff; implementation will be done on a different machine (Linux) with a different agent; the change proposal folder can be copied to the new environment as the single source of truth.
