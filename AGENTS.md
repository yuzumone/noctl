# AGENTS.md — noctl

> This file provides guidance for AI coding agents working on this project. It tracks the current state, architecture, and future goals.

---

## 1. Project Overview

- **Name**: `noctl`
- **Description**: A terminal UI (TUI) tool for browsing and editing Notion databases with high-speed navigation.
- **Language**: Go (1.22+)
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
| **Local Filtering** | ✅ Done | Press `/` in record list to filter results instantly. |
| **Page Detail View** | ✅ Done | Full property list and body content (blocks) rendering with Markdown syntax highlighting via Glamour. |
| **Record Creation** | ✅ Done | Press `n` to create new records based on DB schema. |
| **Record Editing** | ✅ Done | Press `e` to edit existing record properties. |
| **Browser Integration**| ✅ Done | Press `o` to open databases or pages in default browser. |
| **Full-Screen TUI** | ✅ Done | Responsive layout inspired by k9s and Zellij. |
| **Retry Logic** | ✅ Done | Automatic retries on Notion API rate limits (429) and server errors. |

### Note on Image Support
Image rendering was initially implemented using ANSI half-blocks and terminal graphics protocols (Kitty/iTerm2). However, it was removed because:
1. Terminal graphics protocols are inconsistent across environments (TUI vs raw).
2. Large image data caused performance issues and mangled TUI layouts.
3. Half-block rendering provided insufficient quality for general use.
The project now focuses on high-speed text-based workflows.

---

## 3. Project Structure

```
noctl/
├── main.go                    # Entry point: flag handling, config, app launch
├── internal/
│   ├── tui/
│   │   ├── app.go             # Root model and view routing
│   │   ├── dblist.go          # Database selection view
│   │   ├── records.go         # Record list table with local filter
│   │   ├── editor.go          # View/Create/Edit form and block renderer
│   │   ├── browser.go         # OS-native browser opening helper
│   │   ├── keys.go            # Zellij-style footer rendering
│   │   └── styles.go          # Lipgloss style definitions
│   │
│   ├── notion/
│   │   ├── client.go          # API client with retry transport
│   │   ├── database.go        # DB schema and listing
│   │   ├── page.go            # Page CRUD operations
│   │   ├── block.go           # Block retrieval and text conversion
│   │   └── property.go        # Property display conversion
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

### Configuration
`noctl` requires a Notion Internal Integration Token.

1. Create an integration at [notion.so/my-integrations](https://notion.so/my-integrations).
2. Share your databases with the integration.
3. Set the token in `~/.config/noctl/config.yaml`:
```yaml
notion_token: secret_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

### Key Bindings
| Key | Action |
|---|---|
| `j`/`k` | Move selection |
| `Enter` | Select database / View record details |
| `Esc` | Back to previous screen / Cancel |
| `/` | Filter records (local search) |
| `n` | Create new record |
| `e` | Edit current record |
| `o` | Open in web browser |
| `q` | Quit |
| `Ctrl+S` | Save record (in Editor) |

---

## 5. Coding Conventions

- **Error Handling**: Use the `errMsg` type in TUI models to propagate errors to the root model. Prefer descriptive error messages with context.
- **Asynchronous API**: All Notion API calls must be wrapped in `tea.Cmd`.
- **Imports**: Keep imports organized. Use standard library, then external packages, then internal packages.

---

## 6. Future Considerations (TODO)

- [ ] **Pagination**: Support fetching more than 100 records using cursors.
- [ ] **Property Types**: Support more complex types like `Relation`, `Rollup`, and `Date` ranges.
- [ ] **Bulk Operations**: Select multiple records for deletion or property updates.
- [ ] **Configurable Themes**: Allow users to customize Lipgloss styles via YAML.
- [ ] **Search API**: Implement server-side search for large databases.
- [ ] **Deletion**: Add a confirmation dialog to delete records.

---

## 7. Testing

Run unit tests:
```bash
go test ./...
```
- `internal/notion`: Tests for property and block conversion logic.
- `internal/config`: Tests for configuration loading priority.
