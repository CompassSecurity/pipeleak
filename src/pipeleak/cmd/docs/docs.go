package docs

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"gopkg.in/yaml.v3"
)

// getFileName returns the Markdown filename based on command level
func getFileName(cmd *cobra.Command, level int) string {
	switch level {
	case 0:
		return cmd.Name() + ".md" // root command
	case 1:
		return cmd.Short + ".md" // first-level commands
	default:
		return cmd.Name() + ".md" // deeper subcommands
	}
}

// displayName returns the navigation label based on level
func displayName(cmd *cobra.Command, level int) string {
	switch level {
	case 0:
		return cmd.Name() // root
	case 1:
		return cmd.Short // first-level
	default:
		return cmd.Name() // subcommands
	}
}

// generateDocs recursively generates Markdown files
func generateDocs(cmd *cobra.Command, dir string, level int) error {
	var filename string

	if len(cmd.Commands()) > 0 {
		// Command has subcommands → create folder with index.md
		dir = filepath.Join(dir, cmd.Name())
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return err
		}
		filename = filepath.Join(dir, "index.md")
	} else {
		// Leaf command → file in parent folder
		filename = filepath.Join(dir, getFileName(cmd, level))
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := doc.GenMarkdown(cmd, f); err != nil {
		return err
	}

	// Recursively generate subcommands only if any
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		if err := generateDocs(c, dir, level+1); err != nil {
			return err
		}
	}

	return nil
}

// NavEntry represents a single MkDocs navigation entry
type NavEntry struct {
	Label    string
	FilePath string
	Children []*NavEntry
}

// buildNav recursively builds the navigation tree with correct relative paths
func buildNav(cmd *cobra.Command, level int, parentPath string) *NavEntry {
	entry := &NavEntry{
		Label: displayName(cmd, level),
	}

	if len(cmd.Commands()) > 0 {
		// Command has subcommands → folder with index.md
		folder := filepath.Join(parentPath, cmd.Name())
		entry.FilePath = filepath.ToSlash(filepath.Join(folder, "index.md"))
		entry.Children = []*NavEntry{}
		for _, c := range cmd.Commands() {
			if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
				continue
			}
			entry.Children = append(entry.Children, buildNav(c, level+1, folder))
		}
	} else {
		// Leaf command → file in parent folder
		entry.FilePath = filepath.ToSlash(filepath.Join(parentPath, getFileName(cmd, level)))
	}

	return entry
}

// convertNavToYaml recursively converts NavEntry to YAML-friendly format
func convertNavToYaml(entries []*NavEntry) []map[string]interface{} {
	yamlList := []map[string]interface{}{}
	for _, e := range entries {
		if len(e.Children) == 0 {
			yamlList = append(yamlList, map[string]interface{}{
				e.Label: e.FilePath,
			})
		} else {
			yamlList = append(yamlList, map[string]interface{}{
				e.Label: convertNavToYaml(e.Children),
			})
		}
	}
	return yamlList
}

// writeMkdocsYaml generates mkdocs.yml in outputDir
func writeMkdocsYaml(rootCmd *cobra.Command, outputDir string) error {
	rootEntry := buildNav(rootCmd, 0, "")
	nav := convertNavToYaml(rootEntry.Children) // exclude root itself from nav

	mkdocs := map[string]interface{}{
		"site_name": "Pipeleak CLI Docs",
		"docs_dir":  "pipeleak",
		"site_dir":  "site",
		"theme": map[string]interface{}{
			"name": "material",
			"palette": map[string]string{
				"scheme": "slate",
			},
		},
		"extra": map[string]interface{}{
			"highlightjs": true,
		},
		"nav": nav,
	}

	yamlData, err := yaml.Marshal(mkdocs)
	if err != nil {
		return err
	}

	filename := filepath.Join(outputDir, "mkdocs.yml")
	return os.WriteFile(filename, yamlData, 0644)
}

// NewDocsCmd returns a Cobra command to generate CLI Markdown docs and mkdocs.yml
func NewDocsCmd(root *cobra.Command) *cobra.Command {
	var serve bool
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Generate CLI documentation",
		Long:  "Generate Markdown documentation for all commands in this CLI application and mkdocs.yml.",
		Run: func(cmd *cobra.Command, args []string) {
			outputDir := "./cli-docs"

			// Check if outputDir exists, delete recursively if so
			if _, err := os.Stat(outputDir); err == nil {
				log.Info().Msg("Output directory exists, deleting...")
				if err := os.RemoveAll(outputDir); err != nil {
					log.Fatal().Err(err).Msg("Failed to delete existing outputDir")
				}
			}

			if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
				log.Fatal().Err(err).Msg("Failed to create pipeleak directory")
			}

			if err := generateDocs(root, outputDir, 0); err != nil {
				log.Fatal().Err(err).Msg("Failed to generate CLI docs")
			}

			if err := writeMkdocsYaml(root, outputDir); err != nil {
				log.Fatal().Err(err).Msg("Failed to write mkdocs.yml")
			}

			log.Info().Str("folder", outputDir).Msg("Docs and mkdocs.yml successfully generated")

			if serve {
				log.Info().Msg("Running 'mkdocs build' in output folder...")
				// Run mkdocs build in outputDir
				cmdRun := exec.Command("mkdocs", "build")
				cmdRun.Dir = outputDir
				cmdRun.Stdout = os.Stdout
				cmdRun.Stderr = os.Stderr
				if err := cmdRun.Run(); err != nil {
					log.Fatal().Err(err).Msg("Failed to run mkdocs build")
				}

				// Serve outputDir/site with built-in HTTP server
				siteDir := filepath.Join(outputDir, "site")
				log.Info().Msgf("Serving docs %s at http://localhost:8000 ... (Ctrl+C to quit)", siteDir)
				http.Handle("/", http.FileServer(http.Dir(siteDir)))
				if err := http.ListenAndServe(":8000", nil); err != nil {
					log.Fatal().Err(err).Msg("Failed to start HTTP server")
				}
			}
		},
	}

	cmd.Flags().BoolVarP(&serve, "serve", "s", false, "Run 'mkdocs build' in the output folder after generating docs")
	return cmd
}
