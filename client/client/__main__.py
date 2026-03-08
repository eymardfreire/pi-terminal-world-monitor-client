"""
Pi Terminal World Monitor – Terminal client entrypoint.

Runs in a Linux terminal (target: Pi 3B). Connects to backend only; no direct third-party APIs.
"""
import os
import sys

from blessed import Terminal

def main():
    term = Terminal()
    backend_url = os.environ.get("BACKEND_URL", "http://localhost:8000")
    is_tty = term.is_a_tty

    try:
        if is_tty:
            with term.fullscreen(), term.cbreak(), term.hidden_cursor():
                print(term.clear)
                print(term.center(term.bold("Pi Terminal World Monitor")))
                print(term.center(f"Backend: {backend_url}"))
                print(term.center("(Panel cycling and data will be wired in next)"))
                print()
                print(term.center("Press Ctrl+C to exit."))
                try:
                    while True:
                        term.inkey(timeout=1)
                except KeyboardInterrupt:
                    pass
        else:
            # Non-TTY (e.g. CI): plain output, no curses
            print("Pi Terminal World Monitor")
            print(f"Backend: {backend_url}")
            print("(Panel cycling and data will be wired in next)")
            print("Press Ctrl+C to exit.")
            import time
            try:
                while True:
                    time.sleep(1)
            except KeyboardInterrupt:
                pass
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
