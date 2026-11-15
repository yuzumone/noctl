package export

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
	"noctl/internal/notion"
)

type fakeSource struct {
	properties map[string]map[string]any
	calls      map[string]int
	content    *notion.PageMarkdown
}

func (s *fakeSource) ExportPageProperties(_ context.Context, id string) (map[string]any, error) {
	s.calls[id]++
	p, ok := s.properties[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return p, nil
}
func (s *fakeSource) ExportPageReference(ctx context.Context, id string) (map[string]any, string, error) {
	properties, err := s.ExportPageProperties(ctx, id)
	title, _ := properties["title"].(string)
	return properties, title, err
}
func (s *fakeSource) RetrievePageMarkdown(context.Context, string) (*notion.PageMarkdown, error) {
	return s.content, nil
}

const linkedID = "123456781234123412341234567890ab"

func testSource(body string) *fakeSource {
	return &fakeSource{properties: map[string]map[string]any{"root": {"slug": "example", "title": "Example", "draft": false, "zero": float64(0), "empty": "", "nil": nil, "date": "2026-11-03"}, linkedID: {"slug": "日本語 #?", "title": "記事B"}}, calls: map[string]int{}, content: &notion.PageMarkdown{Markdown: body}}
}

func TestPageExport(t *testing.T) {
	s := testSource(`<mention url="https://www.notion.so/` + linkedID + `">記事B</mention>` + "\n\n" + `[again](https://www.notion.so/` + linkedID + `)`)
	doc, err := Page(context.Background(), s, "root", "/articles/{slug}/")
	if err != nil {
		t.Fatal(err)
	}
	target := "/articles/%E6%97%A5%E6%9C%AC%E8%AA%9E%20%23%3F/"
	if !strings.Contains(string(doc.Markdown), target) || s.calls[linkedID] != 1 {
		t.Fatalf("bad resolution/caching: %s, calls %v", doc.Markdown, s.calls)
	}
	parts := strings.SplitN(string(doc.Markdown), "---\n", 3)
	var front map[string]any
	if err := yaml.Unmarshal([]byte(parts[1]), &front); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"draft", "zero", "empty", "nil"} {
		if _, ok := front[key]; !ok {
			t.Fatal("lost", key)
		}
	}
	if _, ok := front["date"].(string); !ok {
		t.Fatalf("date is not a string: %#v", front["date"])
	}
}

func TestMarkdownCodeAndTags(t *testing.T) {
	url := "https://www.notion.so/" + linkedID
	tests := []struct {
		name, input, want string
		fail              bool
	}{
		{"embed", `<embed source="https://example.com?a=1&amp;b=2">Example</embed>`, `[Example](<https://example.com?a=1&b=2>)`, false},
		{"quote", "<quote>one\n\ntwo</quote>", "> one\n> \n> two", false},
		{"nested", `<callout>hello <b>world</b></callout>`, "> hello **world**", false},
		{"inline code", "``<mention url=\"" + url + "\">B</mention> `literal` ``", "<mention url=\"" + url + "\">B</mention>", false},
		{"long fence", "````markdown\n<unknown/>\n```\n````\n", "<unknown/>", false},
		{"tilde fence", "~~~~\n<unknown/>\n~~~\n~~~~\n", "<unknown/>", false},
		{"indented code", "    <unknown/>\n", "<unknown/>", false},
		{"escaped", `\<unknown/>`, `\<unknown/>`, false},
		{"unknown", "<unknown/>", "", true},
		{"container code", "<callout>Example\n\n````\n<unknown/>\n```\n````\n</callout>", "<unknown/>", false},
		{"container inline", "<callout>Use `<unknown/>` here</callout>", "`<unknown/>`", false},
		{"unclosed", "<callout>text", "", true},
		{"shortcode", `{{< embed url="https://example.com" >}}`, `{{</* embed url="https://example.com" */>}}`, false},
		{"image", "![image](" + url + ")", "![image](" + url + ")", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testSource(tt.input)
			r, _ := newResolver(context.Background(), s, "/articles/{slug}/")
			got, err := convertMarkdown(tt.input, r)
			if (err != nil) != tt.fail {
				t.Fatalf("err=%v, output=%s", err, got)
			}
			if !tt.fail && !strings.Contains(got, tt.want) {
				t.Fatalf("got %q want %q", got, tt.want)
			}
			if (strings.Contains(tt.name, "code") || strings.Contains(tt.name, "fence") || tt.name == "image") && len(s.calls) > 0 {
				t.Fatal("fetched inside code/image")
			}
		})
	}
}

