## 1. Add discard state machine and message type

- [x] 1.1 Add discard state constants: `discardIdle`, `discardConfirming`, `discardRunning`, `discardResult` (after archive constants)
- [x] 1.2 Add discard model fields: `discardState`, `discardChangeName`, `discardResultMsg`, `discardResultOk`
- [x] 1.3 Add `discardMsg` struct (same shape as `archiveMsg`: `ok bool`, `output string`)

## 2. Add discard command function

- [x] 2.1 Add `doDiscardChange(projectPath, changeName string) tea.Cmd` that:
  - Creates `changes/discarded/` directory if it doesn't exist (`os.MkdirAll`)
  - Moves `changes/<name>` to `changes/discarded/YYYY-MM-DD-<name>` via `os.Rename`
  - Returns `discardMsg` with success/failure

## 3. Wire up key handler and modal intercepts

- [x] 3.1 Add discard modal intercept block (mirrors archive modal intercept): handle `y`/`n`/`esc` when `discardConfirming`, dismiss on any key when `discardResult`, block input when `discardRunning`
- [x] 3.2 Add `"d"` key case under changes tab: set `discardChangeName` and `discardState = discardConfirming`
- [x] 3.3 Handle `discardMsg` in Update: set result fields, trigger rescan on success

## 4. Render discard modals

- [x] 4.1 Add discard modal rendering in View (mirrors archive modal): confirming (with incomplete task warning), running (spinner), result (success/failure + dismiss)

## 5. Scanner: skip discarded directory

- [x] 5.1 In `ParseProjectInfo` (scan.go), extend the skip condition on line 208 to also skip `e.Name() == "discarded"`

## 6. Nav bar hint

- [x] 6.1 Add `d discard` hint to nav bar when changes tab is active and changes exist

## 7. Verification

- [x] 7.1 Build passes (`go build ./...`)
- [x] 7.2 Tests pass (`go test ./...`)
