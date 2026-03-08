# Design: add-pi-terminal-world-monitor-client

## Context

- **Hardware:** Raspberry Pi 3B, running DietPi (optionally LXDE); target is "getting the most out of older hardware."
- **Inspiration:** [World Monitor](https://github.com/koala73/worldmonitor) — real-time global intelligence dashboard with 435+ feeds, 45 map layers, strategic risk, markets, crypto, conflicts, climate, etc. Running the full web app in Chromium on the Pi is too heavy.
- **Constraint:** No map, no live video; data only, displayed in a terminal. Backend runs on operator's VPS; Pi runs only a thin client that fetches pre-aggregated data.
- **Reference:** PatchNotesLive backend patterns (cache-first, free-tier-friendly APIs, RSS, versioned endpoints) and Weather/WeatherNews panel behavior can inform Weather Watch and similar panels.

## Goals / Non-Goals

- **Goals:**
  - Single, clean terminal UI with distinct panels and color coding.
  - All requested data categories represented (strategic risk, intel, regional news, predictions, energy, think tanks, commodities, markets, economy, trade, supply chain, financial news, tech, crypto, fires, market radar, BTC ETF, stablecoins, conflicts, global giving, climate, weather, world clock).
  - Panel cycling so every category is shown over time despite limited screen space.
  - Readable text-based "Global Situation Map" (region/country + severity/event type, no graphical map).
  - Backend on VPS only; Pi never calls third-party APIs directly.
  - Free APIs and RSS only; cache-first and rate-limit friendly.

- **Non-Goals:**
  - Graphical map or 3D globe.
  - Live video or HLS streams.
  - Local AI/LLM on the Pi.
  - Reusing PatchNotesLive codebase or repo; this is a new repo and a separate backend service.
  - Supporting every World Monitor variant (Tech/Finance/Commodity/Happy); one coherent dashboard that blends the most relevant categories is enough for v0.

## Decisions

### Backend on VPS; Pi as display client only
- **Decision:** All aggregation, RSS parsing, and third-party API calls run on the VPS. The Pi client only calls the backend's API (REST or simple polling). No API keys or heavy logic on the Pi.
- **Why:** Keeps the Pi client lightweight and avoids rate limits / key exposure on the device. Aligns with PatchNotesLive's "Unity never calls third-party APIs directly" pattern.
- **Alternatives:** Pi calls APIs directly — rejected due to rate limits and complexity on constrained hardware.

### Single "dashboard API" surface
- **Decision:** Backend exposes a small set of endpoints (e.g. by panel or by group) that return JSON (or a compact format) ready for the terminal to render. Optionally one "bootstrap" or "all panels" endpoint to minimize round-trips.
- **Why:** Simplifies the client and allows backend to reshape/cache data per panel. Same idea as World Monitor's bootstrap hydration and panel-specific feeds.
- **Alternatives:** Client composes many micro-endpoints — acceptable if the agent prefers it, but fewer round-trips from the Pi is better.

### Panel cycling with configurable duration
- **Decision:** The client cycles through a fixed set of panels (or panel groups) on a timer. Duration per panel (or per "slot") is configurable so the user can tune how long each category stays on screen.
- **Why:** Screen cannot show all panels at once; cycling ensures every category is visible over time. Matches the "cycle the panel with new data categories" requirement.
- **Alternatives:** Scrollable single view — possible but less "dashboard-like"; pagination by key — adds input complexity. Cycling is the primary requirement.

### Color coding by severity and trend
- **Decision:** Use terminal colors (e.g. red/orange/yellow/green and dim/bright) to indicate severity (critical, elevated, monitoring, normal), trend (up/down), and category where useful. Exact palette is left to the implementing agent; consistency across panels is required.
- **Why:** "Everything should be color coated" and World Monitor's use of red/orange/yellow/green for alerts and status.
- **Alternatives:** Monochrome — contradicts requirement; full RGB — may be unnecessary; 8–16 color ANSI is a reasonable default.

### Text-based Global Situation Map
- **Decision:** Backend provides a structured representation of "global situation" (e.g. regions or countries with a severity/alert level and short labels for event types: conflict, hotspot, military, disaster, etc.). Client renders it as readable text (e.g. list or grouped by region), not a graphical map.
- **Why:** User asked for a "readable text-based version of the Global situation map"; World Monitor's map layers (conflict zones, hotspots, military, etc.) and CII/risk scores are the conceptual source.
- **Alternatives:** ASCII-art map — possible but not required; map data only with client-side layout — acceptable if the backend provides enough structure for clear text layout.

### Data sources: free APIs and RSS only
- **Decision:** Backend uses only free-tier or no-key APIs and RSS feeds. World Monitor's public API (`api.worldmonitor.app`) is CORS-restricted for browsers but may be usable from a server; documentation and endpoint list can guide which data categories to replicate via other free sources (e.g. RSS, FRED, Open-Meteo, public conflict/event datasets, etc.).
- **Why:** Cost and simplicity; aligns with "free APIs and RSS" and PatchNotesLive's free-tier-friendly approach.
- **Reference:** World Monitor lists 435+ RSS feeds, GDELT, OpenSky, NASA FIRMS, FRED, EIA, Yahoo Finance, CoinGecko, Polymarket, etc.; many have free tiers or public endpoints. Implementing agent chooses specific sources per panel.

### New repo and separate backend
- **Decision:** This product is implemented in a **new repository**. The backend is a **separate service** on the operator's VPS (not the PatchNotesLive overlay backend). No shared codebase with PatchNotesLive unless the agent explicitly copies patterns or snippets.
- **Why:** User stated "We'll create a new repo for this project as well as a separate backend on my VPS"; scope and deployment are independent.

## Risks / Trade-offs

- **World Monitor API:** If the backend calls `api.worldmonitor.app` from the VPS, respect their ToS and rate limits; they may block non-browser origins. Prefer replicating with direct free APIs and RSS where possible.
- **Pi 3B performance:** Terminal rendering and periodic polling should stay light (e.g. avoid huge JSON payloads or constant redraws). Cursor/alternate screen and minimal redraws are recommended.
- **Panel count:** Many panels imply either short cycle times or long "full cycle" duration. Configurable per-panel or per-slot duration helps; default can favor high-value panels (e.g. strategic risk, intel, world clock) with slightly longer display.

## Migration Plan

- N/A. This is a greenfield project. No migration from PatchNotesLive.

## Open Questions

- Exact terminal stack on Pi (e.g. ncurses, blessed, notcurses, or raw ANSI) — left to implementing agent.
- Whether to support multiple "views" (e.g. focus on intel vs finance) or a single rotating view only — spec allows single cycling view; views can be added later.
- Optional: reuse of PatchNotesLive weather/Open-Meteo or RSS patterns for Weather Watch panel — recommended but not mandatory.
