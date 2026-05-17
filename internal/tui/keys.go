package tui

import (
	"strings"
)

type keyHelp struct {
	key  string
	desc string
}

func renderFooter(width int, helps []keyHelp) string {
	var parts []string
	for _, h := range helps {
		parts = append(parts, KeyStyle.Render(h.key)+KeyDescStyle.Render(h.desc))
	}
	
	footerContent := strings.Join(parts, " ")
	return FooterStyle.Width(width).Render(footerContent)
}
