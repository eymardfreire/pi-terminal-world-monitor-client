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
_weather_cache: TTLCache[str, dict[str, Any]] = TTLCache(maxsize=128, ttl=600)  # 10 min
_gsm_cache: TTLCache[str, dict[str, Any]] = TTLCache(maxsize=4, ttl=300)  # 5 min

# Top cities per continent for Weather Watch: continent -> [(name, lat, lon), ...]
WEATHER_BY_CONTINENT: dict[str, list[tuple[str, float, float]]] = {
    "North America": [
        ("New York", 40.7128, -74.0060),
        ("Los Angeles", 34.0522, -118.2437),
        ("Chicago", 41.8781, -87.6298),
        ("Toronto", 43.6532, -79.3832),
    ],
    "Central America": [
        ("Mexico City", 19.4326, -99.1332),
        ("Guatemala City", 14.6349, -90.5069),
        ("Havana", 23.1136, -82.3666),
    ],
    "South America": [
        ("São Paulo", -23.5505, -46.6333),
        ("Buenos Aires", -34.6037, -58.3816),
        ("Lima", -12.0464, -77.0428),
        ("Bogotá", 4.7110, -74.0721),
    ],
    "Europe": [
        ("London", 51.5074, -0.1278),
        ("Berlin", 52.5200, 13.4050),
        ("Paris", 48.8566, 2.3522),
        ("Madrid", 40.4168, -3.7038),
    ],
    "Africa": [
        ("Cairo", 30.0444, 31.2357),
        ("Lagos", 6.5244, 3.3792),
        ("Johannesburg", -26.2041, 28.0473),
        ("Nairobi", -1.2921, 36.8219),
    ],
    "Middle East": [
        ("Dubai", 25.2048, 55.2708),
        ("Tel Aviv", 32.0853, 34.7818),
        ("Riyadh", 24.7136, 46.6753),
        ("Istanbul", 41.0082, 28.9784),
    ],
    "Asia": [
        ("Tokyo", 35.6762, 139.6503),
        ("Beijing", 39.9042, 116.4074),
        ("Mumbai", 19.0760, 72.8777),
        ("Singapore", 1.3521, 103.8198),
    ],
    "Oceania": [
        ("Sydney", -33.8688, 151.2093),
        ("Melbourne", -37.8136, 144.9631),
        ("Auckland", -36.8509, 174.7645),
    ],
}

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
        "daily": "temperature_2m_max,temperature_2m_min",
        "timezone": "auto",
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
        daily = data.get("daily") or {}
        temp = current.get("temperature_2m")
        code = current.get("weather_code", 0)
        maxes = daily.get("temperature_2m_max") or []
        mins = daily.get("temperature_2m_min") or []
        temp_high = maxes[0] if maxes else None
        temp_low = mins[0] if mins else None
        result = {
            "temp": temp,
            "temp_high": temp_high,
            "temp_low": temp_low,
            "conditions": _weather_code_to_conditions(code),
            "weather_code": code,
        }
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


def _round_temp(t: float | None) -> str:
    if t is None:
        return "—"
    return str(int(round(t)))


@router.get("/weather")
def weather():
    """Weather Watch panel: Open-Meteo current + daily high/low by continent. Client can cycle continents."""
    continents_out: list[dict[str, Any]] = []
    for continent, cities in WEATHER_BY_CONTINENT.items():
        locations: list[dict[str, Any]] = []
        for name, lat, lon in cities:
            one = _fetch_one_weather(lat, lon)
            if one is not None:
                locations.append({
                    "name": name,
                    "temp": _round_temp(one.get("temp")),
                    "temp_high": _round_temp(one.get("temp_high")),
                    "temp_low": _round_temp(one.get("temp_low")),
                    "conditions": one["conditions"],
                    "weather_code": one.get("weather_code", 0),
                })
            else:
                locations.append({
                    "name": name,
                    "temp": "—",
                    "temp_high": "—",
                    "temp_low": "—",
                    "conditions": "No data",
                    "weather_code": 0,
                })
        continents_out.append({"name": continent, "locations": locations})
    return {
        "status": "ok",
        "continents": continents_out,
        "message": "",
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
