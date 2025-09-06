# Implementation Plan

## Phase 1 — Foundations
- Restructure for growth: move entry to `cmd/blt/main.go`; add `internal/` packages.
- Packages: `internal/model` (bullet, journal range), `internal/store` (FS persistence), `internal/ui` (views, keymap), `internal/app` (state, actions).
- Decide storage root: `~/.local/share/blt` (Linux), `%APPDATA%/blt` (Windows), override via `BLT_DATA_DIR`.

Status
- Entry moved to `cmd/blt/main.go` — completed.
- Scaffolds created: `internal/{ui,model,store,app}` — completed.
- Data dir resolver implemented (`BLT_DATA_DIR`, OS defaults) — completed.

Next
- Introduce concrete FS store and begin wiring application state (Phase 2/3).

## Phase 2 — Domain & Storage
- Model bullet types: `Task`, `Done`, `Migrated`, `Scheduled`, `Event`, `Note`, `HighlightImportant`, `HighlightInspiration`.
- File format: JSON Lines per day: `YYYY/MM/DD.jsonl` with one bullet per line; include id, ts, type, text, tags, highlight, schedule date.
- Store API: `LoadDay`, `SaveDay`, `Append`, `Update`, `Delete`; non‑blocking I/O, queue UI updates via `QueueUpdateDraw`.

Status
- Added JSON tags to `model.Bullet` for stable persistence — completed.
- Implemented `internal/store/fsstore.go` with JSONL per-day files — completed.
- Data dir resolution integrated and directories auto-created — completed.
- Verified build with `go build ./...` — completed.

Next
- Begin wiring store into application state and UI list rendering (Phase 3).

## Phase 3 — Core TUI
- Layout: title, controls, main list; reuse `tview.Grid`.
- Navigation: `j/k` move, `g/G` home/end, `PgUp/PgDn` paging.
- Selection state maintained in `internal/app`.

Status
- Wired `FSStore` into UI with fallback to local `.blt-data` — completed.
- Rendered bullets for today in a `tview.List` — completed.
- Implemented `j/k` selection navigation and focus on list — completed.

Next
- Add `g/G` and paging keys; show empty-state message when no items. — completed.
- Begin state-driven refresh methods to re-render list after changes. — completed (`refreshList`).

## Phase 4 — Create & Edit
- Add: `a` opens modal input; create bullet (default Task).
- Edit: `u` or `e` to edit text; `x` delete with confirm.
- Change type: `t` cycle or open type picker.
- Mark done/migrate/schedule: `c` complete; `m` migrate to next day; `s` pick date (date picker modal).

Status
- Add modal implemented (`a`) with persistence and list refresh — completed.
- Edit modal implemented (`e`/`u`) with persistence and refresh — completed.
- Delete confirm (`x`) implemented — completed.
- Complete (`c`), Migrate to next day (`m`), Schedule to date (`s`) implemented with store operations — completed.
 - Type picker (`t`) implemented; updates type and normalizes related fields — completed.
 - Basic tag editing (`#`) implemented with comma-separated input and normalized storage — completed.

Refinements
- Added validation and error modals (empty text, invalid date) — completed.
- Added Escape-to-cancel on modals and dialogs — completed.
- Removed toasts for faster, uncluttered UX; rely on immediate updates and footer note — completed.
- Migration/Scheduling semantics: keep a marked entry on the source day (Migrated/Scheduled) and add a copy with the original type on the target day — completed.

Fixes (Interaction Toggles)
- Complete toggle: pressing `c` on a Task marks Done; pressing `c` again reverts to Task. Prevent completing items that are Migrated or Scheduled. — completed.
- Migrate toggle: pressing `m` on a Task/Event marks current as Migrated and creates a copy next day; pressing `m` again on the Migrated item undoes this (restores original type and deletes the next-day copy). — completed.
- Schedule toggle: pressing `s` on a Task/Event marks current as Scheduled and creates a copy on picked date; pressing `s` again on the Scheduled item undoes this (restores original type and deletes the scheduled copy). — completed.
- Implementation notes: stored the migration target date in `ScheduledFor` for Migrated items to enable reliable undo; clone matching uses text (case-insensitive) and original types Task/Event.

