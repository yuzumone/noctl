package export

import (
	"fmt"
	"html"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	xhtml "golang.org/x/net/html"
)

var tagPattern = regexp.MustCompile(`</?([a-z][a-z0-9-]*)(?:\s[^<>]*?)?\s*/?>`)
var inlineLink = regexp.MustCompile(`(!?\[[^\]\n]*\]\(<?)(https://(?:(?:www\.)?notion\.(?:so|site|com)|app\.notion\.com)/[^\s)>]+)(>?[ \t]*(?:"[^"\n]*"[ \t]*)?\))`)
var referenceLink = regexp.MustCompile(`(?m)^( {0,3}\[[^\]\n]+\]:[ \t]*<?)(https://(?:(?:www\.)?notion\.(?:so|site|com)|app\.notion\.com)/[^\s>]+)`)

// Notion can serialize an empty-alt image with escaped brackets in rich text.
var escapedEmptyImage = regexp.MustCompile(`!\\\[\\\](\((?:<https?://[^<>\r\n]+>|https?://[^()\s]+)\))`)
var styleLineBreak = regexp.MustCompile(`(?i)<br\s*/?>`)
var styleBlock = regexp.MustCompile(`(?s)\\?<style(?:\s[^<>]*?)?>.*?\\?</style>`)
var shortcode = regexp.MustCompile(`(?s)\{\{([<%])\s*(.*?)\s*([>%])\}\}`)

type codeSpan struct{ start, end int }

// lexicalCodeSpans also recognizes code nested inside Notion XML containers,
// which a CommonMark parser otherwise treats as raw HTML.
func lexicalCodeSpans(source string) []codeSpan {
	spans := []codeSpan{}
	for i := 0; i < len(source); {
		lineStart := i == 0 || source[i-1] == '\n'
		if lineStart {
			j := i
			for j < len(source) && j-i < 3 && source[j] == ' ' {
				j++
			}
			if j < len(source) && (source[j] == '`' || source[j] == '~') {
				marker := source[j]
				k := j
				for k < len(source) && source[k] == marker {
					k++
				}
				if k-j >= 3 {
					openingEnd := strings.IndexByte(source[k:], '\n')
					if openingEnd < 0 {
						spans = append(spans, codeSpan{i, len(source)})
						break
					}
					cursor := k + openingEnd + 1
					end := len(source)
					for cursor < len(source) {
						stop := strings.IndexByte(source[cursor:], '\n')
						if stop < 0 {
							stop = len(source) - cursor
						} else {
							stop++
						}
						line := strings.TrimSpace(source[cursor : cursor+stop])
						run := 0
						for run < len(line) && line[run] == marker {
							run++
						}
						if run >= k-j && strings.TrimSpace(line[run:]) == "" {
							end = cursor + stop
							break
						}
						cursor += stop
					}
					spans = append(spans, codeSpan{i, end})
					i = end
					continue
				}
			}
		}
		if source[i] == '`' && (i == 0 || source[i-1] != '\\') {
			j := i
			for j < len(source) && source[j] == '`' {
				j++
			}
			run := source[i:j]
			cursor := j
			for cursor < len(source) {
				offset := strings.Index(source[cursor:], run)
				if offset < 0 {
					break
				}
				start := cursor + offset
				end := start + len(run)
				if (start == 0 || source[start-1] != '`') && (end == len(source) || source[end] != '`') {
					spans = append(spans, codeSpan{i, end})
					i = end
					break
				}
				cursor = end
			}
			if i >= j {
				continue
			}
			i = j
			continue
		}
		i++
	}
	return spans
}

