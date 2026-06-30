package devr

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type lineMsg string
type titleMsg string
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
	levelFilter     level
	hasLevelFilter  bool
	preview         bool
	previewOffset   int
	previewLines    []string
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

func (l level) String() string {
	switch l {
	case levelError:
		return "error"
	case levelWarn:
		return "warn"
	case levelInfo:
		return "info"
	case levelDebug:
		return "debug"
	default:
		return "unknown"
	}
}

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

var levelAliases = map[level][]string{
	levelError: {"ERROR", "error"},
	levelWarn:  {"WARN", "WARNING", "warn", "warning"},
	levelInfo:  {"INFO", "info"},
	levelDebug: {"DEBUG", "debug"},
}

// levelToken finds the keyword to colorize for an entry's own level, so a "warn"
// line with "error" in its message colors "warn", not the unrelated "error".
func levelToken(line string, l level) (string, int) {
	for _, kw := range levelAliases[l] {
		if idx := strings.Index(line, kw); idx >= 0 {
			return kw, idx
		}
	}

	return "", -1
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

	if kw, idx := levelToken(line, e.level); idx >= 0 {
		a := levelAffix[kw]
		line = line[:idx] + a.pre + kw + a.suf + line[idx+len(kw):]
	}

	for _, field := range highlightFields {
		line = highlightJSONField(line, field)
	}

	if filter != "" {
		line = highlightAll(line, filter, filterLower, ansiMatchPre, ansiMatchSuf)
	}

	if search != "" {
		line = highlightAll(line, search, searchLower, ansiSearchPre, ansiSearchSuf)
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

	return line[:start] + ansiMsgPre + line[start:end] + ansiMsgSuf + line[end:]
}

// ansiSeqLen returns the byte length of the CSI escape sequence at the start of
// s, or 0 if s does not begin with one.
func ansiSeqLen(s string) int {
	if len(s) < 2 || s[0] != 0x1b || s[1] != '[' {
		return 0
	}

	for i := 2; i < len(s); i++ {
		if s[i] >= 0x40 && s[i] <= 0x7e {
			return i + 1
		}
	}

	return 0
}

func highlightAll(line, search, searchLower, pre, suf string) string {
	if search == "" {
		return line
	}

	var b strings.Builder

	// Skip over ANSI escape sequences already in the line (level/field coloring)
	// so a search term that occurs inside one isn't matched and spliced apart.
	i := 0
	for i < len(line) {
		if n := ansiSeqLen(line[i:]); n > 0 {
			b.WriteString(line[i : i+n])
			i += n

			continue
		}

		j := i
		for j < len(line) && (line[j] != 0x1b || ansiSeqLen(line[j:]) == 0) {
			j++
		}

		highlightSegment(&b, line[i:j], search, searchLower, pre, suf)
		i = j
	}

	return b.String()
}

func highlightSegment(b *strings.Builder, seg, search, searchLower, pre, suf string) {
	lower := strings.ToLower(seg)

	pos := 0
	for {
		idx := strings.Index(lower[pos:], searchLower)
		if idx < 0 {
			b.WriteString(seg[pos:])
			break
		}

		b.WriteString(seg[pos : pos+idx])
		b.WriteString(pre)
		b.WriteString(seg[pos+idx : pos+idx+len(search)])
		b.WriteString(suf)

		pos += idx + len(search)
	}
}

var (
	styleJSONKey    = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	styleJSONString = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	styleJSONNumber = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	styleJSONBool   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	styleJSONNull   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// ANSI prefix/suffix pairs precomputed from the styles above, so per-line
// rendering concatenates strings instead of invoking lipgloss. They capture the
// color profile detected at init; the viewer never changes it at runtime.
var (
	ansiDimPre, ansiDimSuf       = ansiPair(styleDim)
	ansiKeyPre, ansiKeySuf       = ansiPair(styleJSONKey)
	ansiStringPre, ansiStringSuf = ansiPair(styleJSONString)
	ansiNumberPre, ansiNumberSuf = ansiPair(styleJSONNumber)
	ansiBoolPre, ansiBoolSuf     = ansiPair(styleJSONBool)
	ansiNullPre, ansiNullSuf     = ansiPair(styleJSONNull)
	ansiMatchPre, ansiMatchSuf   = ansiPair(styleMatch)
	ansiSearchPre, ansiSearchSuf = ansiPair(styleSearchHL)
	ansiMsgPre, ansiMsgSuf       = ansiPair(styleMsg)
	ansiErrorPre, ansiErrorSuf   = ansiPair(styleError)
	ansiWarnPre, ansiWarnSuf     = ansiPair(styleWarn)
	ansiInfoPre, ansiInfoSuf     = ansiPair(styleInfo)
	ansiDebugPre, ansiDebugSuf   = ansiPair(styleDebug)
)

type ansiAffix struct{ pre, suf string }

var levelAffix = map[string]ansiAffix{
	"ERROR":   {ansiErrorPre, ansiErrorSuf},
	"error":   {ansiErrorPre, ansiErrorSuf},
	"WARN":    {ansiWarnPre, ansiWarnSuf},
	"WARNING": {ansiWarnPre, ansiWarnSuf},
	"warn":    {ansiWarnPre, ansiWarnSuf},
	"warning": {ansiWarnPre, ansiWarnSuf},
	"INFO":    {ansiInfoPre, ansiInfoSuf},
	"info":    {ansiInfoPre, ansiInfoSuf},
	"DEBUG":   {ansiDebugPre, ansiDebugSuf},
	"debug":   {ansiDebugPre, ansiDebugSuf},
}

func ansiPair(s lipgloss.Style) (string, string) {
	const sentinel = "\x00"

	rendered := s.Render(sentinel)
	idx := strings.Index(rendered, sentinel)

	if idx < 0 {
		return "", ""
	}

	return rendered[:idx], rendered[idx+len(sentinel):]
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

func colorizeJSON(raw string) []string {
	formatted := formatJSON(raw)
	if formatted == raw {
		return []string{raw}
	}

	lines := strings.Split(formatted, "\n")
	out := make([]string, len(lines))

	for i, line := range lines {
		out[i] = colorizeJSONLine(line)
	}

	return out
}

func colorizeJSONLine(line string) string {
	trimmed := strings.TrimSpace(line)
	indent := line[:len(line)-len(trimmed)]

	if trimmed == "{" || trimmed == "}" || trimmed == "}," ||
		trimmed == "[" || trimmed == "]" || trimmed == "]," {
		return indent + ansiDimPre + trimmed + ansiDimSuf
	}

	colonIdx := strings.Index(trimmed, ": ")
	if colonIdx < 0 || !strings.HasPrefix(trimmed, `"`) {
		return indent + colorizeJSONValue(trimmed)
	}

	key := trimmed[:colonIdx]
	val := trimmed[colonIdx+2:]

	return indent + ansiKeyPre + key + ansiKeySuf + ansiDimPre + ": " + ansiDimSuf + colorizeJSONValue(val)
}

func colorizeJSONValue(val string) string {
	clean := strings.TrimSuffix(val, ",")
	comma := ""

	if len(clean) < len(val) {
		comma = ansiDimPre + "," + ansiDimSuf
	}

	switch {
	case strings.HasPrefix(clean, `"`):
		return ansiStringPre + clean + ansiStringSuf + comma
	case clean == "true" || clean == "false":
		return ansiBoolPre + clean + ansiBoolSuf + comma
	case clean == "null":
		return ansiNullPre + clean + ansiNullSuf + comma
	case len(clean) > 0 && (clean[0] >= '0' && clean[0] <= '9' || clean[0] == '-'):
		return ansiNumberPre + clean + ansiNumberSuf + comma
	default:
		return val
	}
}
