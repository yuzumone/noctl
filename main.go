// Package main is the entry point for the noctl application.
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
  noctl [command] [options]

Commands:
  calendar      Open the monthly calendar view for configured databases

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
	var initialState tui.SessionState = tui.ViewDBList

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--help", "-h":
			fmt.Print(usage)
			return
		case "calendar":
			if len(os.Args) > 2 {
				switch os.Args[2] {
				case "--help", "-h":
					fmt.Print(usage)
					return
				}
			}
			initialState = tui.ViewCalendar
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	app := tui.NewAppModel(cfg)
	app.SetInitialState(initialState)

	runProgram(app)
}

func runProgram(app *tui.AppModel) {
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
