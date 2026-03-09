"""
Panel endpoints: World Clock, Weather (Open-Meteo), Global Situation Map, Crypto, News.
Uses cache-first for external calls; placeholders when upstream fails.
"""
import re
import time
import xml.etree.ElementTree as ET
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime, timedelta, timezone
from email.utils import parsedate_to_datetime
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
_crypto_news_cache: TTLCache[str, dict[str, Any]] = TTLCache(maxsize=4, ttl=300)  # 5 min for crypto news RSS
_news_cache: TTLCache[str, dict[str, Any]] = TTLCache(maxsize=2, ttl=300)  # 5 min for news feeds

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
    """Panel keys the client can request. First slot = Crypto (top-left), then Weather, News, World Clock."""
    return {
        "panels": ["crypto", "weather", "news", "world-clock"],
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


# Layer definitions for text-based map (id, display name, icon). Order matches UI.
GSM_LAYER_DEFS: list[dict[str, str]] = [
    {"id": "iran-attacks", "name": "Iran Attacks", "icon": "🎯"},
    {"id": "intel-hotspots", "name": "Intel Hotspots", "icon": "🎯"},
    {"id": "conflict-zones", "name": "Conflict Zones", "icon": "⚔"},
    {"id": "military-bases", "name": "Military Bases", "icon": "🏛"},
    {"id": "nuclear-sites", "name": "Nuclear Sites", "icon": "☢"},
    {"id": "gamma-irradiators", "name": "Gamma Irradiators", "icon": "⚠"},
    {"id": "spaceports", "name": "Spaceports", "icon": "🚀"},
    {"id": "undersea-cables", "name": "Undersea Cables", "icon": "🔌"},
    {"id": "pipelines", "name": "Pipelines", "icon": "🛢"},
    {"id": "ai-datacenters", "name": "AI Data Centers", "icon": "🖥"},
    {"id": "military-activity", "name": "Military Activity", "icon": "✈"},
    {"id": "ship-traffic", "name": "Ship Traffic", "icon": "🚢"},
    {"id": "trade-routes", "name": "Trade Routes", "icon": "⚓"},
    {"id": "aviation", "name": "Aviation", "icon": "✈"},
    {"id": "protests", "name": "Protests", "icon": "📢"},
    {"id": "armed-conflict-events", "name": "Armed Conflict Events", "icon": "⚔"},
    {"id": "displacement-flows", "name": "Displacement Flows", "icon": "👥"},
    {"id": "climate-anomalies", "name": "Climate Anomalies", "icon": "🌫"},
    {"id": "weather-alerts", "name": "Weather Alerts", "icon": "⛈"},
    {"id": "internet-outages", "name": "Internet Outages", "icon": "📡"},
    {"id": "cyber-threats", "name": "Cyber Threats", "icon": "🛡"},
    {"id": "natural-events", "name": "Natural Events", "icon": "🌋"},
    {"id": "fires", "name": "Fires", "icon": "🔥"},
    {"id": "strategic-waterways", "name": "Strategic Waterways", "icon": "⚓"},
    {"id": "economic-centers", "name": "Economic Centers", "icon": "💰"},
    {"id": "critical-minerals", "name": "Critical Minerals", "icon": "💎"},
    {"id": "gps-jamming", "name": "GPS JAMMING", "icon": "📡"},
    {"id": "cii-instability", "name": "CII Instability", "icon": "🌎"},
    {"id": "day-night", "name": "Day/Night", "icon": "🌓"},
]


def _build_global_situation_map() -> dict[str, Any]:
    """Build Global Situation Map from stub/aggregated data. No paid APIs.
    Returns structure for text translation: header (defcon, time_window), alerts by level,
    layers with locations, and regions with severity/events.
    """
    cache_key = "data"
    if cache_key in _gsm_cache:
        return _gsm_cache[cache_key]
    now = datetime.now(timezone.utc)
    # Stub: mirror map semantics — high/elevated/monitoring by location; layers with locations.
    summary = {
        "high": ["Ukraine", "Iran", "Sudan", "Myanmar"],
        "elevated": ["Iraq", "Syria", "Yemen", "Nigeria", "Ethiopia", "Pakistan", "Philippines"],
        "monitoring": ["UK", "France", "Germany", "Poland", "India", "China coast", "USA East", "USA West", "Venezuela", "Colombia", "Kenya", "South Africa"],
    }
    # Layers that have data in stub (active on map). Each: id, name, icon, active, locations.
    layers_with_data = [
        ("conflict-zones", ["Ukraine", "Sudan", "Myanmar", "Syria", "Iraq"]),
        ("intel-hotspots", ["Iran", "Ukraine", "Middle East", "East Asia"]),
        ("iran-attacks", ["Iran", "Iraq", "Israel"]),
        ("military-bases", ["Europe", "Middle East", "USA East", "USA West", "Japan"]),
        ("nuclear-sites", ["Iran", "Middle East", "North Korea"]),
        ("gamma-irradiators", ["Europe", "USA", "Asia"]),
        ("military-activity", ["Ukraine", "Middle East", "South China Sea"]),
    ]
    layers_out = []
    defs_by_id = {d["id"]: d for d in GSM_LAYER_DEFS}
    for layer_id, locations in layers_with_data:
        d = defs_by_id.get(layer_id, {"id": layer_id, "name": layer_id.replace("-", " ").title(), "icon": "•"})
        layers_out.append({
            "id": layer_id,
            "name": d["name"],
            "icon": d["icon"],
            "active": True,
            "locations": locations,
        })
    # Regions (by geography) with severity and event-type labels.
    regions = [
        {"name": "Europe", "severity": "monitoring", "events": ["conflict", "bases", "trade"]},
        {"name": "Middle East", "severity": "elevated", "events": ["conflict", "hotspot", "nuclear", "military"]},
        {"name": "Asia-Pacific", "severity": "monitoring", "events": ["conflict", "military", "trade"]},
        {"name": "Americas", "severity": "normal", "events": ["bases", "economy"]},
        {"name": "Africa", "severity": "elevated", "events": ["conflict", "disaster", "hotspot"]},
    ]
    out = {
        "status": "ok",
        "source": "stub",
        "defcon": 2,
        "defcon_pct": 44,
        "time_window": "7d",
        "updated_utc": now.strftime("%Y-%m-%dT%H:%M:%SZ"),
        "summary": summary,
        "layers": layers_out,
        "regions": regions,
    }
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
STABLECOIN_IDS = "tether,usd-coin,binance-usd,dai,ethena-usde,frax,frax-share,tusd,first-digital-usd"


def _fetch_coingecko_markets(
    page: int = 1, per_page: int = 24, ids: str | None = None, price_change: str = "1h,24h,7d"
) -> list[dict[str, Any]]:
    cache_key = f"markets_p{page}_n{per_page}_{ids or 'all'}_{price_change}"
    if cache_key in _crypto_cache:
        return _crypto_cache[cache_key]

    def _do_fetch() -> list[dict[str, Any]]:
        params: dict[str, Any] = {
            "vs_currency": "usd",
            "order": "market_cap_desc",
            "per_page": per_page,
            "page": page,
            "price_change_percentage": price_change,
        }
        if ids:
            params["ids"] = ids
        with httpx.Client(timeout=10.0) as client:
            r = client.get(f"{COINGECKO_BASE}/coins/markets", params=params)
            r.raise_for_status()
            data = r.json()
        return data if isinstance(data, list) else []

    for attempt in range(2):
        if attempt == 1:
            time.sleep(1)
        try:
            data = _do_fetch()
            if data:
                _crypto_cache[cache_key] = data
            return data
        except Exception:
            if attempt == 0:
                continue
            return []
    return []


# Single fetch of top 56 so client can cycle through more pages; one API call to avoid rate limits
_TOP56_CACHE_KEY = "top56_1h_24h_7d"
TOP_COINS_COUNT = 56


def _fetch_top56_coins() -> list[dict[str, Any]]:
    if _TOP56_CACHE_KEY in _crypto_cache:
        return _crypto_cache[_TOP56_CACHE_KEY]
    for attempt in range(2):
        if attempt == 1:
            time.sleep(1)
        try:
            raw = _fetch_coingecko_markets(page=1, per_page=TOP_COINS_COUNT, price_change="1h,24h,7d")
            if raw:
                _crypto_cache[_TOP56_CACHE_KEY] = raw
                return raw
        except Exception:
            if attempt == 0:
                continue
            return []
    return []


@router.get("/crypto/top")
def crypto_top(range_start: int = 1, per_page: int = 11):
    """Top 56 cryptos by market cap. per_page controls slice size (for resolution-aware clients); range_start is 1-based start index."""
    raw = _fetch_top56_coins()
    if not raw:
        return {"status": "ok", "source": "coingecko", "range": "1-11", "coins": []}
    per_page = max(5, min(25, per_page))
    start_idx = max(0, range_start - 1)
    end_idx = min(start_idx + per_page, TOP_COINS_COUNT)
    slice_raw = raw[start_idx:end_idx]
    coins = []
    for i, c in enumerate(slice_raw):
        rank = start_idx + i + 1
        price = c.get("current_price")
        p1h = c.get("price_change_percentage_1h_in_currency") or c.get("price_change_percentage_1h")
        p24 = c.get("price_change_percentage_24h")
        p7d = c.get("price_change_percentage_7d_in_currency") or c.get("price_change_percentage_7d")
        coins.append({
            "rank": rank,
            "id": c.get("id"),
            "symbol": (c.get("symbol") or "").upper(),
            "name": c.get("name", ""),
            "price": round(price, 4) if price is not None else None,
            "price_1h_pct": round(p1h, 2) if p1h is not None else None,
            "price_24h_pct": round(p24, 2) if p24 is not None else None,
            "price_7d_pct": round(p7d, 2) if p7d is not None else None,
        })
    end = start_idx + len(coins)
    return {"status": "ok", "source": "coingecko", "range": f"{start_idx + 1}-{end}", "coins": coins}


@router.get("/crypto/stablecoins")
def crypto_stablecoins():
    """Stablecoins: status (healthy if all on peg), market cap, volume, per-coin peg health + supply/volume/24h chg."""
    raw = _fetch_coingecko_markets(per_page=20, ids=STABLECOIN_IDS, price_change="24h")
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
        mcap = c.get("market_cap") or 0
        vol = c.get("total_volume") or 0
        p24 = c.get("price_change_percentage_24h")
        coins_out.append({
            "symbol": (c.get("symbol") or "").upper(),
            "name": c.get("name", ""),
            "price": round(price, 4),
            "peg_status": "ON PEG" if on_peg else "OFF PEG",
            "deviation_pct": round(dev, 2),
            "market_cap_b": round(mcap / 1e9, 1) if mcap else None,
            "volume_b": round(vol / 1e9, 3) if vol else None,
            "price_change_24h_pct": round(p24, 2) if p24 is not None else None,
        })
    return {
        "status": "ok",
        "source": "coingecko",
        "status_label": "Healthy" if all_on_peg else "Caution",
        "market_cap_b": round(total_mcap / 1e9, 1),
        "volume_b": round(total_vol / 1e9, 1),
        "coins": coins_out,
    }


GAINERS_LOSERS_COUNT = 28


def _fetch_gainers_losers() -> dict[str, Any]:
    """Top 28 gainers and top 28 losers by 24h price change (exclude stablecoins)."""
    cache_key = "gainers_losers"
    if cache_key in _crypto_cache:
        return _crypto_cache[cache_key]
    # Fetch more than 56 so we have enough after excluding stables
    raw = _fetch_coingecko_markets(per_page=100, price_change="24h")
    if not raw:
        out = {"status": "ok", "source": "coingecko", "gainers": [], "losers": []}
        _crypto_cache[cache_key] = out
        return out
    # Exclude stablecoins (price near 1)
    non_stable = [c for c in raw if c.get("current_price") is not None and abs((c.get("current_price") or 0) - 1.0) > 0.05]
    # Sort by 24h change descending (gainers first)
    non_stable.sort(key=lambda x: (x.get("price_change_percentage_24h") or -1e9), reverse=True)
    gainers = []
    for c in non_stable[:GAINERS_LOSERS_COUNT]:
        p24 = c.get("price_change_percentage_24h")
        gainers.append({
            "symbol": (c.get("symbol") or "").upper(),
            "price": round(c.get("current_price") or 0, 4),
            "change_24h_pct": round(p24, 2) if p24 is not None else None,
        })
    losers = []
    for c in non_stable[-GAINERS_LOSERS_COUNT:]:
        p24 = c.get("price_change_percentage_24h")
        losers.append({
            "symbol": (c.get("symbol") or "").upper(),
            "price": round(c.get("current_price") or 0, 4),
            "change_24h_pct": round(p24, 2) if p24 is not None else None,
        })
    losers.reverse()  # worst first
    out = {"status": "ok", "source": "coingecko", "gainers": gainers, "losers": losers}
    _crypto_cache[cache_key] = out
    return out


@router.get("/crypto/gainers-losers")
def crypto_gainers_losers():
    """Top 28 gainers and top 28 losers by 24h price change (ticker + price)."""
    try:
        return _fetch_gainers_losers()
    except Exception:
        return {"status": "error", "source": "coingecko", "gainers": [], "losers": []}


# Crypto news: public RSS (no API key); no trailing slash to avoid 308 redirect
CRYPTO_NEWS_RSS_URL = "https://www.coindesk.com/arc/outboundfeeds/rss"
CRYPTO_NEWS_MAX_ITEMS = 12


def _fetch_crypto_news() -> dict[str, Any]:
    cache_key = "crypto_news"
    if cache_key in _crypto_news_cache:
        return _crypto_news_cache[cache_key]
    items: list[dict[str, Any]] = []
    try:
        with httpx.Client(timeout=10.0) as client:
            r = client.get(CRYPTO_NEWS_RSS_URL)
            r.raise_for_status()
            root = ET.fromstring(r.text)
        # RSS 2.0: channel -> item; handle default ns; use itertext() so CDATA is included
        def text_of(el: ET.Element) -> str:
            parts = list(el.itertext()) if el.itertext() else []
            if parts:
                return "".join(parts).strip()
            return (el.text or "").strip()

        for tag in root.iter():
            if tag.tag.endswith("item"):
                title = ""
                link = ""
                pub_date = ""
                description = ""
                for child in tag:
                    name = child.tag.split("}")[-1] if "}" in child.tag else child.tag
                    if name == "title":
                        title = text_of(child)
                    elif name == "link":
                        link = text_of(child)
                    elif name == "pubDate":
                        pub_date = text_of(child)
                    elif name == "description":
                        # Feed can have multiple description elements (e.g. description then dc:description); keep first non-empty
                        candidate = text_of(child)
                        if candidate:
                            description = candidate
                    elif name == "encoded":
                        # content:encoded often has full/summary text; use if description empty
                        if not description:
                            candidate = text_of(child)
                            if candidate:
                                description = candidate
                if title:
                    # Strip HTML tags from description for plain-text display
                    if description and "<" in description:
                        description = re.sub(r"<[^>]+>", " ", description)
                        description = " ".join(description.split())
                    items.append({
                        "title": title[:120],
                        "link": link,
                        "pub_date": pub_date,
                        "description": (description[:800] if description else "").strip(),
                    })
                if len(items) >= CRYPTO_NEWS_MAX_ITEMS:
                    break
    except Exception:
        pass
    out = {"status": "ok", "source": "rss", "items": items}
    _crypto_news_cache[cache_key] = out
    return out


@router.get("/crypto/news")
def crypto_news():
    """Crypto news from public RSS (e.g. CoinDesk). Cached 5 min."""
    try:
        return _fetch_crypto_news()
    except Exception:
        return {"status": "error", "source": "rss", "items": [], "message": "News temporarily unavailable."}


# --- News panel: 8 feeds (World, US, Europe, Middle East, Africa, Latin America, Asia-Pacific, Government) ---
NEWS_FEEDS: list[tuple[str, str, str]] = [
    ("world", "World News", "https://feeds.bbci.co.uk/news/world/rss.xml"),
    ("us", "United States", "https://feeds.bbci.co.uk/news/world/us_and_canada/rss.xml"),
    ("europe", "Europe", "https://rss.dw.com/xml/rss-en-eu"),
    ("middle-east", "Middle East", "https://www.aljazeera.com/xml/rss/all.xml"),
    ("africa", "Africa", "https://feeds.bbci.co.uk/news/world/africa/rss.xml"),
    ("latin-america", "Latin America", "https://feeds.bbci.co.uk/news/world/latin_america/rss.xml"),
    ("asia-pacific", "Asia-Pacific", "https://feeds.bbci.co.uk/news/world/asia/rss.xml"),
    ("government", "Government", "https://feeds.bbci.co.uk/news/politics/rss.xml"),
]
NEWS_MAX_ITEMS_PER_FEED = 15
NEWS_NEW_HOURS = 6  # items published in last N hours count as "new" in backlog


def _parse_rss_feed(
    xml_text: str, source_name: str, max_items: int, cutoff_utc: datetime | None
) -> tuple[list[dict[str, Any]], int]:
    """Parse RSS XML; return (items, new_count). new_count = items with pub_date >= cutoff_utc."""
    items: list[dict[str, Any]] = []
    new_count = 0

    def text_of(el: ET.Element) -> str:
        parts = list(el.itertext()) if el.itertext() else []
        if parts:
            return "".join(parts).strip()
        return (el.text or "").strip()

    try:
        root = ET.fromstring(xml_text)
    except Exception:
        return [], 0

    for tag in root.iter():
        if tag.tag.endswith("item"):
            title = ""
            link = ""
            pub_date = ""
            description = ""
            for child in tag:
                name = child.tag.split("}")[-1] if "}" in child.tag else child.tag
                if name == "title":
                    title = text_of(child)
                elif name == "link":
                    link = text_of(child)
                elif name == "pubDate":
                    pub_date = text_of(child)
                elif name == "description":
                    candidate = text_of(child)
                    if candidate:
                        description = candidate
                elif name == "encoded":
                    if not description:
                        candidate = text_of(child)
                        if candidate:
                            description = candidate
            if title:
                if description and "<" in description:
                    description = re.sub(r"<[^>]+>", " ", description)
                    description = " ".join(description.split())
                item = {
                    "title": title[:200],
                    "link": link,
                    "pub_date": pub_date,
                    "description": (description[:800] if description else "").strip(),
                    "source": source_name,
                }
                items.append(item)
                if cutoff_utc and pub_date:
                    try:
                        dt = parsedate_to_datetime(pub_date)
                        if dt.tzinfo is None:
                            dt = dt.replace(tzinfo=timezone.utc)
                        else:
                            dt = dt.astimezone(timezone.utc)
                        if dt >= cutoff_utc:
                            new_count += 1
                    except Exception:
                        pass
                if len(items) >= max_items:
                    break
    return items, new_count


def _fetch_one_news_feed(
    feed_id: str, name: str, url: str, cutoff_utc: datetime | None
) -> dict[str, Any]:
    out: dict[str, Any] = {
        "id": feed_id,
        "name": name,
        "new_count": 0,
        "items": [],
    }
    try:
        with httpx.Client(timeout=12.0) as client:
            r = client.get(url)
            r.raise_for_status()
            items, new_count = _parse_rss_feed(
                r.text, name, NEWS_MAX_ITEMS_PER_FEED, cutoff_utc
            )
            out["items"] = items
            out["new_count"] = new_count
    except Exception:
        pass
    return out


def _fetch_news() -> dict[str, Any]:
    cache_key = "news"
    if cache_key in _news_cache:
        return _news_cache[cache_key]
    now = datetime.now(timezone.utc)
    cutoff = now - timedelta(hours=NEWS_NEW_HOURS)
    feeds_out: list[dict[str, Any]] = []
    with ThreadPoolExecutor(max_workers=8) as executor:
        futures = {
            executor.submit(_fetch_one_news_feed, fid, name, url, cutoff): fid
            for fid, name, url in NEWS_FEEDS
        }
        for fut in as_completed(futures):
            try:
                feeds_out.append(fut.result())
            except Exception:
                pass
    # Keep feed order
    by_id = {f["id"]: f for f in feeds_out}
    feeds_ordered = [by_id[fid] for fid, _n, _u in NEWS_FEEDS if fid in by_id]
    out = {"status": "ok", "source": "rss", "feeds": feeds_ordered}
    _news_cache[cache_key] = out
    return out


@router.get("/news")
def news():
    """Eight news feeds (World, US, Europe, Middle East, Africa, Asia-Pacific, Energy, Government). Each feed has new_count (backlog) and items (title, link, pub_date, description, source). Cached 5 min."""
    try:
        return _fetch_news()
    except Exception:
        return {"status": "error", "source": "rss", "feeds": [], "message": "News temporarily unavailable."}


# BTC ETF Tracker: stub data matching World Monitor layout (Net Flow, Est. Flow, Total Vol, ETFs; table TICKER, ISSUER, EST. FLOW, VOLUME, CHANGE)
# Wire a real source (e.g. Farside/Blockworks) when available; replace _btc_etf_stub().
BTC_ETF_STUB: list[dict[str, Any]] = [
    {"ticker": "IBIT", "issuer": "BlackRock", "est_flow_m": -220.2, "volume_m": 57.0, "change_pct": -4.43},
    {"ticker": "FBTC", "issuer": "Fidelity", "est_flow_m": -34.1, "volume_m": 5.8, "change_pct": -4.42},
    {"ticker": "ARKB", "issuer": "ARK/21Shares", "est_flow_m": -9.8, "volume_m": 4.3, "change_pct": -4.40},
    {"ticker": "BITB", "issuer": "Bitwise", "est_flow_m": -11.3, "volume_m": 3.1, "change_pct": -4.44},
    {"ticker": "GBTC", "issuer": "Grayscale", "est_flow_m": -12.4, "volume_m": 2.3, "change_pct": -4.45},
    {"ticker": "HODL", "issuer": "VanEck", "est_flow_m": -3.3, "volume_m": 1.7, "change_pct": -4.37},
    {"ticker": "BRRR", "issuer": "Valkyrie", "est_flow_m": -0.364, "volume_m": 0.189, "change_pct": -4.43},
    {"ticker": "EZBC", "issuer": "Franklin", "est_flow_m": -0.529, "volume_m": 0.134, "change_pct": -4.49},
    {"ticker": "BTCO", "issuer": "Invesco", "est_flow_m": -0.391, "volume_m": 0.058, "change_pct": -4.47},
    {"ticker": "BTCW", "issuer": "WisdomTree", "est_flow_m": -0.176, "volume_m": 0.024, "change_pct": -4.39},
]


def _btc_etf_stub() -> dict[str, Any]:
    est_total = sum(e["est_flow_m"] for e in BTC_ETF_STUB)
    total_vol = sum(e["volume_m"] for e in BTC_ETF_STUB)
    etfs_down = sum(1 for e in BTC_ETF_STUB if (e.get("change_pct") or 0) < 0)
    etfs_up = len(BTC_ETF_STUB) - etfs_down
    return {
        "status": "ok",
        "source": "stub",
        "net_flow_label": "NET OUTFLOW" if est_total < 0 else "NET INFLOW",
        "est_flow_m": round(abs(est_total), 1),
        "total_vol_m": round(total_vol, 1),
        "etfs_up": etfs_up,
        "etfs_down": etfs_down,
        "etfs": BTC_ETF_STUB,
    }


@router.get("/crypto/btc-etf")
def crypto_btc_etf():
    """BTC ETF Tracker: header (Net Flow, Est. Flow, Total Vol, ETFs) + list of ETFs. Stub data; wire real source when available."""
    return _btc_etf_stub()
