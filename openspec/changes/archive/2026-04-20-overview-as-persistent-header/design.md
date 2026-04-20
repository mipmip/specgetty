## Context

Current detail panel layout:
```
┌─ Detail ──────────────────────────────────┐
│ [overview] [specs] [changes] [archive]... │
│ (tab content fills remaining space)       │
└───────────────────────────────────────────┘
```

New layout:
```
┌─ Detail ──────────────────────────────────┐
│ /home/pim/cVibeCoding/specgetty           │
│ Specs: 2  Changes: 7 active  Archived: 1 │
│ [specs] [changes] [archive] [tasks] [config] │
│ (tab content fills remaining space)       │
└───────────────────────────────────────────┘
```

## Goals / Non-Goals

**Goals:**
- Persistent 2-line header: project path + stats line
- Remove overview tab entirely
- Tab bar shifts down, tab content height reduced by 2 lines

**Non-Goals:**
- Showing active change names or archived change names in the header (too verbose)

## Decisions

### 1. Header is 2 lines: path + stats
Line 1: full project path in bold blue (existing `headerStyle`). Line 2: stats in dim style. Compact and always useful.

### 2. Remove renderOverview entirely
The active changes list and archived list were the bulk of the overview. Both are now accessible through their own tabs. No information is lost.

### 3. Tab constants renumber
Remove `tabOverview = 0`. Specs becomes 0, changes 1, archive 2, tasks 3, config 4. Number keys 1-5 map to these.

## Risks / Trade-offs

- **Trade-off**: Losing 2 lines of tab content height for the header. Acceptable — the context is more valuable than 2 extra lines of scrollable content.
