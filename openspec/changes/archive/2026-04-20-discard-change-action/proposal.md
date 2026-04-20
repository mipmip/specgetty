## Why

When a change is explored or partially worked on but then abandoned (e.g., the approach was wrong, requirements shifted, insights invalidated the direction), there's no way to remove it from the active changes list without archiving it. Archiving pollutes the archive with dead-end work that was never completed. Users need a way to explicitly discard abandoned changes, keeping the archive clean for actually completed work.

## What Changes

- **Add** `d` keybinding on the Changes tab to discard the selected change
- **Add** confirmation modal (same pattern as archive) warning about incomplete tasks and showing the destination
- **Move** the change directory from `changes/<name>/` to `changes/discarded/YYYY-MM-DD-<name>/`
- `spg` handles the move directly via `os.Rename` — no `openspec` CLI dependency
- **Scanner ignores** the `discarded/` directory entirely — discarded changes never appear in the UI
- Triggers a rescan after successful discard (same as archive)

## Capabilities

### New Capabilities
- `discard-change`: Discard an active change by pressing `d`, moving it to `changes/discarded/` with a date prefix

### Modified Capabilities
- `keyboard-navigation`: New `d` key on Changes tab
- `change-scanning`: Scanner skips `discarded/` directory when discovering changes

## Impact

- **ui/ui.go**: Add discard state constants, discard key handler, confirmation modal, `doDiscardChange()` command, discard message type
- **scanner/find.go**: Skip `discarded/` directory during change discovery (same as `archive/` is skipped)
