package docs

import (
	pkgdocs "github.com/CompassSecurity/pipeleak/pkg/docs"
	"github.com/spf13/cobra"
)

func NewDocsCmd(root *cobra.Command) *cobra.Command {
	var serve bool
	var githubPages bool

	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Generate CLI documentation",
		Long:  "Generates documentation for all commands. Must be run in an environment where 'mkdocs' is installed.",
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

	cmd.Flags().BoolVarP(&serve, "serve", "s", false, "Serve documentation after building")
	cmd.Flags().BoolVarP(&githubPages, "github-pages", "g", false, "Build for GitHub Pages")

	return cmd
}
