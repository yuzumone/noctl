package main

import (
	"fmt"
	"os"

	"noctl/internal/config"
	"noctl/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

const usage = `noctl - A terminal UI tool for browsing Notion databases.

Usage:
  noctl [options]

Options:
  -h, --help    Show this help message

Configuration:
  noctl looks for a Notion token in:
  1. Environment variable: NOCTL_NOTION_TOKEN
  2. Config file: ~/.config/noctl/config.yaml
  3. Config file: ~/.noctl.yaml

Example config file (~/.config/noctl/config.yaml):
  notion_token: secret_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
`

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--help", "-h":
			fmt.Print(usage)
			return
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(tui.NewAppModel(cfg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
