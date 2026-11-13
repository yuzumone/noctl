package export

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
)

func TestStyledTableExport(t *testing.T) {
	source := `<table header-row="true">
<colgroup>
<col width="376.328125" />
<col width="233.328125" />
<col width="330.3333333333333" />
</colgroup>
<tr><td>Image</td><td>Date</td><td>Price</td></tr>
<tr><td>!\[\](https://img.example.com/photo.webp)</td><td>2026-10-22</td><td>1200</td></tr>
<tr><td><mention url="https://www.notion.so/` + linkedID + `">Related</mention></td><td><b>bold</b><br />next</td><td>` + "`a|b`" + `</td></tr>
</table>`
	s := testSource(source)
	s.properties[linkedID]["slug"] = "related"
	doc, err := Page(context.Background(), s, "root", "/articles/{slug}/")
	if err != nil {
		t.Fatal(err)
	}
	body := strings.SplitN(string(doc.Markdown), "---\n", 3)[2]
	for _, expected := range []string{
		"| Image | Date | Price |",
		"| --- | --- | --- |",
		"| ![](https://img.example.com/photo.webp) | 2026-10-22 | 1200 |",
		"| [Related](</articles/related/>) | **bold**<br>next | `a\\|b` |",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("missing %q in %s", expected, body)
		}
	}
	var rendered bytes.Buffer
	md := goldmark.New(goldmark.WithExtensions(extension.Table), goldmark.WithRendererOptions(goldmarkhtml.WithUnsafe()))
	if err := md.Convert([]byte(body), &rendered); err != nil {
		t.Fatal(err)
	}
	result := rendered.String()
	if strings.Count(result, "<th>") != 3 || strings.Count(result, "<td>") != 6 || !strings.Contains(result, "<code>a|b</code>") || !strings.Contains(result, `<img src="https://img.example.com/photo.webp" alt="">`) {
		t.Fatalf("invalid rendered table: %s", result)
	}
}

func TestTableHeadersAndCells(t *testing.T) {
	tests := []struct{ name, source, want string }{
		{"no header", `<table><tr><td>A</td><td>B</td></tr></table>`, "|  |  |\n| --- | --- |\n| A | B |"},
		{"header column", `<table header-column="true"><tr><td>A</td><td>B</td></tr></table>`, "| **A** | B |"},
		{"cell escapes", `<table header-row="true"><tr><th>Header</th></tr><tr><td>a\|b | c<br>d</td></tr></table>`, "| a\\|b \\| c<br>d |"},
		{"empty cell", `<table header-row="true"><tr><td>Header</td></tr><tr><td /></tr></table>`, "| Header |\n| --- |\n|  |"},
		{"code is literal", "```html\n<table><col width=\"100\" /></table>\n```", "<table><col width=\"100\" /></table>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testSource(tt.source)
			r, _ := newResolver(context.Background(), s, "")
			got, err := convertMarkdown(tt.source, r)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(got, tt.want) {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMalformedTableFails(t *testing.T) {
	for _, source := range []string{
		`<table><tr><td>missing close</td></tr>`,
		`<table><tr><td>A</td></tr><tr><td>B</td><td>C</td></tr></table>`,
		`<table><tr><td colspan="2">merged</td></tr></table>`,
		`<table><td>no row</td></table>`,
		`<table><tr><td><unknown /></td></tr></table>`,
		`<table>unexpected text<tr><td>A</td></tr></table>`,
	} {
		s := testSource(source)
		if _, err := Page(context.Background(), s, "root", ""); err == nil {
			t.Fatalf("accepted invalid table %s", source)
		}
	}
}

func TestStyleBlockRendersAsCSS(t *testing.T) {
	source := `\<style>
.post-content td \{
text-align: center;
\}
\</style>`
	doc, err := Page(context.Background(), testSource(source), "root", "")
	if err != nil {
		t.Fatal(err)
	}
	body := strings.SplitN(string(doc.Markdown), "---\n", 3)[2]
	var rendered bytes.Buffer
	md := goldmark.New(goldmark.WithRendererOptions(goldmarkhtml.WithUnsafe()))
	if err := md.Convert([]byte(body), &rendered); err != nil {
		t.Fatal(err)
	}
	expected := "<style>\n.post-content td {\ntext-align: center;\n}\n</style>"
	if !strings.Contains(rendered.String(), expected) {
		t.Fatalf("CSS was not rendered as a style element: %s", rendered.String())
	}
}
