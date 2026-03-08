## ADDED Requirements

### Requirement: Panel-based layout
The terminal client SHALL display data in discrete panels that act as containers for organized data. Layout and arrangement are implementation-defined but SHALL be clean and readable on a terminal (e.g. Pi 3B).

#### Scenario: Panels visible during normal operation
- **WHEN** the client is running and connected to the backend
- **THEN** at least one panel is visible at a time with clear boundaries (e.g. title, content area)
- **AND** content is readable at typical terminal sizes (e.g. 80x24 or 1280x720 pixel equivalent)

### Requirement: Panel cycling
The terminal client SHALL cycle through the set of data panels (or panel groups) on a configurable timer so that over time all data categories are shown despite limited screen space.

#### Scenario: Cycle advances to next panel
- **WHEN** the configured duration for the current panel elapses
- **THEN** the client switches to the next panel in the sequence
- **AND** the sequence loops so that every category is shown repeatedly

#### Scenario: Full set of categories represented
- **WHEN** cycling is enabled
- **THEN** the rotation includes all requested panel categories (Strategic Risk, Intel, Live Intelligence, Work News, continental panels, Predictions, Energy and Resources, Think Tanks, Commodities, Markets, Economy Indicators, Trade Policy, Supply Chain, Financial News, Technology, Crypto, Fires, Market Radar, BTC ETF Tracker, Stablecoins, Armed Conflict Events, Global Giving, Climate Anomalies, Weather Watch, World Clock)
- **AND** Global Situation Map can be included as one of the cycled views

### Requirement: Color coding
The terminal client SHALL apply color coding across panels to indicate severity (e.g. critical, elevated, monitoring, normal), trend (e.g. up/down for markets), and category where useful. Colors SHALL be consistent across panels.

#### Scenario: Severity visible in panel content
- **WHEN** the backend returns items with severity or status
- **THEN** the client renders them with distinct colors (e.g. red/orange/yellow/green or equivalent)
- **AND** the same severity level uses the same color in different panels

#### Scenario: Trend visible for numeric data
- **WHEN** panel data includes numeric changes (e.g. price, percentage)
- **THEN** positive and negative trends are distinguished by color (e.g. green/red or equivalent)

### Requirement: Backend-only data source
The terminal client SHALL obtain all dashboard data from the backend API only. It SHALL NOT call third-party APIs or RSS feeds directly.

#### Scenario: No direct external calls from client
- **WHEN** the client needs data for any panel
- **THEN** it requests that data from the backend only
- **AND** no API keys or third-party URLs are required in the client configuration

### Requirement: Weather Watch panel
The terminal client SHALL include a Weather Watch panel that displays weather data (e.g. cities, conditions, temperatures) in a form similar to PatchNotesLive's weather panel, with color coding where appropriate.

#### Scenario: Weather data displayed
- **WHEN** the Weather Watch panel is active
- **THEN** it shows at least city names and current conditions (and optionally temps)
- **AND** data is sourced from the backend, which may use e.g. Open-Meteo or equivalent free API

### Requirement: World Clock panel
The terminal client SHALL include a World Clock panel that displays current time (and optionally multiple time zones). It MAY be combined with other content (e.g. in a header/footer) if the layout supports it.

#### Scenario: Time visible when World Clock is shown
- **WHEN** the World Clock panel (or slot) is visible
- **THEN** at least one current time is displayed (e.g. UTC or local)
- **AND** the source of time is the backend or system time; no direct NTP required from client if backend provides time
