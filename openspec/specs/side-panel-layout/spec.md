# side-panel-layout Specification

## Purpose
TBD - created by archiving change side-panel-detail-layout. Update Purpose after archive.
## Requirements
### Requirement: Horizontal split layout
The TUI SHALL display a narrow left panel and a wide right panel side by side, with an optional log panel spanning the full width below.

#### Scenario: Normal terminal width
- **WHEN** the terminal width is 60 columns or more
- **THEN** the left panel SHALL occupy approximately 30% of width (min 20, max 40 columns) and the right panel SHALL occupy the remaining width

#### Scenario: Narrow terminal
- **WHEN** the terminal width is less than 60 columns
- **THEN** the TUI SHALL display a message indicating the terminal is too small

### Requirement: Project names as basenames
The left panel SHALL display project directory basenames instead of full paths.

#### Scenario: Unique basenames
- **WHEN** all detected projects have unique directory basenames
- **THEN** the left panel SHALL show only the basename (e.g. `specgetty` instead of `/home/pim/cVibeCoding/specgetty`)

#### Scenario: Duplicate basenames
- **WHEN** two or more projects share the same directory basename
- **THEN** the left panel SHALL append the parent directory name to disambiguate (e.g. `specgetty (cVibeCoding)` and `specgetty (cForks)`)

### Requirement: Detail panel shows project info and file listing
The detail panel SHALL display a persistent header with the project path and stats above the tab bar, visible regardless of which tab is active.

#### Scenario: Header always visible
- **WHEN** any tab is active in the detail panel
- **THEN** the project path and stats line SHALL be visible above the tab bar

#### Scenario: Stats line content
- **WHEN** a project is selected
- **THEN** the stats line SHALL show spec count, active changes count, and archived changes count

#### Scenario: No project selected
- **WHEN** no projects are found
- **THEN** the right panel SHALL display "No project selected."

### Requirement: Navigation works across both panels
Tab switching and cursor navigation SHALL work in the horizontal layout.

#### Scenario: Tab between panels
- **WHEN** the user presses tab
- **THEN** focus SHALL cycle between the left panel and the right panel (and log panel if visible)

#### Scenario: Cursor navigation in left panel
- **WHEN** the left panel is focused and user presses j/k
- **THEN** the project cursor SHALL move and the right panel SHALL update to show the newly selected project's contents

