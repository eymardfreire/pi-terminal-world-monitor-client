# Handoff: Pi terminal World Monitor–style dashboard

This folder is a **change proposal only**. Implementation will be done on a **different machine (Linux)** with a **different agent**. Copy this entire change folder to the target environment and use it as the single source of truth.

## What to copy

Copy the whole directory:

```
openspec/changes/add-pi-terminal-world-monitor-client/
```

Include all of:

- `proposal.md` — why and what
- `design.md` — context, goals, decisions, risks
- `tasks.md` — implementation checklist
- `specs/` — requirement deltas for backend, client, and global situation map
- `HANDOFF.md` — this file

## How to use it on the Linux machine

1. **Create a new repo** for the project (backend + terminal client can be two subdirs or two repos; design leaves that to the implementer).
2. **Read in order:** `proposal.md` → `design.md` → `tasks.md` → `specs/*/spec.md`.
3. **Implement** by following `tasks.md`; mark items `[x]` as done.
4. **Backend** runs on your VPS; **client** runs on the Pi 3B (DietPi/Linux). No code lives in the PatchNotesLive repo.
5. **Reference:** [World Monitor](https://github.com/koala73/worldmonitor) for data categories and ideas; [api.worldmonitor.app](https://github.com/koala73/worldmonitor#programmatic-api-access) for optional API reference (use from VPS only; prefer free APIs and RSS where possible).
6. **PatchNotesLive:** Existing backend/weather/RSS patterns can be used as reference for Weather Watch and cache-first behavior; no requirement to share code.

## Panel list (for quick reference)

Strategic Risk Overview, Intel Feed, Live Intelligence, Work News, then one panel per continent (Africa, Americas, Asia-Pacific, Europe, Latin America, Middle East, etc.), Predictions, Energy and Resources, Think Tanks, Commodities, Markets, Economy Indicators, Trade Policy, Supply Chain, Financial News, Technology, Crypto, Fires, Market Radar, BTC ETF Tracker, Stablecoins, Armed Conflict Events, Global Giving, Climate Anomalies, Weather Watch (similar to PatchNotesLive), World Clock. Plus a **text-based Global Situation Map** view. All panels **color-coded**; **panel cycling** so every category is shown over time.

## Validation (optional)

If the target environment has OpenSpec tooling and you copied the folder under an `openspec/changes/` tree:

```bash
openspec validate add-pi-terminal-world-monitor-client --strict
```

Otherwise, use the proposal and specs as normative requirements for the implementation.
