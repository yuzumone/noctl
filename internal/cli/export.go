// Package cli implements non-interactive noctl commands.
package cli

import (
	"fmt"

	"noctl/internal/config"
	pageexport "noctl/internal/export"
	"noctl/internal/notion"

	"github.com/spf13/cobra"
)

// NewExportCmd constructs the CLI-only export command.
func NewExportCmd() *cobra.Command {
	exportCmd := &cobra.Command{Use: "export", Short: "Export Notion pages as Markdown", SilenceUsage: true, SilenceErrors: true}
	var format, output, linkTemplate string
	var dryrun, force bool
	var database bool
	pageCmd := &cobra.Command{
		Use: "page <page-id>", Short: "Export one Notion page", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if format != "hugo" {
				return fmt.Errorf("unsupported format %q (supported: hugo)", format)
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if cfg.NotionToken == "" {
				return fmt.Errorf("notion token is required; set NOCTL_NOTION_TOKEN or notion_token in your config")
			}
			client := notion.NewClient(cfg.NotionToken)
			var docs []*pageexport.Document
			if database {
				docs, err = pageexport.Database(cmd.Context(), client, args[0], linkTemplate)
			} else {
				var doc *pageexport.Document
				doc, err = pageexport.Page(cmd.Context(), client, args[0], linkTemplate)
				docs = []*pageexport.Document{doc}
			}
			if err != nil {
				return err
			}
			for _, doc := range docs {
				for _, warning := range doc.Warnings {
					if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "Warning:", warning); err != nil {
						return err
					}
				}
				if dryrun {
					_, err = cmd.OutOrStdout().Write(doc.Markdown)
					if err != nil {
						return err
					}
					continue
				}
				path, err := pageexport.Write(output, doc, force)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(cmd.ErrOrStderr(), "Exported", path)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	exportCmd.PersistentFlags().StringVar(&format, "format", "hugo", "Output format (hugo)")
	exportCmd.PersistentFlags().StringVar(&output, "output", "./", "Output directory; filename is <slug>.md")
	exportCmd.PersistentFlags().StringVar(&linkTemplate, "link-template", "", "Site-relative link template using referenced properties, e.g. /articles/{slug}/")
	exportCmd.PersistentFlags().BoolVar(&dryrun, "dryrun", false, "Print Markdown without writing files")
	exportCmd.PersistentFlags().BoolVar(&force, "force", false, "Replace an existing regular output file")
	databaseCmd := &cobra.Command{Use: "database <database-id>", Aliases: []string{"db"}, Short: "Export all pages in a Notion database", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		database = true
		return pageCmd.RunE(cmd, args)
	}}
	exportCmd.AddCommand(pageCmd, databaseCmd)
	return exportCmd
}
