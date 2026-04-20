## Context

The archive feature provides `a` → confirmation modal → `openspec archive` CLI call → rescan. The discard feature mirrors this pattern but is simpler: no CLI dependency, just a direct `os.Rename`.

```
┌─ Changes tab ─────────────────────────────────┐
│  ▶ add-websocket-support  (3/5)               │
│    fix-auth-bug           (2/2)               │
│                                               │
│  [a] archive   [d] discard                    │
└───────────────────────────────────────────────┘

Press d:
┌─────────────────────────────────────────┐
│  Discard "add-websocket-support"?       │
│                                         │
│  ⚠ 2 incomplete task(s)                │
│                                         │
│  Will be moved to changes/discarded/    │
│  (y/n)                                  │
└─────────────────────────────────────────┘
```

## Goals / Non-Goals

**Goals:**
- `d` key on Changes tab triggers discard flow
- Confirmation modal with incomplete task warning (same pattern as archive)
- Move change directory to `changes/discarded/YYYY-MM-DD-<name>/`
- `spg` handles the move directly — no external CLI dependency
- Scanner skips `discarded/` directory (no UI visibility)
- Rescan after successful discard

**Non-Goals:**
- Showing discarded changes anywhere in the UI (no "Discarded" tab)
- Undo/restore from within the UI (manual recovery from disk is sufficient)
- Deleting the directory permanently

## Decisions

### 1. Mirror archive state machine pattern
Add `discardIdle`, `discardConfirming`, `discardRunning`, `discardResult` constants — identical pattern to archive. Keeps the codebase consistent and the implementation predictable.

### 2. Direct filesystem move via os.Rename
Unlike archive (which shells out to `openspec archive`), discard uses `os.Rename` directly. This means:
- No dependency on the openspec CLI being installed
- Simpler error handling (just the rename error)
- `discardRunning` state is brief but kept for consistency

### 3. Scanner skips "discarded" alongside "archive"
In `ParseProjectInfo` (scan.go line 208), the condition `e.Name() == "archive"` becomes `e.Name() == "archive" || e.Name() == "discarded"`. Minimal change, same pattern.

### 4. Date-prefixed directory naming
Same convention as archive: `YYYY-MM-DD-<name>`. Allows sorting by discard date if someone browses the directory manually.

### 5. Confirmation modal always shown
Even for changes with no tasks, the modal confirms intent. For changes with incomplete tasks, the modal shows the count — same UX as archive but with "Discard" wording.

## Risks / Trade-offs

- **Trade-off**: No undo in the UI. Acceptable — discarded changes are still on disk, and restoring is a manual `mv` operation. Adding undo would over-complicate a feature meant for dead-end work.
