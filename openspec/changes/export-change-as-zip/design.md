## Context

Specgetty already has two change actions: archive (`a`) and discard (`d`), both following the same state machine pattern (idle → confirming → running → result). Export follows this exact pattern but is non-destructive — it reads the change directory and writes a zip to `~/`.

The change directory path is not stored in `ChangeInfo` but can be reconstructed:
- Active: `<projectPath>/openspec/changes/<name>/`
- Archived: `<projectPath>/openspec/changes/archive/<datePrefix>-<name>/`

For archived changes, the directory name includes a date prefix (e.g., `2026-04-02-overview-tab-with-stats`). The scanner stores the full prefixed name in `ChangeInfo.Name` for archived changes.

## Goals / Non-Goals

**Goals:**
- Export any change (active or archived) as a self-contained zip file
- Follow established UI patterns (state machine, confirmation modal, result feedback)
- Zip preserves internal directory structure with the change name as root folder
- For archived changes, strip the date prefix from the root folder name in the zip (avoids redundancy since the zip filename already has a date)

**Non-Goals:**
- Configurable export path (always `~/`)
- Selective artifact export (always exports everything)
- Export multiple changes at once
- Integration with git or PR tooling

## Decisions

### State machine follows archive/discard pattern
Use the same 4-state machine: `exportIdle → exportConfirming → exportRunning → exportResult`. This keeps the code consistent and the UX predictable.

### Use Go stdlib `archive/zip`
No external dependency needed. The zip creation walks the change directory and adds each file with its relative path.

### Zip root folder uses semantic name
For active changes, the root folder is the change name as-is (e.g., `overview-tab-with-stats/`). For archived changes, strip the `YYYY-MM-DD-` prefix so the zip contains just the semantic name.

### Filename format: `<name>-<today>.zip`
The semantic name (date-prefix stripped for archived) plus today's date. Example: `overview-tab-with-stats-2026-04-22.zip`. This means re-exporting overwrites the same file on the same day, which is fine.

### Source path reconstruction
- Active changes: `<projectPath>/openspec/changes/<ChangeInfo.Name>/`
- Archived changes: `<projectPath>/openspec/changes/archive/<ChangeInfo.Name>/`

The scanner already stores the full directory name (with date prefix) in `ChangeInfo.Name` for archived changes, and uses `ArchiveDate.IsZero()` to distinguish active from archived.

### Export available on both changes and archive tabs
The `e` key works on the changes tab (active changes) and the archive tab (archived changes). The selected change is determined by `changeCursor` or `archiveCursor` depending on the active tab.

## Risks / Trade-offs

- **Overwrite without warning**: If the zip file already exists at `~/`, it gets silently overwritten. Acceptable because it's the user's own export and the filename includes the date.
- **Large changes**: A change with many specs or large artifacts could produce a sizable zip. Unlikely in practice — these are markdown files.
