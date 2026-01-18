---
allowed-tools: Bash(./scripts/release.sh:*), Bash(git:*), Bash(gh:*), Bash(cargo:*), Bash(tar:*), Bash(shasum:*), Read, Grep, Edit, Write
description: Release a new binary version of the claude-wrapper
argument-hint: [version number (e.g., 1.2.3)]
---

# Release Claude Wrapper Binary

Based on the `Instructions` below, take the `Variables`, follow the `Workflow` section to execute the release script for the claude-wrapper. Then follow the `Report` section to report the results of your work.

## Variables

VERSION: $1

## Instructions

- Intelligently handle the release process based on the current state of the repository
- Check if the version is already set in Cargo.toml and if the tag exists
- The version must follow semantic versioning format (e.g., 1.2.3)
- The release process includes:
  - Validating prerequisites (git, gh CLI, cargo, docker)
  - Updating version in Cargo.toml (if needed)
  - Running tests and quality checks (cargo test, clippy, fmt)
  - Building binaries for multiple platforms (macOS, Linux)
  - Generating checksums for all binaries
  - Creating release notes with changelog
  - Creating git commit and tag
  - Pushing to GitHub
  - Creating GitHub release with all artifacts
- Be flexible and handle edge cases:
  - Version already matches but no tag exists
  - Tag already exists
  - Commits need to be pushed before tagging
  - Build failures on specific platforms
- Use the automated script when appropriate, but also handle manual steps when needed

## Workflow

1. Change to the claude-wrapper directory
2. Check the current version in Cargo.toml to understand what version is currently set
3. Check if the requested version tag already exists in git
4. If the version in Cargo.toml matches $VERSION and the tag doesn't exist:
   - Push any uncommitted/unpushed changes first
   - Create the tag manually: `git tag -a v$VERSION -m "Release v$VERSION"`
   - Push the tag: `git push origin v$VERSION`
   - Run the build and release steps manually
5. If the version in Cargo.toml doesn't match $VERSION:
   - Execute the full release script: `./scripts/release.sh $VERSION`
6. If the tag already exists:
   - Inform the user and ask if they want to delete and recreate it
7. Handle any errors gracefully:
   - If version bump commit fails due to no changes, continue with tag creation
   - If builds fail for specific platforms, continue with successful ones
   - If GitHub release creation fails, provide manual instructions

## Report

Provide a concise summary including:
- The version number that was released
- The tag created (e.g., v1.2.3)
- The platforms that were successfully built
- The GitHub release URL
- Any warnings or issues encountered during the process
