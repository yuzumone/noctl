# AGENTS.md — noctl

> This file provides guidance for AI coding agents working on this project. It tracks the current state, architecture, and future goals.

---

## 1. Project Overview

- **Name**: `noctl`
- **Description**: A terminal UI (TUI) tool for browsing and editing Notion databases with high-speed navigation.
- **Language**: Go (1.26+)
- **Category**: CLI / TUI Application

### Goals
- Fast, keyboard-driven interface for Notion.
- Support for viewing and editing common property types.
- Robust handling of large databases with local filtering.
- Minimalistic but highly functional design.

---

## 2. Features Implemented

| Feature | Status | Notes |
|---|---|---|
| **Database Listing** | ✅ Done | Fetches and lists all accessible databases. |
| **Record Browsing** | ✅ Done | Table view of database records with dynamic column widths. |
| **Local Filtering** | ✅ Done | Press `/` in record list to filter results locally. |
| **Pagination** | ✅ Done | Automatically fetches all records from a database using cursors. |
| **Omnisearch** | ✅ Done | Global server-side search across all shared pages and databases (`o` key). |
| **Calendar View** | ✅ Done | Multi-database monthly calendar with automatic date property detection (`calendar` subcommand). |
| **Page Detail View** | ✅ Done | Full property list with Nerd Font icons and body content rendering. |
| **Content Editing** | ✅ Done | Advanced parsing of Markdown into Notion blocks (headings, quotes, todos, etc.). |
| **Record Creation** | ✅ Done | Press `a` (add) to create new records based on DB schema. |
| **Record Editing** | ✅ Done | Press `e` to edit existing record properties or `E` to edit content in external editor. |
| **Browser Integration**| ✅ Done | Press `b` (browser) to open databases or pages in default browser. |
| **Full-Screen TUI** | ✅ Done | Responsive layout with navigation history stack and popup overlays. |
| **Layered Rendering** | ✅ Done | Custom overlay logic to show popups without hiding the background main view. |
| **CI/CD** | ✅ Done | GitHub Actions for automated formatting (go fmt), linting (golangci-lint), and testing. |
| **Retry Logic** | ✅ Done | Automatic retries on Notion API rate limits (429) and server errors. |

---

## 3. Project Structure

```
noctl/
├── main.go                    # Entry point: Cobra-based command handling, config, app launch
├── .golangci.yml              # Linter configuration
├── .github/workflows/ci.yml   # GitHub Actions workflow (fmt, vet, lint, test)
├── internal/
│   ├── tui/
│   │   ├── app.go             # Root model, view routing, history, and popup overlay logic
│   │   ├── dblist.go          # Database selection view
│   │   ├── records.go         # Record list table with local filter
│   │   ├── calendar.go        # Multi-database monthly calendar view
│   │   ├── table_selector.go  # Table-based record selection popup (used in Calendar)
│   │   ├── editor.go          # View/Create/Edit form and block renderer
│   │   ├── omnisearch.go      # Global search popup component
│   │   ├── selector.go        # Option selection popup component for Select/MultiSelect properties
│   │   ├── confirm.go         # Yes/No confirmation dialog popup component
│   │   ├── browser.go         # OS-native browser opening helper
│   │   ├── keys.go            # Zellij-style footer rendering
│   │   └── styles.go          # Lipgloss style definitions
│   │
│   ├── notion/
│   │   ├── client.go          # API client with retry transport and raw HTTP search
│   │   ├── database.go        # DB schema and listing
│   │   ├── page.go            # Page CRUD operations
│   │   ├── block.go           # Block parsing, MD conversion, and surgical updates
│   │   └── property.go        # Property display and icon mapping
│   │
│   └── config/
│       └── config.go          # Viper-based configuration loading
```

---

## 4. Usage

### Installation
```bash
go build -o noctl .
```

### Key Bindings
| Key | Action |
|---|---|
| `hjkl` / `↑`/`↓`/`←`/`→` / `Ctrl+n`/`Ctrl+p`| Move selection (Daily movement in Calendar) |
| `H` / `L` | Prev/Next Month (in Calendar) |
| `Enter` | Select / View details / Confirm |
| `Esc` | Back to previous screen (pop history) / Cancel |
| `o` | **Omnisearch** (Global search popup) |
| `b` | Open in web **browser** |
| `/` | Local filter (in list views) |
| `a` | Add new record |
| `e` | Edit properties |
| `E` | Edit body content in external editor |
| `q` | Quit (shows confirmation dialog) |
| `Ctrl+S` | Save record (in Editor) |

---

## 5. Coding Conventions

- **State Management**: Use the pointer receiver `(*AppModel)` for the root model. Track navigation using the `history` stack.
- **Error Handling**: Use the `errMsg` type in TUI models. Explicitly check return values (enforced by CI).
- **Asynchronous API**: All Notion API calls must be wrapped in `tea.Cmd`.
- **Styling**: Prefer `lipgloss` for all UI components. Avoid hardcoded background colors to respect terminal themes. Use `UnsetBackground()`.
- **Layered Rendering**: Use `renderWithPopup` in `app.go` to overlay popups on the active background view.
- **Documentation**: All exported symbols must have a doc comment (enforced by CI).

---

## 6. Testing

Run unit tests:
```bash
go test ./...
```
- `internal/notion`: Tests for block parsing and markdown conversion.
- `internal/config`: Tests for configuration loading.
- `golangci-lint`: Run locally via `golangci-lint run`.
