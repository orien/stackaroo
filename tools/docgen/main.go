package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	stackcmd "codeberg.org/orien/stackaroo/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func main() {
	outputDir := filepath.Join("docs", "user", "reference", "cli")

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		log.Fatalf("create output directory: %v", err)
	}

	if err := cleanMarkdown(outputDir); err != nil {
		log.Fatalf("clean output directory: %v", err)
	}

	root := stackcmd.RootCommand()
	root.DisableAutoGenTag = true
	setDisableAutoGenTag(root)

	if err := doc.GenMarkdownTreeCustom(root, outputDir, filePrepender, linkHandler); err != nil {
		log.Fatalf("generate markdown documentation: %v", err)
	}
}

func cleanMarkdown(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".md") {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

func setDisableAutoGenTag(cmd *cobra.Command) {
	for _, child := range cmd.Commands() {
		child.DisableAutoGenTag = true
		setDisableAutoGenTag(child)
	}
}

func filePrepender(filename string) string {
	return ""
}

func linkHandler(name string) string {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	base = strings.ReplaceAll(base, " ", "-")
	return strings.ToLower(base)
}
