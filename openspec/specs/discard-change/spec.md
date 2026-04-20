# discard-change Specification

## Purpose
Allows users to discard abandoned active changes, moving them to a `discarded/` directory instead of polluting the archive with incomplete work.

## Requirements

### Requirement: Discard keybinding
The system SHALL provide a `d` keybinding that initiates discarding when the changes tab is active and a change is selected.

#### Scenario: Press d on changes tab with a change selected
- **WHEN** the user presses `d` while the changes tab is focused and a change is highlighted
- **THEN** the system SHALL show a confirmation modal for discarding that change

#### Scenario: Press d when no changes exist
- **WHEN** the user presses `d` on the changes tab with no active changes
- **THEN** the system SHALL do nothing

#### Scenario: Press d on a different tab
- **WHEN** the user presses `d` while a tab other than changes is active
- **THEN** the system SHALL do nothing

### Requirement: Incomplete task warning
The system SHALL warn the user when discarding a change that has incomplete tasks.

#### Scenario: Change has incomplete tasks
- **WHEN** the user initiates a discard for a change where `TasksDone < TasksTotal`
- **THEN** the confirmation modal SHALL display the count of incomplete tasks and ask "Discard anyway? (y/n)"

#### Scenario: Change has all tasks complete or no tasks
- **WHEN** the user initiates a discard for a change where `TasksDone == TasksTotal` or `TasksTotal == 0`
- **THEN** the confirmation modal SHALL display "Discard <name>? (y/n)" without a task warning

### Requirement: Confirmation modal
The system SHALL require explicit confirmation before executing the discard.

#### Scenario: User confirms with y
- **WHEN** the user presses `y` on the confirmation modal
- **THEN** the system SHALL execute the discard operation

#### Scenario: User cancels with n or escape
- **WHEN** the user presses `n` or `escape` on the confirmation modal
- **THEN** the system SHALL dismiss the modal and return to normal state

### Requirement: Discard execution
The system SHALL move the change directory to `changes/discarded/YYYY-MM-DD-<name>/` using a direct filesystem rename.

#### Scenario: Successful discard
- **WHEN** the rename succeeds
- **THEN** the system SHALL display a success modal and rescan the project

#### Scenario: Target directory already exists
- **WHEN** the target path `changes/discarded/YYYY-MM-DD-<name>/` already exists
- **THEN** the system SHALL display a failure modal with an appropriate error message

#### Scenario: Discarded directory does not exist yet
- **WHEN** the `changes/discarded/` directory does not exist
- **THEN** the system SHALL create it before performing the rename

### Requirement: Result feedback
The system SHALL display the discard result in a modal that dismisses on any keypress.

#### Scenario: Dismiss result modal
- **WHEN** the user presses any key while the result modal is shown
- **THEN** the modal SHALL be dismissed and the UI SHALL return to normal state

### Requirement: Nav bar hint
The nav bar SHALL show a `d discard` hint when the changes tab is active and changes exist.

#### Scenario: Changes tab active with changes
- **WHEN** the changes tab is active and the project has active changes
- **THEN** the nav bar SHALL include the `d discard` keybinding hint

### Requirement: Scanner skips discarded directory
The scanner SHALL ignore the `discarded/` subdirectory within `changes/`, the same way it ignores `archive/`.

#### Scenario: Discarded directory exists
- **WHEN** scanning active changes in `openspec/changes/`
- **THEN** the scanner SHALL skip any entry named `discarded`
