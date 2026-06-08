package gui

import (
	"github.com/drycool/md_reader_go/internal/toc"
	"github.com/drycool/md_reader_go/internal/viewer"
)

// State holds the GUI application state.
type State struct {
	TOCData   toc.TOCData
	Files     map[string][]string
	Results   []viewer.SearchResult
	AllItems  []ListItem
	BasePath  string
}

// ListItem represents a single item in the sidebar list.
type ListItem struct {
	Title    string
	FilePath string
	Line     int
	Score    int
	IsHeader bool // true = header entry, false = file-level entry
}

// NewState creates a new empty state.
func NewState() *State {
	return &State{
		TOCData:  make(toc.TOCData),
		Files:    make(map[string][]string),
		Results:  nil,
		AllItems: nil,
	}
}