func TestIncompleteAndInvalidExport(t *testing.T) {
	for _, kind := range []string{"truncated", "unknown", "slug", "template"} {
		t.Run(kind, func(t *testing.T) {
			s := testSource("body")
			template := ""
			switch kind {
			case "truncated":
				s.content.Truncated = true
			case "unknown":
				s.content.UnknownBlockIDs = []string{"block"}
			case "slug":
				s.properties["root"]["slug"] = "../escape"
			case "template":
				template = "/articles/{slug"
			}
			if _, err := Page(context.Background(), s, "root", template); err == nil {
				t.Fatal("accepted invalid export")
			}
		})
	}
}

func TestLinkFallbacks(t *testing.T) {
	for _, suffix := range []string{"#block", "", "?pvs=4"} {
		s := testSource("")
		delete(s.properties, linkedID)
		r, _ := newResolver(context.Background(), s, "/articles/{slug}/")
		link := "https://www.notion.so/" + linkedID + suffix
		got, _ := r.resolve(link, "B")
		r.resolve(link, "B")
		if got != link || len(r.warnings) != 1 {
			t.Fatalf("%q %v", got, r.warnings)
		}
		if s.calls[linkedID] > 1 {
			t.Fatal("repeated failed fetch")
		}
	}
	s := testSource("")
	r, _ := newResolver(context.Background(), s, "")
	r.resolve("https://www.notion.so/"+linkedID, "")
	if len(s.calls) != 0 {
		t.Fatal("unexpected fetch without template")
	}
}

