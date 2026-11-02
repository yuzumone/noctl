package tui

import (
	"html"
	"regexp"
	"strings"
)

var notionTag = regexp.MustCompile(`</?([a-z][a-z0-9-]*)(?:\s[^<>]*?)?\s*/?>`)
var notionURL = regexp.MustCompile(`(?:url|source)="([^"]*)"`)
var protectedMarkdown = regexp.MustCompile("(?s)(`+)([^`]*)(`+)")

// markdownForDisplay adapts common Notion tags for Glamour. The original
// document is kept separately and is the only content passed to the editor/API.
func markdownForDisplay(markdown string) string {
	var out strings.Builder
	fence := ""
	for _, line := range strings.SplitAfter(markdown, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			marker := trimmed[:3]
			switch fence {
			case "":
				fence = marker
			case marker:
				fence = ""
			}
			out.WriteString(line)
			continue
		}
		if fence != "" {
			out.WriteString(line)
			continue
		}
		// Do not interpret tags inside inline code.
		last := 0
		for _, loc := range protectedMarkdown.FindAllStringIndex(line, -1) {
			out.WriteString(displayNotionTags(line[last:loc[0]]))
			out.WriteString(line[loc[0]:loc[1]])
			last = loc[1]
		}
		out.WriteString(displayNotionTags(line[last:]))
	}
	return out.String()
}

func displayNotionTags(text string) string {
	return notionTag.ReplaceAllStringFunc(text, func(tag string) string {
		name := notionTag.FindStringSubmatch(tag)[1]
		closing := strings.HasPrefix(tag, "</")
		switch name {
		case "b", "strong":
			return "**"
		case "i", "em":
			return "*"
		case "s", "del":
			return "~~"
		case "code":
			return "`"
		case "br":
			return "  \n"
		case "h1", "h2", "h3", "h4":
			if closing {
				return "\n\n"
			}
			return strings.Repeat("#", int(name[1]-'0')) + " "
		case "callout":
			if closing {
				return "\n\n"
			}
			return "💡 "
		case "quote":
			if closing {
				return "\n\n"
			}
			return "> "
		case "li":
			if closing {
				return "\n"
			}
			return "- "
		case "p", "toggle", "columns", "column", "ul", "ol", "synced-block":
			return "\n"
		case "u", "span":
			return ""
		case "page", "database", "mention", "file", "pdf", "audio", "video", "bookmark", "embed", "unknown":
			if closing {
				return ""
			}
			if match := notionURL.FindStringSubmatch(tag); len(match) > 1 {
				return "[" + name + "](" + html.UnescapeString(match[1]) + ") "
			}
		}
		// Preserve unrecognized tags so their content remains visible.
		return tag
	})
}
