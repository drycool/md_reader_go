// Package toc provides markdown header parsing and table-of-contents generation.
package toc

import (
	"regexp"
	"strings"

	"github.com/drycool/md_reader_go/internal/logger"
)

// HeaderInfo represents a single markdown header.
type HeaderInfo struct {
	Title      string // The header text without # markers
	Level      int    // Header level (1-6)
	LineNumber int    // 0-based line number in the file
}

// TOCData maps file paths to their header lists.
type TOCData map[string][]HeaderInfo

// headerRegex matches markdown headers: # through ######.
var headerRegex = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)

// parseHeadersFromLines parses markdown headers from file lines.
// Ignores headers inside fenced code blocks (``` or ~~~) and indented code blocks.
func parseHeadersFromLines(lines []string) []HeaderInfo {
	var headers []HeaderInfo

	inFencedBlock := false
	fenceChar := "" // "`" or "~"
	fenceLen := 0

	for i, line := range lines {
		// Track fenced code blocks (``` and ~~~)
		if isFenceLine(line) {
			char, count := getFenceInfo(line)
			if !inFencedBlock {
				inFencedBlock = true
				fenceChar = char
				fenceLen = count
				continue
			}
			// Closing fence: must match opening char and be >= opening length
			if char == fenceChar && count >= fenceLen {
				inFencedBlock = false
				fenceChar = ""
				fenceLen = 0
			}
			continue
		}

		// Skip lines inside fenced code blocks
		if inFencedBlock {
			continue
		}

		// Skip indented code blocks (4 spaces or 1 tab)
		if isIndentedCodeLine(line) {
			continue
		}

		matches := headerRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		level := len(matches[1])
		title := strings.TrimSpace(matches[2])

		// Strip trailing # characters (common in some markdown styles)
		title = strings.TrimRight(title, "# ")
		title = strings.TrimSpace(title)

		headers = append(headers, HeaderInfo{
			Title:      title,
			Level:      level,
			LineNumber: i,
		})
	}

	return headers
}

// isFenceLine checks if a line starts a fenced code block.
// Markdown spec says 3+ backticks/tildes, but many parsers accept 2+.
func isFenceLine(line string) bool {
	trimmed := strings.TrimLeft(line, " ")
	if len(trimmed) < 2 {
		return false
	}
	// Check for 2+ backticks or 3+ tildes as opening fence
	if strings.HasPrefix(trimmed, "``") || strings.HasPrefix(trimmed, "~~~") {
		return true
	}
	return false
}

// getFenceInfo returns the fence character and its count for a fence line.
func getFenceInfo(line string) (string, int) {
	trimmed := strings.TrimLeft(line, " ")
	char := string(trimmed[0])
	count := 0
	for _, c := range trimmed {
		if string(c) == char {
			count++
		} else {
			break
		}
	}
	return char, count
}

// isIndentedCodeLine checks if a line looks like an indented code block.
// Indented code blocks start with 4 spaces or a tab.
func isIndentedCodeLine(line string) bool {
	if line == "" {
		return false
	}
	// Check for leading 4 spaces
	spaceCount := 0
	for _, c := range line {
		if c == ' ' {
			spaceCount++
			if spaceCount == 4 {
				return true
			}
		} else if c == '\t' {
			return true
		} else {
			return false
		}
	}
	return false
}

// BuildTOC builds a table of contents from a map of file paths to file lines.
func BuildTOC(files map[string][]string) TOCData {
	log := logger.GetLogger("toc")
	defer logger.FuncLog(log, "BuildTOC")()

	toc := make(TOCData)
	for path, lines := range files {
		headers := parseHeadersFromLines(lines)
		if len(headers) > 0 {
			toc[path] = headers
		}
	}

	totalHeaders := 0
	for _, headers := range toc {
		totalHeaders += len(headers)
	}
	log.Debug("TOC built", "files", len(toc), "headers", totalHeaders)

	return toc
}

// ValidateTOCStructure checks that the TOC has proper nesting.
// Returns true if the TOC structure is valid.
func ValidateTOCStructure(toc TOCData) bool {
	for path, headers := range toc {
		if !validateHeaderSequence(headers) {
			logger.GetLogger("toc").Warn("Invalid header sequence", "file", path)
			return false
		}
	}
	return true
}

// validateHeaderSequence checks that headers are properly nested.
// Headers shouldn't jump by more than 1 level at a time (e.g., ### then # is fine,
// but # then ###### is unusual).
func validateHeaderSequence(headers []HeaderInfo) bool {
	if len(headers) == 0 {
		return true
	}

	prevLevel := headers[0].Level
	for i := 1; i < len(headers); i++ {
		curr := headers[i]
		// Allow jumps up (e.g., ### -> ## -> #), but warn on jumps down > 1
		if curr.Level > prevLevel+1 {
			return false
		}
		prevLevel = curr.Level
	}
	return true
}

// FindSectionBounds returns the start and end line numbers for a header's section.
// The section starts at the header line and ends before the next header at the same
// or higher level, or at end of file.
func FindSectionBounds(lines []string, headerLine int) (int, int) {
	if headerLine < 0 || headerLine >= len(lines) {
		return headerLine, headerLine
	}

	start := headerLine
	currentLevel := 0

	// Determine the level of the current header
	if matches := headerRegex.FindStringSubmatch(lines[headerLine]); matches != nil {
		currentLevel = len(matches[1])
	}

	// Find the end
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if matches := headerRegex.FindStringSubmatch(lines[i]); matches != nil {
			nextLevel := len(matches[1])
			if nextLevel <= currentLevel {
				end = i
				break
			}
		}
	}

	return start, end
}
