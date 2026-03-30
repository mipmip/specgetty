package ui

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mipmip/specgetty/src/scanner"
)

const (
	viewRepo   = 0
	viewStatus = 1
	viewLog    = 2
)

// Message types

type scanMsg struct {
	projects scanner.ProjectMap
	err      error
}

type logMsg string

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
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("2")). // green
			Foreground(lipgloss.Color("0")). // black
			Width(0)                         // set dynamically

	normalStyle = lipgloss.NewStyle()

	activeBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("2")) // green

	inactiveBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")) // gray

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
)

type model struct {
	config          *scanner.Config
	ignoreDirErrors bool
	projects        scanner.ProjectMap
	repoPaths       []string
	cursor          int
	activeView      int
	scanning        bool
	err             error
	spinner         spinner.Model
	statusViewport  viewport.Model
	logViewport     viewport.Model
	logContent      string
	fileCursor      int
	filePaths       []string
	logVisible      bool
	logShownOnce    bool
	pendingKey      string
	width           int
	height          int
	program         *tea.Program
	inTmux          bool
	version         string
}

func newModel(config *scanner.Config, ignoreDirErrors bool, version string) model {
	s := spinner.New()
	s.Spinner = spinner.Dot

	return model{
		config:          config,
		ignoreDirErrors: ignoreDirErrors,
		scanning:        true,
		inTmux:          os.Getenv("TMUX") != "",
		version:         version,
		spinner:         s,
		statusViewport:  viewport.New(0, 0),
		logViewport:     viewport.New(0, 0),
	}
}

func (m model) Init() tea.Cmd {
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

		// Handle pending 'g' chord
		key := msg.String()
		if m.pendingKey == "g" {
			m.pendingKey = ""
			if key == "g" {
				switch m.activeView {
				case viewRepo:
					m.cursor = 0
					m.updateFileList()
				case viewStatus:
					m.fileCursor = 0
				case viewLog:
					m.logViewport.GotoTop()
				}
				return m, nil
			}
		}

		switch key {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "s":
			m.scanning = true
			cmds = append(cmds, m.doScan())
		case "enter":
			cmd := m.doEdit()
			if cmd != nil {
				return m, cmd
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
					m.activeView = viewRepo
				}
				m.recalcLayout()
			}
		case "tab":
			if m.logVisible {
				m.activeView = (m.activeView + 1) % 3
			} else {
				if m.activeView == viewRepo {
					m.activeView = viewStatus
				} else {
					m.activeView = viewRepo
				}
			}
		case "g":
			m.pendingKey = "g"
		case "G":
			switch m.activeView {
			case viewRepo:
				if len(m.repoPaths) > 0 {
					m.cursor = len(m.repoPaths) - 1
					m.updateFileList()
				}
			case viewStatus:
				if len(m.filePaths) > 0 {
					m.fileCursor = len(m.filePaths) - 1
				}
			case viewLog:
				m.logViewport.GotoBottom()
			}
		case "pgdown", "ctrl+f":
			half := m.halfPage()
			switch m.activeView {
			case viewRepo:
				m.cursor = min(m.cursor+half, len(m.repoPaths)-1)
				m.updateFileList()
			case viewStatus:
				if len(m.filePaths) > 0 {
					m.fileCursor = min(m.fileCursor+half, len(m.filePaths)-1)
				}
			case viewLog:
				m.logViewport.LineDown(half)
			}
		case "pgup", "ctrl+b":
			half := m.halfPage()
			switch m.activeView {
			case viewRepo:
				m.cursor = max(m.cursor-half, 0)
				m.updateFileList()
			case viewStatus:
				if len(m.filePaths) > 0 {
					m.fileCursor = max(m.fileCursor-half, 0)
				}
			case viewLog:
				m.logViewport.LineUp(half)
			}
		case "up", "k":
			if m.activeView == viewRepo {
				if m.cursor > 0 {
					m.cursor--
					m.updateFileList()
				}
			} else if m.activeView == viewStatus {
				if len(m.filePaths) > 0 && m.fileCursor > 0 {
					m.fileCursor--
				}
			} else if m.activeView == viewLog {
				m.logViewport.LineUp(1)
			}
		case "down", "j":
			if m.activeView == viewRepo {
				if m.cursor < len(m.repoPaths)-1 {
					m.cursor++
					m.updateFileList()
				}
			} else if m.activeView == viewStatus {
				if len(m.filePaths) > 0 && m.fileCursor < len(m.filePaths)-1 {
					m.fileCursor++
				}
			} else if m.activeView == viewLog {
				m.logViewport.LineDown(1)
			}
		}

	case scanMsg:
		m.scanning = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.projects = msg.projects
			m.repoPaths = make([]string, 0, len(m.projects))
			for r := range m.projects {
				m.repoPaths = append(m.repoPaths, r)
			}
			sort.Strings(m.repoPaths)
			if m.cursor >= len(m.repoPaths) {
				m.cursor = max(0, len(m.repoPaths)-1)
			}
			m.recalcLayout()
			m.updateFileList()
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