Next
- Optional: add type picker (`t`) and tags/highlights editing.
- Improve input validation and feedback (e.g., show parse errors for date).
 - Consider showing tags inline and/or a tag filter (Phase 5).

## Phase 5 — Views & Filters
- Date scope: day/week/month; keys `1/2/3` toggle scope.
- Date navigation: `[`/`]` previous/next period; `d` jump-to-date modal.
- Filters: `/` text filter, `:` type filter, `#` tag filter; show active filters in status.

Status
- Added period state (day/week/month) and range loading across days — completed.
- Wired keys: `1/2/3` scope, `[` and `]` prev/next, `d` jump-to-date — completed.
- Implemented filters: text (`/`), type multi-select (`:`), tag list (`F`) with live summary — completed.
- List shows date prefix for non-day scopes; controls/footer reflect active filters — completed.
- Editing actions disabled outside Day with informative toast — completed.

Optional Follow-ups
- Persisted period, filters, and last date across runs (prefs.json) — completed.
- Sidebar summary shows counts by type for the visible range — completed.

Next
- Optional: persist last scope/filters between runs.
- Add week/month summaries (counts by type) in sidebar.

## Phase 6 — Help & UX
- `?` opens help overlay listing bindings and examples.
- Persistent controls footer kept concise; reflect context-sensitive hints.

Status
- Help overlay added with key summaries, navigation, filters, and modification keys — completed.
- All modals close with Esc; toasts provide feedback — completed.

Enhancements
- Dynamic keybinds footer: controls now adapt to the selected item type in Day view. For example, hide `[m] Migrate` on completed items; hide `[c] Complete` on scheduled/migrated/other non-task types; show `[m]` on migrated and `[s]` on scheduled to support undo. Also wired selection-change updates to refresh the footer — completed.
- Help overlay updated to reflect toggle behavior and context-aware actions: `c` toggles Task/Done, `m` undoes on Migrated, `s` undoes on Scheduled; notes that action keys appear only when applicable — completed.
- Footer labeling refined: action labels toggle with state — `[c] Complete` becomes `[c] In progress` on completed items; `[m] Migrate` becomes `[m] Unmigrate` on migrated; `[s] Schedule` becomes `[s] Unschedule` on scheduled — completed.
- Selection preservation: after actions (complete/migrate/schedule/edit/delete), the list preserves the currently selected item using ID+date tracking, avoiding jump-to-top behavior — completed.

Help Overlay Rework
- Replaced bordered modal with borderless, padded overlay and top/bottom rules for a cleaner look — completed.
- Enabled wrapping for small terminal widths and simplified line layout — completed.

Type Selector Rework — Completed
- Implemented compact, borderless overlay for type selection wrapped with top/bottom rules; no full-screen modal.
- Restricted choices to semantic types only: Task, Event, Note, Important, Inspiration (removed Done/Completed, Migrated, Scheduled).
- Added visible hints within overlay: "[j/k] Move  [enter] Select  [esc] Cancel"; global keys disabled while open.
- Centered fixed-size overlay sized to content to avoid scrolling; on very small terminals, j/k and arrows scroll the list.
- Fix: Disabled dynamic color parsing on the hints line so square brackets render literally, ensuring keybinds display correctly.

Navigation Tweak
- Added `T` shortcut to jump to today; updated help and keybinding docs — completed.

Layout Rework
- Removed summary sidebar for a minimalist layout. The list is now centered within a fixed-width middle column for better use of space across terminal sizes — completed.
- Header redesigned: two lines — first line shows app name; second line shows current scope and date range plus percent completed (based on Task/Done within visible items) — completed.
- Header styling: ensured second line uses default background (no green fill) — completed.
- Header layout refined: single-line header with left/right alignment; replaced full border with a bottom rule only — completed.

