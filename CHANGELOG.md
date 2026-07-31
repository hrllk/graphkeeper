# Changelog

## alpha.5 - 2026-07-31

### Added

- Added version-aware CLI behavior with `--help` and `--version`, including
  documented exit codes and release-build version injection.
- Added a section-aware hidden hotkey popup. It now shows the global shortcuts
  and the shortcuts for the active section, with scrolling and paging for
  smaller terminals.

### Changed

- Updated the TUI color theme to use ANSI-compatible colors so the graph,
  overlays, markers, and status labels render consistently across terminals.
- Centralized popup and theme styling to keep the visual language consistent
  across confirmations, searches, loading states, and branch actions.
- Added focused tests for CLI parsing, hidden-hotkey navigation, rendering,
  keyboard handling, model behavior, and Git repository operations.

### Documentation

- Added the CLI reference and build-and-run guide.
- Added project context, architecture decisions, and the current feedback
  task list for ongoing development.

## alpha.4

- Initial versioned CLI baseline.
