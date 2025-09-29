package docs

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"gopkg.in/yaml.v3"
)

func getFileName(cmd *cobra.Command, level int) string {
	switch level {
	case 1:
		if cmd.GroupID != "" {
			return cmd.GroupID + ".md"
		}
		return cmd.Name() + ".md"
	default:
		return cmd.Name() + ".md"
	}
}

func displayName(cmd *cobra.Command, level int) string {
	titleCaser := cases.Title(language.Und, cases.NoLower)
	switch level {
	case 1:
		if cmd.GroupID != "" {
			return titleCaser.String(cmd.GroupID)
		}
		return titleCaser.String(cmd.Name())
	default:
		return titleCaser.String(cmd.Name())
	}
}

func generateDocs(cmd *cobra.Command, dir string, level int) error {
	var filename string

	if len(cmd.Commands()) > 0 {
		dir = filepath.Join(dir, cmd.Name())
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return err
		}
		filename = filepath.Join(dir, "index.md")
	} else {
		filename = filepath.Join(dir, getFileName(cmd, level))
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	customLinkHandler := func(s string) string {
		if s == "pipeleak.md" {
			return "/"
		}

		s = strings.TrimPrefix(s, "pipeleak_")
		s = strings.TrimSuffix(s, ".md")
		s = strings.ReplaceAll(s, "_", "/")
		return "/" + s
	}

	if err := doc.GenMarkdownCustom(cmd, f, customLinkHandler); err != nil {
		return err
	}

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

type NavEntry struct {
	Label    string
	FilePath string
	Children []*NavEntry
}

func buildNav(cmd *cobra.Command, level int, parentPath string) *NavEntry {
	entry := &NavEntry{
		Label: displayName(cmd, level),
	}

	if len(cmd.Commands()) > 0 {
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
		entry.FilePath = filepath.ToSlash(filepath.Join(parentPath, getFileName(cmd, level)))
	}

	return entry
}

func convertNavToYaml(entries []*NavEntry) []map[string]interface{} {
	yamlList := []map[string]interface{}{}
	for _, e := range entries {
		navPath := e.FilePath
		if len(navPath) >= 9 && navPath[:9] == "pipeleak/" {
			navPath = navPath[9:]
		}
		if len(e.Children) == 0 {
			if filepath.Ext(navPath) == ".md" {
				navPath = navPath[:len(navPath)-3]
			}
			yamlList = append(yamlList, map[string]interface{}{
				e.Label: navPath,
			})
		} else {
			yamlList = append(yamlList, map[string]interface{}{
				e.Label: convertNavToYaml(e.Children),
			})
		}
	}
	return yamlList
}

func writeMkdocsYaml(rootCmd *cobra.Command, outputDir string) error {
	rootEntry := buildNav(rootCmd, 0, "")
	nav := convertNavToYaml(rootEntry.Children)
	// Add hardcoded Introduction entry at the top
	introEntry := map[string]interface{}{"Introduction": "/introduction/getting_started/"}
	methodologyEntry := map[string]interface{}{
		"Methodology": []map[string]interface{}{
			{"GitLab": "/methodology/gitlab/"},
		},
	}
	nav = append([]map[string]interface{}{introEntry, methodologyEntry}, nav...)

	assetsDir := filepath.Join(outputDir, "pipeleak", "assets")
	if err := os.MkdirAll(assetsDir, os.ModePerm); err != nil {
		return err
	}
	srcLogo := filepath.Join("..", "..", "docs", "logo.png")
	dstLogo := filepath.Join(assetsDir, "logo.png")
	logoData, err := os.ReadFile(srcLogo)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dstLogo, logoData, 0644); err != nil {
		return err
	}

	mkdocs := map[string]interface{}{
		"site_name": "Pipeleak CLI Docs",
		"docs_dir":  "pipeleak",
		"site_dir":  "site",
		"theme": map[string]interface{}{
			"name":    "material",
			"logo":    "assets/logo.png",
			"favicon": "assets/logo.png",
			"palette": map[string]string{
				"scheme":  "slate",
				"primary": "green",
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

var serve bool
var rootCmd *cobra.Command

func NewDocsCmd(root *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Generate CLI documentation",
		Long:  "Generate Markdown documentation for all commands in this CLI application and mkdocs.yml. Must be run in an environment where 'mkdocs' is installed.",
		Example: `
# Generate docs and serve them at http://localhost:8000
pipeleak docs --serve
		`,
		Run: Docs,
	}

	cmd.Flags().BoolVarP(&serve, "serve", "s", false, "Run 'mkdocs build' in the output folder after generating docs")
	rootCmd = root
	return cmd
}

func copySubfolders(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			srcPath := filepath.Join(srcDir, entry.Name())
			dstPath := filepath.Join(dstDir, entry.Name())
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, os.ModePerm); err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func Docs(cmd *cobra.Command, args []string) {
	if _, err := os.Stat("main.go"); os.IsNotExist(err) {
		log.Fatal().Msg("Run this command from the project src/pipeleak directory.")
	}

	outputDir := "./cli-docs"

	if _, err := os.Stat(outputDir); err == nil {
		log.Info().Msg("Output directory exists, deleting...")
		if err := os.RemoveAll(outputDir); err != nil {
			log.Fatal().Err(err).Msg("Failed to delete existing outputDir")
		}
	}

	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		log.Fatal().Err(err).Msg("Failed to create pipeleak directory")
	}

	if err := copySubfolders("../../docs", filepath.Join(outputDir, "pipeleak")); err != nil {
		log.Fatal().Err(err).Msg("Failed to copy docs subfolders")
	}

	rootCmd.DisableAutoGenTag = true
	if err := generateDocs(rootCmd, outputDir, 0); err != nil {
		log.Fatal().Err(err).Msg("Failed to generate CLI docs")
	}

	if err := writeMkdocsYaml(rootCmd, outputDir); err != nil {
		log.Fatal().Err(err).Msg("Failed to write mkdocs.yml")
	}

	log.Info().Str("folder", outputDir).Msg("Markdown successfully generated")

	log.Info().Msg("Running 'mkdocs build' in output folder...")
	cmdRun := exec.Command("mkdocs", "build")
	cmdRun.Dir = outputDir
	cmdRun.Stdout = os.Stdout
	cmdRun.Stderr = os.Stderr
	if err := cmdRun.Run(); err != nil {
		log.Fatal().Err(err).Msg("Failed to run mkdocs build")
	}

	if serve {
		siteDir := filepath.Join(outputDir, "site")
		log.Info().Msgf("Serving docs %s at http://localhost:8000 ... (Ctrl+C to quit)", siteDir)
		http.Handle("/", http.FileServer(http.Dir(siteDir)))
		if err := http.ListenAndServe(":8000", nil); err != nil {
			log.Fatal().Err(err).Msg("Failed to start HTTP server")
		}
	}
}
