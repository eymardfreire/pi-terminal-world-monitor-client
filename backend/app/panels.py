"""
Panel endpoints: World Clock, Weather (Open-Meteo), Global Situation Map.
Uses cache-first for external calls; placeholders when upstream fails.
"""
from datetime import datetime, timezone
from typing import Any

import httpx
from cachetools import TTLCache
from fastapi import APIRouter

router = APIRouter(prefix="/panels", tags=["panels"])

# In-memory cache for external APIs (TTL seconds, max entries)
_weather_cache: TTLCache[str, dict[str, Any]] = TTLCache(maxsize=64, ttl=600)  # 10 min
_gsm_cache: TTLCache[str, dict[str, Any]] = TTLCache(maxsize=4, ttl=300)  # 5 min

# Cities for Weather Watch (name, lat, lon)
WEATHER_LOCATIONS = [
    ("London", 51.5074, -0.1278),
    ("New York", 40.7128, -74.0060),
    ("Tokyo", 35.6762, 139.6503),
    ("Berlin", 52.5200, 13.4050),
]

# WMO weather code -> short label
WEATHER_CODE_LABELS: dict[int, str] = {
    0: "Clear",
    1: "Mainly clear",
    2: "Partly cloudy",
    3: "Overcast",
    45: "Fog",
    48: "Deposit rime fog",
    51: "Light drizzle",
    53: "Drizzle",
    55: "Dense drizzle",
    61: "Slight rain",
    63: "Rain",
    65: "Heavy rain",
    71: "Slight snow",
    73: "Snow",
    75: "Heavy snow",
    80: "Slight showers",
    81: "Showers",
    82: "Violent showers",
    85: "Slight snow showers",
    86: "Heavy snow showers",
    95: "Thunderstorm",
    96: "Thunderstorm + hail",
    99: "Thunderstorm + heavy hail",
}


def _weather_code_to_conditions(code: int) -> str:
    return WEATHER_CODE_LABELS.get(code, f"Code {code}")


def _fetch_one_weather(lat: float, lon: float) -> dict[str, Any] | None:
    url = "https://api.open-meteo.com/v1/forecast"
    params = {
        "latitude": lat,
        "longitude": lon,
        "current": "temperature_2m,weather_code",
    }
    cache_key = f"{lat:.4f},{lon:.4f}"
    if cache_key in _weather_cache:
        return _weather_cache[cache_key]
    try:
        with httpx.Client(timeout=8.0) as client:
            r = client.get(url, params=params)
            r.raise_for_status()
            data = r.json()
        current = data.get("current") or {}
        temp = current.get("temperature_2m")
        code = current.get("weather_code", 0)
        result = {"temp": temp, "conditions": _weather_code_to_conditions(code)}
        _weather_cache[cache_key] = result
        return result
    except Exception:
        return None


@router.get("")
def list_panels():
    """Panel keys the client can request. Used for discovery and cycling."""
    return {
        "panels": ["world-clock", "weather", "global-situation-map"],
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
    """Weather Watch panel: Open-Meteo current conditions for configured cities."""
    locations: list[dict[str, str]] = []
    for name, lat, lon in WEATHER_LOCATIONS:
        one = _fetch_one_weather(lat, lon)
        if one is not None:
            temp = one["temp"]
            locations.append({
                "name": name,
                "temp": str(int(round(temp))) if temp is not None else "—",
                "conditions": one["conditions"],
            })
        else:
            locations.append({"name": name, "temp": "—", "conditions": "No data"})
    return {
        "status": "ok" if locations else "partial",
        "locations": locations,
        "message": "" if locations else "Open-Meteo unavailable.",
    }


def _build_global_situation_map() -> dict[str, Any]:
    """Build Global Situation Map from stub/aggregated data. No paid APIs."""
    cache_key = "data"
    if cache_key in _gsm_cache:
        return _gsm_cache[cache_key]
    # Stub structure: regions with severity and event types (spec 2.3 / 3.5).
    # Real pipeline would aggregate conflict/news feeds; for now return defined shape.
    regions = [
        {"name": "Europe", "severity": "monitoring", "events": ["diplomacy", "trade"]},
        {"name": "Middle East", "severity": "elevated", "events": ["conflict", "hotspot"]},
        {"name": "Asia-Pacific", "severity": "monitoring", "events": ["trade", "military"]},
        {"name": "Americas", "severity": "normal", "events": ["economy"]},
        {"name": "Africa", "severity": "elevated", "events": ["conflict", "disaster"]},
    ]
    out = {"status": "ok", "source": "stub", "regions": regions}
    _gsm_cache[cache_key] = out
    return out


@router.get("/global-situation-map")
def global_situation_map():
    """Text-based Global Situation Map: regions/countries with severity and event-type labels."""
    try:
        return _build_global_situation_map()
    except Exception:
        return {
            "status": "error",
            "source": "stub",
            "regions": [],
            "message": "Data temporarily unavailable.",
        }
