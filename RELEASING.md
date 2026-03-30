# Releasing specgetty

This document describes the release process for specgetty.

## Overview

Releases are automated via GitHub Actions and goreleaser. When a `v*` tag is pushed to GitHub, the release workflow builds binaries for all supported platforms and creates a GitHub Release with the artifacts.

## Supported Platforms

- Linux (amd64, arm64)
- macOS (amd64, arm64)

## Pre-Release Checklist

Before creating a release, ensure:

- [ ] All tests pass: `make test`
- [ ] The application builds: `make build`
- [ ] `CHANGELOG.md` is updated with the new version's changes
- [ ] The `[Unreleased]` section has been moved to a versioned section

## Creating a Release

The easiest way to create a release is using the release script:

```bash
./scripts/release.sh
```

This will interactively:
1. Check for a clean working tree
2. Prompt for version bump type (patch/minor/major)
3. Update `src/VERSION`
4. Update `CHANGELOG.md`
5. Update `flake.nix` vendorHash (if nix is available)
6. Commit, tag, and push

### Prerequisites

- [gum](https://github.com/charmbracelet/gum) — interactive CLI prompts
- [nix](https://nixos.org/) — optional, for vendorHash updates

## Manual Release

If you prefer to release manually:

### 1. Update the Changelog

Edit `CHANGELOG.md`:
- Rename the `[Unreleased]` section to `[X.Y.Z] - YYYY-MM-DD`
- Add a new empty `[Unreleased]` section at the top

### 2. Update the Version

```bash
echo "X.Y.Z" > src/VERSION
```

### 3. Commit and Tag

```bash
git add src/VERSION CHANGELOG.md
git commit -m "chore: release vX.Y.Z"
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin HEAD:master
git push origin vX.Y.Z
```

### 4. Wait for the Release

The GitHub Action will automatically:
1. Build binaries for all platforms
2. Create checksums
3. Create a GitHub Release with all artifacts

## Local Testing

```bash
# Validate goreleaser config
goreleaser check

# Build a snapshot (doesn't create a release)
goreleaser build --snapshot --clean

# Check the built binaries
ls -la dist/
```

## Troubleshooting

### Release workflow failed

1. Check the Actions tab on GitHub for error details
2. Common issues:
   - Missing `GITHUB_TOKEN` permissions (should be automatic)
   - goreleaser config errors (run `goreleaser check` locally)
   - Build failures (run `make build` locally)

### Tag already exists

If you need to redo a release:
```bash
git tag -d vX.Y.Z
git push origin :refs/tags/vX.Y.Z
# Delete the GitHub Release manually via the web UI
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin vX.Y.Z
```

### Version not showing in binary

Ensure the tag follows the `vX.Y.Z` format. goreleaser extracts the version from the git tag and injects it via ldflags.

Test locally:
```bash
go build -ldflags "-X main.version=test" -o spg ./src
./spg --version
```
