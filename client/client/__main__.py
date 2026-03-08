"""
Pi Terminal World Monitor – Terminal client entrypoint.

Runs in a Linux terminal (target: Pi 3B). Connects to backend only; no direct third-party APIs.
"""
import os
import re
import sys
import time

import httpx
from blessed import Terminal

from client.config import get_backend_url, get_cycle_seconds
from client.palette import SEVERITY, TREND, UI

REQUEST_TIMEOUT = 10.0

# Grid layout: minimum size so panels don't break on small/resized terminals
MIN_PANEL_WIDTH = 28
MIN_PANEL_HEIGHT = 7   # title, sep, content lines, bottom
PANEL_GAP = 1
HEADER_LINES = 2
FOOTER_LINES = 1


def fetch_json(base_url: str, path: str) -> dict | None:
    url = base_url.rstrip("/") + path
    try:
        r = httpx.get(url, timeout=REQUEST_TIMEOUT)
        r.raise_for_status()
        return r.json()
    except Exception:
        return None


def _style(term, attr: str, text: str) -> str:
    """Wrap text in a blessed attribute using concatenation (avoids callable form for Python 3.14/curses).
    Supports composite attributes like 'bold_cyan' (applied left to right)."""
    normal = getattr(term, "normal", "")
    parts = attr.split("_")
    seq = ""
    for part in parts:
        s = getattr(term, part, None)
        if s is not None:
            try:
                seq = seq + s
            except TypeError:
                pass
    if not seq:
        return text
    try:
        return seq + text + normal
    except TypeError:
        return text


def render_world_clock(term, data: dict) -> list[str]:
    lines = []
    if data.get("status") != "ok":
        return [_style(term, UI["placeholder"], "No data")]
    lines.append(_style(term, "bold", "UTC ") + data.get("utc", ""))
    for z in data.get("zones", []):
        lines.append(f"  {z.get('name', '')}: {z.get('date', '')} {z.get('time', '')}")
    return lines


def render_weather(term, data: dict) -> list[str]:
    lines = []
    if data.get("status") == "placeholder":
        lines.append(_style(term, UI["placeholder"], data.get("message", "Weather coming soon.")))
    else:
        locs = data.get("locations", [])
        if not locs:
            lines.append(_style(term, UI["placeholder"], "No locations yet."))
        for loc in locs:
            lines.append(f"  {loc.get('name', '')}: {loc.get('temp', '')}° {loc.get('conditions', '')}")
    return lines


PANEL_RENDERERS = {
    "world-clock": ("World Clock", render_world_clock),
    "weather": ("Weather Watch", render_weather),
}


