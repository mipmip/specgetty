## Why

The overview tab duplicates information that's already available in other tabs (active changes → changes tab, archived → archive tab). Having it as a separate tab wastes a tab slot and hides useful context when browsing other tabs. Moving the essential overview info into a persistent header means users always see the project path and stats regardless of which tab they're on.

## What Changes

- **Remove** the `overview` tab from the tab bar
- **Add** a persistent header area at the top of the detail panel showing:
  - Project path (full path, styled)
  - Stats line: spec count, active changes count, archived count
- The header is always visible above the tab bar, regardless of which tab is active
- Tab indices shift: specs becomes tab 1, changes tab 2, etc.
- The active changes list and recently archived list are removed — they're accessible via the changes and archive tabs

## Capabilities

### New Capabilities

### Modified Capabilities
- `side-panel-layout`: Detail panel now has a persistent header area above the tab bar

## Impact

- **ui/ui.go**: Remove `tabOverview` constant, remove `renderOverview()`, add header rendering in `renderDetailPanel()`, shift tab constants, reduce tab content height to account for header lines
