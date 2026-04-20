package ui

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mipmip/specgetty/src/scanner"
	"github.com/mipmip/specgetty/src/watcher"
)

const (
	viewProjects = 0
	viewDetail   = 1
	viewLog      = 2
)

const (
	tabSpecs   = 0
	tabChanges = 1
	tabArchive = 2
	tabConfig  = 3
)

const (
	archiveIdle       = 0
	archiveConfirming = 1
	archiveRunning    = 2
	archiveResult     = 3
)

const (
	discardIdle       = 0
	discardConfirming = 1
	discardRunning    = 2
	discardResult     = 3
)

const (
	exportIdle       = 0
	exportConfirming = 1
	exportRunning    = 2
	exportResult     = 3
)

var tabNames = []string{"specs", "changes", "archive", "config"}

// Message types

type scanMsg struct {
	projects scanner.ProjectMap
	err      error
}

type logMsg string

// fsChangeMsg is sent when the filesystem watcher detects changes in the openspec directory.
type fsChangeMsg struct{}

// logWriter sends log output as tea messages to the program.
type logWriter struct {
	program *tea.Program
}

func (w logWriter) Write(p []byte) (n int, err error) {
	w.program.Send(logMsg(string(p)))
	return len(p), nil
}

// Styles

var (
	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("2")). // green
			Foreground(lipgloss.Color("0")). // black
			Width(0)                         // set dynamically

	normalStyle = lipgloss.NewStyle()

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("1")). // red
			Padding(1, 2).
			Align(lipgloss.Center)

	navBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252"))

	navBarKeyStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("2")).
			Bold(true)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("4")) // blue

	activeTabStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("2")).
			Foreground(lipgloss.Color("0")).
			Bold(true).
			Padding(0, 1)

	inactiveTabStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("236")).
				Foreground(lipgloss.Color("250")).
				Padding(0, 1)

	sectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("3")) // yellow

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250"))
)

type model struct {
	config          *scanner.Config
	ignoreDirErrors bool
	projects        scanner.ProjectMap
	repoPaths       []string
	displayNames    []string
	cursor          int
	activeView      int
	scanning        bool
	err             error
	spinner         spinner.Model
	detailViewport  viewport.Model
	logViewport     viewport.Model
	logContent      string
	detailTab        int
	specCursor       int
	changeCursor      int
	changeArtifactTab int
	archiveCursor     int
	archiveArtifactTab int
	fileCursor      int
	filePaths       []string
	zoomed          bool
	initialZoomPath string // set via CLI --zoom, triggers single-project scan
	fullScanDone    bool   // tracks whether a full scan has been run
	logVisible        bool
	logShownOnce      bool
	pendingKey        string
	archiveState      int
	archiveChangeName string
	archiveResultMsg  string
	archiveResultOk   bool
	discardState      int
	discardChangeName string
	discardResultMsg  string
	discardResultOk   bool
	exportState       int
	exportChangeName  string
	exportResultMsg   string
	exportResultOk    bool
	exportIsArchived  bool
	width           int
	height          int
	program         *tea.Program
	version         string
	watcher         *watcher.Watcher
}

func newModel(config *scanner.Config, ignoreDirErrors bool, version string) model {
	s := spinner.New()
	s.Spinner = spinner.Dot

	return model{
		config:          config,
		ignoreDirErrors: ignoreDirErrors,
		scanning:        true,
		version:         version,
		spinner:         s,
		detailViewport:  viewport.New(0, 0),
		logViewport:     viewport.New(0, 0),
	}
}