def _grid_size(term) -> tuple[int, int, int, int]:
    """Return (cols, rows, panel_w, panel_h) so panels fit and don't break on resize."""
    cols = max(1, (term.width + PANEL_GAP) // (MIN_PANEL_WIDTH + PANEL_GAP))
    available_h = term.height - HEADER_LINES - FOOTER_LINES
    rows = max(1, available_h // MIN_PANEL_HEIGHT)
    panel_w = max(MIN_PANEL_WIDTH, (term.width - (cols - 1) * PANEL_GAP) // cols)
    panel_h = MIN_PANEL_HEIGHT
    return cols, rows, panel_w, panel_h


def _strip_ansi(s: str) -> str:
    return re.sub(r"\x1b\[[0-9;]*m", "", s)


def _panel_lines(term, title: str, content_lines: list[str], panel_w: int, panel_h: int) -> list[str]:
    """Build the list of lines for one panel box. Every line is exactly panel_w visible chars so borders close."""
    border_seq = getattr(term, UI["border"], "") or ""
    border_off = getattr(term, "normal", "")
    inner_w = max(2, panel_w - 4)
    max_content = max(0, panel_h - 4)

    def make_line(inner: str) -> str:
        """Inner must have visible length <= inner_w; we pad to inner_w so full line is exactly panel_w."""
        plain = _strip_ansi(inner)
        if len(plain) > inner_w:
            inner = plain[: inner_w - 1] + "…"
            plain = inner
        padded = inner + " " * max(0, inner_w - len(plain))
        return border_seq + "│ " + padded + " │" + border_off

    out = []
    top = "┌" + "─" * (panel_w - 2) + "┐"
    out.append(border_seq + top + border_off)
    tit = (title[: inner_w]).ljust(inner_w)
    out.append(border_seq + "│ " + tit + " │" + border_off)
    mid = "├" + "─" * (panel_w - 2) + "┤"
    out.append(border_seq + mid + border_off)

    for i in range(max_content):
        if i < len(content_lines):
            line = content_lines[i] if isinstance(content_lines[i], str) else str(content_lines[i])
            out.append(make_line(line))
        else:
            out.append(border_seq + "│ " + " " * inner_w + " │" + border_off)

    bot = "└" + "─" * (panel_w - 2) + "┘"
    out.append(border_seq + bot + border_off)
    return out


def _draw_grid(term, panels: list[str], base: str, cycle_seconds: int, index: int, cycle_offset: int) -> None:
    """Draw app header, grid of panel boxes, and footer. Fetch all data first, then clear and draw (avoids blank flash)."""
    cols, rows, panel_w, panel_h = _grid_size(term)
    inner_w = max(2, panel_w - 4)
    max_content = max(0, panel_h - 4)
    gap = " " * PANEL_GAP

    num_cells = cols * rows
    start = (cycle_offset % max(1, len(panels))) if panels else 0
    visible = [panels[(start + i) % len(panels)] for i in range(num_cells)]

    # Fetch all panel data before clearing the screen so we never show a blank grid
    panel_line_buffers: list[list[str]] = []
    for key in visible:
        title, render_fn = PANEL_RENDERERS.get(
            key, ("?", lambda t, d: [_style(t, UI["placeholder"], "?")])
        )
        data = fetch_json(base, f"/panels/{key}") or {}
        raw = render_fn(term, data)
        content = list(raw[:max_content])
        while len(content) < max_content:
            content.append("")
        panel_line_buffers.append(_panel_lines(term, title, content, panel_w, panel_h))

    # Now clear and draw in one go
    print(term.clear)
    print(_style(term, UI["app_title"], "Pi Terminal World Monitor"))
    print()

    # Emit grid row by row: each "row" of panels has panel_h lines; each line is concat of panel lines
    for row in range(rows):
        for line_idx in range(panel_h):
            row_line = ""
            for col in range(cols):
                cell_idx = row * cols + col
                lines = panel_line_buffers[cell_idx]
                if line_idx < len(lines):
                    row_line += lines[line_idx]
                else:
                    row_line += " " * panel_w
                if col < cols - 1:
                    row_line += gap
            print(row_line)

    print()
    footer = f"Next in {cycle_seconds}s  ·  Q quit  ·  Ctrl+C exit"
    print(_style(term, UI["footer"], footer))


def run_dashboard(term, backend_url: str, is_tty: bool, cycle_seconds: int) -> None:
    base = backend_url.rstrip("/")
    panel_list = fetch_json(base, "/panels")
    if not panel_list or "panels" not in panel_list:
        panels = ["world-clock", "weather"]
    else:
        panels = [p for p in panel_list["panels"] if p in PANEL_RENDERERS]
    if not panels:
        panels = ["world-clock", "weather"]

    index = 0
    while True:
        if is_tty:
            _draw_grid(term, panels, base, cycle_seconds, index, index)
        else:
            # Non-TTY: print each panel in sequence
            for i, key in enumerate(panels):
                title, render_fn = PANEL_RENDERERS.get(key, ("?", lambda t, d: ["?"]))
                data = fetch_json(base, f"/panels/{key}") or {}
                print(f"[{title}]")
                for line in render_fn(term, data):
                    print(line)
                print()

        if is_tty:
            try:
                key = term.inkey(timeout=cycle_seconds)
                if key and key.lower() == "q":
                    break
            except KeyboardInterrupt:
                break
        else:
            try:
                time.sleep(cycle_seconds)
            except KeyboardInterrupt:
                break

        index += 1


def main():
    term = Terminal()
    backend_url = get_backend_url()
    cycle_seconds = get_cycle_seconds()
    is_tty = term.is_a_tty

    try:
        if is_tty:
            with term.fullscreen(), term.cbreak(), term.hidden_cursor():
                if fetch_json(backend_url, "/health") is None:
                    print(term.clear)
                    print(_style(term, UI["error"], "Cannot reach backend: " + backend_url))
                    print(_style(term, UI["placeholder"], "Start the backend or set BACKEND_URL. Press any key to exit."))
                    term.inkey()
                    return 1
                run_dashboard(term, backend_url, is_tty=True, cycle_seconds=cycle_seconds)
        else:
            if fetch_json(backend_url, "/health") is None:
                print("Cannot reach backend:", backend_url)
                return 1
            run_dashboard(term, backend_url, is_tty=False, cycle_seconds=cycle_seconds)
    except KeyboardInterrupt:
        return 0
    except Exception as e:
        if "setupterm" in str(e).lower() or "curses" in str(e).lower():
            print("Pi Terminal World Monitor (no TTY)")
            print(f"Backend: {backend_url}")
            print("Run in a real terminal for full TUI.")
            return 0
        raise

    return 0


if __name__ == "__main__":
    sys.exit(main())
