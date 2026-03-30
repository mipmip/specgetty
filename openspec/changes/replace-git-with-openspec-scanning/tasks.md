## 1. Scanner data model

- [x] 1.1 Replace `RepoStatus` and `MultiGitStatus` in `scan.go` with `ProjectStatus` (containing `[]FileEntry`) and `ProjectMap` (`map[string]ProjectStatus`)
- [x] 1.2 Remove `GoGitStatus` and `GitStatus` functions from `scan.go`
- [x] 1.3 Create `ListOpenSpecContents(dir string) ([]FileEntry, error)` that recursively reads `openspec/` and returns relative paths with a dir/file indicator
- [x] 1.4 Rewrite `Scan()` to call `ListOpenSpecContents` on each found project instead of running git status
- [x] 1.5 Remove `excluded.go` and `excluded_test.go` entirely

## 2. Directory walker

- [x] 2.1 Change `find.go` callback to match `openspec` directory name instead of `.git`
- [x] 2.2 Update walker comments and function docs to reference OpenSpec instead of git

## 3. Config cleanup

- [x] 3.1 Remove `GitIgnore` field from `Config` struct in `scan.go`
- [x] 3.2 Remove `gitignore` section from embedded `config.yml`
- [x] 3.3 Remove excluder creation from `Scan()` function

## 4. UI changes

- [x] 4.1 Update `model` struct: replace `scanner.MultiGitStatus` with `scanner.ProjectMap`, remove `diffViewport` and diff-related fields
- [x] 4.2 Update `scanMsg` to use the new `ProjectMap` type
- [x] 4.3 Remove `diffMsg` type and all diff-fetching logic (`fetchDiff`, `colorizeDiff`)
- [x] 4.4 Remove diff-related styles (`diffAddedStyle`, `diffDeletedStyle`, `diffHunkStyle`, `diffMetaStyle`)
- [x] 4.5 Rewrite `renderFileList` to show `d`/`f` indicator + path instead of git status codes
- [x] 4.6 Rewrite `renderStatusContent` to be a single file list panel (no split with diff)
- [x] 4.7 Remove `doEditFile` and `canEditSelectedFile` (git-file-specific actions)
- [x] 4.8 Remove `e` keybinding and update nav bar keys
- [x] 4.9 Update `renderRepoList` empty message from "No dirty repositories found." to "No OpenSpec projects found."
- [x] 4.10 Update `renderPanel` title from "Repositories" to "Projects" and "Status" to "Contents"

## 5. Dependencies and tests

- [x] 5.1 Remove `github.com/go-git/go-git/v5` from imports and run `go mod tidy`
- [x] 5.2 Update `scan_test.go` and `find_test.go` for the new detection and data model
- [x] 5.3 Update `ui_test.go` for the changed types
- [x] 5.4 Verify `make build` succeeds and `spg --version` prints correctly
