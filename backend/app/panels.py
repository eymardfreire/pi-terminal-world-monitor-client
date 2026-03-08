"""
Panel endpoints: World Clock, Weather (Open-Meteo), Global Situation Map.
Uses cache-first for external calls; placeholders when upstream fails.
"""
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime, timezone
from typing import Any

import httpx
from cachetools import TTLCache
from fastapi import APIRouter

# Parallel fetches for weather (avoids client timeout when many cities)
_weather_executor = ThreadPoolExecutor(max_workers=12)

router = APIRouter(prefix="/panels", tags=["panels"])

# In-memory cache for external APIs (TTL seconds, max entries)
_weather_cache: TTLCache[str, dict[str, Any]] = TTLCache(maxsize=128, ttl=600)  # 10 min
_gsm_cache: TTLCache[str, dict[str, Any]] = TTLCache(maxsize=4, ttl=300)  # 5 min
_crypto_cache: TTLCache[str, dict[str, Any]] = TTLCache(maxsize=8, ttl=90)  # 90s for crypto (CoinGecko rate limit)

# Top cities per continent for Weather Watch: continent -> [(name, lat, lon), ...]
# Order defines cycle: North America → Central America → ... → Oceania → repeat
WEATHER_BY_CONTINENT: dict[str, list[tuple[str, float, float]]] = {
    "North America": [
        ("New York", 40.7128, -74.0060),
        ("Los Angeles", 34.0522, -118.2437),
        ("Chicago", 41.8781, -87.6298),
        ("Toronto", 43.6532, -79.3832),
        ("Miami", 25.7617, -80.1918),
        ("Vancouver", 49.2827, -123.1207),
    ],
    "Central America": [
        ("Mexico City", 19.4326, -99.1332),
        ("Guatemala City", 14.6349, -90.5069),
        ("Havana", 23.1136, -82.3666),
        ("San José", 9.9281, -84.0907),
        ("Panama City", 8.9824, -79.5199),
    ],
    "South America": [
        ("São Paulo", -23.5505, -46.6333),
        ("Buenos Aires", -34.6037, -58.3816),
        ("Lima", -12.0464, -77.0428),
        ("Bogotá", 4.7110, -74.0721),
        ("Santiago", -33.4489, -70.6693),
        ("Caracas", 10.4806, -66.9036),
    ],
    "Europe": [
        ("London", 51.5074, -0.1278),
        ("Berlin", 52.5200, 13.4050),
        ("Paris", 48.8566, 2.3522),
        ("Madrid", 40.4168, -3.7038),
        ("Rome", 41.9028, 12.4964),
        ("Amsterdam", 52.3676, 4.9041),
    ],
    "Africa": [
        ("Cairo", 30.0444, 31.2357),
        ("Lagos", 6.5244, 3.3792),
        ("Johannesburg", -26.2041, 28.0473),
        ("Nairobi", -1.2921, 36.8219),
        ("Casablanca", 33.5731, -7.5898),
        ("Accra", 5.6037, -0.1870),
    ],
    "Middle East": [
        ("Dubai", 25.2048, 55.2708),
        ("Tel Aviv", 32.0853, 34.7818),
        ("Riyadh", 24.7136, 46.6753),
        ("Istanbul", 41.0082, 28.9784),
        ("Tehran", 35.6892, 51.3890),
        ("Doha", 25.2854, 51.5310),
    ],
    "Asia": [
        ("Tokyo", 35.6762, 139.6503),
        ("Beijing", 39.9042, 116.4074),
        ("Mumbai", 19.0760, 72.8777),
        ("Singapore", 1.3521, 103.8198),
        ("Seoul", 37.5665, 126.9780),
        ("Bangkok", 13.7563, 100.5018),
    ],
    "Oceania": [
        ("Sydney", -33.8688, 151.2093),
        ("Melbourne", -37.8136, 144.9631),
        ("Auckland", -36.8509, 174.7645),
        ("Brisbane", -27.4698, 153.0251),
        ("Perth", -31.9505, 115.8605),
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
    """Panel keys the client can request. First slot = Crypto (top-left), then Weather, GSM, World Clock."""
    return {
        "panels": ["crypto", "weather", "global-situation-map", "world-clock"],
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


def _weather_location_entry(name: str, lat: float, lon: float) -> dict[str, Any]:
    """Build one location dict; used for parallel fetch."""
    one = _fetch_one_weather(lat, lon)
    if one is not None:
        return {
            "name": name,
            "temp": _round_temp(one.get("temp")),
            "temp_high": _round_temp(one.get("temp_high")),
            "temp_low": _round_temp(one.get("temp_low")),
            "conditions": one["conditions"],
            "weather_code": one.get("weather_code", 0),
        }
    return {
        "name": name,
        "temp": "—",
        "temp_high": "—",
        "temp_low": "—",
        "conditions": "No data",
        "weather_code": 0,
    }


@router.get("/weather")
def weather():
    """Weather Watch panel: Open-Meteo current + daily high/low by continent. Fetches in parallel to avoid timeout."""
    # Flatten to (continent, name, lat, lon) preserving order; fetch in parallel
    tasks: list[tuple[str, str, float, float]] = []
    for continent, cities in WEATHER_BY_CONTINENT.items():
        for name, lat, lon in cities:
            tasks.append((continent, name, lat, lon))
    results: list[tuple[str, dict[str, Any]]] = []
    futures = {_weather_executor.submit(_weather_location_entry, n, la, lo): (cont, n, la, lo) for cont, n, la, lo in tasks}
    for fut in as_completed(futures):
        cont, name, _lat, _lon = futures[fut]
        try:
            entry = fut.result()
            results.append((cont, entry))
        except Exception:
            results.append((cont, {"name": name, "temp": "—", "temp_high": "—", "temp_low": "—", "conditions": "No data", "weather_code": 0}))
    # Reassemble by continent in original order
    order = list(WEATHER_BY_CONTINENT.keys())
    by_continent: dict[str, list[dict[str, Any]]] = {c: [] for c in order}
    for cont, entry in results:
        by_continent[cont].append(entry)
    # Preserve city order within each continent (results are unordered; re-sort by task order)
    continents_out = []
    for continent in order:
        cities = WEATHER_BY_CONTINENT[continent]
        names_order = [c[0] for c in cities]
        locs = by_continent[continent]
        locs_sorted = sorted(locs, key=lambda x: names_order.index(x["name"]) if x["name"] in names_order else 999)
        continents_out.append({"name": continent, "locations": locs_sorted})
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


# --- Crypto panel (CoinGecko free API; no key) ---

COINGECKO_BASE = "https://api.coingecko.com/api/v3"
STABLECOIN_IDS = "tether,usd-coin,binance-usd,dai,ethena-usde,frax,frax-share,tusd"


def _fetch_coingecko_markets(page: int = 1, per_page: int = 24, ids: str | None = None) -> list[dict[str, Any]]:
    cache_key = f"markets_p{page}_n{per_page}_{ids or 'all'}"
    if cache_key in _crypto_cache:
        return _crypto_cache[cache_key]
    try:
        params: dict[str, Any] = {
            "vs_currency": "usd",
            "order": "market_cap_desc",
            "per_page": per_page,
            "page": page,
        }
        if ids:
            params["ids"] = ids
        with httpx.Client(timeout=10.0) as client:
            r = client.get(f"{COINGECKO_BASE}/coins/markets", params=params)
            r.raise_for_status()
            data = r.json()
        _crypto_cache[cache_key] = data
        return data
    except Exception:
        return []


@router.get("/crypto/top")
def crypto_top(range_start: int = 1):
    """Top cryptos by market cap. range_start=1 returns ranks 1-12, range_start=13 returns 13-24. Price, 24h%, 7d% (if available)."""
    page = 1 if range_start <= 12 else 2
    per_page = 12
    raw = _fetch_coingecko_markets(page=page, per_page=per_page)
    coins = []
    for i, c in enumerate(raw):
        rank = (page - 1) * per_page + i + 1
        price = c.get("current_price")
        p24 = c.get("price_change_percentage_24h")
        # 7d not in markets endpoint; use 24h for now or leave null
        p7d = c.get("price_change_percentage_7d_in_currency")
        coins.append({
            "rank": rank,
            "id": c.get("id"),
            "symbol": (c.get("symbol") or "").upper(),
            "name": c.get("name", ""),
            "price": round(price, 4) if price is not None else None,
            "price_24h_pct": round(p24, 2) if p24 is not None else None,
            "price_7d_pct": round(p7d, 2) if p7d is not None else None,
        })
    return {"status": "ok", "source": "coingecko", "range": f"{range_start}-{range_start + len(coins) - 1}", "coins": coins}


@router.get("/crypto/stablecoins")
def crypto_stablecoins():
    """Stablecoins: status (healthy if all on peg), market cap, volume, per-coin peg health."""
    raw = _fetch_coingecko_markets(per_page=20, ids=STABLECOIN_IDS)
    if not raw:
        return {
            "status": "ok",
            "source": "coingecko",
            "status_label": "No data",
            "market_cap_b": None,
            "volume_b": None,
            "coins": [],
        }
    total_mcap = sum(c.get("market_cap") or 0 for c in raw)
    total_vol = sum(c.get("total_volume") or 0 for c in raw)
    coins_out = []
    all_on_peg = True
    for c in raw:
        price = c.get("current_price") or 0
        dev = abs(price - 1.0) * 100
        on_peg = dev <= 0.5
        if not on_peg:
            all_on_peg = False
        coins_out.append({
            "symbol": (c.get("symbol") or "").upper(),
            "name": c.get("name", ""),
            "price": round(price, 4),
            "peg_status": "ON PEG" if on_peg else "OFF PEG",
            "deviation_pct": round(dev, 2),
        })
    return {
        "status": "ok",
        "source": "coingecko",
        "status_label": "Healthy" if all_on_peg else "Caution",
        "market_cap_b": round(total_mcap / 1e9, 1),
        "volume_b": round(total_vol / 1e9, 1),
        "coins": coins_out,
    }


@router.get("/crypto/btc-etf")
def crypto_btc_etf():
    """BTC ETF Tracker: placeholder for World Monitor–style stats (flows, AUM, etc.). Next agent: wire real source."""
    return {
        "status": "ok",
        "source": "stub",
        "message": "BTC ETF data TBD – see docs/HANDOFF-PROGRESS.md",
        "etfs": [],
        "total_flows_24h": None,
        "total_aum": None,
    }
