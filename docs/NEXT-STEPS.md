# Next steps – Pi Terminal World Monitor

Prioritized list for the next development session. See **HANDOFF-PROGRESS.md** for full context.

## 1. ~~Real Weather data (backend)~~ Done

- Open-Meteo wired in `backend/app/panels.py`; 10 min cache; London, New York, Tokyo, Berlin. Returns `locations: [{ "name", "temp", "conditions" }]`.

## 2. ~~Global Situation Map (backend + client)~~ Done

- Backend: `GET /panels/global-situation-map` with `regions[]` (name, severity, events). Stub data; pipeline ready for real feeds.
- Go client: panel `global-situation-map` with severity colors (red/yellow/cyan/green).

## 3. More panels and cache (backend)

- **Tasks:** 3.1 (cache-first), 3.2 (one source per panel), 2.2 (schemas).
- **Backend:** Add in-memory cache (e.g. `cachetools.TTLCache`) for external HTTP/RSS; backoff on rate limit. Add one or two more panel endpoints (e.g. strategic risk from RSS, or markets/crypto from a free API). Document response shapes in code or OpenAPI.

## 4. Go client: extend grid and panels

- **Task:** 4.3 – For each new backend panel, add a renderer in `client-go/main.go` and include in the panel list. Optionally make grid rows/cols configurable via env.

## 5. Validate on Pi 3B

- **Task:** 7.4 – Cross-compile Go client for ARM, run on DietPi, check CPU/memory and readability. Update **docs/INSTALL-PI.md** if build or run steps change.

---

After completing a batch of tasks, update `openspec/changes/add-pi-terminal-world-monitor-client/tasks.md` (mark `[x]`) and, if useful, add a short note to **HANDOFF-PROGRESS.md**.
