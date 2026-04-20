## 1. Remove overview tab

- [x] 1.1 Remove `tabOverview` constant, renumber remaining tabs: specs=0, changes=1, archive=2, tasks=3, config=4
- [x] 1.2 Update `tabNames` slice to remove "overview"
- [x] 1.3 Remove `renderOverview()` function
- [x] 1.4 Remove `tabOverview` case from `renderDetailPanel` switch
- [x] 1.5 Update number key bindings (1-5) for new tab indices
- [x] 1.6 Update nav bar hint from "1-6" to "1-5"

## 2. Add persistent header

- [x] 2.1 In `renderDetailPanel`, render project path (headerStyle) + stats line (dimStyle) before the tab bar
- [x] 2.2 Reduce tab content height by 2 to account for header lines
- [x] 2.3 Ensure header renders correctly in both normal and zoom mode

## 3. Tests and verification

- [x] 3.1 Update `TestRenderTabHeader` for new tab names (no "overview")
- [x] 3.2 Verify build and all tests pass
