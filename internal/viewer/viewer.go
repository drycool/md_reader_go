// Package viewer provides interactive viewing of markdown files with fuzzy search.
package viewer

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/drycool/md_reader_go/internal/loader"
	"github.com/drycool/md_reader_go/internal/logger"
	"github.com/drycool/md_reader_go/internal/toc"
	"github.com/drycool/md_reader_go/internal/validator"
)

// SectionView represents a view of a markdown file section.
type SectionView struct {
	FilePath string
	Title    string
	Start    int // Start line (inclusive)
	End      int // End line (exclusive)
	Content  []string
}

// SearchResult represents a fuzzy search result.
type SearchResult struct {
	FilePath string
	Title    string
	Line     int
	Score    int    // Match score (higher = better match)
	LineText string // The matched line text
}

// Viewer provides interactive viewer functionality.
type Viewer struct {
	log      *logger.Logger
	loader   *loader.Loader
	validator *validator.PathValidator
	inputVal  *validator.InputValidator
}

// NewViewer creates a new Viewer.
func NewViewer() *Viewer {
	return &Viewer{
		log:       logger.GetLogger("viewer"),
		loader:    loader.NewLoader(),
		validator: validator.NewPathValidator(),
		inputVal:  validator.NewInputValidator(),
	}
}

// LoadAndBuildTOC loads files and builds a TOC in one operation.
func (v *Viewer) LoadAndBuildTOC(path string) (toc.TOCData, map[string][]string, error) {
	log := v.log
	ld := v.loader

	var files map[string][]string
	var err error

	if ld.IsSingleFile(path) {
		lines, loadErr := ld.LoadSingleFile(path)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		absPath, _ := filepath.Abs(path)
		files = map[string][]string{absPath: lines}
	} else {
		files, err = ld.LoadMarkdownFilesRecursive(path)
		if err != nil {
			return nil, nil, err
		}
	}

	tableOfContents := toc.BuildTOC(files)

	log.Info("Loaded and built TOC",
		"path", path,
		"files", len(files),
		"headers", countHeaders(tableOfContents),
	)

	return tableOfContents, files, nil
}

// FuzzySearch performs a case-insensitive search across all headers.
// Uses Levenshtein distance for fuzzy matching, plus exact/prefix/contains tiers.
// Returns results sorted by relevance.
func (v *Viewer) FuzzySearch(tableOfContents toc.TOCData, query string) []SearchResult {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil
	}

	var results []SearchResult
	maxDistance := max(2, len(query)/3) // Fuzzy tolerance

	for filePath, headers := range tableOfContents {
		// Also match against filename (basename without extension)
		baseName := strings.ToLower(filepath.Base(filePath))
		baseNoExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))
		fileScore := 0
		if baseNoExt == query {
			fileScore = 95
		} else if strings.Contains(baseNoExt, query) {
			fileScore = 65
		} else if strings.HasPrefix(baseNoExt, query) {
			fileScore = 85
		}

		for _, h := range headers {
			lowerTitle := strings.ToLower(h.Title)
			score := 0

			// Scoring tiers:
			// 100 = exact match
			// 90  = exact word match
			// 80  = prefix match
			// 70  = contains match + low Levenshtein
			// 60  = contains match
			// 40  = Levenshtein within tolerance
			// 25  = word match
			// 0   = no match

			if lowerTitle == query {
				score = 100
			} else if strings.HasPrefix(lowerTitle, query+" ") || strings.HasSuffix(lowerTitle, " "+query) {
				score = 90
			} else if strings.HasPrefix(lowerTitle, query) {
				score = 80
			} else if strings.Contains(lowerTitle, query) {
				levDist := levenshteinDistance(lowerTitle, query)
				if levDist <= maxDistance {
					score = 70
				} else {
					score = 60
				}
			} else if wordMatch(lowerTitle, query) {
				score = 25
			} else {
				// Levenshtein-based fuzzy match as fallback
				levDist := levenshteinDistance(lowerTitle, query)
				if levDist <= maxDistance {
					// Higher score for closer matches
					score = int(math.Round(float64(40) * (1.0 - float64(levDist)/float64(maxDistance+1))))
					if score < 10 {
						score = 10
					}
				} else {
					continue
				}
			}

			// Use the higher of header score or file score
			if fileScore > score {
				score = fileScore
			}

			results = append(results, SearchResult{
				FilePath: filePath,
				Title:    h.Title,
				Line:     h.LineNumber,
				Score:    score,
			})
		}
	}

	// Sort by score (descending), then alphabetically for ties
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score ||
				(results[j].Score == results[i].Score && results[j].Title < results[i].Title) {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results
}

