"""
Stub panel endpoints for deployment testing.
Real data sources will be wired in tasks 3.x.
"""
from datetime import datetime, timezone
from fastapi import APIRouter

router = APIRouter(prefix="/panels", tags=["panels"])


@router.get("")
def list_panels():
    """Panel keys the client can request. Used for discovery and cycling."""
    return {
        "panels": ["world-clock", "weather"],
        "status": "ok",
    }


@router.get("/world-clock")
def world_clock():
    """Current server time and optional timezones. Client can show in World Clock panel."""
    now = datetime.now(timezone.utc)
    return {
        "status": "ok",
        "source": "server",
        "utc": now.strftime("%Y-%m-%dT%H:%M:%SZ"),
        "zones": [
            {"name": "UTC", "time": now.strftime("%H:%M"), "date": now.strftime("%Y-%m-%d")},
            {"name": "Local", "time": now.strftime("%H:%M"), "date": now.strftime("%Y-%m-%d")},
        ],
    }


@router.get("/weather")
def weather():
    """Placeholder for Weather Watch panel. Real data from Open-Meteo etc. in task 3.4."""
    return {
        "status": "placeholder",
        "locations": [],
        "message": "Weather sources will be wired in backend task 3.4.",
    }
