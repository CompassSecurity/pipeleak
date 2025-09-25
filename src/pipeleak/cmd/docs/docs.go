package docs

import (
	"os"

	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func NewDocsCmd(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "docs",
		Short: "Generate CLI documentation",
		Long:  "Generate Markdown documentation for all commands in this CLI application.",
		Run: func(cmd *cobra.Command, args []string) {
			outputDir := "./cli-docs"

			if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
				log.Fatal().Err(err).Msg("Failed to create cli-docs directory")
			}

			err := doc.GenMarkdownTree(root, outputDir)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to generate cli docs")
			}

			log.Info().Str("folder", outputDir).Msg("Docs successfully generated")
		},
	}
}