Command Bar Input
- Replaced modal dialogs with an inline, bordered input bar above the keybinds: add/edit text, schedule date, jump-to-date, filters, type and tag editing, and delete confirmation now use the bottom input without covering content — completed.
- Added configurable center width via preferences (`center_width` in `prefs.json`); default 80. App preserves unknown preference fields when saving — completed.

Interaction Fixes (Inputs)
- Disable global keybinds while an input is active; only Enter confirms and Esc cancels — completed.
- Remove blue background from inputs; apply default background to input fields and delete confirm — completed.
- Show accurate keybinds while input is active: footer displays `[enter] Confirm  [esc] Cancel` — completed.
- Align delete confirmation with Enter/Esc behavior (Enter confirms, Esc cancels) — completed.
- Delete confirmation no longer shows a box; uses footer prompt with Enter/Esc and blocks other keys — completed.

Keybinding Simplification
- Edit uses `[e]` only (removed `[u]`) across handlers, footer, and help — completed.

Borders & Rules
- Replaced input box borders with a minimal top-and-bottom rule to eliminate doubled borders — completed.
- Replaced header full border with bottom rule only — completed.

## Phase 7 — Testing & Tooling
- Extract pure logic for tests: store, filtering, date ranges, migrations.
- Add table-driven tests under `internal/.../*_test.go`.
- Commands: `go fmt ./...`, `go vet ./...`, `go test ./... -cover`.

## Phase 8 — CLI Mode
- Implemented a non-TUI interface to add, list, filter, and modify bullets using commands and flags, reusing `internal/app` and `internal/store`.

- Commands:
  - `blt add [--type task|event|note|important|inspiration] --text "..." | --note "..." [--date YYYY-MM-DD] [--data-dir PATH]`
  - `blt list [--timespan day|week|month] [--date YYYY-MM-DD] [--type ...] [--tags ...] [--text ...] [--json] [--data-dir PATH]`
  - `blt delete <index> [context flags]`
  - `blt complete <index> [context flags]`
  - `blt migrate <index> [context flags]`
  - `blt schedule <index> --to YYYY-MM-DD [context flags]`
  - `blt edit <index> --set "new text" [context flags]`

- Context flags:
  - `--timespan`, `--date`, `--type`, `--tags`, `--text`, and `--data-dir` define the same context as in the TUI filters. Index refers to the position within that context.

- Output:
  - Default: human-readable list (prefix + text + tags) and date prefix in non-day spans.
  - `--json`: machine-readable JSON array of entries with id, date, type, text, tags.

- Implementation notes:
  - Subcommand parsing built with stdlib `flag` in `cmd/blt/cli.go`. `main.go` routes to CLI when a known subcommand is present; otherwise loads TUI.
  - `add` writes directly via `store.Append` with the chosen type; all other mutations reuse `app` index-based methods for consistency.
  - Exit codes: non-zero on parsing/validation/storage errors; messages printed to stderr.

- Acceptance: Verified `go build`, and basic flows for list/add/delete/complete/migrate/schedule/edit compile and follow the expected semantics.

## Phase 9 — Releases
- Strategy: Tag-based releases via GitHub Actions building cross-platform binaries and attaching them to the GitHub Release.
- Targets: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, windows/arm64.
- Workflow: `.github/workflows/release.yml` triggers on `v*` tags (and manually), builds with `CGO_ENABLED=0`, names artifacts `blt_<os>_<arch>(.exe)`, and publishes using `softprops/action-gh-release`.
- Next: Optionally add a separate CI workflow for lint/test on PRs; optionally generate SHA256 checksums and a release body from CHANGELOG.

## Milestones & Acceptance
- M1: Load/save day, list view with `j/k`, add task.
- M2: Edit/delete/change type; complete/migrate/schedule with date picker.
- M3: Day/week/month scopes, filters, jump-to-date.
- M4: Help overlay, tests ≥ core logic coverage, docs updated.