func (m *model) recalcLayout() {
	if m.width == 0 || m.height == 0 {
		return
	}
	innerWidth := m.width - 2

	statusHeight := m.statusPanelHeight()
	if statusHeight > 0 {
		m.statusViewport.Width = innerWidth
		m.statusViewport.Height = statusHeight
	}

	logHeight := m.logPanelHeight()
	if logHeight > 0 {
		m.logViewport.Width = innerWidth
		m.logViewport.Height = logHeight
	}
}

func (m model) repoPanelHeight() int {
	n := len(m.repoPaths)
	if n == 0 {
		n = 1
	}
	maxH := (m.height - 6) / 2
	if n > maxH {
		return maxH
	}
	return n
}

func (m model) statusPanelHeight() int {
	repoH := m.repoPanelHeight() + 2
	logH := m.logPanelHeight()
	if logH > 0 {
		logH += 2
	}
	remaining := m.height - repoH - logH - 1
	if remaining < 3 {
		return 3
	}
	return remaining - 2
}

func (m model) logPanelHeight() int {
	if !m.logVisible {
		return 0
	}
	return min(10, (m.height-6)/3)
}

func (m model) halfPage() int {
	switch m.activeView {
	case viewStatus:
		return max(1, m.statusViewport.Height/2)
	case viewLog:
		return max(1, m.logViewport.Height/2)
	default:
		return max(1, m.repoPanelHeight()/2)
	}
}

// updateFileList rebuilds the file list for the currently selected project.
func (m *model) updateFileList() {
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

func (m model) doEdit() tea.Cmd {
	if len(m.repoPaths) == 0 || m.cursor >= len(m.repoPaths) {
		return nil
	}
	currentProject := m.repoPaths[m.cursor]
	if currentProject == "" {
		return nil
	}

	cmdStr := strings.Replace(m.config.EditCommand, "%WORKING_DIRECTORY", currentProject, -1)
	if cmdStr == "" {
		return nil
	}

	if m.inTmux {
		return func() tea.Msg {
			_ = exec.Command("tmux", "display-popup", "-E", "-w", "80%", "-h", "80%", "-d", currentProject).Run()
			return nil
		}
	}

	args := strings.Fields(cmdStr)
	if len(args) == 0 {
		return nil
	}
	c := exec.Command(args[0], args[1:]...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return nil
	})
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}
	if m.height < 20 {
		return "Terminal too small. Need at least 20 lines."
	}

	innerWidth := m.width - 2

	// Project panel
	repoContent := m.renderRepoList(innerWidth)
	repoPanel := m.renderPanel(viewRepo, innerWidth, m.repoPanelHeight(), repoContent)

	// Contents panel
	statusContent := m.renderStatusContent(innerWidth, m.statusPanelHeight())
	statusPanel := m.renderPanel(viewStatus, innerWidth, m.statusPanelHeight(), statusContent)

	// Nav bar
	navBar := m.renderNavBar()

	var view string
	if m.logVisible {
		logPanel := m.renderPanel(viewLog, innerWidth, m.logPanelHeight(), m.logViewport.View())
		view = lipgloss.JoinVertical(lipgloss.Left, repoPanel, statusPanel, logPanel, navBar)
	} else {
		view = lipgloss.JoinVertical(lipgloss.Left, repoPanel, statusPanel, navBar)
	}

	// Modal overlays
	if m.scanning {
		modal := modalStyle.Width(20).Render(m.spinner.View() + " Scanning...")
		view = placeOverlay(m.width, m.height, modal, view)
	}
	if m.err != nil {
		errText := fmt.Sprintf("Error: %v", m.err)
		modal := modalStyle.Width(m.width * 3 / 4).Render(errText)
		view = placeOverlay(m.width, m.height, modal, view)
	}

	return view
}

