package devr

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func clearWarningAfter() tea.Cmd {
	return tea.Tick(2*time.Second, func(_ time.Time) tea.Msg {
		return clearWarningMsg{}
	})
}

func (m logViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		return m, nil

	case clearWarningMsg:
		m.ctrlCOnce = false
		return m, nil

	case processExitMsg:
		m.markCrashed()
		return m, nil

	case lineMsg:
		m.appendLine(parseLine(string(msg)))
		return m, nil

	case tea.KeyMsg:
		if quit, cmd := m.handleCtrlC(msg.String()); quit || cmd != nil {
			return m, cmd
		}

		m.ctrlCOnce = false

		if m.mode == modeHelp {
			m.mode = modeNormal
			return m, nil
		}

		if m.mode == modeSearch {
			return m.updateSearch(msg)
		}

		return m.updateNormal(msg)
	}

	return m, nil
}

func (m logViewModel) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.exit = exitDetach
		return m, tea.Quit
	case "up", "k":
		m.follow = false
		if m.cursor > 0 {
			m.cursor--
			m.ensureVisible()
		}
	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			m.ensureVisible()
		}

		if m.cursor == len(m.filtered)-1 {
			m.follow = true
		}
	case "ctrl+d":
		m.moveBy(m.logHeight() / 2)
	case "ctrl+u":
		m.moveBy(-m.logHeight() / 2)
	case "pgdown", "ctrl+f":
		m.moveBy(m.logHeight())
	case "pgup", "ctrl+b":
		m.moveBy(-m.logHeight())
	case "H":
		m.follow = false
		m.cursor = m.offset
	case "M":
		m.follow = false

		mid := m.offset + m.logHeight()/2
		if mid >= len(m.filtered) {
			mid = max(0, len(m.filtered)-1)
		}

		m.cursor = mid
	case "L":
		m.follow = false

		bot := m.offset + m.logHeight() - 1
		if bot >= len(m.filtered) {
			bot = max(0, len(m.filtered)-1)
		}

		m.cursor = bot
	case "G":
		m.follow = true
		m.cursor = max(0, len(m.filtered)-1)
		m.ensureVisible()
	case "g":
		m.follow = false
		m.cursor = 0
		m.offset = 0
	case "enter":
		m.appendNewEntry(logEntry{raw: "", level: levelUnknown})
	case "y":
		if m.cursor < len(m.filtered) {
			idx := m.filtered[m.cursor]
			copyToClipboard(formatJSON(m.lines[idx].raw))
		}
	case "tab":
		m.preview = !m.preview
	case "?":
		m.mode = modeHelp
	case "/":
		m.mode = modeSearch
		m.searchBuf = m.filter
	case "1":
		m.setFilter("error")
	case "2":
		m.setFilter("warn")
	case "3":
		m.setFilter("info")
	case "4":
		m.setFilter("debug")
	case "0":
		m.setFilter("")
	case "w":
		m.wrap = !m.wrap
	case "alt+enter":
		m.appendNewEntry(logEntry{
			raw:      "── MARKER ──",
			level:    levelUnknown,
			isMarker: true,
		})
	}

	return m, nil
}

func (m logViewModel) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.mode = modeNormal
		m.setFilter(m.searchBuf)
	case "esc":
		m.mode = modeNormal
		m.searchBuf = m.filter
	case "backspace":
		if len(m.searchBuf) > 0 {
			runes := []rune(m.searchBuf)
			m.searchBuf = string(runes[:len(runes)-1])
		}

		m.setFilter(m.searchBuf)
	case "ctrl+u":
		m.searchBuf = ""
		m.setFilter("")
	default:
		if msg.Type == tea.KeyRunes {
			m.searchBuf += string(msg.Runes)
			m.setFilter(m.searchBuf)
		}
	}

	return m, nil
}

func (m *logViewModel) handleCtrlC(key string) (bool, tea.Cmd) {
	if key != "ctrl+c" {
		return false, nil
	}

	if m.ctrlCOnce {
		m.exit = exitStop
		return true, tea.Quit
	}

	// Require confirmation before stopping the managed process from the viewer.
	m.ctrlCOnce = true

	return false, clearWarningAfter()
}

func (m *logViewModel) moveBy(delta int) {
	m.follow = false

	m.cursor += delta

	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
		m.follow = true
	}

	if m.cursor < 0 {
		m.cursor = 0
	}

	m.ensureVisible()
}

func (m *logViewModel) appendLine(entry logEntry) {
	m.lines = append(m.lines, entry)
	if !m.matchFilter(entry) {
		return
	}

	m.filtered = append(m.filtered, len(m.lines)-1)
	if m.follow {
		m.cursor = len(m.filtered) - 1
		m.ensureVisible()
	}
}

func (m *logViewModel) appendNewEntry(entry logEntry) {
	m.follow = true
	m.appendLine(entry)
}

func (m *logViewModel) markCrashed() {
	m.title = "CRASHED"
	m.follow = false
}

func (m *logViewModel) setFilter(f string) {
	m.filter = f
	m.filterLower = strings.ToLower(f)
	m.refilter()

	if m.mode != modeSearch {
		m.follow = true
	}

	if len(m.filtered) > 0 {
		m.cursor = len(m.filtered) - 1
	} else {
		m.cursor = 0
	}

	m.ensureVisible()
}

func (m *logViewModel) refilter() {
	m.filtered = m.filtered[:0]
	for i, e := range m.lines {
		if m.matchFilter(e) {
			m.filtered = append(m.filtered, i)
		}
	}
}

func (m logViewModel) matchFilter(e logEntry) bool {
	if m.filter == "" {
		return true
	}

	return strings.Contains(e.lower, m.filterLower)
}