func (m model) Init() tea.Cmd {
	if m.initialZoomPath != "" {
		return tea.Batch(
			m.spinner.Tick,
			m.doScanSingle(m.initialZoomPath),
		)
	}
	return tea.Batch(
		m.spinner.Tick,
		m.doScan(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalcLayout()

	case tea.KeyMsg:
		if m.scanning {
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			}
			return m, nil
		}

		if m.err != nil {
			m.err = nil
			return m, nil
		}

		// Archive modal intercepts all keys
		if m.archiveState == archiveConfirming {
			switch msg.String() {
			case "y":
				m.archiveState = archiveRunning
				projectPath := m.repoPaths[m.cursor]
				cmds = append(cmds, doArchiveChange(projectPath, m.archiveChangeName))
			case "n", "esc":
				m.archiveState = archiveIdle
			}
			return m, tea.Batch(cmds...)
		}
		if m.archiveState == archiveResult {
			m.archiveState = archiveIdle
			return m, nil
		}
		if m.archiveState == archiveRunning {
			return m, nil
		}

		// Export modal intercepts all keys
		if m.exportState == exportConfirming {
			switch msg.String() {
			case "y":
				m.exportState = exportRunning
				projectPath := m.repoPaths[m.cursor]
				cmds = append(cmds, doExportChange(projectPath, m.exportChangeName, m.exportIsArchived))
			case "n", "esc":
				m.exportState = exportIdle
			}
			return m, tea.Batch(cmds...)
		}
		if m.exportState == exportResult {
			m.exportState = exportIdle
			return m, nil
		}
		if m.exportState == exportRunning {
			return m, nil
		}

		// Discard modal intercepts all keys
		if m.discardState == discardConfirming {
			switch msg.String() {
			case "y":
				m.discardState = discardRunning
				projectPath := m.repoPaths[m.cursor]
				cmds = append(cmds, doDiscardChange(projectPath, m.discardChangeName))
			case "n", "esc":
				m.discardState = discardIdle
			}
			return m, tea.Batch(cmds...)
		}
		if m.discardState == discardResult {
			m.discardState = discardIdle
			return m, nil
		}
		if m.discardState == discardRunning {
			return m, nil
		}

		key := msg.String()
		if m.pendingKey == "g" {
			m.pendingKey = ""
			if key == "g" {
				switch m.activeView {
				case viewProjects:
					m.cursor = 0
					m.updateFileList()
				case viewDetail:
					m.fileCursor = 0
				case viewLog:
					m.logViewport.GotoTop()
				}
				return m, nil
			}
		}

		switch key {
		case "q", "ctrl+c":
			m.stopWatcher()
			return m, tea.Quit
		case "enter":
			if m.zoomed {
				// Exit zoom
				m.stopWatcher()
				m.zoomed = false
				m.activeView = viewProjects
				if !m.fullScanDone {
					m.scanning = true
					cmds = append(cmds, m.doScan())
				}
			} else if m.activeView == viewProjects && len(m.repoPaths) > 0 {
				// Enter zoom
				m.zoomed = true
				m.activeView = viewDetail
				if cmd := m.startWatcher(m.repoPaths[m.cursor]); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		case "esc":
			if m.zoomed {
				m.stopWatcher()
				m.zoomed = false
				m.activeView = viewProjects
				if !m.fullScanDone {
					m.scanning = true
					cmds = append(cmds, m.doScan())
				}
			}
		case "s":
			m.scanning = true
			if m.zoomed && len(m.repoPaths) > 0 {
				cmds = append(cmds, m.doScanSingle(m.repoPaths[m.cursor]))
			} else {
				cmds = append(cmds, m.doScan())
			}
		case "l":
			m.logVisible = !m.logVisible
			if m.logVisible {
				m.recalcLayout()
				if !m.logShownOnce {
					m.logShownOnce = true
					m.logViewport.GotoBottom()
				}
			} else {
				if m.activeView == viewLog {
					m.activeView = viewProjects
				}
				m.recalcLayout()
			}
		case "tab":
			if m.zoomed {
				// No panel switching in zoom mode
			} else if m.logVisible {
				m.activeView = (m.activeView + 1) % 3
			} else {
				if m.activeView == viewProjects {
					m.activeView = viewDetail
				} else {
					m.activeView = viewProjects
				}
			}
		case "g":
			m.pendingKey = "g"
		case "G":
			switch m.activeView {
			case viewProjects:
				if len(m.repoPaths) > 0 {
					m.cursor = len(m.repoPaths) - 1
					m.updateFileList()
				}
			case viewDetail:
				if len(m.filePaths) > 0 {
					m.fileCursor = len(m.filePaths) - 1
				}
			case viewLog:
				m.logViewport.GotoBottom()
			}
		case "pgdown", "ctrl+f":
			half := m.halfPage()
			switch m.activeView {
			case viewProjects:
				m.cursor = min(m.cursor+half, len(m.repoPaths)-1)
				m.updateFileList()
			case viewDetail:
				if len(m.filePaths) > 0 {
					m.fileCursor = min(m.fileCursor+half, len(m.filePaths)-1)
				}
			case viewLog:
				m.logViewport.LineDown(half)
			}
		case "pgup", "ctrl+b":
			half := m.halfPage()
			switch m.activeView {
			case viewProjects:
				m.cursor = max(m.cursor-half, 0)
				m.updateFileList()
			case viewDetail:
				if len(m.filePaths) > 0 {
					m.fileCursor = max(m.fileCursor-half, 0)
				}
			case viewLog:
				m.logViewport.LineUp(half)
			}
		case "left":
			if m.activeView == viewDetail {
				if m.detailTab == tabChanges && len(m.currentChanges()) > 0 && m.changeArtifactTab > 0 {
					m.changeArtifactTab--
				} else if m.detailTab == tabArchive && len(m.currentArchivedChanges()) > 0 && m.archiveArtifactTab > 0 {
					m.archiveArtifactTab--
				} else if m.detailTab > 0 {
					m.detailTab--
					m.changeArtifactTab = 0
					m.archiveArtifactTab = 0
				}
			}
		case "right":
			if m.activeView == viewDetail {
				if m.detailTab == tabChanges && len(m.currentChanges()) > 0 {
					maxTab := m.changeArtifactTabCount() - 1
					if m.changeArtifactTab < maxTab {
						m.changeArtifactTab++
					} else if m.detailTab < len(tabNames)-1 {
						m.detailTab++
						m.changeArtifactTab = 0
					}
				} else if m.detailTab == tabArchive && len(m.currentArchivedChanges()) > 0 {
					maxTab := m.archiveArtifactTabCount() - 1
					if m.archiveArtifactTab < maxTab {
						m.archiveArtifactTab++
					} else if m.detailTab < len(tabNames)-1 {
						m.detailTab++
						m.archiveArtifactTab = 0
					}
				} else if m.detailTab < len(tabNames)-1 {
					m.detailTab++
				}
			}
		case "1":
			if m.activeView == viewDetail {
				m.detailTab = tabSpecs
			}
		case "2":
			if m.activeView == viewDetail {
				m.detailTab = tabChanges
			}
		case "3":
			if m.activeView == viewDetail {
				m.detailTab = tabArchive
			}
		case "4":
			if m.activeView == viewDetail {
				m.detailTab = tabConfig
			}
		case "up", "k":
			if m.activeView == viewProjects {
				if m.cursor > 0 {
					m.cursor--
					m.updateFileList()
				}
			} else if m.activeView == viewDetail {
				if m.detailTab == tabSpecs {
					if m.specCursor > 0 {
						m.specCursor--
					}
				} else if m.detailTab == tabChanges {
					if m.changeCursor > 0 {
						m.changeCursor--
						m.changeArtifactTab = 0
					}
				} else if m.detailTab == tabArchive {
					if m.archiveCursor > 0 {
						m.archiveCursor--
						m.archiveArtifactTab = 0
					}
				} else if len(m.filePaths) > 0 && m.fileCursor > 0 {
					m.fileCursor--
				}
			} else if m.activeView == viewLog {
				m.logViewport.LineUp(1)
			}
		case "down", "j":
			if m.activeView == viewProjects {
				if m.cursor < len(m.repoPaths)-1 {
					m.cursor++
					m.updateFileList()
				}
			} else if m.activeView == viewDetail {
				if m.detailTab == tabSpecs {
					specNames := m.currentSpecNames()
					if m.specCursor < len(specNames)-1 {
						m.specCursor++
					}
				} else if m.detailTab == tabChanges {
					changes := m.currentChanges()
					if m.changeCursor < len(changes)-1 {
						m.changeCursor++
						m.changeArtifactTab = 0
					}
				} else if m.detailTab == tabArchive {
					archived := m.currentArchivedChanges()
					if m.archiveCursor < len(archived)-1 {
						m.archiveCursor++
						m.archiveArtifactTab = 0
					}
				} else if len(m.filePaths) > 0 && m.fileCursor < len(m.filePaths)-1 {
					m.fileCursor++
				}
			} else if m.activeView == viewLog {
				m.logViewport.LineDown(1)
			}
		case "a":
			if m.detailTab == tabChanges {
				changes := m.currentChanges()
				if len(changes) > 0 && m.changeCursor < len(changes) {
					m.archiveChangeName = changes[m.changeCursor].Name
					m.archiveState = archiveConfirming
				}
			}
		case "d":
			if m.detailTab == tabChanges {
				changes := m.currentChanges()
				if len(changes) > 0 && m.changeCursor < len(changes) {
					m.discardChangeName = changes[m.changeCursor].Name
					m.discardState = discardConfirming
				}
			}
		case "e":
			if m.detailTab == tabChanges {
				changes := m.currentChanges()
				if len(changes) > 0 && m.changeCursor < len(changes) {
					m.exportChangeName = changes[m.changeCursor].Name
					m.exportIsArchived = false
					m.exportState = exportConfirming
				}
			} else if m.detailTab == tabArchive {
				archived := m.currentArchivedChanges()
				if len(archived) > 0 && m.archiveCursor < len(archived) {
					m.exportChangeName = archived[m.archiveCursor].Name
					m.exportIsArchived = true
					m.exportState = exportConfirming
				}
			}
		}

	case archiveMsg:
		m.archiveResultOk = msg.ok
		m.archiveResultMsg = msg.output
		m.archiveState = archiveResult
		if msg.ok && len(m.repoPaths) > 0 {
			cmds = append(cmds, m.doScanSingle(m.repoPaths[m.cursor]))
		}

	case discardMsg:
		m.discardResultOk = msg.ok
		m.discardResultMsg = msg.output
		m.discardState = discardResult
		if msg.ok && len(m.repoPaths) > 0 {
			cmds = append(cmds, m.doScanSingle(m.repoPaths[m.cursor]))
		}

	case exportMsg:
		m.exportResultOk = msg.ok
		m.exportResultMsg = msg.output
		m.exportState = exportResult

	case fsChangeMsg:
		if m.zoomed && len(m.repoPaths) > 0 {
			m.scanning = true
			cmds = append(cmds, m.doScanSingle(m.repoPaths[m.cursor]))
			if m.watcher != nil {
				cmds = append(cmds, waitForFsChange(m.watcher))
			}
		}

	case scanMsg:
		m.scanning = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			// Track if this was a full scan (more than 1 project or not zoomed)
			if !m.zoomed || len(msg.projects) > 1 {
				m.fullScanDone = true
			}
			m.projects = msg.projects
			m.repoPaths = make([]string, 0, len(m.projects))
			for r := range m.projects {
				m.repoPaths = append(m.repoPaths, r)
			}
			sort.Strings(m.repoPaths)
			m.displayNames = projectDisplayNames(m.repoPaths)
			if m.cursor >= len(m.repoPaths) {
				m.cursor = max(0, len(m.repoPaths)-1)
			}
			m.recalcLayout()
			m.updateFileList()

			// Start watcher after initial scan if zoomed via CLI
			if m.zoomed && m.watcher == nil && len(m.repoPaths) > 0 {
				if cmd := m.startWatcher(m.repoPaths[m.cursor]); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}

	case logMsg:
		m.logContent += string(msg)
		m.logViewport.SetContent(m.logContent)
		m.logViewport.GotoBottom()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// projectDisplayNames returns basenames for each path, disambiguating duplicates
// by appending the parent directory name.
func projectDisplayNames(paths []string) []string {
	names := make([]string, len(paths))
	baseCounts := make(map[string]int)

	for _, p := range paths {
		base := filepath.Base(p)
		baseCounts[base]++
	}

	for i, p := range paths {
		base := filepath.Base(p)
		if baseCounts[base] > 1 {
			parent := filepath.Base(filepath.Dir(p))
			names[i] = base + " (" + parent + ")"
		} else {
			names[i] = base
		}
	}
	return names
}

func (m model) leftPanelWidth() int {
	w := m.width * 3 / 10
	if w < 20 {
		w = 20
	}
	if w > 40 {
		w = 40
	}
	return w
}

func (m model) rightPanelWidth() int {
	return m.width - m.leftPanelWidth() - 1 // 1 for gap between panels
}

func (m *model) recalcLayout() {
	if m.width == 0 || m.height == 0 {
		return
	}

	rightInner := m.rightPanelWidth() - 2 // border
	panelH := m.mainPanelHeight()

	m.detailViewport.Width = rightInner
	m.detailViewport.Height = panelH

	logHeight := m.logPanelHeight()
	if logHeight > 0 {
		m.logViewport.Width = m.width - 2
		m.logViewport.Height = logHeight
	}
}

func (m model) mainPanelHeight() int {
	logH := m.logPanelHeight()
	if logH > 0 {
		logH += 2 // border
	}
	remaining := m.height - logH - 1 // -1 for nav bar
	if remaining < 5 {
		return 3
	}
	return remaining - 2 // -border
}

func (m model) logPanelHeight() int {
	if !m.logVisible {
		return 0
	}
	return min(10, (m.height-6)/3)
}

func (m model) halfPage() int {
	switch m.activeView {
	case viewDetail:
		return max(1, m.mainPanelHeight()/2)
	case viewLog:
		return max(1, m.logViewport.Height/2)
	default:
		return max(1, m.mainPanelHeight()/2)
	}
}

func (m model) currentSpecNames() []string {
	if len(m.repoPaths) == 0 {
		return nil
	}
	return m.projects[m.repoPaths[m.cursor]].Info.SpecNames
}

func (m model) currentChanges() []scanner.ChangeInfo {
	if len(m.repoPaths) == 0 {
		return nil
	}
	return m.projects[m.repoPaths[m.cursor]].Info.Changes
}

func (m model) changeArtifactTabCount() int {
	changes := m.currentChanges()
	if m.changeCursor >= len(changes) {
		return 0
	}
	ci := changes[m.changeCursor]
	count := len(ci.ArtifactFiles) // .md files
	if len(ci.SpecNames) > 0 {
		count++ // specs sub-tab
	}
	return count
}

func (m model) changeArtifactTabNames() []string {
	changes := m.currentChanges()
	if m.changeCursor >= len(changes) {
		return nil
	}
	ci := changes[m.changeCursor]
	var names []string
	for _, f := range ci.ArtifactFiles {
		names = append(names, strings.TrimSuffix(f, ".md"))
	}
	if len(ci.SpecNames) > 0 {
		names = append(names, "specs")
	}
	return names
}

func (m model) currentArchivedChanges() []scanner.ChangeInfo {
	if len(m.repoPaths) == 0 {
		return nil
	}
	return m.projects[m.repoPaths[m.cursor]].Info.ArchivedChanges
}

func (m model) archiveArtifactTabCount() int {
	archived := m.currentArchivedChanges()
	if m.archiveCursor >= len(archived) {
		return 0
	}
	ci := archived[m.archiveCursor]
	count := len(ci.ArtifactFiles)
	if len(ci.SpecNames) > 0 {
		count++
	}
	return count
}

func (m *model) updateFileList() {
	m.specCursor = 0
	m.changeCursor = 0
	m.changeArtifactTab = 0
	m.archiveCursor = 0
	m.archiveArtifactTab = 0
	if len(m.repoPaths) == 0 {
		m.filePaths = nil
		m.fileCursor = 0
		return
	}
	currentProject := m.repoPaths[m.cursor]
	st, ok := m.projects[currentProject]
	if !ok || len(st.Files) == 0 {
		m.filePaths = nil
		m.fileCursor = 0
		return
	}

	m.filePaths = make([]string, 0, len(st.Files))
	for _, f := range st.Files {
		m.filePaths = append(m.filePaths, f.Path)
	}
	m.fileCursor = 0
}

func (m model) doScan() tea.Cmd {
	config := m.config
	ignoreDirErrors := m.ignoreDirErrors
	return func() tea.Msg {
		projects, err := scanner.Scan(config, ignoreDirErrors)
		return scanMsg{projects: projects, err: err}
	}
}

func (m model) doScanSingle(projectPath string) tea.Cmd {
	return func() tea.Msg {
		files, err := scanner.ListOpenSpecContents(projectPath)
		if err != nil {
			return scanMsg{err: err}
		}
		info := scanner.ParseProjectInfo(projectPath)
		projects := scanner.ProjectMap{
			projectPath: scanner.ProjectStatus{
				Files: files,
				Info:  info,
			},
		}
		return scanMsg{projects: projects}
	}
}

func waitForFsChange(w *watcher.Watcher) tea.Cmd {
	return func() tea.Msg {
		_, ok := <-w.Events()
		if !ok {
			return nil // watcher closed
		}
		return fsChangeMsg{}
	}
}

func (m *model) startWatcher(projectPath string) tea.Cmd {
	openspecDir := filepath.Join(projectPath, "openspec")
	w, err := watcher.New(openspecDir)
	if err != nil {
		log.Printf("watcher: failed to start: %v", err)
		return nil
	}
	m.watcher = w
	return waitForFsChange(w)
}

func (m *model) stopWatcher() {
	if m.watcher != nil {
		m.watcher.Close()
		m.watcher = nil
	}
}

type archiveMsg struct {
	ok     bool
	output string
}

type discardMsg struct {
	ok     bool
	output string
}

type exportMsg struct {
	ok     bool
	output string
}

func doArchiveChange(projectPath string, changeName string) tea.Cmd {
	return func() tea.Msg {
		if _, err := exec.LookPath("openspec"); err != nil {
			return archiveMsg{ok: false, output: "openspec CLI not found. Install it to enable archiving."}
		}
		cmd := exec.Command("openspec", "archive", changeName, "-y")
		cmd.Dir = projectPath
		out, err := cmd.CombinedOutput()
		if err != nil {
			return archiveMsg{ok: false, output: strings.TrimSpace(string(out))}
		}
		return archiveMsg{ok: true, output: strings.TrimSpace(string(out))}
	}
}

func doDiscardChange(projectPath string, changeName string) tea.Cmd {
	return func() tea.Msg {
		changesDir := filepath.Join(projectPath, "openspec", "changes")
		discardedDir := filepath.Join(changesDir, "discarded")
		if err := os.MkdirAll(discardedDir, 0755); err != nil {
			return discardMsg{ok: false, output: fmt.Sprintf("Failed to create discarded directory: %v", err)}
		}
		datePrefix := time.Now().Format("2006-01-02")
		target := filepath.Join(discardedDir, datePrefix+"-"+changeName)
		if _, err := os.Stat(target); err == nil {
			return discardMsg{ok: false, output: fmt.Sprintf("Target already exists: %s", datePrefix+"-"+changeName)}
		}
		src := filepath.Join(changesDir, changeName)
		if err := os.Rename(src, target); err != nil {
			return discardMsg{ok: false, output: fmt.Sprintf("Failed to move: %v", err)}
		}
		return discardMsg{ok: true, output: fmt.Sprintf("Discarded \"%s\"", changeName)}
	}
}

var archiveDatePrefix = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-`)

func exportSemanticName(changeName string, isArchived bool) string {
	if isArchived {
		return archiveDatePrefix.ReplaceAllString(changeName, "")
	}
	return changeName
}

func exportDestPath(semanticName string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dateStr := time.Now().Format("2006-01-02")
	return filepath.Join(home, semanticName+"-"+dateStr+".zip")
}

func doExportChange(projectPath string, changeName string, isArchived bool) tea.Cmd {
	return func() tea.Msg {
		var srcDir string
		if isArchived {
			srcDir = filepath.Join(projectPath, "openspec", "changes", "archive", changeName)
		} else {
			srcDir = filepath.Join(projectPath, "openspec", "changes", changeName)
		}

		if _, err := os.Stat(srcDir); err != nil {
			return exportMsg{ok: false, output: fmt.Sprintf("Source not found: %s", srcDir)}
		}

		semanticName := exportSemanticName(changeName, isArchived)
		destPath := exportDestPath(semanticName)

		zipFile, err := os.Create(destPath)
		if err != nil {
			return exportMsg{ok: false, output: fmt.Sprintf("Failed to create zip: %v", err)}
		}
		defer zipFile.Close()

		w := zip.NewWriter(zipFile)
		defer w.Close()

		err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			relPath, err := filepath.Rel(srcDir, path)
			if err != nil {
				return err
			}
			zipPath := filepath.Join(semanticName, relPath)

			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}
			header.Name = zipPath
			header.Method = zip.Deflate

			writer, err := w.CreateHeader(header)
			if err != nil {
				return err
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(writer, file)
			return err
		})

		if err != nil {
			os.Remove(destPath)
			return exportMsg{ok: false, output: fmt.Sprintf("Failed to create zip: %v", err)}
		}

		return exportMsg{ok: true, output: fmt.Sprintf("Exported to %s", destPath)}
	}
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}
	if m.width < 60 || m.height < 20 {
		return "Terminal too small. Need at least 60x20."
	}

	panelH := m.mainPanelHeight()

	var mainRow string
	if m.zoomed {
		// Full-width detail panel
		fullW := m.width - 2
		detailContent := m.renderDetailPanel(fullW, panelH)
		mainRow = m.renderPanel(viewDetail, fullW, panelH, detailContent)
	} else {
		leftW := m.leftPanelWidth() - 2
		rightW := m.rightPanelWidth() - 2

		projectContent := m.renderProjectList(leftW, panelH)
		leftPanel := m.renderPanel(viewProjects, m.leftPanelWidth()-2, panelH, projectContent)

		detailContent := m.renderDetailPanel(rightW, panelH)
		rightPanel := m.renderPanel(viewDetail, m.rightPanelWidth()-2, panelH, detailContent)

		mainRow = lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	}

	// Nav bar
	navBar := m.renderNavBar()

	var view string
	if m.logVisible {
		logPanel := m.renderPanel(viewLog, m.width-2, m.logPanelHeight(), m.logViewport.View())
		view = lipgloss.JoinVertical(lipgloss.Left, mainRow, logPanel, navBar)
	} else {
		view = lipgloss.JoinVertical(lipgloss.Left, mainRow, navBar)
	}

	// Modal overlays
	if m.scanning {
		modal := modalStyle.Width(40).Render(m.spinner.View() + " Scanning for OpenSpec sources...")
		view = placeOverlay(m.width, m.height, modal, view)
	}
	if m.err != nil {
		errText := fmt.Sprintf("Error: %v", m.err)
		modal := modalStyle.Width(m.width * 3 / 4).Render(errText)
		view = placeOverlay(m.width, m.height, modal, view)
	}

	// Archive modals
	switch m.archiveState {
	case archiveConfirming:
		var content string
		changes := m.currentChanges()
		if m.changeCursor < len(changes) {
			ci := changes[m.changeCursor]
			if ci.TasksTotal > 0 && ci.TasksDone < ci.TasksTotal {
				incomplete := ci.TasksTotal - ci.TasksDone
				content = fmt.Sprintf("⚠ %d incomplete task(s) in \"%s\"\n\nArchive anyway? (y/n)", incomplete, m.archiveChangeName)
			} else {
				content = fmt.Sprintf("Archive \"%s\"? (y/n)", m.archiveChangeName)
			}
		}
		modal := modalStyle.Width(50).Render(content)
		view = placeOverlay(m.width, m.height, modal, view)
	case archiveRunning:
		modal := modalStyle.Width(40).Render(m.spinner.View() + " Archiving...")
		view = placeOverlay(m.width, m.height, modal, view)
	case archiveResult:
		var prefix string
		if m.archiveResultOk {
			prefix = "✓ "
		} else {
			prefix = "✗ "
		}
		content := prefix + m.archiveResultMsg + "\n\nPress any key to dismiss."
		modal := modalStyle.Width(m.width * 3 / 4).Render(content)
		view = placeOverlay(m.width, m.height, modal, view)
	}

	// Discard modals
	switch m.discardState {
	case discardConfirming:
		var content string
		changes := m.currentChanges()
		if m.changeCursor < len(changes) {
			ci := changes[m.changeCursor]
			if ci.TasksTotal > 0 && ci.TasksDone < ci.TasksTotal {
				incomplete := ci.TasksTotal - ci.TasksDone
				content = fmt.Sprintf("⚠ %d incomplete task(s) in \"%s\"\n\nDiscard anyway? (y/n)", incomplete, m.discardChangeName)
			} else {
				content = fmt.Sprintf("Discard \"%s\"? (y/n)", m.discardChangeName)
			}
		}
		modal := modalStyle.Width(50).Render(content)
		view = placeOverlay(m.width, m.height, modal, view)
	case discardRunning:
		modal := modalStyle.Width(40).Render(m.spinner.View() + " Discarding...")
		view = placeOverlay(m.width, m.height, modal, view)
	case discardResult:
		var prefix string
		if m.discardResultOk {
			prefix = "✓ "
		} else {
			prefix = "✗ "
		}
		content := prefix + m.discardResultMsg + "\n\nPress any key to dismiss."
		modal := modalStyle.Width(m.width * 3 / 4).Render(content)
		view = placeOverlay(m.width, m.height, modal, view)
	}

	// Export modals
	switch m.exportState {
	case exportConfirming:
		semanticName := exportSemanticName(m.exportChangeName, m.exportIsArchived)
		destPath := exportDestPath(semanticName)
		content := fmt.Sprintf("Export \"%s\"?\n\n→ %s\n\n(y/n)", semanticName, destPath)
		modal := modalStyle.Width(60).Render(content)
		view = placeOverlay(m.width, m.height, modal, view)
	case exportRunning:
		modal := modalStyle.Width(40).Render(m.spinner.View() + " Exporting...")
		view = placeOverlay(m.width, m.height, modal, view)
	case exportResult:
		var prefix string
		if m.exportResultOk {
			prefix = "✓ "
		} else {
			prefix = "✗ "
		}
		content := prefix + m.exportResultMsg + "\n\nPress any key to dismiss."
		modal := modalStyle.Width(m.width * 3 / 4).Render(content)
		view = placeOverlay(m.width, m.height, modal, view)
	}

	return padToHeight(view, m.height)
}

func (m model) renderProjectList(width int, height int) string {
	if len(m.repoPaths) == 0 {
		return "No OpenSpec projects found."
	}

	var b strings.Builder
	offset := 0
	if m.cursor >= height {
		offset = m.cursor - height + 1
	}

	end := offset + height
	if end > len(m.repoPaths) {
		end = len(m.repoPaths)
	}

	for i := offset; i < end; i++ {
		if i > offset {
			b.WriteString("\n")
		}
		name := m.displayNames[i]
		if i == m.cursor {
			b.WriteString(selectedStyle.Width(width).Render(name))
		} else {
			b.WriteString(normalStyle.Render(name))
		}
	}
	return b.String()
}

func (m model) renderTabHeader(width int) string {
	var b strings.Builder
	for i, name := range tabNames {
		if i > 0 {
			b.WriteString(" ")
		}
		if i == m.detailTab {
			b.WriteString(activeTabStyle.Render(name))
		} else {
			b.WriteString(inactiveTabStyle.Render(name))
		}
	}
	return b.String()
}

func (m model) renderDetailPanel(width int, height int) string {
	if len(m.repoPaths) == 0 {
		return "No project selected."
	}

	currentProject := m.repoPaths[m.cursor]
	info := m.projects[currentProject].Info

	var b strings.Builder

	// Persistent header: project path + stats (1 char padding all sides)
	statsLine := fmt.Sprintf("Specs: %d  Changes: %d active  Archived: %d",
		info.SpecCount, len(info.ActiveChanges), len(info.ArchivedChanges))
	if info.TasksTotal > 0 {
		statsLine += fmt.Sprintf("  Tasks: %d/%d", info.TasksDone, info.TasksTotal)
	}
	headerContent := headerStyle.Render(currentProject) + "\n" + dimStyle.Render(statsLine)
	b.WriteString(lipgloss.NewStyle().Padding(1, 1).Render(headerContent))
	b.WriteString("\n")

	// Tab header
	b.WriteString(m.renderTabHeader(width))
	b.WriteString("\n")

	// Tab content (height minus header 4 lines (2 content + 2 padding) and tab header 1 line)
	contentHeight := height - 5
	if contentHeight < 1 {
		contentHeight = 1
	}

	switch m.detailTab {
	case tabSpecs:
		b.WriteString(m.renderSpecsTab(width, contentHeight))
	case tabChanges:
		b.WriteString(m.renderChangesTab(width, contentHeight))
	case tabArchive:
		b.WriteString(m.renderArchiveTab(width, contentHeight))
	case tabConfig:
		b.WriteString(m.renderConfigTab(width, contentHeight))
	default:
		b.WriteString(m.renderNotImplemented(width, contentHeight))
	}

	return b.String()
}

// renderChangeList is the shared renderer for both changes and archive tabs.
func renderChangeList(changes []scanner.ChangeInfo, cursor int, artifactTab int, isActive bool, showDate bool, width int, height int) string {
	if len(changes) == 0 {
		if showDate {
			return dimStyle.Render("No archived changes")
		}
		return dimStyle.Render("No active changes")
	}

	listWidth := width * 3 / 10
	if listWidth < 20 {
		listWidth = 20
	}
	contentWidth := width - listWidth - 1

	// Render change list
	var listB strings.Builder
	offset := 0
	if cursor >= height {
		offset = cursor - height + 1
	}
	end := offset + height
	if end > len(changes) {
		end = len(changes)
	}

	for i := offset; i < end; i++ {
		if i > offset {
			listB.WriteString("\n")
		}
		ci := changes[i]
		label := ci.Name
		if showDate && !ci.ArchiveDate.IsZero() {
			label = ci.ArchiveDate.Format("2006-01-02") + "  " + label
		} else if ci.TasksTotal > 0 {
			label += fmt.Sprintf(" (%d/%d)", ci.TasksDone, ci.TasksTotal)
		}
		if isActive && i == cursor {
			listB.WriteString(selectedStyle.Width(listWidth).Render(label))
		} else {
			listB.WriteString(normalStyle.Render(label))
		}
	}

	// Render right side: sub-tab header + content
	var rightB strings.Builder
	ci := changes[cursor]

	// Build sub-tab names
	var subTabNames []string
	for _, f := range ci.ArtifactFiles {
		subTabNames = append(subTabNames, strings.TrimSuffix(f, ".md"))
	}
	if len(ci.SpecNames) > 0 {
		subTabNames = append(subTabNames, "specs")
	}

	// Sub-tab header
	for i, name := range subTabNames {
		if i > 0 {
			rightB.WriteString(" ")
		}
		if i == artifactTab {
			rightB.WriteString(activeTabStyle.Render(name))
		} else {
			rightB.WriteString(inactiveTabStyle.Render(name))
		}
	}
	rightB.WriteString("\n")

	if artifactTab < len(ci.ArtifactFiles) {
		filename := ci.ArtifactFiles[artifactTab]
		content := ci.ArtifactContents[filename]

		if filename == "tasks.md" && ci.TasksTotal > 0 {
			statsLine := fmt.Sprintf("Tasks: %d/%d complete\n\n", ci.TasksDone, ci.TasksTotal)
			rightB.WriteString(sectionHeaderStyle.Render(statsLine))
		}

		rightB.WriteString(renderMarkdown(content, contentWidth))
	} else if len(ci.SpecNames) > 0 {
		for i, name := range ci.SpecNames {
			if i > 0 {
				rightB.WriteString("\n")
			}
			rightB.WriteString(sectionHeaderStyle.Render(name))
			rightB.WriteString("\n")
			if content, ok := ci.SpecContents[name]; ok {
				rightB.WriteString(renderMarkdown(content, contentWidth))
			} else {
				rightB.WriteString(dimStyle.Render("  No spec.md found"))
			}
			rightB.WriteString("\n")
		}
	}

	changeList := truncateContent(listB.String(), height)
	changeContent := truncateContent(rightB.String(), height)

	leftBox := lipgloss.NewStyle().Width(listWidth).Height(height).MaxHeight(height).Render(changeList)
	rightBox := lipgloss.NewStyle().Width(contentWidth).Height(height).MaxHeight(height).Render(changeContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, " ", rightBox)
}

func (m model) renderChangesTab(width int, height int) string {
	isActive := m.activeView == viewDetail && m.detailTab == tabChanges
	return renderChangeList(m.currentChanges(), m.changeCursor, m.changeArtifactTab, isActive, false, width, height)
}

func (m model) renderArchiveTab(width int, height int) string {
	isActive := m.activeView == viewDetail && m.detailTab == tabArchive
	return renderChangeList(m.currentArchivedChanges(), m.archiveCursor, m.archiveArtifactTab, isActive, true, width, height)
}

func (m model) renderSpecsTab(width int, height int) string {
	if len(m.repoPaths) == 0 {
		return "No project selected."
	}

	info := m.projects[m.repoPaths[m.cursor]].Info

	if len(info.SpecNames) == 0 {
		return dimStyle.Render("No specs found")
	}

	// Split: ~30% for spec list, ~70% for content
	listWidth := width * 3 / 10
	if listWidth < 15 {
		listWidth = 15
	}
	contentWidth := width - listWidth - 1 // 1 for gap

	// Render spec list
	var listB strings.Builder
	offset := 0
	if m.specCursor >= height {
		offset = m.specCursor - height + 1
	}
	end := offset + height
	if end > len(info.SpecNames) {
		end = len(info.SpecNames)
	}

	isActive := m.activeView == viewDetail && m.detailTab == tabSpecs
	for i := offset; i < end; i++ {
		if i > offset {
			listB.WriteString("\n")
		}
		name := info.SpecNames[i]
		if isActive && i == m.specCursor {
			listB.WriteString(selectedStyle.Width(listWidth).Render(name))
		} else {
			listB.WriteString(normalStyle.Render(name))
		}
	}

	// Render spec content
	var contentB strings.Builder
	selectedSpec := info.SpecNames[m.specCursor]
	specContent, ok := info.SpecContents[selectedSpec]
	if !ok || specContent == "" {
		contentB.WriteString(dimStyle.Render("No spec.md found"))
	} else {
		contentB.WriteString(renderMarkdown(specContent, contentWidth))
	}

	specList := truncateContent(listB.String(), height)
	specContentStr := truncateContent(contentB.String(), height)

	// Force fixed dimensions on both sides to prevent wrapping/flickering
	leftBox := lipgloss.NewStyle().Width(listWidth).Height(height).MaxHeight(height).Render(specList)
	rightBox := lipgloss.NewStyle().Width(contentWidth).Height(height).MaxHeight(height).Render(specContentStr)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, " ", rightBox)
}

func (m model) renderConfigTab(width int, height int) string {
	if len(m.repoPaths) == 0 {
		return "No project selected."
	}

	currentProject := m.repoPaths[m.cursor]
	info := m.projects[currentProject].Info

	if info.ConfigFile == "" {
		return dimStyle.Render("No project configuration found")
	}

	var b strings.Builder

	// File source indicator
	b.WriteString(dimStyle.Render("openspec/" + info.ConfigFile))
	b.WriteString("\n\n")

	// Render based on file type
	if strings.HasSuffix(info.ConfigFile, ".md") {
		b.WriteString(renderMarkdown(info.ConfigContent, width))
	} else {
		b.WriteString(renderYAML(info.ConfigContent, width))
	}

	return b.String()
}

var (
	mdHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("6")) // cyan

	mdBoldStyle = lipgloss.NewStyle().Bold(true)

	mdItalicStyle = lipgloss.NewStyle().Italic(true)

	yamlKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("6")) // cyan

	yamlCommentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("250")) // light gray

	yamlValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")) // green
)

func renderMarkdown(content string, width int) string {
	lines := strings.Split(content, "\n")
	var b strings.Builder

	for i, line := range lines {
		if i > 0 {
			b.WriteString("\n")
		}

		trimmed := strings.TrimSpace(line)

		// Headers
		if strings.HasPrefix(trimmed, "#") {
			b.WriteString(mdHeaderStyle.Render(trimmed))
			continue
		}

		// List items
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			b.WriteString("  " + renderInlineMarkdown(trimmed))
			continue
		}

		// Regular text with inline formatting
		b.WriteString(renderInlineMarkdown(line))
	}

	return b.String()
}

func renderInlineMarkdown(line string) string {
	result := line

	// Bold: **text**
	for {
		start := strings.Index(result, "**")
		if start == -1 {
			break
		}
		end := strings.Index(result[start+2:], "**")
		if end == -1 {
			break
		}
		end += start + 2
		bold := result[start+2 : end]
		result = result[:start] + mdBoldStyle.Render(bold) + result[end+2:]
	}

	// Italic: _text_
	for {
		start := strings.Index(result, "_")
		if start == -1 {
			break
		}
		end := strings.Index(result[start+1:], "_")
		if end == -1 {
			break
		}
		end += start + 1
		italic := result[start+1 : end]
		result = result[:start] + mdItalicStyle.Render(italic) + result[end+1:]
	}

	return result
}

func renderYAML(content string, width int) string {
	lines := strings.Split(content, "\n")
	var b strings.Builder

	for i, line := range lines {
		if i > 0 {
			b.WriteString("\n")
		}

		trimmed := strings.TrimSpace(line)

		// Comment lines
		if strings.HasPrefix(trimmed, "#") {
			b.WriteString(yamlCommentStyle.Render(line))
			continue
		}

		// Key: value lines
		colonIdx := strings.Index(line, ":")
		if colonIdx > 0 {
			// Check for inline comment
			key := line[:colonIdx]
			rest := line[colonIdx:]

			commentIdx := strings.Index(rest, " #")
			if commentIdx > 0 {
				value := rest[:commentIdx]
				comment := rest[commentIdx:]
				b.WriteString(yamlKeyStyle.Render(key))
				b.WriteString(yamlValueStyle.Render(value))
				b.WriteString(yamlCommentStyle.Render(comment))
			} else {
				b.WriteString(yamlKeyStyle.Render(key))
				b.WriteString(yamlValueStyle.Render(rest))
			}
			continue
		}

		// Plain lines (list items, etc)
		b.WriteString(line)
	}

	return b.String()
}

func (m model) renderNotImplemented(width int, height int) string {
	return "\n\n" + dimStyle.Render("  Not yet implemented")
}

// truncateContent ensures content is exactly maxLines tall — truncates or pads.
func truncateContent(content string, maxLines int) string {
	lines := strings.Split(content, "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	for len(lines) < maxLines {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// padToHeight ensures the final rendered view is exactly the given height.
func padToHeight(view string, height int) string {
	lines := strings.Split(view, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (m model) renderPanel(view int, width int, height int, content string) string {
	content = truncateContent(content, height)
	var title string
	switch view {
	case viewProjects:
		title = " Projects "
	case viewDetail:
		if m.zoomed && len(m.displayNames) > 0 && m.cursor < len(m.displayNames) {
			title = " Detail (" + m.displayNames[m.cursor] + ") "
		} else {
			title = " Detail "
		}
	case viewLog:
		title = " Log "
	}

	borderColor := lipgloss.Color("240")
	if m.activeView == view {
		borderColor = lipgloss.Color("2")
	}

	border := lipgloss.RoundedBorder()
	titleStyled := lipgloss.NewStyle().Foreground(borderColor).Bold(true).Render(title)
	topBorder := border.TopLeft +
		strings.Repeat(border.Top, 1) +
		titleStyled +
		strings.Repeat(border.Top, max(0, width-lipgloss.Width(title)-2)) +
		border.TopRight

	boxStyle := lipgloss.NewStyle().
		Border(border).
		BorderTop(false).
		BorderForeground(borderColor).
		Width(width).
		Height(height).
		MaxHeight(height + 2) // +2 for border lines

	rendered := topBorder + "\n" + boxStyle.Render(content)
	// Ensure the panel is exactly height+2 lines (title + border top + content area + border bottom)
	return padToHeight(rendered, height+2)
}

func (m model) renderNavBar() string {
	var keys []struct{ key, action string }
	if m.zoomed {
		keys = []struct{ key, action string }{
			{"q", "quit"},
			{"esc", "back"},
			{"s", "scan"},
			{"jk/\u2191\u2193", "navigate"},
			{"\u2190\u2192/1-4", "tabs"},
			{"l", "log"},
			{"gg/G", "jump"},
		}
	} else {
		keys = []struct{ key, action string }{
			{"q", "quit"},
			{"enter", "zoom"},
			{"s", "scan"},
			{"tab", "switch"},
			{"jk/\u2191\u2193", "navigate"},
			{"\u2190\u2192/1-4", "tabs"},
			{"l", "log"},
			{"gg/G", "jump"},
		}
	}
	if m.detailTab == tabChanges && len(m.currentChanges()) > 0 {
		keys = append(keys, struct{ key, action string }{"a", "archive"})
		keys = append(keys, struct{ key, action string }{"d", "discard"})
		keys = append(keys, struct{ key, action string }{"e", "export"})
	}
	if m.detailTab == tabArchive && len(m.currentArchivedChanges()) > 0 {
		keys = append(keys, struct{ key, action string }{"e", "export"})
	}

	var left strings.Builder
	for i, k := range keys {
		if i > 0 {
			left.WriteString(navBarStyle.Render("  "))
		}
		left.WriteString(navBarKeyStyle.Render(k.key))
		left.WriteString(navBarStyle.Render(" " + k.action))
	}

	right := navBarStyle.Render("specgetty " + m.version)

	bar := lipgloss.PlaceHorizontal(
		m.width,
		lipgloss.Left,
		left.String()+strings.Repeat(" ", max(0, m.width-lipgloss.Width(left.String())-lipgloss.Width(right)))+right,
		lipgloss.WithWhitespaceBackground(lipgloss.Color("236")),
	)

	return bar
}

func placeOverlay(width, height int, modal, background string) string {
	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceBackground(lipgloss.NoColor{}),
	)
}

func Run(config *scanner.Config, ignoreDirErrors bool, version string, initialZoomPath string) error {
	m := newModel(config, ignoreDirErrors, version)
	if initialZoomPath != "" {
		m.initialZoomPath = initialZoomPath
		m.zoomed = true
		m.activeView = viewDetail
	}
	p := tea.NewProgram(m, tea.WithAltScreen())

	m.program = p
	log.SetOutput(logWriter{program: p})

	_, err := p.Run()
	return err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
