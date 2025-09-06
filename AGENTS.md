# Repository Guidelines

## Project Structure & Module Organization
- `main.go`: Entry point; TUI built with `tview`/`tcell`.
- `go.mod` / `go.sum`: Module metadata (`github.com/rdo34/blt`).
- Binary: `blt` (local build artifact). Consider future packages under `internal/` and CLI subcommands under `cmd/blt/` as the project grows.

## Build, Test, and Development Commands
- Build: `go build -o blt .` — compiles the TUI binary.
- Run: `go run .` — runs the app without producing a binary.
- Format: `go fmt ./...` — formats all Go files.
- Vet: `go vet ./...` — catches common issues.
- Test: `go test ./...` — runs unit tests (none yet).
- Coverage: `go test ./... -cover` — prints coverage summary.

## Coding Style & Naming Conventions
- Formatting: Use `gofmt` (`go fmt`). No unformatted diffs in PRs.
- Imports: Standard, then third‑party, then local; group with blank lines.
- Naming: Exported identifiers use `CamelCase` and clear nouns/verbs; unexported use `camelCase`.
- UI constants/keys: Prefer `const` (e.g., `DefaultControls`).
- Linting: Prefer `go vet`; optional `golangci-lint` if available.

## Testing Guidelines
- Framework: Standard `testing` package.
- Location: Place tests alongside code as `*_test.go`.
- Naming: `TestXxx` for unit tests; table‑driven where helpful.
- Coverage: Aim for meaningful coverage of non‑UI logic. For TUI code, extract logic into pure functions where possible.

## Commit & Pull Request Guidelines
- Commits: Use Conventional Commits where possible: `feat:`, `fix:`, `docs:`, `refactor:`, `chore:`, `test:`. Keep scope small and messages imperative.
- Branching: Short, descriptive branches (e.g., `feat/list-pane`, `fix/escape-key`).
- PRs: Include summary, rationale, screenshots/GIFs for UI changes, and any tradeoffs. Link related issues. Ensure `go fmt`, `go vet`, and tests pass.

## Architecture & Configuration Notes
- UI: Built with `tview` grid layout and `tcell` events. Keep input handling centralized and avoid blocking operations on the UI thread.
- Dependencies: Managed via Go modules; prefer minimal additions.
- Cross‑platform: Target modern terminals; test on macOS/Linux. Windows support depends on terminal capabilities.

## Preferences & Keybindings
- Preferences: Saved to `prefs.json` under the data dir (`BLT_DATA_DIR` or OS default). Persists period (day/week/month), filters (text/type/tags), and last date.
- Keys (navigation): `j/k`, `PgUp/PgDn`, `g/G`, `1/2/3` (Day/Week/Month), `[`/`]` (prev/next), `d` (jump to date), `?` (help).
 - Keys (navigation): `j/k`, `PgUp/PgDn`, `g/G`, `1/2/3` (Day/Week/Month), `[`/`]` (prev/next), `d` (jump to date), `T` (today), `?` (help).
- Keys (filters): `/` text, `:` type (multi‑select), `F` tags.
- Keys (modify, Day view only): `a` add, `e` edit, `x` delete, `c` complete, `m` migrate, `s` schedule, `t` change type, `#` edit tags.

## Project

This `blt` app is a minimalist TUI for bullet-journalling.

It should feature a simple list view of bullets with `j`/`k` scrolling and the ability to add and edit bullets.

It should have a simplified bullet system:
- Tasks
- Completed tasks
- Migrated tasks (moved to tomorrow)
- Task scheduled (moved to a specific date)
- Events
- Notes
- Important highlights
- Inspiration highlights

As well as the ability to edit bullets:
- Delete
- Change type
- Highlight
- Tag
- Mark as completed, migrated or reschedule (with the ability to pick the reschedule date)

It also should have the ability to change the viewed list:
- Switch between day, week or month
- Filter by text, type or tags
- View a specific date (past or future)

Controls should always be a simple as possible and either displayed on screen permanently or on press of a `?` help key.

Data should be stored in the filesystem.
