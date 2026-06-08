// Package main is the entry point for the md-reader CLI application.
package main

import (
	"fmt"
	"os"

	"github.com/drycool/md_reader_go/internal/gui"
	"github.com/drycool/md_reader_go/internal/logger"
	"github.com/drycool/md_reader_go/internal/viewer"
	"github.com/spf13/cobra"
)

var (
	debug    bool
	logFile  string
	recursive bool
)

var rootCmd = &cobra.Command{
	Use:   "md-reader",
	Short: "Interactive CLI viewer for Markdown documentation",
	Long: `md-reader is a comprehensive tool for viewing and navigating
Markdown documentation files. It supports interactive search,
table of contents generation, and section-based viewing.

Supports both single files and entire directories of markdown files.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		level := "INFO"
		if debug {
			level = "DEBUG"
		}
		if err := logger.SetupLogging(level, logFile, ""); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to setup logging: %v\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Default: show help
		cmd.Help()
	},
}

var openCmd = &cobra.Command{
	Use:   "open [path]",
	Short: "Open markdown files in interactive mode",
	Long: `Opens a markdown file or directory in interactive terminal mode.
Allows searching headers and viewing sections with fuzzy search.

Examples:
  md-reader open document.md        # Open a single file
  md-reader open ./docs/            # Open a directory of markdown files
  md-reader open .                  # Open current directory`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		log := logger.GetLogger("cli")
		log.Info("Opening files", "path", path, "recursive", recursive)

		v := viewer.NewViewer()
		return v.InteractiveView(path)
	},
}

var tocCmd = &cobra.Command{
	Use:   "toc [path]",
	Short: "Display table of contents for markdown files",
	Long: `Parses markdown headers and displays the table of contents
for the specified file or directory.

Examples:
  md-reader toc document.md          # Show TOC for a single file
  md-reader toc ./docs/              # Show TOC for all files in a directory`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		log := logger.GetLogger("cli")
		log.Info("Building TOC", "path", path)

		v := viewer.NewViewer()
		tableOfContents, _, err := v.LoadAndBuildTOC(path)
		if err != nil {
			return fmt.Errorf("failed to build TOC: %w", err)
		}

		if len(tableOfContents) == 0 {
			fmt.Println("No headers found.")
			return nil
		}

		viewer.PrintTOC(tableOfContents)
		return nil
	},
}

var statsCmd = &cobra.Command{
	Use:   "stats [path]",
	Short: "Show statistics about markdown files",
	Long: `Analyzes markdown files and displays statistics:
number of files, headers, lines, etc.

Examples:
  md-reader stats ./docs/            # Show stats for a directory
  md-reader stats document.md        # Show stats for a single file`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		v := viewer.NewViewer()
		tableOfContents, files, err := v.LoadAndBuildTOC(path)
		if err != nil {
			return fmt.Errorf("failed to analyze: %w", err)
		}

		totalLines := 0
		totalFiles := len(files)
		totalHeaders := 0
		largestFile := ""
		maxLines := 0

		for fp, lines := range files {
			totalLines += len(lines)
			if len(lines) > maxLines {
				maxLines = len(lines)
				largestFile = fp
			}
		}

		for _, headers := range tableOfContents {
			totalHeaders += len(headers)
		}

		fmt.Printf("  Files:       %d\n", totalFiles)
		fmt.Printf("  Total lines: %d\n", totalLines)
		fmt.Printf("  Headers:     %d\n", totalHeaders)
		if largestFile != "" {
			fmt.Printf("  Largest:     %s (%d lines)\n", largestFile, maxLines)
		}

		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("md-reader v0.1.0 — Go port")
		fmt.Println("Interactive Markdown documentation viewer")
	},
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")
	rootCmd.PersistentFlags().StringVarP(&logFile, "log-file", "l", "", "Log to file instead of stderr")

	// Command-specific flags
	openCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Search directories recursively (default: non-recursive)")

	// Add commands
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(tocCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(guiCmd)
}

var guiCmd = &cobra.Command{
	Use:   "gui [path]",
	Short: "Open graphical interface for markdown files",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		v := viewer.NewViewer()
		a := gui.NewApp(v)
		a.Show(path)
		return nil
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
