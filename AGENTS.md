# AGENTS.md - noctl

This file provides guidance for AI coding agents working on this project. Keep it focused on how to work safely and consistently in this repository.

---

## Project Summary

- **Name**: `noctl`
- **Description**: A terminal UI (TUI) tool for browsing and editing Notion databases with high-speed navigation.
- **Language**: Go. Follow the Go version declared in `go.mod`.
- **Category**: CLI / TUI application.

`noctl` aims to provide a fast, vim-inspired keyboard-driven Notion interface with robust database browsing, local filtering, page detail views, and record editing.

---

## How To Work In This Repo

- Prefer small, focused changes that match the existing package boundaries.
- Keep Notion API access, request/response conversion, retry behavior, and property/block conversion in `internal/notion`.
- Keep Bubble Tea models, routing, keyboard handling, rendering, and popup behavior in `internal/tui`.
- Keep configuration loading and config-specific validation in `internal/config`.
- Wrap asynchronous work, especially Notion API calls, in `tea.Cmd`.
- Preserve existing keybindings unless the task explicitly changes them.
- Avoid hardcoded terminal background colors. Prefer `lipgloss` styles that respect the user's terminal theme and use `UnsetBackground()` where appropriate.
- Add or update tests when changing parsing, conversion, config loading, or behavior that can be checked without an interactive terminal.

---

## Architecture Notes

```
noctl/
|-- main.go                    # Cobra entry point, config loading, app launch
|-- .golangci.yml              # Linter configuration
|-- .github/workflows/
|   |-- ci.yml                 # Formatting, vet, build, lint, and tests
|   `-- release.yml            # Release workflow
|-- Makefile                   # Multi-platform build and packaging orchestration
|-- VERSION                    # Version source
`-- internal/
    |-- tui/                   # Bubble Tea models, views, key handling, rendering
    |-- notion/                # Notion client, database/page/block/property logic
    `-- config/                # Viper-based configuration loading
```

Important files:

- `internal/tui/app.go`: root model, view routing, history stack, and popup overlay rendering.
- `internal/tui/records.go`: record table view and local filtering.
- `internal/tui/editor.go`: page detail, create/edit form, and content rendering.
- `internal/tui/calendar.go`: multi-database monthly calendar view.
- `internal/notion/client.go`: Notion API client and retry transport.
- `internal/notion/block.go`: Markdown/block conversion and content update logic.
- `internal/notion/property.go`: property display and icon mapping.

---

## TUI Rules

- Use pointer receivers for root app state: `(*AppModel)`.
- Maintain navigation through the `history` stack.
- Treat vim-style navigation as the baseline interaction model.
- Render popups through `renderWithPopup` so overlays do not erase the background view.
- Use the existing popup components for selection and confirmation flows where possible.
- Keep list filtering local unless the feature explicitly requires server-side search.
- Keep browser-opening behavior centralized in `internal/tui/browser.go`.

---

## Current User-Facing Features

- Database listing.
- Record browsing with dynamic column widths.
- Local filtering with `/`.
- Pagination through Notion cursors.
- Global omnisearch with `o`.
- Multi-database calendar view through the `calendar` subcommand.
- Page detail view with property list and body rendering.
- Record creation with `a`.
- Record property editing with `e`.
- Page content editing in an external editor with `E`.
- Browser integration with `b`.
- Confirmation dialog on quit with `q`.

---

## Testing And CI

CI is defined in `.github/workflows/ci.yml`. When relevant to the change, run the same checks locally:

```bash
go mod download
```

```bash
x=$(gofmt -l .); if [ -n "$x" ]; then echo "Following files are not formatted:"; echo "$x"; exit 1; fi
```

```bash
go vet ./...
```

```bash
go build -v -o /dev/null ./main.go
```

```bash
go test -v ./...
```

```bash
golangci-lint run
```

Notes:

- `golangci-lint` runs in a separate CI job using the version configured in the GitHub Actions workflow.
- Exported Go symbols must have doc comments because linting enforces this.
- Tests currently focus on packages such as `internal/notion` and `internal/config`.
