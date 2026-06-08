// Package validator provides safe path, file, and input validation
// to prevent path traversal, control characters, and other security issues.
package validator

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/drycool/md_reader_go/internal/exceptions"
)

// Constants mirroring the Python PathValidator.
const (
	MaxPathLength = 260
	MaxInputLength = 1000
)

// MarkdownExtensions is the set of valid markdown file extensions.
var MarkdownExtensions = map[string]bool{
	".md":        true,
	".markdown":  true,
	".mdown":     true,
	".mkd":       true,
}

// dangerousPatterns are checked against paths for security.
// Note: ':' is allowed because Windows drive letters use it (e.g. D:\).
// We validate colon position separately below.
var dangerousPatterns = []*regexp.Regexp{
	// Path traversal (..)
	regexp.MustCompile(`\.\.[\\/]`),
	// Control characters 0x00-0x1f
	regexp.MustCompile(`[\x00-\x1f]`),
	// Windows-invalid characters (excluding ':' which is valid in drive letters)
	regexp.MustCompile(`[<>"|?*]`),
}

// PathValidator validates file system paths for safety.
type PathValidator struct{}

// NewPathValidator creates a new PathValidator.
func NewPathValidator() *PathValidator {
	return &PathValidator{}
}

// ValidatePath validates a path for safety and correctness.
// If mustExist is true, the path must exist on disk.
func (v *PathValidator) ValidatePath(path string, mustExist bool) (string, error) {
	if path == "" {
		return "", exceptions.NewPathValidationError(path, "Path cannot be empty")
	}

	// Length check
	if len(path) > MaxPathLength {
		return "", exceptions.NewPathValidationError(
			path,
			"Path too long",
			MaxPathMsg(),
		)
	}

	// Check against dangerous patterns
	for _, pattern := range dangerousPatterns {
		if pattern.MatchString(path) {
			return "", exceptions.NewPathValidationError(
				path,
				"Path contains dangerous pattern",
			)
		}
	}

	// Validate colon usage: only allowed as drive letter separator (e.g. D:\)
	// Any other colon is invalid on Windows.
	if idx := strings.Index(path, ":"); idx >= 0 {
		if idx != 1 || !unicode.IsLetter(rune(path[0])) {
			return "", exceptions.NewPathValidationError(
				path,
				"Invalid colon in path (only drive letters like C:\\ are allowed)",
			)
		}
	}

	// Normalize the path
	cleaned := filepath.Clean(path)
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", exceptions.NewPathValidationError(
			path,
			"Invalid path format",
			err.Error(),
		)
	}

	// Check existence if required
	if mustExist {
		if _, err := os.Stat(abs); os.IsNotExist(err) {
			return "", exceptions.NewPathValidationError(
				abs,
				"Path does not exist",
			)
		}
	}

	return abs, nil
}

// ValidateFilePath validates a path to a file.
// If checkReadable is true, also checks that the file is readable.
func (v *PathValidator) ValidateFilePath(path string, checkReadable bool) (string, error) {
	validated, err := v.ValidatePath(path, true)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(validated)
	if err != nil {
		return "", exceptions.NewPathValidationError(
			validated,
			"Cannot stat path",
			err.Error(),
		)
	}

	if info.IsDir() {
		return "", exceptions.NewPathValidationError(
			validated,
			"Path is not a file",
		)
	}

	if checkReadable {
		f, err := os.Open(validated)
		if err != nil {
			return "", exceptions.NewFileLoadError(
				validated,
				"File is not readable",
				err.Error(),
			)
		}
		f.Close()
	}

	return validated, nil
}

// ValidateDirectoryPath validates a path to a directory.
func (v *PathValidator) ValidateDirectoryPath(path string, checkReadable bool) (string, error) {
	validated, err := v.ValidatePath(path, true)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(validated)
	if err != nil {
		return "", exceptions.NewPathValidationError(
			validated,
			"Cannot stat path",
			err.Error(),
		)
	}

	if !info.IsDir() {
		return "", exceptions.NewPathValidationError(
			validated,
			"Path is not a directory",
		)
	}

	if checkReadable {
		f, err := os.Open(validated)
		if err != nil {
			return "", exceptions.NewPathValidationError(
				validated,
				"Directory is not readable",
				err.Error(),
			)
		}
		f.Close()
	}

	return validated, nil
}

// IsMarkdownFile checks if a file has a markdown extension.
func (v *PathValidator) IsMarkdownFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return MarkdownExtensions[ext]
}

// ValidateMarkdownFile validates that a path points to a readable markdown file.
func (v *PathValidator) ValidateMarkdownFile(path string) (string, error) {
	validated, err := v.ValidateFilePath(path, true)
	if err != nil {
		return "", err
	}

	if !v.IsMarkdownFile(validated) {
		return "", exceptions.NewPathValidationError(
			validated,
			"File is not a Markdown file",
		)
	}

	return validated, nil
}

// --- Input Validator ---

// InputValidator validates user input.
type InputValidator struct{}

// NewInputValidator creates a new InputValidator.
func NewInputValidator() *InputValidator {
	return &InputValidator{}
}

// ValidateSearchQuery validates and sanitizes a search query.
func (iv *InputValidator) ValidateSearchQuery(query string) (string, error) {
	if !isString(query) {
		return "", exceptions.NewUserInputError(
			query,
			"Query must be a string",
		)
	}

	cleaned := strings.TrimSpace(query)
	if cleaned == "" {
		return "", exceptions.NewUserInputError(
			query,
			"Query cannot be empty",
		)
	}

	if len(cleaned) > MaxInputLength {
		return "", exceptions.NewUserInputError(
			query[:min(len(query), 50)]+"...",
			"Query too long",
		)
	}

	// Check for control characters
	for _, r := range cleaned {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return "", exceptions.NewUserInputError(
				cleaned,
				"Query contains control characters",
			)
		}
	}

	return cleaned, nil
}

// --- File Content Validator ---

// FileContentValidator validates file content.
type FileContentValidator struct{}

// NewFileContentValidator creates a new FileContentValidator.
func NewFileContentValidator() *FileContentValidator {
	return &FileContentValidator{}

}

// MaxFileSize is the maximum file size to load (10MB).
const MaxFileSize = 10 * 1024 * 1024

// ValidateFileSize checks that a file doesn't exceed the maximum size.
func (fcv *FileContentValidator) ValidateFileSize(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return exceptions.NewFileLoadError(path, "Cannot stat file", err.Error())
	}

	if info.Size() > MaxFileSize {
		return exceptions.NewFileLoadError(
			path,
			"File too large",
		)
	}
	return nil
}

// --- helpers ---

func isString(v any) bool {
	_, ok := v.(string)
	return ok
}

// MaxPathMsg returns a message about max path length.
func MaxPathMsg() string {
	return "Path too long (max 260 characters)"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
