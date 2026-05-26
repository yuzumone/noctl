// Package main is the entry point for the noctl application.
package main

import (
	"fmt"
	"os"

	"noctl/internal/config"
	"noctl/internal/tui"
	"noctl/internal/version"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "noctl",
	Short: "A terminal UI tool for browsing Notion databases",
	Long: `noctl is a fast, keyboard-driven terminal user interface for 
browsing and editing Notion databases.`,
	Run: func(cmd *cobra.Command, args []string) {
		runApp(tui.ViewDBList)
	},
}

var calendarCmd = &cobra.Command{
	Use:   "calendar",
	Aliases: []string{"c"},
	Short: "Open the monthly calendar view for configured databases",
	Run: func(cmd *cobra.Command, args []string) {
		runApp(tui.ViewCalendar)
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Aliases: []string{"v"},
	Short: "Print the version number of noctl",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("noctl %s\n", version.Get())
	},
}

func init() {
	rootCmd.AddCommand(calendarCmd)
	rootCmd.AddCommand(versionCmd)
}

func runApp(initialState tui.SessionState) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	app := tui.NewAppModel(cfg)
	app.SetInitialState(initialState)

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
