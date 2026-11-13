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
				return fmt.Errorf("Notion token is required; set NOCTL_NOTION_TOKEN or notion_token in your config")
			}
			doc, err := pageexport.Page(cmd.Context(), notion.NewClient(cfg.NotionToken), args[0], linkTemplate)
			if err != nil {
				return err
			}
			for _, warning := range doc.Warnings {
				fmt.Fprintln(cmd.ErrOrStderr(), "Warning:", warning)
			}
			if dryrun {
				_, err = cmd.OutOrStdout().Write(doc.Markdown)
				return err
			}
			path, err := pageexport.Write(output, doc, force)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.ErrOrStderr(), "Exported", path)
			return nil
		},
	}
	pageCmd.Flags().StringVar(&format, "format", "hugo", "Output format (hugo)")
	pageCmd.Flags().StringVar(&output, "output", "./", "Output directory; filename is <slug>.md")
	pageCmd.Flags().StringVar(&linkTemplate, "link-template", "", "Site-relative link template using referenced properties, e.g. /articles/{slug}/")
	pageCmd.Flags().BoolVar(&dryrun, "dryrun", false, "Print Markdown without writing files")
	pageCmd.Flags().BoolVar(&force, "force", false, "Replace an existing regular output file")
	exportCmd.AddCommand(pageCmd)
	return exportCmd
}
