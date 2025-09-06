# BLT — Minimal TUI Bullet Journal

BLT is a minimalist terminal user interface (TUI) for bullet‑journalling. It focuses on fast keyboard navigation, simple lists, and lightweight on‑disk storage so your notes remain portable and easy to back up.

## Features
- Day/Week/Month views with quick navigation.
- Add, edit, delete, complete, migrate, and schedule tasks.
- Change item type (task, done, event, note, highlights).
- Tags with inline display and filters (text/type/tag).
- Persistent preferences (period, filters, last date).
- Simple JSONL file storage on your filesystem.

## Install & Run
- Prerequisite: Go 1.25+ (see `go.mod`).
- Build: `make build` or `go build -o blt ./cmd/blt`
- Run: `make run` or `./blt`
- Format/Vet/Test: `make fmt`, `make vet`, `make test`

## Data & Preferences
- Data dir (override with `BLT_DATA_DIR`):
  - Linux: `~/.local/share/blt`
  - macOS: `~/Library/Application Support/blt`
  - Windows: `%APPDATA%\blt`
- Storage format: JSON Lines per day at `YYYY/MM/DD.jsonl` (one item per line).
- Preferences: `prefs.json` in the data dir (period, filters, last date).

## Keybindings (Quick)
- Movement: `j/k`, `PgUp/PgDn`, `g/G`
- Scope: `1` Day, `2` Week, `3` Month, `[` Prev, `]` Next, `d` Jump
- Filters: `/` Text, `:` Type (toggle), `F` Tags
- Modify (Day view only): `a` Add, `e/u` Edit, `x` Delete, `c` Complete, `m` Migrate, `s` Schedule, `t` Type, `#` Tags
- Help: `?`

## Development
- Code style: standard Go; use `make fmt vet` before sending PRs.
- Tests: standard `testing` (`make test`).
- Structure:
  - `cmd/blt/`: entrypoint
  - `internal/ui`: TUI, keybindings, overlays
  - `internal/app`: state, actions, filters, periods
  - `internal/model`: domain types
  - `internal/store`: filesystem store and preferences

Notes
- Works best on modern terminals (Linux/macOS). Windows support depends on terminal capabilities.
- In restricted environments without a default data dir, BLT may fall back to a local `.blt-data` folder.
