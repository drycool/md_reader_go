// Package loader handles loading markdown files from disk.
package loader

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/drycool/md_reader_go/internal/exceptions"
	"github.com/drycool/md_reader_go/internal/logger"
	"github.com/drycool/md_reader_go/internal/validator"
)

// Loader handles loading markdown files.
type Loader struct {
	pathValidator *validator.PathValidator
	contentValidator *validator.FileContentValidator
	log            *logger.Logger
	loadedCount    int
	failedCount    int
}

// NewLoader creates a new Loader.
func NewLoader() *Loader {
	return &Loader{
		pathValidator:    validator.NewPathValidator(),
		contentValidator: validator.NewFileContentValidator(),
		log:              logger.GetLogger("loader"),
		loadedCount:      0,
		failedCount:      0,
	}
}

// LoadSingleFile loads a single markdown file and returns its content as lines.
func (l *Loader) LoadSingleFile(filePath string) ([]string, error) {
	defer logger.FuncLog(l.log, "LoadSingleFile")()

	// Validate the path
	validatedPath, err := l.pathValidator.ValidateMarkdownFile(filePath)
	if err != nil {
		l.failedCount++
		return nil, err
	}

	// Check file size
	if err := l.contentValidator.ValidateFileSize(validatedPath); err != nil {
		l.failedCount++
		return nil, err
	}

	// Read file
	lines, err := readFileLines(validatedPath)
	if err != nil {
		l.failedCount++
		return nil, exceptions.NewFileLoadError(
			validatedPath,
			"Failed to read file",
			err.Error(),
		)
	}

	l.loadedCount++
	l.log.Debug("Loaded file", "path", validatedPath, "lines", len(lines))
	return lines, nil
}

// LoadMarkdownFiles loads all markdown files from a directory (non-recursive).
// Returns a map of file paths to their content lines.
func (l *Loader) LoadMarkdownFiles(basePath string) (map[string][]string, error) {
	defer logger.FuncLog(l.log, "LoadMarkdownFiles")()

	validatedPath, err := l.pathValidator.ValidateDirectoryPath(basePath, true)
	if err != nil {
		return nil, err
	}

	files := make(map[string][]string)

	entries, err := os.ReadDir(validatedPath)
	if err != nil {
		return nil, exceptions.NewFileLoadError(
			validatedPath,
			"Failed to read directory",
			err.Error(),
		)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(validatedPath, entry.Name())
		if !l.pathValidator.IsMarkdownFile(filePath) {
			continue
		}

		lines, err := l.LoadSingleFile(filePath)
		if err != nil {
			l.log.Warn("Skipping file", "path", filePath, "error", err)
			continue
		}

		files[filePath] = lines
	}

	l.log.Info("Loaded markdown files",
		"directory", validatedPath,
		"loaded", len(files),
		"total_entries", len(entries),
	)
	return files, nil
}

// LoadMarkdownFilesRecursive loads all markdown files recursively from a directory.
func (l *Loader) LoadMarkdownFilesRecursive(basePath string) (map[string][]string, error) {
	defer logger.FuncLog(l.log, "LoadMarkdownFilesRecursive")()

	validatedPath, err := l.pathValidator.ValidateDirectoryPath(basePath, true)
	if err != nil {
		return nil, err
	}

	files := make(map[string][]string)

	err = filepath.Walk(validatedPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			l.log.Warn("Error accessing path", "path", path, "error", err)
			return nil // Skip inaccessible files
		}

		if info.IsDir() {
			return nil
		}

		if !l.pathValidator.IsMarkdownFile(path) {
			return nil
		}

		lines, loadErr := l.LoadSingleFile(path)
		if loadErr != nil {
			l.log.Warn("Skipping file", "path", path, "error", loadErr)
			return nil
		}

		files[path] = lines
		return nil
	})

	if err != nil {
		return files, fmt.Errorf("error walking directory: %w", err)
	}

	l.log.Info("Loaded markdown files recursively",
		"directory", validatedPath,
		"loaded", len(files),
	)
	return files, nil
}

// IsSingleFile checks if a path points to a single markdown file (not a directory).
func (l *Loader) IsSingleFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir() && l.pathValidator.IsMarkdownFile(path)
}

// GetStats returns loading statistics.
func (l *Loader) GetStats() map[string]int {
	return map[string]int{
		"loaded_files": l.loadedCount,
		"failed_files": l.failedCount,
	}
}

// ResetStats resets the loading statistics.
func (l *Loader) ResetStats() {
	l.loadedCount = 0
	l.failedCount = 0
}

// readFileLines reads a file and returns its lines.
func readFileLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	// Increase max line length (default is 64KB)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return lines, fmt.Errorf("error scanning file: %w", err)
	}

	// Handle empty file
	if lines == nil {
		lines = []string{}
	}

	return lines, nil
}

// DetectEncoding is a placeholder for encoding detection.
// In Go, we handle UTF-8 by default; users with other encodings
// should convert files or use a library like golang.org/x/text.
func DetectEncoding(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Read first 512 bytes for BOM detection
	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil {
		return "utf-8", nil // assume UTF-8 on read error
	}

	buf = buf[:n]

	// Check for BOM
	if len(buf) >= 3 && buf[0] == 0xEF && buf[1] == 0xBB && buf[2] == 0xBF {
		return "utf-8-bom", nil
	}
	if len(buf) >= 2 && buf[0] == 0xFF && buf[1] == 0xFE {
		return "utf-16le", nil
	}
	if len(buf) >= 2 && buf[0] == 0xFE && buf[1] == 0xFF {
		return "utf-16be", nil
	}

	// Check for valid UTF-8
	if !isValidUTF8(buf) {
		return "unknown", nil
	}

	return "utf-8", nil
}

// isValidUTF8 checks if a byte slice is valid UTF-8.
func isValidUTF8(buf []byte) bool {
	i := 0
	for i < len(buf) {
		if buf[i] < 0x80 {
			i++
			continue
		}
		// Multi-byte sequence
		if buf[i] >= 0xC0 && buf[i] <= 0xDF {
			if i+1 >= len(buf) || buf[i+1]&0xC0 != 0x80 {
				return false
			}
			i += 2
		} else if buf[i] >= 0xE0 && buf[i] <= 0xEF {
			if i+2 >= len(buf) || buf[i+1]&0xC0 != 0x80 || buf[i+2]&0xC0 != 0x80 {
				return false
			}
			i += 3
		} else if buf[i] >= 0xF0 && buf[i] <= 0xF4 {
			if i+3 >= len(buf) || buf[i+1]&0xC0 != 0x80 || buf[i+2]&0xC0 != 0x80 || buf[i+3]&0xC0 != 0x80 {
				return false
			}
			i += 4
		} else {
			return false
		}
	}
	return true
}

// --- Convenience helper ---

// IsMarkdownFile checks extension directly.
func IsMarkdownFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return validator.MarkdownExtensions[ext]
}
