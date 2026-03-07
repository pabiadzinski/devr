package devr

import (
	"encoding/json"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type lineMsg string
type clearWarningMsg struct{}
type processExitMsg struct{}

type logViewMode int

const (
	modeNormal logViewMode = iota
	modeSearch
	modeHelp
)

type exitAction int

const (
	exitDetach exitAction = iota
	exitStop
)

type logViewModel struct {
	lines           []logEntry
	filtered        []int
	cursor          int
	offset          int
	width           int
	height          int
	filter          string
	filterLower     string
	preview         bool
	follow          bool
	mode            logViewMode
	searchBuf       string
	exit            exitAction
	ctrlCOnce       bool
	done            chan struct{}
	title           string
	wrap            bool
	highlightFields []string
}

var (
	styleError   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	styleWarn    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	styleInfo    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	styleDebug   = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	styleTime    = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	styleDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleMatch   = lipgloss.NewStyle().Background(lipgloss.Color("11")).Foreground(lipgloss.Color("0"))
	styleCtrlC   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	styleHelpKey = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	styleSearch  = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	styleFollow  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	styleMsg     = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
)

func newModel() logViewModel {
	return logViewModel{
		follow: true,
		done:   make(chan struct{}),
	}
}

func (m logViewModel) Init() tea.Cmd {
	return nil
}

type logEntry struct {
	raw      string
	lower    string
	level    level
	isMarker bool
}

type level int

const (
	levelDebug level = iota
	levelInfo
	levelWarn
	levelError
	levelUnknown
)

func parseLine(line string) logEntry {
	e := logEntry{raw: line, lower: strings.ToLower(line), level: levelUnknown}

	switch {
	case strings.Contains(e.lower, `"level":"error"`) || strings.Contains(e.lower, " error "):
		e.level = levelError
	case strings.Contains(e.lower, `"level":"warn"`) || strings.Contains(e.lower, `"level":"warning"`) || strings.Contains(e.lower, " warn "):
		e.level = levelWarn
	case strings.Contains(e.lower, `"level":"info"`) || strings.Contains(e.lower, " info "):
		e.level = levelInfo
	case strings.Contains(e.lower, `"level":"debug"`) || strings.Contains(e.lower, " debug "):
		e.level = levelDebug
	}

	return e
}

var levelKeywords = []string{
	"ERROR", "error",
	"WARN", "WARNING", "warn", "warning",
	"INFO", "info",
	"DEBUG", "debug",
}

var levelStyles = map[string]lipgloss.Style{
	"ERROR": styleError, "error": styleError,
	"WARN": styleWarn, "WARNING": styleWarn, "warn": styleWarn, "warning": styleWarn,
	"INFO": styleInfo, "info": styleInfo,
	"DEBUG": styleDebug, "debug": styleDebug,
}

func (e logEntry) render(selected bool, search, searchLower string, width int, highlightFields []string) string {
	if e.isMarker {
		markerWidth := width - 5 // account for padding and cursor
		if markerWidth < 10 {
			markerWidth = 10
		}

		marker := strings.Repeat("━", markerWidth)

		if selected {
			return styleMatch.Render(" >> " + marker)
		}

		return styleWarn.Render(" " + marker)
	}

	line := e.raw

	if selected {
		line = "  >> " + line
	} else {
		line = " " + line
	}

	for _, kw := range levelKeywords {
		if idx := strings.Index(line, kw); idx >= 0 {
			line = line[:idx] + levelStyles[kw].Render(kw) + line[idx+len(kw):]

			break
		}
	}

	for _, field := range highlightFields {
		line = highlightJSONField(line, field)
	}

	if search != "" {
		line = highlightAll(line, search, searchLower)
	}

	return line
}

func highlightJSONField(line, field string) string {
	pattern := `"` + field + `":"`

	start := strings.Index(line, pattern)
	if start < 0 {
		return line
	}

	start += len(pattern)
	end := start

	for end < len(line) {
		if line[end] == '"' && (end == start || line[end-1] != '\\') {
			break
		}

		end++
	}

	if end >= len(line) {
		return line
	}

	content := line[start:end]
	highlighted := line[:start] + styleMsg.Render(content) + line[end:]

	return highlighted
}

func highlightAll(line, search, searchLower string) string {
	lower := strings.ToLower(line)

	var b strings.Builder

	pos := 0

	for {
		idx := strings.Index(lower[pos:], searchLower)
		if idx < 0 {
			b.WriteString(line[pos:])
			break
		}

		b.WriteString(line[pos : pos+idx])
		b.WriteString(styleMatch.Render(line[pos+idx : pos+idx+len(search)]))
		pos += idx + len(search)
	}

	return b.String()
}

func formatJSON(raw string) string {
	var obj any
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return raw
	}

	pretty, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return raw
	}

	return string(pretty)
}