func (m model) renderRepoList(width int) string {
	if len(m.repoPaths) == 0 {
		return "No OpenSpec projects found."
	}

	var b strings.Builder
	h := m.repoPanelHeight()
	offset := 0
	if m.cursor >= h {
		offset = m.cursor - h + 1
	}

	end := offset + h
	if end > len(m.repoPaths) {
		end = len(m.repoPaths)
	}

	for i := offset; i < end; i++ {
		if i > offset {
			b.WriteString("\n")
		}
		line := m.repoPaths[i]
		if i == m.cursor {
			styled := selectedStyle.Width(width).Render(line)
			b.WriteString(styled)
		} else {
			b.WriteString(normalStyle.Render(line))
		}
	}
	return b.String()
}

func (m model) renderFileList(width int, height int) string {
	if len(m.filePaths) == 0 {
		return "No files."
	}

	currentProject := m.repoPaths[m.cursor]
	st := m.projects[currentProject]

	// Build a lookup from path to FileEntry
	entryMap := make(map[string]scanner.FileEntry, len(st.Files))
	for _, f := range st.Files {
		entryMap[f.Path] = f
	}

	var b strings.Builder
	offset := 0
	if m.fileCursor >= height {
		offset = m.fileCursor - height + 1
	}
	end := offset + height
	if end > len(m.filePaths) {
		end = len(m.filePaths)
	}

	isActive := m.activeView == viewStatus
	for i := offset; i < end; i++ {
		if i > offset {
			b.WriteString("\n")
		}
		path := m.filePaths[i]
		entry := entryMap[path]
		indicator := "f"
		if entry.IsDir {
			indicator = "d"
		}
		line := fmt.Sprintf(" %s  %s", indicator, path)
		if isActive && i == m.fileCursor {
			b.WriteString(selectedStyle.Width(width).Render(line))
		} else {
			b.WriteString(normalStyle.Render(line))
		}
	}
	return b.String()
}

func (m model) renderStatusContent(innerWidth int, height int) string {
	return m.renderFileList(innerWidth, height)
}

func (m model) renderPanel(view int, width int, height int, content string) string {
	var title string
	switch view {
	case viewRepo:
		title = " Projects "
	case viewStatus:
		title = " Contents "
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
		Height(height)

	return topBorder + "\n" + boxStyle.Render(content)
}

func (m model) renderNavBar() string {
	keys := []struct{ key, action string }{
		{"q", "quit"},
		{"s", "scan"},
		{"enter", "open"},
		{"tab", "switch"},
		{"jk/\u2191\u2193", "navigate"},
		{"l", "log"},
		{"pgup/dn", "scroll"},
		{"gg/G", "jump"},
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

// placeOverlay renders a modal centered over the background.
func placeOverlay(width, height int, modal, background string) string {
	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceBackground(lipgloss.NoColor{}),
	)
}

func Run(config *scanner.Config, ignoreDirErrors bool, version string) error {
	m := newModel(config, ignoreDirErrors, version)
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
