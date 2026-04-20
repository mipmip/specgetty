## MODIFIED Requirements

### Requirement: Detail panel shows project info and file listing
The detail panel SHALL display a persistent header with the project path and stats above the tab bar, visible regardless of which tab is active.

#### Scenario: Header always visible
- **WHEN** any tab is active in the detail panel
- **THEN** the project path and stats line SHALL be visible above the tab bar

#### Scenario: Stats line content
- **WHEN** a project is selected
- **THEN** the stats line SHALL show spec count, active changes count, and archived changes count

## REMOVED Requirements

### Requirement: Overview tab
**Reason**: Overview information moved to persistent header. Active changes and archived lists available via their own tabs.
**Migration**: Stats visible in header. Change/archive lists accessible via changes and archive tabs.
