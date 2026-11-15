package export

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var placeholder = regexp.MustCompile(`\{([^{}]+)\}`)
var uuid = regexp.MustCompile(`(?i)([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}|[0-9a-f]{32})$`)

type cachedProperties struct {
	properties map[string]any
	title      string
	err        error
}
type resolver struct {
	ctx      context.Context
	source   Source
	template string
	cache    map[string]cachedProperties
	warnings []string
	warned   map[string]bool
}

func newResolver(ctx context.Context, source Source, template string) (*resolver, error) {
	rest := placeholder.ReplaceAllString(template, "x")
	if strings.ContainsAny(rest, "{}\r\n\x00") {
		return nil, fmt.Errorf("invalid link template")
	}
	if template != "" && (!strings.HasPrefix(template, "/") || strings.HasPrefix(template, "//") || strings.ContainsAny(template, "?#\\")) {
		return nil, fmt.Errorf("link template must be a site-relative path, such as /articles/{slug}/")
	}
	return &resolver{ctx: ctx, source: source, template: template, cache: map[string]cachedProperties{}, warned: map[string]bool{}}, nil
}

func normalizeID(id string) string { return strings.ToLower(strings.ReplaceAll(id, "-", "")) }

func (r *resolver) warn(link, reason string) {
	key := link + reason
	if !r.warned[key] {
		r.warnings = append(r.warnings, fmt.Sprintf("link %s: %s; keeping Notion URL", link, reason))
		r.warned[key] = true
	}
}

func (r *resolver) resolve(link, label string) (string, string) {
	if r.template == "" {
		return link, label
	}
	u, err := url.Parse(link)
	if err != nil || u.Scheme != "https" || (u.Host != "www.notion.so" && u.Host != "notion.so" && u.Host != "www.notion.site" && u.Host != "notion.site" && u.Host != "app.notion.com" && u.Host != "www.notion.com" && u.Host != "notion.com") {
		return link, label
	}
	if u.Fragment != "" {
		r.warn(link, "block anchors are unsupported")
		return link, label
	}
	match := uuid.FindStringSubmatch(strings.TrimRight(u.Path, "/"))
	var item cachedProperties
	// Some imported links retain a site path under Notion's app host.
	// These carry a slug rather than an API page ID.
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if u.Host == "app.notion.com" && len(parts) == 2 && parts[0] == "articles" && validSlug(parts[1]) {
		item.properties = map[string]any{"slug": parts[1]}
	} else {
		if len(match) == 0 {
			r.warn(link, "cannot identify page ID")
			return link, label
		}
		id := normalizeID(match[1])
		var ok bool
		item, ok = r.cache[id]
		if !ok {
			item.properties, item.title, item.err = r.source.ExportPageReference(r.ctx, id)
			r.cache[id] = item
		}
	}
	if item.err != nil {
		r.warn(link, "cannot retrieve referenced page: "+item.err.Error())
		return link, label
	}
	if label == "" {
		label = item.title
	}
	if label == "" {
		if title, ok := item.properties["title"].(string); ok {
			label = title
		}
	}
	failed := false
	target := placeholder.ReplaceAllStringFunc(r.template, func(token string) string {
		key := token[1 : len(token)-1]
		value := item.properties[key]
		var s string
		switch v := value.(type) {
		case string:
			s = v
		case float64:
			s = fmt.Sprint(v)
		case int:
			s = fmt.Sprint(v)
		default:
			failed = true
		}
		if strings.TrimSpace(s) == "" || s == "." || s == ".." {
			failed = true
		}
		// Escape a complete path segment, including characters PathEscape leaves literal.
		escaped := url.PathEscape(s)
		escaped = strings.NewReplacer("(", "%28", ")", "%29").Replace(escaped)
		return escaped
	})
	if failed {
		r.warn(link, "missing, empty, or non-scalar template property")
		return link, label
	}
	return target, label
}
