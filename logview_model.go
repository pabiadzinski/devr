package devr

import (
	"encoding/json"
	"fmt"
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
	modeHighlightSearch
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
	search          string
	searchLower     string
	highlightFields []string
	parser          logParser
}

var (
	styleError    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	styleWarn     = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	styleInfo     = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	styleDebug    = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	styleTime     = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	styleDim      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleMatch    = lipgloss.NewStyle().Background(lipgloss.Color("11")).Foreground(lipgloss.Color("0"))
	styleCtrlC    = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	styleHelpKey  = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	styleSearch   = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	styleFollow   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	styleSearchHL = lipgloss.NewStyle().Background(lipgloss.Color("5")).Foreground(lipgloss.Color("15"))
	styleMsg      = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
)

func newModel() logViewModel {
	return logViewModel{
		follow: true,
		wrap:   true,
		done:   make(chan struct{}),
		parser: newLogParser(DefaultConfig().Logs),
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

type logParser struct {
	format     string
	levelField string
	levelMap   map[string]level
}

func parseLine(line string) logEntry {
	return newLogParser(DefaultConfig().Logs).Parse(line)
}

func newLogParser(cfg ConfigLogs) logParser {
	defaults := DefaultConfig().Logs
	if cfg.Format == "" {
		cfg.Format = defaults.Format
	}

	if cfg.LevelField == "" {
		cfg.LevelField = defaults.LevelField
	}

	if len(cfg.LevelValues.Error) == 0 {
		cfg.LevelValues.Error = defaults.LevelValues.Error
	}

	if len(cfg.LevelValues.Warn) == 0 {
		cfg.LevelValues.Warn = defaults.LevelValues.Warn
	}

	if len(cfg.LevelValues.Info) == 0 {
		cfg.LevelValues.Info = defaults.LevelValues.Info
	}

	if len(cfg.LevelValues.Debug) == 0 {
		cfg.LevelValues.Debug = defaults.LevelValues.Debug
	}

	parser := logParser{
		format:     cfg.Format,
		levelField: cfg.LevelField,
		levelMap:   make(map[string]level),
	}

	addLevelAliases(parser.levelMap, levelError, cfg.LevelValues.Error)
	addLevelAliases(parser.levelMap, levelWarn, cfg.LevelValues.Warn)
	addLevelAliases(parser.levelMap, levelInfo, cfg.LevelValues.Info)
	addLevelAliases(parser.levelMap, levelDebug, cfg.LevelValues.Debug)

	return parser
}

func addLevelAliases(dst map[string]level, target level, aliases []string) {
	for _, alias := range aliases {
		dst[strings.ToLower(alias)] = target
	}
}

func (p logParser) Parse(line string) logEntry {
	entry := logEntry{raw: line, lower: strings.ToLower(line), level: levelUnknown}

	switch p.format {
	case "json":
		return p.parseJSON(entry)
	case "text":
		return p.parseText(entry)
	default:
		if parsed, ok := p.parseJSONIfPossible(entry); ok {
			return parsed
		}

		return p.parseText(entry)
	}
}

func (p logParser) parseJSONIfPossible(entry logEntry) (logEntry, bool) {
	parsed, err := p.parseJSONObject(entry.raw)
	if err != nil {
		return entry, false
	}

	entry.level = p.lookupLevel(parsed[p.levelField])

	return entry, true
}

func (p logParser) parseJSON(entry logEntry) logEntry {
	parsed, err := p.parseJSONObject(entry.raw)
	if err != nil {
		return p.parseText(entry)
	}

	entry.level = p.lookupLevel(parsed[p.levelField])

	return entry
}

func (p logParser) parseText(entry logEntry) logEntry {
	if value, ok := extractKeyValue(entry.raw, p.levelField); ok {
		entry.level = p.lookupLevel(value)
		if entry.level != levelUnknown {
			return entry
		}
	}

	for _, field := range strings.Fields(entry.lower) {
		token := strings.Trim(field, `"'[](){}:;,`)
		if level := p.lookupLevel(token); level != levelUnknown {
			entry.level = level
			return entry
		}
	}

	return entry
}

func (p logParser) parseJSONObject(line string) (map[string]any, error) {
	var obj map[string]any
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		return nil, err
	}

	return obj, nil
}

func (p logParser) lookupLevel(v any) level {
	if v == nil {
		return levelUnknown
	}

	key := strings.ToLower(strings.TrimSpace(fmt.Sprint(v)))
	if level, ok := p.levelMap[key]; ok {
		return level
	}

	return levelUnknown
}

func extractKeyValue(line, key string) (string, bool) {
	prefix := key + "="
	for _, part := range strings.Fields(line) {
		if !strings.HasPrefix(part, prefix) {
			continue
		}

		value := strings.TrimPrefix(part, prefix)

		return strings.Trim(value, `"'`), true
	}

	return "", false
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

func (e logEntry) render(selected bool, filter, filterLower, search, searchLower string, width int, highlightFields []string) string {
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

	if filter != "" {
		line = highlightAll(line, filter, filterLower, styleMatch)
	}

	if search != "" {
		line = highlightAll(line, search, searchLower, styleSearchHL)
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

func highlightAll(line, search, searchLower string, style lipgloss.Style) string {
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
		b.WriteString(style.Render(line[pos+idx : pos+idx+len(search)]))
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
