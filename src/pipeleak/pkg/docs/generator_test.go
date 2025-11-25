package docs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// Test getFileName logic for level handling and GroupID usage.
func TestGetFileName(t *testing.T) {
	cmdNoGroup := &cobra.Command{Use: "scan", Short: "scan"}
	cmdGroup := &cobra.Command{Use: "enum", GroupID: "enumeration"}

	assert.Equal(t, "scan.md", getFileName(cmdNoGroup, 1))
	assert.Equal(t, "enumeration.md", getFileName(cmdGroup, 1))
	assert.Equal(t, "scan.md", getFileName(cmdNoGroup, 2)) // level >1 ignores GroupID logic
}

// Test displayName title casing and GroupID preference.
func TestDisplayName(t *testing.T) {
	cmdNoGroup := &cobra.Command{Use: "list"}
	cmdGroup := &cobra.Command{Use: "enum", GroupID: "gitlab pentest"}

	assert.Equal(t, "List", displayName(cmdNoGroup, 1))
	assert.Equal(t, "Gitlab Pentest", displayName(cmdGroup, 1)) // Title case applied
	assert.Equal(t, "Enum", displayName(cmdGroup, 2))           // deeper level uses Name
}

// buildNav should create index.md for commands with children and .md file for leaves.
func TestBuildNav(t *testing.T) {
	root := &cobra.Command{Use: "pipeleak"}
	parent := &cobra.Command{Use: "alpha"}
	leaf := &cobra.Command{Use: "scan", Run: func(cmd *cobra.Command, args []string) {}}
	parent.AddCommand(leaf)
	root.AddCommand(parent)

	entry := buildNav(root, 0, "")
	assert.Equal(t, "Pipeleak", entry.Label)
	assert.Len(t, entry.Children, 1)
	child := entry.Children[0]
	assert.Equal(t, "Alpha", child.Label)
	assert.Equal(t, filepath.ToSlash("pipeleak/alpha/index.md"), child.FilePath)
	assert.Len(t, child.Children, 1)
	grand := child.Children[0]
	assert.Equal(t, filepath.ToSlash("pipeleak/alpha/scan.md"), grand.FilePath)
}

// convertNavToYaml should trim pipeleak/ prefix and .md suffix.
func TestConvertNavToYaml(t *testing.T) {
	entries := []*NavEntry{
		{Label: "Alpha", FilePath: filepath.ToSlash("pipeleak/alpha/index.md"), Children: []*NavEntry{}},
		{Label: "Beta", FilePath: filepath.ToSlash("pipeleak/beta/leaf.md"), Children: []*NavEntry{}},
	}
	yamlList := convertNavToYaml(entries)
	// Entries become maps with label key
	assert.Len(t, yamlList, 2)
	// Validate trimming and removal of extension
	alphaMap := yamlList[0]
	betaMap := yamlList[1]
	assert.Equal(t, "alpha/index", alphaMap["Alpha"])
	assert.Equal(t, "beta/leaf", betaMap["Beta"])
}

// writeMkdocsYaml should create mkdocs.yml with expected keys and nav entries.
func TestWriteMkdocsYaml(t *testing.T) {
	root := &cobra.Command{Use: "pipeleak"}
	alpha := &cobra.Command{Use: "alpha", Run: func(cmd *cobra.Command, args []string) {}}
	deepParent := &cobra.Command{Use: "beta"}
	deepChild := &cobra.Command{Use: "deep", Run: func(cmd *cobra.Command, args []string) {}}
	deepParent.AddCommand(deepChild)
	root.AddCommand(alpha)
	root.AddCommand(deepParent)

	tmpDir := t.TempDir()
	// Change working directory to module root (src/pipeleak) so relative ../../docs resolves correctly
	wd, _ := os.Getwd()
	rootDir := filepath.Clean(filepath.Join(wd, "..", ".."))
	old := wd
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("failed to chdir to root: %v", err)
	}
	defer func() { _ = os.Chdir(old) }()

	err := writeMkdocsYaml(root, tmpDir, false)
	assert.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(tmpDir, "mkdocs.yml"))
	assert.NoError(t, err)

	var parsed map[string]interface{}
	err = yaml.Unmarshal(data, &parsed)
	assert.NoError(t, err)

	// Basic keys exist
	assert.Equal(t, "Pipeleak - Pipeline Secrets Scanner", parsed["site_name"])
	assert.Equal(t, "pipeleak", parsed["docs_dir"])

	// Nav structure assertions
	navAny, ok := parsed["nav"].([]interface{})
	assert.True(t, ok)
	assert.Equal(t, 4, len(navAny)) // intro, methodology, alpha, beta

	// Introduction entry first
	introMap, ok := navAny[0].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, introMap, "Introduction")
	assert.Equal(t, "/introduction/getting_started/", introMap["Introduction"])

	// Methodology second
	methMap, ok := navAny[1].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, methMap, "Methodology")

	// Command entries appear after methodology
	foundAlpha := false
	foundBeta := false
	for _, item := range navAny[2:] {
		if m, ok := item.(map[string]interface{}); ok {
			if _, ok := m["Alpha"]; ok {
				foundAlpha = true
			}
			if _, ok := m["Beta"]; ok {
				foundBeta = true
			}
		}
	}
	assert.True(t, foundAlpha)
	assert.True(t, foundBeta)
}

// writeMkdocsYaml with GithubPages should prefix nav paths.
func TestWriteMkdocsYaml_GithubPagesPrefix(t *testing.T) {
	root := &cobra.Command{Use: "pipeleak"}
	root.AddCommand(&cobra.Command{Use: "alpha", Run: func(cmd *cobra.Command, args []string) {}})

	tmpDir := t.TempDir()
	wd, _ := os.Getwd()
	rootDir := filepath.Clean(filepath.Join(wd, "..", ".."))
	old := wd
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("failed to chdir to root: %v", err)
	}
	defer func() { _ = os.Chdir(old) }()

	err := writeMkdocsYaml(root, tmpDir, true)
	assert.NoError(t, err)
	data, err := os.ReadFile(filepath.Join(tmpDir, "mkdocs.yml"))
	assert.NoError(t, err)
	var parsed map[string]interface{}
	err = yaml.Unmarshal(data, &parsed)
	assert.NoError(t, err)
	navAny := parsed["nav"].([]interface{})
	introMap := navAny[0].(map[string]interface{})
	assert.Equal(t, "/pipeleak/introduction/getting_started/", introMap["Introduction"])
}
