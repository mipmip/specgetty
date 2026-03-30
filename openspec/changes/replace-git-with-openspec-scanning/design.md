## Context

Specgetty is forked from dirty-repo-scanner. The scanner package currently walks directory trees looking for `.git/` directories, then runs `git status` on each match to find "dirty" repos. The UI displays a list of dirty repos on the left, a file list with git status codes in the middle, and a diff viewport on the right.

The tool needs to be repurposed: instead of finding dirty git repos, it finds projects that contain an `openspec/` directory and shows their OpenSpec contents.

## Goals / Non-Goals

**Goals:**
- Replace `.git` detection with `openspec/` directory detection in the walker
- Replace the git-status data model with a simple directory listing model
- Remove the diff panel — replace with a plain listing of openspec subdirs and files
- Remove the `go-git` dependency entirely
- Keep the existing TUI structure (bubbletea, panels, navigation) intact

**Non-Goals:**
- Parsing or interpreting OpenSpec YAML/Markdown files (future work)
- Showing change status, spec validation, or any semantic OpenSpec understanding
- Changing the config file format beyond removing the `gitignore` section
- Modifying the walker's glob/exclude/symlink logic

## Decisions

### 1. Detection target: `openspec/` directory
Walk the directory tree and match on a directory named `openspec` instead of `.git`. When found, emit the parent directory as a "project". This is a minimal change in `find.go` — swap the name check from `.git` to `openspec`.

**Alternative considered**: Looking for `openspec/config.yaml` specifically. Rejected because just checking for the directory is simpler and matches how the `.git` detection worked.

### 2. Data model: replace `git.Status` with a flat file list
Replace `RepoStatus` (wrapping `git.Status`) with a new `ProjectStatus` struct containing a `[]FileEntry` where each entry has a path and whether it's a directory. Replace `MultiGitStatus` with `map[string]ProjectStatus`.

This is read via `os.ReadDir` recursively on the `openspec/` subdirectory — no external dependencies needed.

### 3. Remove diff panel, show file listing instead
The right-side diff viewport is removed entirely. The status panel becomes a single file/directory listing of the `openspec/` contents (no split view). Each line shows a directory indicator or plain filename — no git status codes.

### 4. Remove `excluded.go` entirely
The excluder filtered git status entries based on gitignore patterns. With no git status, this is unnecessary. The walker's directory-level exclude logic (in `find.go`) remains untouched.

### 5. Remove `go-git` dependency
All imports of `github.com/go-git/go-git/v5` are removed. Run `go mod tidy` to clean up `go.mod`/`go.sum`.

## Risks / Trade-offs

- **Risk**: Recursive listing of large `openspec/` directories could be slow → Mitigation: OpenSpec directories are typically small (specs, changes, config). No mitigation needed now.
- **Trade-off**: Removing diff panel simplifies the UI but loses the split-panel layout. This is intentional — the split layout can be reintroduced later when there's meaningful content to show on the right side.
