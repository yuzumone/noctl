# noctl

`noctl` is a fast, keyboard-driven Terminal User Interface (TUI) tool for browsing and editing Notion databases. It's designed for efficiency, allowing you to manage your Notion data without leaving your terminal.

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

Ensure you have Go 1.27.4 or later installed.

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
| `noctl export page <page-id>` | Export a page as Hugo Markdown. |
| `noctl version` | Print the current version. |
| `noctl --help` | Show usage information. |

### Export a page for Hugo

```sh
# Write ./<slug>.md (the default output directory is ./).
noctl export page <page-id> --format hugo

# Write into an existing content directory and resolve page references.
noctl export page <page-id> --output ./content/articles \
  --link-template '/articles/{slug}/'

# Read and convert the page without writing any files.
noctl export page <page-id> --link-template '/articles/{slug}/' --dryrun

# Explicitly allow replacing an existing regular file.
noctl export page <page-id> --output ./content/articles --force
```

Export is CLI-only and uses the existing Notion token configuration. Share the
page and any referenced pages with that integration. Only page IDs are accepted
as arguments; use `noctl export database <database-id>` (or `export db`) to export all pages. Database export accepts the same flags, converts all pages before writing, and rejects duplicate slugs. Files are written individually; a write failure may leave earlier files exported. The supported format is
`hugo`, which is also the default.

Property names become YAML front matter keys without renaming or Hugo-specific
mapping. A non-empty string property named `slug` is required, including for
`--dryrun`. Slugs must be single filenames: path separators, control characters,
`<>:"|?*`, surrounding whitespace, and trailing dots are rejected. The exporter
creates missing output directories, but never falls back to the title, renames old
files, or deletes previous exports. Existing directories and symlinks cannot be overwritten.
Completed output is published from a temporary file in the destination directory;
failures before publication leave an existing output unchanged. Without `--force`,
a file created concurrently is not overwritten. Redirecting stdout with a shell
bypasses these protections; use `--output` for file output.

Property conversion:

| Notion type | YAML value |
|---|---|
| title, rich_text | Concatenated plain text, without styling |
| number, checkbox | Number or boolean (including `0` and `false`) |
| select, status | Option name |
| multi_select | Array of option names |
| url, email, phone_number | String |
| date | Start string; `{start, end, time_zone}` with only present fields when extra information exists |
| people, relation | Array of user or page IDs |
| created_by, last_edited_by | User ID |
| created_time, last_edited_time | Timestamp string |
| files | Array of `{name, url}` objects; files are not downloaded |
| unique_id | `{prefix, number}` |
| formula, rollup | Converted calculation result, recursively for arrays |

Empty strings, empty arrays, null values, false, and zero are retained. Unknown
property types and incomplete calculations fail the export. Paginated properties
are retrieved completely before conversion. The Notion schema must use names and
value types suitable for Hugo (for example, a single date for Hugo's `date`).

Common Notion tags are converted to Markdown: callouts/quotes to blockquotes,
toggles to HTML details, and embeds/bookmarks to ordinary links. Ordinary
Markdown and code are preserved; literal Hugo shortcode syntax is escaped so
Hugo does not execute it. Unknown or unsupported Notion tags fail rather than
silently losing content. Image URLs are preserved without downloading or
rewriting; Notion-hosted expiring URLs will still expire.
Raw `<style>` blocks are preserved, and Markdown punctuation escapes added by
Notion within escaped style blocks are removed. Style examples in code stay literal.

With `--link-template`, page mentions and ordinary Notion Markdown links use the
referenced page's properties to expand placeholders such as `{slug}`. Values
must be non-empty strings or numbers and are URL-escaped as individual path
segments. The template must be a site-relative path, for example
`/articles/{slug}/`. Referenced properties are cached for one execution; referenced
bodies are not retrieved. Missing properties, inaccessible pages, and block
anchors keep their Notion URLs and produce warnings. The exporter does not
verify that the resulting public URLs exist or that linked pages are published.
Without this option, references retain Notion URLs without extra API requests.

Truncated Markdown, unknown blocks, and failed property retrieval prevent output.
No concurrent-edit detection is performed. Warnings and saved paths go to stderr;
`--dryrun` writes only the completed document to stdout. Warnings allow a successful
exit; retrieval, conversion, validation, and write failures return a nonzero exit.

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

Page bodies use Notion's official Markdown API with `markdown_version=v2`.
The external editor receives the original Notion-flavored Markdown, including
Notion-specific tags. Saving replaces the page body after checking that it has
not changed since editing began. Child-page and database deletion protection
remains enabled. Incomplete pages can be viewed but cannot be edited here.
If saving fails or Notion reports warnings, the edited file is retained and its
path is shown so the content can be recovered. Common tags are adapted for the
terminal preview; complex Notion layouts may display differently.

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
