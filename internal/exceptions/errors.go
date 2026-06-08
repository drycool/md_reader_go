// Package exceptions provides custom error types for the md-reader application.
// This mirrors the Python md_reader.exceptions module hierarchy.
package exceptions

import (
	"fmt"
)

// MdReaderError is the base error type for all md-reader errors.
type MdReaderError struct {
	Message string
	Details string
	Err     error
}

func (e *MdReaderError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s | Details: %s", e.Message, e.Details)
	}
	return e.Message
}

func (e *MdReaderError) Unwrap() error {
	return e.Err
}

// NewMdReaderError creates a new MdReaderError.
func NewMdReaderError(msg string, details ...string) *MdReaderError {
	d := ""
	if len(details) > 0 {
		d = details[0]
	}
	return &MdReaderError{Message: msg, Details: d}
}

// --- Concrete Error Types ---

// FileLoadError represents errors during file loading.
type FileLoadError struct {
	MdReaderError
	FilePath string
}

func NewFileLoadError(filePath, msg string, details ...string) *FileLoadError {
	d := ""
	if len(details) > 0 {
		d = details[0]
	}
	return &FileLoadError{
		MdReaderError: MdReaderError{
			Message: fmt.Sprintf("Error loading file '%s': %s", filePath, msg),
			Details: d,
		},
		FilePath: filePath,
	}
}

// PathValidationError represents errors during path validation.
type PathValidationError struct {
	MdReaderError
	Path string
}

func NewPathValidationError(path, msg string, details ...string) *PathValidationError {
	d := ""
	if len(details) > 0 {
		d = details[0]
	}
	return &PathValidationError{
		MdReaderError: MdReaderError{
			Message: fmt.Sprintf("Invalid path '%s': %s", path, msg),
			Details: d,
		},
		Path: path,
	}
}

// MarkdownParsingError represents errors during markdown parsing.
type MarkdownParsingError struct {
	MdReaderError
	FilePath   string
	LineNumber int
}

func NewMarkdownParsingError(filePath string, lineNum int, msg string, details ...string) *MarkdownParsingError {
	d := ""
	if len(details) > 0 {
		d = details[0]
	}
	return &MarkdownParsingError{
		MdReaderError: MdReaderError{
			Message: fmt.Sprintf("Markdown parsing error in '%s' at line %d: %s", filePath, lineNum, msg),
			Details: d,
		},
		FilePath:   filePath,
		LineNumber: lineNum,
	}
}

// TocBuildError represents errors during TOC building.
type TocBuildError struct {
	MdReaderError
	FilePath string
}

func NewTocBuildError(msg string, filePath ...string) *TocBuildError {
	fp := ""
	if len(filePath) > 0 {
		fp = filePath[0]
	}
	fullMsg := msg
	if fp != "" {
		fullMsg = fmt.Sprintf("TOC build error for '%s': %s", fp, msg)
	}
	return &TocBuildError{
		MdReaderError: MdReaderError{Message: fullMsg},
		FilePath:      fp,
	}
}

// ConfigurationError represents configuration errors.
type ConfigurationError struct {
	MdReaderError
	Setting string
}

func NewConfigurationError(setting, msg string, details ...string) *ConfigurationError {
	d := ""
	if len(details) > 0 {
		d = details[0]
	}
	return &ConfigurationError{
		MdReaderError: MdReaderError{
			Message: fmt.Sprintf("Configuration error for '%s': %s", setting, msg),
			Details: d,
		},
		Setting: setting,
	}
}

// UserInputError represents invalid user input.
type UserInputError struct {
	MdReaderError
	InputValue string
}

func NewUserInputError(inputValue, msg string, details ...string) *UserInputError {
	d := ""
	if len(details) > 0 {
		d = details[0]
	}
	return &UserInputError{
		MdReaderError: MdReaderError{
			Message: fmt.Sprintf("Invalid user input '%s': %s", inputValue, msg),
			Details: d,
		},
		InputValue: inputValue,
	}
}

// CacheError represents cache operation errors.
type CacheError struct {
	MdReaderError
	Operation string
}

func NewCacheError(operation, msg string, details ...string) *CacheError {
	d := ""
	if len(details) > 0 {
		d = details[0]
	}
	return &CacheError{
		MdReaderError: MdReaderError{
			Message: fmt.Sprintf("Cache error during '%s': %s", operation, msg),
			Details: d,
		},
		Operation: operation,
	}
}

// IsFileLoadError checks if an error is a FileLoadError.
func IsFileLoadError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*FileLoadError)
	return ok
}

// IsPathValidationError checks if an error is a PathValidationError.
func IsPathValidationError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*PathValidationError)
	return ok
}

// IsUserInputError checks if an error is a UserInputError.
func IsUserInputError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*UserInputError)
	return ok
}
