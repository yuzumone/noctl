# noctl

`noctl` is a fast, keyboard-driven Terminal User Interface (TUI) tool for browsing and editing Notion databases. It's designed for efficiency, allowing you to manage your Notion data without leaving your terminal.

![noctl Calendar](https://via.placeholder.com/800x400?text=noctl+Calendar+View+Placeholder)

## Features

- 🚀 **High Speed**: Built with Go for maximum performance.
- 📅 **Unified Calendar**: View multiple Notion databases in a single, aggregated monthly calendar.
- 🔍 **Omnisearch**: Global search across all your shared pages and databases.
- 📑 **Table View**: Dynamic, responsive table view for database records.
- 📝 **Markdown Editor**: Edit page content using familiar Markdown syntax in your preferred external editor.
- 🎨 **Themeable**: Customizable colors to match your terminal theme.
- ⌨️ **Vim-style Keys**: Intuitive navigation with `hjkl` and other keyboard shortcuts.

## Installation

### From Source

Ensure you have Go 1.26 or later installed.

```bash
git clone https://github.com/yourusername/noctl.git
cd noctl
go build -o noctl .
```

## Configuration

`noctl` looks for configuration in the following locations:
1. Environment variable: `NOCTL_NOTION_TOKEN`
2. `~/.config/noctl/config.yaml`
3. `~/.noctl.yaml`

### Example `config.yaml`

```yaml
notion_token: secret_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
default_database_id: "your_default_db_uuid"
calendar_database_ids:
  - "uuid_1"
  - "uuid_2"

theme:
  accent: "#7D56F4"
  select_bg: "#5F00FF"
```

## Usage

### Commands

| Command | Description |
|---|---|
| `noctl` | Launch the main TUI (starts with database list). |
| `noctl calendar` | Launch the integrated multi-database calendar view. |
| `noctl version` | Print the current version. |
| `noctl --help` | Show usage information. |

### Key Bindings

#### General
| Key | Action |
|---|---|
| `q` | Quit (with confirmation) |
| `Esc` | Back to previous screen / Cancel |
| `o` | Open **Omnisearch** |
| `Ctrl+c` | Force quit |

#### Navigation & Lists
| Key | Action |
|---|---|
| `hjkl` / `↑↓←→` | Move selection |
| `Enter` | Select / View details / Confirm |
| `b` | Open in web **browser** (in list views) |
| `/` | Filter items locally |
| `a` | Add new record |
| `e` | Edit properties |

#### Calendar View
| Key | Action |
|---|---|
| `hjkl` / `↑↓←→` | Move selected day |
| `H` / `L` | Previous / Next month |
| `Enter` | Show list of records for the selected day |

#### Editor
| Key | Action |
|---|---|
| `E` | Edit body content in external editor (`$EDITOR`) |
| `Ctrl+s` | Save changes |

## Development

### Running Tests
```bash
go test ./...
```

### Linting
```bash
golangci-lint run ./...
```

### Building with Version
```bash
go build -ldflags "-X noctl/internal/version.Version=v0.1.0" -o noctl .
```

## License

[MIT License](LICENSE)