// levenshteinDistance computes the Levenshtein edit distance between two strings.
// Uses optimized O(n*m) DP with O(min(n,m)) memory.
func levenshteinDistance(s, t string) int {
	// Use runes for proper Unicode support
	sRunes := []rune(s)
	tRunes := []rune(t)
	n := len(sRunes)
	m := len(tRunes)

	if n == 0 {
		return m
	}
	if m == 0 {
		return n
	}

	// Ensure first dimension is the shorter string for memory efficiency
	if n > m {
		sRunes, tRunes = tRunes, sRunes
		n, m = m, n
	}

	// Single row DP
	prev := make([]int, n+1)
	curr := make([]int, n+1)

	// Initialize
	for i := 0; i <= n; i++ {
		prev[i] = i
	}

	for j := 1; j <= m; j++ {
		curr[0] = j
		for i := 1; i <= n; i++ {
			cost := 1
			if sRunes[i-1] == tRunes[j-1] {
				cost = 0
			}
			curr[i] = min3(
				prev[i]+1,     // deletion
				curr[i-1]+1,   // insertion
				prev[i-1]+cost, // substitution
			)
		}
		prev, curr = curr, prev
	}

	return prev[n]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// wordMatch checks if the query matches any word boundary in the title.
func wordMatch(lowerTitle, query string) bool {
	words := strings.Fields(lowerTitle)
	for _, word := range words {
		if strings.HasPrefix(word, query) || strings.Contains(word, query) {
			return true
		}
	}
	return false
}

// GetSectionContent retrieves the content of a section.
func (v *Viewer) GetSectionContent(files map[string][]string, filePath string, start, end int) SectionView {
	lines, ok := files[filePath]
	if !ok {
		return SectionView{
			FilePath: filePath,
			Content:  []string{"[File not found]"},
		}
	}

	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start > end {
		return SectionView{
			FilePath: filePath,
			Content:  []string{"[Invalid section bounds]"},
		}
	}

	title := filepath.Base(filePath)
	content := make([]string, end-start)
	copy(content, lines[start:end])

	return SectionView{
		FilePath: filePath,
		Title:    title,
		Start:    start,
		End:      end,
		Content:  content,
	}
}

// InteractiveView starts an interactive terminal session for browsing markdown files.
func (v *Viewer) InteractiveView(basePath string) error {
	log := v.log
	defer logger.FuncLog(log, "InteractiveView")()

	// Load files and build TOC
	tableOfContents, _, err := v.LoadAndBuildTOC(basePath)
	if err != nil {
		return fmt.Errorf("failed to load files: %w", err)
	}

	if len(tableOfContents) == 0 {
		fmt.Println("No markdown files found with headers.")
		return nil
	}

	fmt.Printf("Loaded %d file(s) with headers.\n", len(tableOfContents))
	fmt.Println("Enter a search query to find sections, or press Ctrl+C to exit.")

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\nSearch: ")
		scanned := scanner.Scan()
		if !scanned {
			break
		}

		query := scanner.Text()
		if strings.TrimSpace(query) == "" {
			continue
		}

		// Validate input
		_, err := v.inputVal.ValidateSearchQuery(query)
		if err != nil {
			fmt.Printf("Invalid input: %v\n", err)
			continue
		}

		results := v.FuzzySearch(tableOfContents, query)
		if len(results) == 0 {
			fmt.Println("No matches found.")
			continue
		}

		// Display results
		fmt.Printf("\nFound %d match(es):\n", len(results))
		for i, r := range results {
			if i >= 20 {
				fmt.Printf("  ... and %d more\n", len(results)-20)
				break
			}
			fmt.Printf("  [%d] %s — %s (line %d)\n", i+1, r.Title, filepath.Base(r.FilePath), r.Line+1)
		}

		fmt.Print("\nView section number (or Enter to skip): ")
		scanned = scanner.Scan()
		if !scanned {
			break
		}

		selection := strings.TrimSpace(scanner.Text())
		if selection == "" {
			continue
		}

		var idx int
		if _, err := fmt.Sscanf(selection, "%d", &idx); err != nil || idx < 1 || idx > len(results) {
			fmt.Println("Invalid selection.")
			continue
		}

		result := results[idx-1]

		// Reload files to get content
		files, err := v.loader.LoadMarkdownFilesRecursive(filepath.Dir(result.FilePath))
		if err != nil {
			fmt.Printf("Error reloading files: %v\n", err)
			continue
		}

		// Find section bounds
		lines, ok := files[result.FilePath]
		if !ok {
			fmt.Println("File not found in loaded data.")
			continue
		}

		start, end := toc.FindSectionBounds(lines, result.Line)
		section := v.GetSectionContent(files, result.FilePath, start, end)

		// Display the section
		v.printSection(section)
	}

	return scanner.Err()
}

// printSection displays a section view with line numbers.
func (v *Viewer) printSection(section SectionView) {
	fmt.Printf("\n═══════════════════════════════════════════\n")
	fmt.Printf("  %s\n", section.Title)
	fmt.Printf("  Lines %d-%d\n", section.Start+1, section.End)
	fmt.Printf("═══════════════════════════════════════════\n\n")

	for i, line := range section.Content {
		// Skip empty lines at start
		if line == "" && i < 2 {
			continue
		}
		fmt.Printf("%4d │ %s\n", section.Start+i+1, line)
	}
	fmt.Println()
}

// PrintTOC displays the table of contents in a human-readable format.
func PrintTOC(tableOfContents toc.TOCData) {
	for filePath, headers := range tableOfContents {
		fmt.Printf("\n%s:\n", filepath.Base(filePath))
		for _, h := range headers {
			indent := strings.Repeat("  ", h.Level-1)
			fmt.Printf("  %s# %s (line %d)\n", indent, h.Title, h.LineNumber+1)
		}
	}
}

func countHeaders(toc toc.TOCData) int {
	count := 0
	for _, headers := range toc {
		count += len(headers)
	}
	return count
}
