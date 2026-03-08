# Pi Terminal World Monitor – Go client (tview)

Terminal client written in Go using [tview](https://github.com/rivo/tview). Fetches panel data from the backend and displays a responsive grid with proper borders and layout.

## Prerequisites

- Go 1.21 or later (`go version`)

## Build

```bash
cd client-go
go mod tidy
go build -o pi-world-monitor-client .
```

On Raspberry Pi (e.g. DietPi), either build on the Pi or cross-compile from another machine:

```bash
# From Linux amd64, build for Pi (ARM 32-bit)
GOOS=linux GOARCH=arm GOARM=7 go build -o pi-world-monitor-client .
```

## Run

```bash
export BACKEND_URL=http://your-vps:8000
./pi-world-monitor-client
```

**Environment:**

- `BACKEND_URL` – Backend API base URL (default: `http://localhost:8000`).
- `CYCLE_SECONDS` – Seconds between data refresh (default: `8`).

Press **Q** to quit.

## Layout

- 2×2 grid of panels (World Clock and Weather Watch), each with a bordered box and title.
- Footer shows refresh countdown and exit hint.
- tview handles resizing and keeps panel borders closed.
