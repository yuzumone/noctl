package export

import (
	"fmt"
	"strings"

	xhtml "golang.org/x/net/html"
)

// parseTable consumes Notion's styled table while letting the normal parser
// convert inline content in each cell. Column widths and colors are presentation
// metadata and are omitted from the Markdown table.
func (p *tagParser) parseTable(attrs map[string]string) (string, error) {
	var rows [][]string
	var row []string
	inRow, inColumns := false, false
	for p.position < len(p.source) {
		loc := tagPattern.FindStringIndex(p.source[p.position:])
		if loc == nil {
			break
		}
		start, stop := p.position+loc[0], p.position+loc[1]
		if strings.TrimSpace(p.source[p.position:start]) != "" {
			return "", fmt.Errorf("unexpected text outside a table cell")
		}
		raw := p.source[start:stop]
		p.position = stop
		tokenizer := xhtml.NewTokenizer(strings.NewReader(raw))
		typ := tokenizer.Next()
		token := tokenizer.Token()
		closing := typ == xhtml.EndTagToken
		switch token.Data {
		case "colgroup":
			if inRow || closing != inColumns {
				return "", fmt.Errorf("invalid table column group")
			}
			inColumns = !closing
		case "col":
			if closing || !inColumns {
				return "", fmt.Errorf("column outside a table column group")
			}
		case "tr":
			if inColumns || closing != inRow {
				return "", fmt.Errorf("invalid table row")
			}
			if closing {
				if len(row) == 0 {
					return "", fmt.Errorf("empty table row")
				}
				rows = append(rows, row)
			} else {
				row = nil
			}
			inRow = !closing
		case "td", "th":
			if closing || !inRow {
				return "", fmt.Errorf("cell outside a table row")
			}
			for _, a := range token.Attr {
				if (a.Key == "colspan" || a.Key == "rowspan") && a.Val != "1" {
					return "", fmt.Errorf("merged table cells are unsupported")
				}
			}
			var cell string
			if typ != xhtml.SelfClosingTagToken {
				var err error
				cell, err = p.parse(token.Data)
				if err != nil {
					return "", err
				}
			}
			// Restore code before escaping pipes so a pipe inside a code span cannot
			// become an extra Markdown column.
			cell = p.restore(cell)
			cell = escapeTableCell(cell)
			if cell != "" && (token.Data == "th" || attrs["header-column"] == "true" && len(row) == 0) {
				cell = "**" + cell + "**"
			}
			row = append(row, cell)
		case "table":
			if !closing || inRow || inColumns {
				return "", fmt.Errorf("invalid or nested table")
			}
			return renderTable(rows, attrs["header-row"] == "true")
		default:
			return "", fmt.Errorf("unsupported tag <%s> inside table", token.Data)
		}
	}
	return "", fmt.Errorf("unclosed Notion tag <table>")
}

func escapeTableCell(cell string) string {
	lines := strings.Split(strings.TrimSpace(cell), "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	cell = strings.Join(lines, "<br>")
	var out strings.Builder
	backslashes := 0
	for _, r := range cell {
		if r == '|' && backslashes%2 == 0 {
			out.WriteByte('\\')
		}
		out.WriteRune(r)
		if r == '\\' {
			backslashes++
		} else {
			backslashes = 0
		}
	}
	return out.String()
}

func renderTable(rows [][]string, header bool) (string, error) {
	if len(rows) == 0 {
		return "", fmt.Errorf("empty table")
	}
	columns := len(rows[0])
	for _, row := range rows {
		if len(row) != columns {
			return "", fmt.Errorf("table rows have inconsistent column counts")
		}
	}
	if !header {
		// Pipe tables require a header. An empty one preserves all original data rows.
		rows = append([][]string{make([]string, columns)}, rows...)
	}
	var out strings.Builder
	out.WriteString("\n\n")
	for i, row := range rows {
		out.WriteString("| ")
		out.WriteString(strings.Join(row, " | "))
		out.WriteString(" |\n")
		if i == 0 {
			out.WriteString("|")
			for range columns {
				out.WriteString(" --- |")
			}
			out.WriteByte('\n')
		}
	}
	out.WriteByte('\n')
	return out.String(), nil
}
