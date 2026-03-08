## ADDED Requirements

### Requirement: Panel data API
The backend SHALL expose a REST (or equivalent) API that returns pre-aggregated data for each dashboard panel category, so the terminal client can fetch data without calling third-party APIs.

#### Scenario: Client fetches panel data
- **WHEN** the client requests data for a given panel (e.g. strategic risk, intel, a regional news panel, commodities, world clock)
- **THEN** the backend returns a JSON (or agreed) payload conforming to a defined schema for that panel
- **AND** data is derived from backend-side aggregation only (RSS and free APIs), not from the client

#### Scenario: Graceful degradation when upstream fails
- **WHEN** an upstream source (RSS or free API) is unavailable or rate-limited
- **THEN** the backend returns cached or placeholder data when available
- **AND** response includes a clear status or label so the client can show e.g. "DELAYED" or "CACHE"

### Requirement: Cache-first and rate-limit friendly
The backend SHALL use a cache-first strategy for all external requests and SHALL respect rate limits and backoff for free-tier APIs and RSS feeds.

#### Scenario: Repeated requests for same panel
- **WHEN** the client (or any caller) requests the same panel data within the cache TTL
- **THEN** the backend serves from cache when possible
- **AND** external calls are not repeated beyond the backend's refresh policy

#### Scenario: Upstream rate limit
- **WHEN** an upstream returns rate-limit or backoff response
- **THEN** the backend backs off and uses last-known data
- **AND** does not expose API keys or credentials to the client

### Requirement: Data categories for all requested panels
The backend SHALL provide data for the following panel categories (each may be one endpoint or part of a grouped response): Strategic Risk Overview, Intel Feed, Live Intelligence, Work News, one panel per continent (e.g. Africa, Americas, Asia-Pacific, Europe, Latin America, Middle East), Predictions, Energy and Resources, Think Tanks, Commodities, Markets, Economy Indicators, Trade Policy, Supply Chain, Financial News, Technology, Crypto, Fires, Market Radar, BTC ETF Tracker, Stablecoins, Armed Conflict Events, Global Giving, Climate Anomalies, Weather Watch, World Clock.

#### Scenario: At least one source per category
- **WHEN** a panel category is requested
- **THEN** the backend returns data from at least one free API or RSS source, or a documented placeholder with status
- **AND** response shape is consistent so the client can render the panel

### Requirement: Global Situation Map data
The backend SHALL provide a dedicated data structure for the text-based Global Situation Map: regions or countries with severity level and event-type labels (e.g. conflict, hotspot, military, disaster), suitable for text rendering without a graphical map.

#### Scenario: Client fetches global situation
- **WHEN** the client requests the Global Situation Map
- **THEN** the backend returns a structured list (or tree) of regions/countries with severity and event-type fields
- **AND** the structure is sufficient for the client to produce a readable text summary (e.g. grouped by region)

### Requirement: Free APIs and RSS only
The backend SHALL use only free-tier or no-key APIs and RSS feeds for data collection. No paid or key-required sources SHALL be required for core functionality.

#### Scenario: No paid dependency
- **WHEN** the backend is deployed with only free-tier or public configuration
- **THEN** all panel endpoints return data or a clear "no data"/placeholder state
- **AND** no feature requires a paid API key to function
