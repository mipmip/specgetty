## Why

Specgetty was forked from dirty-repo-scanner and still contains all the git-oriented scanning and UI logic. The tool's purpose is now to find and display OpenSpec projects, not dirty git repos. The core scanning, data model, and UI all need to be rewired to look for `openspec/` directories instead of `.git/` directories, and the detail pane needs to show OpenSpec contents instead of git status/diffs.

## What Changes

- **BREAKING**: Replace `.git` directory detection in `find.go` with `openspec/` directory detection
- **BREAKING**: Replace `git.Status`-based data model (`RepoStatus`, `MultiGitStatus`) with an OpenSpec project model that lists subdirs and files under `openspec/`
- **BREAKING**: Remove git status columns (staged/worktree codes) from the file list pane
- **BREAKING**: Remove diff fetching and diff viewport entirely
- Replace the right-side diff panel with a simple file/directory listing of the `openspec/` contents
- Remove `go-git` dependency from `go.mod`
- Remove `excluded.go` (git-specific ignore logic) — not needed for OpenSpec scanning
- Update config to remove `gitignore` section (no longer relevant)

## Capabilities

### New Capabilities
- `openspec-scanning`: Scan directories to find projects containing an `openspec/` subdirectory and list their contents

### Modified Capabilities

## Impact

- **scanner package**: `scan.go`, `find.go`, `excluded.go` — major rewrite of all three files
- **ui package**: `ui.go` — remove diff panel, change file list to show openspec contents, update status content rendering
- **Dependencies**: Remove `github.com/go-git/go-git/v5` from `go.mod`/`go.sum`
- **Config**: `config.yml` and `Config` struct lose the `gitignore` section
- **Tests**: All existing scanner tests will need updating for the new detection logic
