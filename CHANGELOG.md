# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project adheres to Semantic Versioning.

## [0.1.0] - 2025-09-06

### Added
- Minimalist TUI for bullet journalling with list view, `j/k` navigation, paging (`PgUp/PgDn`), `g/G` home/end.
- Views: Day, Week, Month (`1/2/3`) with prev/next navigation (`[`/`]`) and jump-to-date (`d`).
- CRUD: Add (`a`), Edit (`e`), Delete (`x`).
- Actions: Complete (`c`, toggle Task/Done), Migrate (`m`, toggle on Migrated), Schedule (`s`, toggle on Scheduled), Type picker (`t`), Tags editor (`#`).
- Filters: Text (`/`), Type (`:` multi-select), Tags (`F`).
- Help overlay (`?`) with grouped, readable bindings.
- Today shortcut: `T` jumps to today.
- Footer shows context-aware keybinds and active filters.
- File-based storage (JSON Lines per day) under data dir (`BLT_DATA_DIR` or OS default). Preferences persisted (period, filters, last date, center width).
- CLI mode: `list`, `add`, `delete`, `complete`, `migrate`, `schedule`, `edit` with `--timespan`, `--date`, `--type`, `--tags`, `--text`, `--json`, `--data-dir` flags.
- CLI absolute day indexes: list prints day-local indexes; mutations require `--date` and operate by day index.

### Changed
- Inputs use a compact, inline style without blue/double borders; global keys disabled while input is active; footer shows only `[enter]`/`[esc]`.
- Delete confirmation uses a footer-only prompt (no input box).
- Header uses a bottom rule instead of a full border for a cleaner look.
- Type selector reworked to a compact, borderless overlay with visible hints; only semantic types (Task, Event, Note, Important, Inspiration) are selectable.
- Help overlay reworked to a borderless, wrapped layout that fits small widths.

### Fixed
- Keybinds shown correctly when inputs are active; tview color tags no longer interfere with bracketed hints.
- Type selector hints render correctly by disabling dynamic color parsing for literal brackets.

[0.1.0]: https://github.com/rdo34/blt/releases/tag/v0.1.0