// protectCode masks code content using the Markdown parser's source positions.
// Arbitrary fence lengths and multiline code spans are preserved when restored.
func protectCode(source string) (string, func(string) string, error) {
	if strings.ContainsRune(source, 0) {
		return "", nil, fmt.Errorf("Markdown contains NUL")
	}
	data := []byte(source)
	tree := goldmark.DefaultParser().Parse(text.NewReader(data))
	spans := []codeSpan{}
	err := ast.Walk(tree, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n.(type) {
		case *ast.CodeSpan:
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if t, ok := c.(*ast.Text); ok {
					spans = append(spans, codeSpan{t.Segment.Start, t.Segment.Stop})
				}
			}
			return ast.WalkSkipChildren, nil
		case *ast.CodeBlock, *ast.FencedCodeBlock:
			for i := 0; i < n.Lines().Len(); i++ {
				s := n.Lines().At(i)
				spans = append(spans, codeSpan{s.Start, s.Stop})
			}
			if f, ok := n.(*ast.FencedCodeBlock); ok && f.Info != nil {
				s := f.Info.Segment
				spans = append(spans, codeSpan{s.Start, s.Stop})
			}
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return "", nil, err
	}
	spans = append(spans, lexicalCodeSpans(source)...)
	// Merge overlapping parser and lexical ranges.
	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
	merged := []codeSpan{}
	for _, s := range spans {
		if len(merged) > 0 && s.start <= merged[len(merged)-1].end {
			if s.end > merged[len(merged)-1].end {
				merged[len(merged)-1].end = s.end
			}
		} else {
			merged = append(merged, s)
		}
	}
	spans = merged
	// Reverse application avoids changing parser offsets.
	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
	var saved []string
	for i := len(spans) - 1; i >= 0; i-- {
		s := spans[i]
		var masked strings.Builder
		for _, line := range strings.SplitAfter(source[s.start:s.end], "\n") {
			if line == "" {
				continue
			}
			content := strings.TrimSuffix(line, "\n")
			token := "\x00" + strconv.Itoa(len(saved)) + "\x00"
			saved = append(saved, content)
			masked.WriteString(token)
			if strings.HasSuffix(line, "\n") {
				masked.WriteByte('\n')
			}
		}
		source = source[:s.start] + masked.String() + source[s.end:]
	}
	restore := func(s string) string {
		for i, original := range saved {
			s = strings.ReplaceAll(s, "\x00"+strconv.Itoa(i)+"\x00", original)
		}
		return s
	}
	return source, restore, nil
}

// preserveStyles keeps raw CSS out of the Notion tag converter. When Notion
// serializes a style block as escaped Markdown, recover its original punctuation.
func preserveStyles(source string) (string, func(string) string) {
	var blocks []string
	masked := styleBlock.ReplaceAllStringFunc(source, func(block string) string {
		if strings.HasPrefix(block, `\<style`) {
			block = string(util.UnescapePunctuations([]byte(block)))
			block = styleLineBreak.ReplaceAllString(block, "\n")
		}
		token := "\x00style" + strconv.Itoa(len(blocks)) + "\x00"
		blocks = append(blocks, block)
		return token
	})
	restore := func(source string) string {
		for i, block := range blocks {
			source = strings.ReplaceAll(source, "\x00style"+strconv.Itoa(i)+"\x00", block)
		}
		return source
	}
	return masked, restore
}

func convertMarkdown(source string, links *resolver) (string, error) {
	masked, restore, err := protectCode(source)
	if err != nil {
		return "", err
	}
	masked, restoreStyles := preserveStyles(masked)
	// Normalize Notion's escaped empty-alt images while code is still masked.
	masked = escapedEmptyImage.ReplaceAllString(masked, "![]${1}")
	// Rewrite original Markdown links before introducing links from Notion tags.
	masked = inlineLink.ReplaceAllStringFunc(masked, func(s string) string {
		m := inlineLink.FindStringSubmatch(s)
		if strings.HasPrefix(m[1], "!") {
			return s
		}
		target, _ := links.resolve(html.UnescapeString(m[2]), "")
		return m[1] + target + m[3]
	})
	masked = referenceLink.ReplaceAllStringFunc(masked, func(s string) string {
		m := referenceLink.FindStringSubmatch(s)
		target, _ := links.resolve(html.UnescapeString(m[2]), "")
		return m[1] + target
	})
	parser := tagParser{source: masked, links: links, restore: restore}
	body, err := parser.parse("")
	if err != nil {
		return "", err
	}
	body = restore(restoreStyles(body))
	// Hugo evaluates shortcodes even inside fenced code. Escape literal syntax.
	body = shortcode.ReplaceAllStringFunc(body, func(s string) string {
		m := shortcode.FindStringSubmatch(s)
		if strings.HasPrefix(m[2], "/*") {
			return s
		}
		return "{{" + m[1] + "/* " + m[2] + " */" + m[3] + "}}"
	})
	return body, nil
}

type tagParser struct {
	source   string
	position int
	links    *resolver
	restore  func(string) string
}

