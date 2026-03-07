package devr

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func (m logViewModel) View() string {
	if m.width == 0 {
		return "loading..."
	}

	if m.mode == modeHelp {
		return m.renderHelp()
	}

	var b strings.Builder

	viewH := m.logHeight()

	b.WriteString(m.headerText())
	b.WriteString("\n")

	usedLines := 0

	for i := m.offset; i < len(m.filtered) && usedLines < viewH; i++ {
		idx := m.filtered[i]
		line := m.lines[idx].render(i == m.cursor, m.filter, m.filterLower, m.width, m.highlightFields)

		if m.wrap && m.width > 0 {
			line = ansi.Hardwrap(line, m.width, true)
		}

		for _, wl := range strings.Split(line, "\n") {
			if usedLines >= viewH {
				break
			}

			b.WriteString(wl)
			b.WriteString("\n")

			usedLines++
		}
	}

	for i := usedLines; i < viewH; i++ {
		b.WriteString("\n")
	}

	if m.preview && m.cursor < len(m.filtered) {
		idx := m.filtered[m.cursor]
		previewH := m.height - viewH - 2
		preview := formatJSON(m.lines[idx].raw)
		lines := strings.Split(preview, "\n")

		b.WriteString(styleDim.Render(strings.Repeat("─", m.width)))
		b.WriteString("\n")

		for i := 0; i < previewH && i < len(lines); i++ {
			b.WriteString(styleTime.Render(lines[i]))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(m.footerText())

	return b.String()
}

func (m logViewModel) headerText() string {
	return styleDim.Render(fmt.Sprintf(" %d lines | filter: %s | ?=help",
		len(m.filtered), filterLabel(m.filter)))
}

func (m logViewModel) footerText() string {
	switch {
	case m.ctrlCOnce:
		return styleCtrlC.Render(" Press Ctrl+C again to stop the process")
	case m.mode == modeSearch:
		cursor := styleSearch.Render("▏")
		return styleSearch.Render(" /") + m.searchBuf + cursor
	default:
		return m.statusFooter()
	}
}

func (m logViewModel) statusFooter() string {
	followIndicator := ""
	if m.follow {
		followIndicator = styleFollow.Render(" ●FOLLOW")
	}

	titlePart := ""
	if m.title != "" {
		titlePart = styleHelpKey.Render(" "+m.title) + " "
	}

	wrapIndicator := ""
	if m.wrap {
		wrapIndicator = styleFollow.Render(" WRAP")
	}

	return titlePart + styleDim.Render(fmt.Sprintf("%d/%d", m.cursor+1, len(m.filtered))) + followIndicator + wrapIndicator + " "
}

func (m logViewModel) renderHelp() string {
	keys := []struct{ key, desc string }{
		{"j/k, ↑/↓", "Move up/down"},
		{"Ctrl+D", "Half page down"},
		{"Ctrl+U", "Half page up"},
		{"Ctrl+F, PgDn", "Page down"},
		{"Ctrl+B, PgUp", "Page up"},
		{"g", "Go to top"},
		{"G", "Go to bottom (follow)"},
		{"H", "Top of screen"},
		{"M", "Middle of screen"},
		{"L", "Bottom of screen"},
		{"/ + text", "Search / filter"},
		{"Esc", "Cancel search"},
		{"Ctrl+U (search)", "Clear search"},
		{"1-4", "Filter by level (error/warn/info/debug)"},
		{"0", "Clear filter"},
		{"w", "Toggle line wrap"},
		{"Alt+Enter", "Insert marker line"},
		{"Tab", "Toggle JSON preview"},
		{"Enter", "Insert blank line"},
		{"y", "Copy line to clipboard"},
		{"q, Esc", "Detach (process keeps running)"},
		{"Ctrl+C ×2", "Stop process and exit"},
	}

	var b strings.Builder

	title := styleHelpKey.UnsetWidth().Render(" Keyboard Shortcuts")
	b.WriteString("\n" + title + "\n")
	b.WriteString(styleDim.Render(" "+strings.Repeat("─", 38)) + "\n\n")

	for _, k := range keys {
		fmt.Fprintf(&b, "  %s  %s\n",
			styleHelpKey.Width(16).Render(k.key),
			k.desc)
	}

	b.WriteString("\n" + styleDim.Render(" Press any key to close"))

	return b.String()
}

func filterLabel(f string) string {
	if f == "" {
		return "all"
	}

	return f
}

func (m logViewModel) logHeight() int {
	h := m.height - 3
	if m.preview {
		h /= 2
	}

	if h < 1 {
		return 1
	}

	return h
}

func (m *logViewModel) countVisualLines(filteredIdx int) int {
	if filteredIdx >= len(m.filtered) {
		return 0
	}

	idx := m.filtered[filteredIdx]
	line := m.lines[idx].render(false, m.filter, m.filterLower, m.width, m.highlightFields)

	if m.wrap && m.width > 0 {
		wrapped := ansi.Hardwrap(line, m.width, true)
		return len(strings.Split(wrapped, "\n"))
	}

	return 1
}

func (m *logViewModel) ensureVisible() {
	viewH := m.logHeight()

	if m.cursor < m.offset {
		m.offset = m.cursor
		return
	}

	neededLines := 0
	for i := m.offset; i <= m.cursor && i < len(m.filtered); i++ {
		neededLines += m.countVisualLines(i)
	}

	for neededLines > viewH && m.offset < m.cursor {
		neededLines -= m.countVisualLines(m.offset)
		m.offset++
	}
}
