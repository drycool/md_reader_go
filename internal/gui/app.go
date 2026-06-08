package gui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/drycool/md_reader_go/internal/toc"
	"github.com/drycool/md_reader_go/internal/viewer"
)

// NavItem represents an item in the sidebar navigation.
type NavItem struct {
	Title    string
	FilePath string
	Line     int
	Level    int
}

// App represents the GUI application state and components.
type App struct {
	window fyne.Window
	viewer *viewer.Viewer

	// Components
	searchEntry *widget.Entry
	navList     *widget.List
	contentArea *widget.RichText

	// Data
	currentPath string
	tocData     toc.TOCData
	files       map[string][]string
	navItems    []NavItem
}

// NewApp creates a new GUI application instance.
func NewApp(v *viewer.Viewer) *App {
	return &App{
		viewer: v,
	}
}

// Show initializes and displays the main GUI window.
func (a *App) Show(path string) {
	myApp := app.New()
	a.window = myApp.NewWindow("MD Reader")
	a.currentPath = path

	// Initialize components
	a.searchEntry = widget.NewEntry()
	a.searchEntry.SetPlaceHolder("Search headers...")
	a.searchEntry.OnChanged = a.onSearchChanged

	a.navList = widget.NewList(
		func() int { return len(a.navItems) },
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			item := a.navItems[id]
			label := obj.(*widget.Label)
			indent := strings.Repeat("  ", item.Level-1)
			if item.Level == 1 {
				label.TextStyle = fyne.TextStyle{Bold: true}
			} else {
				label.TextStyle = fyne.TextStyle{Bold: false}
			}
			label.SetText(fmt.Sprintf("%s%s", indent, item.Title))
		},
	)
	a.navList.OnSelected = a.onNavItemSelected

	a.contentArea = widget.NewRichTextFromMarkdown("# Welcome\nSelect a file or search to begin.")

	// Navigation Button
	openBtn := widget.NewButton("Open Folder", func() {
		d := dialog.NewFolderOpen(func(list fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, a.window)
				return
			}
			if list == nil {
				return
			}
			a.loadData(list.Path())
		}, a.window)
		d.Resize(fyne.NewSize(600, 400))
		d.Show()
	})

	// Layout
	topBar := container.NewVBox(openBtn, a.searchEntry)
	sidebar := container.NewBorder(topBar, nil, nil, nil, a.navList)
	split := container.NewHSplit(sidebar, container.NewScroll(a.contentArea))
	split.Offset = 0.3

	a.window.SetContent(split)
	a.window.Resize(fyne.NewSize(1000, 600))

	// Initial data load
	a.loadData(path)

	if len(a.navItems) > 0 {
		a.navList.Select(0)
	} else {
		a.contentArea.ParseMarkdown("# No Markdown files found\nTry opening a directory that contains `.md` files with headers (e.g., `# Header`).")
	}

	a.window.ShowAndRun()
}

func (a *App) loadData(path string) {
	fmt.Printf("Loading data from: %s\n", path)
	tocData, files, err := a.viewer.LoadAndBuildTOC(path)
	if err != nil {
		fmt.Printf("Load error: %v\n", err)
		a.contentArea.ParseMarkdown(fmt.Sprintf("# Error\nFailed to load files: %v", err))
		return
	}

	fmt.Printf("Loaded %d files, %d total headers\n", len(files), countHeaders(tocData))
	a.tocData = tocData
	a.files = files
	a.buildInitialNav()
}

func countHeaders(data toc.TOCData) int {
	c := 0
	for _, h := range data {
		c += len(h)
	}
	return c
}

func (a *App) buildInitialNav() {
	var items []NavItem
	
	// Sort file paths for consistent display
	var paths []string
	for p := range a.tocData {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, p := range paths {
		headers := a.tocData[p]
		// Add file header as a NavItem
		items = append(items, NavItem{
			Title:    filepath.Base(p),
			FilePath: p,
			Line:     0,
			Level:    1,
		})
		
		for _, h := range headers {
			items = append(items, NavItem{
				Title:    h.Title,
				FilePath: p,
				Line:     h.LineNumber,
				Level:    h.Level + 1, // Indent headers under file
			})
		}
	}

	a.navItems = items
	a.navList.Refresh()
}

func (a *App) onSearchChanged(query string) {
	if query == "" {
		a.buildInitialNav()
		return
	}

	results := a.viewer.FuzzySearch(a.tocData, query)
	var items []NavItem
	for _, r := range results {
		items = append(items, NavItem{
			Title:    r.Title,
			FilePath: r.FilePath,
			Line:     r.Line,
			Level:    1,
		})
	}

	a.navItems = items
	a.navList.Refresh()
}

func (a *App) onNavItemSelected(id widget.ListItemID) {
	item := a.navItems[id]
	lines, ok := a.files[item.FilePath]
	if !ok {
		return
	}

	start, end := toc.FindSectionBounds(lines, item.Line)
	sectionContent := strings.Join(lines[start:end], "\n")
	
	// Prepend title and metadata
	header := fmt.Sprintf("# %s\n*File: %s (Lines %d-%d)*\n\n---\n\n", 
		item.Title, filepath.Base(item.FilePath), start+1, end)
	
	a.contentArea.ParseMarkdown(header + sectionContent)
}