func TestWrite(t *testing.T) {
	dir := t.TempDir()
	doc := &Document{Slug: "example", Markdown: []byte("new")}
	path, err := Write(dir, doc, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Write(dir, doc, false); err == nil {
		t.Fatal("overwrote existing file")
	}
	doc.Markdown = []byte("replacement")
	if _, err := Write(dir, doc, true); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if string(b) != "replacement" {
		t.Fatal(string(b))
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(dir, "other")
	if err := os.WriteFile(other, []byte("safe"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(other, path); err != nil {
		t.Fatal(err)
	}
	if _, err := Write(dir, doc, true); err == nil {
		t.Fatal("replaced symlink")
	}
	b, _ = os.ReadFile(other)
	if string(b) != "safe" {
		t.Fatal("changed symlink destination")
	}
	nested := filepath.Join(dir, "missing", "articles")
	created, err := Write(nested, doc, false)
	if err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(created); err != nil || string(b) != "replacement" {
		t.Fatalf("nested output: %q, %v", b, err)
	}
	if _, err := Write(filepath.Join(other, "articles"), doc, false); err == nil {
		t.Fatal("accepted regular file as parent directory")
	}
}

func TestEscapedEmptyImages(t *testing.T) {
	image := `!\[\](https://img.example.com/photo.webp)`
	tests := []struct{ name, source, want string }{
		{"plain", image, `![](https://img.example.com/photo.webp)`},
		{"angle destination", `!\[\](<https://img.example.com/a(b).webp>)`, `![](<https://img.example.com/a(b).webp>)`},
		{"normal image", `![caption](https://img.example.com/photo.webp)`, `![caption](https://img.example.com/photo.webp)`},
		{"inline code", "`" + image + "`", "`" + image + "`"},
		{"fenced code", "```markdown\n" + image + "\n```", "```markdown\n" + image + "\n```"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testSource(tt.source)
			resolver, err := newResolver(context.Background(), s, "")
			if err != nil {
				t.Fatal(err)
			}
			got, err := convertMarkdown(tt.source, resolver)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStyleBlocks(t *testing.T) {
	escaped := "\\<style>  \n.post-content table \\{  \ntable-layout: fixed;  \n\\}  \n\\</style>"
	raw := "<style>  \n.post-content table {  \ntable-layout: fixed;  \n}  \n</style>"
	tests := []struct{ name, source, want string }{
		{"Notion escaped CSS", escaped, raw},
		{"Notion CSS line breaks", `\<style><br />.post-content td \{<br />text-align: center;<br />\}<br />\</style>`, "<style>\n.post-content td {\ntext-align: center;\n}\n</style>"},
		{"raw CSS", raw, raw},
		{"inline style", `<style>.post-content td { color: red; }</style>`, `<style>.post-content td { color: red; }</style>`},
		{"CSS escapes", `<style>.icon::before { content: "\2605"; }</style>`, `<style>.icon::before { content: "\2605"; }</style>`},
		{"inline code", "`" + `\<style>a \{ color: red; \}\</style>` + "`", "`" + `\<style>a \{ color: red; \}\</style>` + "`"},
		{"fenced code", "```html\n" + escaped + "\n```", "```html\n" + escaped + "\n```"},
		{"other literal tags", `\<example> \{literal\} \</example>`, `\<example> \{literal\} \</example>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := testSource(tt.source)
			resolver, err := newResolver(context.Background(), source, "")
			if err != nil {
				t.Fatal(err)
			}
			got, err := convertMarkdown(tt.source, resolver)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInlineCodeTags(t *testing.T) {
	for _, tt := range []struct{ input, want string }{
		{`<code>$HOME/.config/direnv/direnvrc</code>`, "`$HOME/.config/direnv/direnvrc`"},
		{`あとは <code>use flake</code> です`, "あとは `use flake` です"},
		{"<code>a`b</code>", "``a`b``"},
		{"<code>`x`</code>", "`` `x` ``"},
		{`<code>&lt;span&gt;&amp;</code>`, "`<span>&`"},
		{`<code><span>literal</span></code>`, "`<span>literal</span>`"},
		{`<code> x </code>`, "`  x  `"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			r, err := newResolver(context.Background(), testSource(""), "")
			if err != nil {
				t.Fatal(err)
			}
			got, err := convertMarkdown(tt.input, r)
			if err != nil || got != tt.want {
				t.Fatalf("got %q, error %v; want %q", got, err, tt.want)
			}
		})
	}
}

func TestImportedArticleLinks(t *testing.T) {
	for _, tt := range []struct{ template, link, want string }{
		{"/articles/{slug}/", "https://app.notion.com/articles/example/", "/articles/example/"},
		{"/posts/{slug}/", "https://app.notion.com/articles/example/", "/posts/example/"},
		{"", "https://app.notion.com/articles/example/", "https://app.notion.com/articles/example/"},
		{"/articles/{title}/", "https://app.notion.com/articles/example/", "https://app.notion.com/articles/example/"},
		{"/articles/{slug}/", "https://app.notion.com/articles/example/#block", "https://app.notion.com/articles/example/#block"},
		{"/articles/{slug}/", "https://example.com/articles/example/", "https://example.com/articles/example/"},
	} {
		t.Run(tt.template+tt.link, func(t *testing.T) {
			s := testSource("")
			r, err := newResolver(context.Background(), s, tt.template)
			if err != nil {
				t.Fatal(err)
			}
			got, label := r.resolve(tt.link, "Article")
			if got != tt.want || label != "Article" {
				t.Fatalf("got %q %q", got, label)
			}
			if len(s.calls) != 0 {
				t.Fatal("slug links must not call page API")
			}
		})
	}
}
