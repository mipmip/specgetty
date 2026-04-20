## Why

OpenSpec changes capture the full reasoning behind a feature — proposal, design, specs, tasks. When contributing to a foreign repo via PR, there's no way to attach this context. Reviewers only see the code diff. Exporting a change as a zip lets you attach the complete spec context to a PR so reviewers understand the *why* behind the code.

## What Changes

- Add an `e` keybinding in the TUI that exports the currently selected change as a zip file
- Works from both the changes tab (active changes) and the archive tab (archived changes)
- Zips the entire change directory preserving internal structure
- Writes to `~/<change-name>-<YYYY-MM-DD>.zip` using today's date
- Shows a confirmation modal before exporting, with the destination path
- For archived changes, strips the date prefix from the directory name inside the zip to avoid redundancy

## Capabilities

### New Capabilities
- `export-change`: Keybinding, confirmation modal, zip creation, and result feedback for exporting a change

### Modified Capabilities

## Impact

- `src/ui/ui.go`: New state machine for export flow (like archive/discard), `e` keybinding handler, confirmation and result modals, nav bar hint
- New Go standard library dependency: `archive/zip` (stdlib, no external dep)
- No changes to scanner or watcher
