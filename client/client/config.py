"""
Client configuration from environment.
"""
import os


def get_cycle_seconds() -> int:
    """Seconds to show each panel before cycling. From CYCLE_SECONDS env, default 8."""
    raw = os.environ.get("CYCLE_SECONDS", "8")
    try:
        n = int(raw)
        return max(1, min(120, n))
    except ValueError:
        return 8


def get_backend_url() -> str:
    """Backend API base URL. From BACKEND_URL env, default localhost:8000."""
    return os.environ.get("BACKEND_URL", "http://localhost:8000").rstrip("/")
