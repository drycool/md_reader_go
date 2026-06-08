package gui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ContentPanel manages the markdown content display area.
type ContentPanel struct {
	RichText *widget.RichText
	Scroll   fyne.CanvasObject
}

// NewContentPanel creates a new scrollable RichText content panel.
func NewContentPanel() *ContentPanel {
	rt := widget.NewRichTextFromMarkdown("# Welcome\n\nSelect a file or header from the sidebar to view its content.")
	rt.Wrapping = fyne.TextWrapWord

	scroll := container.NewVScroll(rt)

	return &ContentPanel{
		RichText: rt,
		Scroll:   scroll,
	}
}

// SetMarkdown updates the content panel with new markdown text.
func (cp *ContentPanel) SetMarkdown(md string) {
	if md == "" {
		md = "_No content available._"
	}
	cp.RichText.ParseMarkdown(md)
	cp.Refresh()
}

// Refresh forces a redraw of the content panel.
func (cp *ContentPanel) Refresh() {
	cp.RichText.Refresh()
	cp.Scroll.Refresh()
}

// ShowSection displays a specific section from the loaded files.
func (cp *ContentPanel) ShowSection(files map[string][]string, filePath string, startLine int) {
	lines, ok := files[filePath]
	if !ok {
		cp.SetMarkdown("**File not found:** " + filePath)
		return
	}

	// Find the end of this section (next header or EOF)
	endLine := len(lines)
	for i := startLine + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "#") {
			endLine = i
			break
		}
	}

	if startLine >= len(lines) {
		startLine = len(lines) - 1
	}
	if startLine < 0 {
		startLine = 0
	}

	section := lines[startLine:endLine]
	md := strings.Join(section, "\n")
	cp.SetMarkdown(md)
}
