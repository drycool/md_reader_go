package gui

import (
	"fmt"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/drycool/md_reader_go/internal/toc"
	"github.com/drycool/md_reader_go/internal/viewer"
)

// Sidebar manages the search entry and navigation list.
type Sidebar struct {
	SearchEntry *widget.Entry
	List        *widget.List
	Items       []ListItem
	onSelect    func(item ListItem)
}

// NewSidebar creates a new sidebar with search and list.
func NewSidebar(onSelect func(ListItem)) *Sidebar {
	sb := &Sidebar{
		Items:    nil,
		onSelect: onSelect,
	}

	sb.SearchEntry = widget.NewEntry()
	sb.SearchEntry.SetPlaceHolder("Search headers & files...")
	sb.SearchEntry.OnChanged = sb.onSearchChanged

	sb.List = widget.NewList(
		func() int { return len(sb.Items) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("template")
			label.Truncation = fyne.TextTruncateEllipsis
			return label
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(sb.Items) {
				item := sb.Items[id]
				label := obj.(*widget.Label)
				if item.Score > 0 {
					label.SetText(fmt.Sprintf("[%d] %s", item.Score, item.Title))
				} else {
					label.SetText(item.Title)
				}
				if !item.IsHeader {
					label.TextStyle = fyne.TextStyle{Bold: true}
				} else {
					label.TextStyle = fyne.TextStyle{}
				}
				label.Refresh()
			}
		},
	)

	sb.List.OnSelected = func(id widget.ListItemID) {
		if id < len(sb.Items) && sb.onSelect != nil {
			sb.onSelect(sb.Items[id])
		}
	}

	return sb
}

// Container returns the sidebar layout.
func (sb *Sidebar) Container() *fyne.Container {
	return container.NewBorder(sb.SearchEntry, nil, nil, nil, sb.List)
}

// LoadTOC populates the sidebar with all headers from the TOC.
func (sb *Sidebar) LoadTOC(tocData toc.TOCData) {
	var items []ListItem

	for filePath, headers := range tocData {
		baseName := filepath.Base(filePath)
		items = append(items, ListItem{
			Title:    "📄 " + baseName,
			FilePath: filePath,
			Line:     0,
			Score:    0,
			IsHeader: false,
		})

		for _, h := range headers {
			indent := strings.Repeat("  ", h.Level-1)
			items = append(items, ListItem{
				Title:    indent + h.Title,
				FilePath: filePath,
				Line:     h.LineNumber,
				Score:    0,
				IsHeader: true,
			})
		}
	}

	sb.Items = items
	sb.List.Refresh()
}

// UpdateResults replaces sidebar items with search results.
func (sb *Sidebar) UpdateResults(results []viewer.SearchResult) {
	var items []ListItem
	for _, r := range results {
		items = append(items, ListItem{
			Title:    r.Title,
			FilePath: r.FilePath,
			Line:     r.Line,
			Score:    r.Score,
			IsHeader: true,
		})
	}
	sb.Items = items
	sb.List.Refresh()
}

// onSearchChanged is called when the search entry text changes.
// This will be wired to the viewer's FuzzySearch by the app.
func (sb *Sidebar) onSearchChanged(query string) {
	// Handled externally by App via callback
}

// SetOnSearch sets an external handler for search changes.
func (sb *Sidebar) SetOnSearch(fn func(query string)) {
	sb.SearchEntry.OnChanged = fn
}
