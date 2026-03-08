## ADDED Requirements

### Requirement: Text-based Global Situation Map
The system SHALL provide a readable text-based representation of the global situation (regions/countries with alert level and key event types), without requiring a graphical map. The backend SHALL supply the data; the client SHALL render it as text.

#### Scenario: Backend provides structured situation data
- **WHEN** the client requests the Global Situation Map
- **THEN** the backend returns a structure that includes regions or countries with at least severity/alert level and event-type labels (e.g. conflict, hotspot, military, disaster)
- **AND** the structure is suitable for text-only display (e.g. list or grouped by region)

#### Scenario: Client renders as readable text
- **WHEN** the client displays the Global Situation Map view
- **THEN** it shows a text summary (e.g. region name, severity, and event types) that a user can read without a map
- **AND** severity levels are color-coded consistently with the rest of the dashboard

#### Scenario: Map view fits panel cycling
- **WHEN** the Global Situation Map is included in the panel rotation
- **THEN** it is shown for the same configurable duration as other panels
- **AND** it uses the same panel container and color scheme as other panels
