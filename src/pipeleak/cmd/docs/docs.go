package docs

import (
	pkgdocs "github.com/CompassSecurity/pipeleak/pkg/docs"
	"github.com/spf13/cobra"
)

// NewDocsCmd creates a new docs command
func NewDocsCmd(root *cobra.Command) *cobra.Command {
	var serve bool
	var githubPages bool

	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Generate CLI documentation",
		Long:  "Generate Markdown documentation for all commands in this CLI application and mkdocs.yml. Must be run in an environment where 'mkdocs' is installed.",
		Example: `
# Generate docs and serve them at http://localhost:8000
pipeleak docs --serve
		`,
		Run: func(cmd *cobra.Command, args []string) {
			pkgdocs.Generate(pkgdocs.GenerateOptions{
				RootCmd:     root,
				Serve:       serve,
				GithubPages: githubPages,
			})
		},
	}

	cmd.Flags().BoolVarP(&serve, "serve", "s", false, "Run 'mkdocs build' in the output folder after generating docs")
	cmd.Flags().BoolVarP(&githubPages, "github-pages", "g", false, "Build for GitHub Pages")

	return cmd
}