func (p *tagParser) parse(end string) (string, error) {
	var out strings.Builder
	for p.position < len(p.source) {
		loc := tagPattern.FindStringSubmatchIndex(p.source[p.position:])
		if loc == nil {
			out.WriteString(p.source[p.position:])
			p.position = len(p.source)
			break
		}
		start, stop := p.position+loc[0], p.position+loc[1]
		out.WriteString(p.source[p.position:start])
		raw := p.source[start:stop]
		p.position = stop
		// Escaped literal tags are not Notion syntax.
		slashes := 0
		for i := start - 1; i >= 0 && p.source[i] == '\\'; i-- {
			slashes++
		}
		if slashes%2 != 0 {
			out.WriteString(raw)
			continue
		}
		tokenizer := xhtml.NewTokenizer(strings.NewReader(raw))
		typ := tokenizer.Next()
		token := tokenizer.Token()
		name := token.Data
		if typ == xhtml.EndTagToken {
			if name != end {
				return "", fmt.Errorf("unexpected closing tag </%s>", name)
			}
			return out.String(), nil
		}
		attrs := map[string]string{}
		for _, a := range token.Attr {
			attrs[a.Key] = a.Val
		}
		if name == "table" && typ == xhtml.StartTagToken {
			table, err := p.parseTable(attrs)
			if err != nil {
				return "", err
			}
			out.WriteString(table)
			continue
		}
		self := typ == xhtml.SelfClosingTagToken || name == "br"
		child := ""
		if !self && name == "mention" {
			closing := strings.Index(p.source[p.position:], "</mention>")
			if closing < 0 {
				return "", fmt.Errorf("unclosed Notion tag <mention>")
			}
			child = html.UnescapeString(p.source[p.position : p.position+closing])
			p.position += closing + len("</mention>")
		} else if !self {
			var err error
			child, err = p.parse(name)
			if err != nil {
				return "", err
			}
		}
		converted, err := p.convert(name, attrs, child)
		if err != nil {
			return "", err
		}
		out.WriteString(converted)
	}
	if end != "" {
		return "", fmt.Errorf("unclosed Notion tag <%s>", end)
	}
	return out.String(), nil
}

func markdownLink(label, target string) string {
	if label == "" {
		label = target
	}
	label = strings.NewReplacer("\\", "\\\\", "[", "\\[", "]", "\\]", "*", "\\*", "_", "\\_", "`", "\\`", "<", "&lt;", ">", "&gt;", "\n", " ").Replace(label)
	target = strings.NewReplacer("<", "%3C", ">", "%3E", "\n", "%0A", "\r", "%0D").Replace(target)
	return "[" + label + "](<" + target + ">)"
}

func (p *tagParser) convert(name string, a map[string]string, child string) (string, error) {
	switch name {
	case "b":
		return "**" + child + "**", nil
	case "i":
		return "*" + child + "*", nil
	case "s":
		return "~~" + child + "~~", nil
	case "u", "span":
		return child, nil
	case "p":
		return child + "\n\n", nil
	case "h1", "h2", "h3", "h4":
		return strings.Repeat("#", int(name[1]-'0')) + " " + child + "\n\n", nil
	case "br":
		return "  \n", nil
	case "quote", "callout":
		return "> " + strings.ReplaceAll(strings.Trim(child, "\n"), "\n", "\n> ") + "\n\n", nil
	case "columns", "column", "synced-block", "tabs", "tab":
		return child + "\n\n", nil
	case "ul", "ol":
		if name == "ol" {
			lines := strings.Split(strings.TrimRight(child, "\n"), "\n")
			for i, line := range lines {
				if strings.HasPrefix(line, "- ") {
					lines[i] = "1. " + line[2:]
				}
			}
			child = strings.Join(lines, "\n")
		}
		return child + "\n\n", nil
	case "li":
		return "- " + strings.ReplaceAll(strings.TrimRight(child, "\n"), "\n", "\n  ") + "\n", nil
	case "toggle":
		parts := strings.SplitN(strings.Trim(child, "\n"), "\n", 2)
		details := ""
		if len(parts) > 1 {
			details = parts[1]
		}
		return "<details>\n<summary>" + html.EscapeString(parts[0]) + "</summary>\n\n" + details + "\n\n</details>\n\n", nil
	case "todo":
		checked := " "
		if a["checked"] == "true" {
			checked = "x"
		}
		return "- [" + checked + "] " + child + "\n", nil
	case "equation":
		return "$$\n" + child + "\n$$\n\n", nil
	case "mention", "page", "database", "bookmark", "embed", "file", "pdf", "audio", "video", "link-preview", "synced-block-reference":
		target := a["url"]
		if target == "" {
			target = a["source"]
		}
		if target == "" {
			return "", fmt.Errorf("<%s> has no URL", name)
		}
		label := strings.TrimSpace(child)
		if name == "mention" && a["type"] != "database" || name == "page" {
			target, label = p.links.resolve(target, label)
		}
		return markdownLink(label, target), nil
	case "image":
		target := a["source"]
		if target == "" {
			return "", fmt.Errorf("image has no source")
		}
		return "!" + markdownLink(strings.TrimSpace(child), target) + "\n\n", nil
	case "date":
		value := a["start"]
		if value == "" {
			return "", fmt.Errorf("date has no start")
		}
		if a["start-time"] != "" {
			value += " " + a["start-time"]
		}
		if a["end"] != "" {
			value += " – " + a["end"]
		}
		if a["end-time"] != "" {
			value += " " + a["end-time"]
		}
		if a["time-zone"] != "" {
			value += " " + a["time-zone"]
		}
		return value, nil
	default:
		return "", fmt.Errorf("unsupported Notion tag <%s>", name)
	}
}
