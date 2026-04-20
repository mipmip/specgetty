# export-change Specification

## Purpose
Allows users to export a change (active or archived) as a zip file for attaching to foreign PRs.

## ADDED Requirements

### Requirement: Export keybinding
The system SHALL provide an `e` keybinding that initiates export when the changes tab or archive tab is active and a change is selected.

#### Scenario: Press e on changes tab with a change selected
- **WHEN** the user presses `e` while the changes tab is focused and a change is highlighted
- **THEN** the system SHALL show a confirmation modal for exporting that change

#### Scenario: Press e on archive tab with a change selected
- **WHEN** the user presses `e` while the archive tab is focused and an archived change is highlighted
- **THEN** the system SHALL show a confirmation modal for exporting that archived change

#### Scenario: Press e when no changes exist
- **WHEN** the user presses `e` on the changes or archive tab with no entries
- **THEN** the system SHALL do nothing

#### Scenario: Press e on a different tab
- **WHEN** the user presses `e` while a tab other than changes or archive is active
- **THEN** the system SHALL do nothing

### Requirement: Confirmation modal
The system SHALL require explicit confirmation before creating the zip file.

#### Scenario: Modal shows destination path
- **WHEN** the confirmation modal is displayed
- **THEN** it SHALL show the change name and the full destination path (e.g., `~/overview-tab-with-stats-2026-04-22.zip`)

#### Scenario: User confirms with y
- **WHEN** the user presses `y` on the confirmation modal
- **THEN** the system SHALL execute the export operation

#### Scenario: User cancels with n or escape
- **WHEN** the user presses `n` or `escape` on the confirmation modal
- **THEN** the system SHALL dismiss the modal and return to normal state

### Requirement: Zip creation
The system SHALL create a zip file containing the entire change directory with preserved internal structure.

#### Scenario: Active change export
- **WHEN** exporting an active change named `my-feature`
- **THEN** the zip SHALL contain a root folder `my-feature/` with all files and subdirectories from `openspec/changes/my-feature/`

#### Scenario: Archived change export
- **WHEN** exporting an archived change with directory name `2026-04-02-my-feature`
- **THEN** the zip SHALL contain a root folder `my-feature/` (date prefix stripped) with all files and subdirectories from `openspec/changes/archive/2026-04-02-my-feature/`

#### Scenario: Zip filename format
- **WHEN** a change is exported on date 2026-04-22
- **THEN** the zip file SHALL be named `<semantic-name>-2026-04-22.zip` where semantic-name has any date prefix stripped

#### Scenario: Zip destination
- **WHEN** a change is exported
- **THEN** the zip file SHALL be written to the user's home directory (`~/`)

#### Scenario: Existing file at destination
- **WHEN** a zip file with the same name already exists at `~/`
- **THEN** the system SHALL overwrite it

### Requirement: Result feedback
The system SHALL display the export result in a modal that dismisses on any keypress.

#### Scenario: Successful export
- **WHEN** the zip file is created successfully
- **THEN** the system SHALL display a success modal showing the destination path

#### Scenario: Failed export
- **WHEN** the zip creation fails (e.g., permission error, disk full)
- **THEN** the system SHALL display a failure modal with the error message

#### Scenario: Dismiss result modal
- **WHEN** the user presses any key while the result modal is shown
- **THEN** the modal SHALL be dismissed and the UI SHALL return to normal state

### Requirement: Nav bar hint
The nav bar SHALL show an `e export` hint when the changes or archive tab is active and entries exist.

#### Scenario: Changes tab active with changes
- **WHEN** the changes tab is active and the project has active changes
- **THEN** the nav bar SHALL include the `e export` keybinding hint

#### Scenario: Archive tab active with archived changes
- **WHEN** the archive tab is active and the project has archived changes
- **THEN** the nav bar SHALL include the `e export` keybinding hint

#### Scenario: No entries on current tab
- **WHEN** the changes or archive tab is active but has no entries
- **THEN** the nav bar SHALL NOT include the `e export` hint
