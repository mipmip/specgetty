## 1. Export state machine and model fields

- [x] 1.1 Add export state constants (`exportIdle`, `exportConfirming`, `exportRunning`, `exportResult`)
- [x] 1.2 Add model fields: `exportState`, `exportChangeName`, `exportResultMsg`, `exportResultOk`, `exportIsArchived`
- [x] 1.3 Add `exportMsg` message type (ok bool, output string)

## 2. Keybinding and confirmation

- [x] 2.1 Handle `e` keypress on changes tab — set `exportChangeName` from `changeCursor`, `exportIsArchived = false`, transition to `exportConfirming`
- [x] 2.2 Handle `e` keypress on archive tab — set `exportChangeName` from `archiveCursor`, `exportIsArchived = true`, transition to `exportConfirming`
- [x] 2.3 Handle `y`/`n`/`escape` in `exportConfirming` state — confirm triggers export, cancel returns to idle

## 3. Zip creation

- [x] 3.1 Implement `doExportChange(projectPath, changeName string, isArchived bool) tea.Cmd` that creates the zip
- [x] 3.2 Resolve source directory: active → `openspec/changes/<name>/`, archived → `openspec/changes/archive/<name>/`
- [x] 3.3 Strip date prefix from archived change name for the zip root folder and filename
- [x] 3.4 Walk source directory, add all files to zip with relative paths under `<semanticName>/`
- [x] 3.5 Write zip to `~/<semanticName>-<YYYY-MM-DD>.zip`

## 4. Result handling and UI

- [x] 4.1 Handle `exportMsg` in Update — set result fields, transition to `exportResult`
- [x] 4.2 Render confirmation modal showing change name and destination path
- [x] 4.3 Render result modal (success with path, or error message)
- [x] 4.4 Dismiss result modal on any keypress, return to `exportIdle`

## 5. Nav bar

- [x] 5.1 Show `e export` hint when changes tab is active and has entries
- [x] 5.2 Show `e export` hint when archive tab is active and has entries
