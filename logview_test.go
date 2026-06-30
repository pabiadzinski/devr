package devr

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogViewLineMsgFollowsWhenEnabled(t *testing.T) {
	m := newModel()
	m.height = 10
	m.width = 80

	updated, _ := m.Update(lineMsg(`{"level":"info","msg":"ready"}`))
	got := updated.(logViewModel)

	require.Len(t, got.lines, 1)
	assert.Equal(t, []int{0}, got.filtered)
	assert.Equal(t, 0, got.cursor)
	assert.True(t, got.follow)
}

func TestLogViewLineMsgDoesNotMoveCursorWhenFollowDisabled(t *testing.T) {
	m := newModel()
	m.height = 10
	m.width = 80
	m.follow = false
	m.lines = []logEntry{parseLine(`{"level":"info","msg":"first"}`)}
	m.filtered = []int{0}
	m.cursor = 0

	updated, _ := m.Update(lineMsg(`{"level":"info","msg":"second"}`))
	got := updated.(logViewModel)

	require.Len(t, got.lines, 2)
	assert.Equal(t, []int{0, 1}, got.filtered)
	assert.Equal(t, 0, got.cursor)
	assert.False(t, got.follow)
}

func TestLogViewSetFilterUpdatesFilteredAndCursor(t *testing.T) {
	m := newModel()
	m.height = 10
	m.width = 80
	m.lines = []logEntry{
		parseLine(`{"level":"info","msg":"ready"}`),
		parseLine(`{"level":"error","msg":"boom"}`),
		parseLine(`{"level":"error","msg":"again"}`),
	}
	m.refilter()

	m.setFilter("error")

	assert.Equal(t, "error", m.filter)
	assert.Equal(t, "error", m.filterLower)
	assert.Equal(t, []int{1, 2}, m.filtered)
	assert.Equal(t, 1, m.cursor)
	assert.True(t, m.follow)
}

func TestLogViewSearchModeUpdatesBufferAndFilter(t *testing.T) {
	m := newModel()
	m.height = 10
	m.width = 80
	m.mode = modeSearch
	m.lines = []logEntry{
		parseLine(`{"level":"info","msg":"ready"}`),
		parseLine(`{"level":"error","msg":"boom"}`),
	}
	m.refilter()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	got := updated.(logViewModel)
	assert.Equal(t, "e", got.searchBuf)
	assert.Equal(t, "e", got.filter)

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	got = updated.(logViewModel)
	assert.Equal(t, "er", got.searchBuf)
	assert.Equal(t, []int{1}, got.filtered)

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	got = updated.(logViewModel)
	assert.Equal(t, "e", got.searchBuf)
	assert.Equal(t, "e", got.filter)
}

func TestLogViewCtrlCRequiresConfirmation(t *testing.T) {
	m := newModel()

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	got := updated.(logViewModel)

	require.NotNil(t, cmd)
	assert.True(t, got.ctrlCOnce)
	assert.Equal(t, exitDetach, got.exit)

	updated, cmd = got.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	got = updated.(logViewModel)

	require.NotNil(t, cmd)
	assert.Equal(t, exitStop, got.exit)
}

func TestLogViewProcessExitStopsFollowAndMarksCrash(t *testing.T) {
	m := newModel()
	m.follow = true
	m.title = "WATCH"

	updated, _ := m.Update(processExitMsg{})
	got := updated.(logViewModel)

	assert.Equal(t, "CRASHED", got.title)
	assert.False(t, got.follow)
}

func TestLogViewFooterTextReflectsModes(t *testing.T) {
	m := newModel()
	m.filtered = []int{0, 1}
	m.cursor = 1
	m.follow = true
	m.title = "RUN"

	assert.Contains(t, m.footerText(), "RUN")

	m.mode = modeSearch
	m.searchBuf = "err"
	assert.Contains(t, m.footerText(), "/err")

	m.mode = modeNormal
	m.ctrlCOnce = true
	assert.Contains(t, m.footerText(), "Press Ctrl+C again")
}

func TestLogParserJSONCustomLevelField(t *testing.T) {
	parser := newLogParser(ConfigLogs{
		Format:     "json",
		LevelField: "severity",
		LevelValues: ConfigLogLevels{
			Error: []string{"critical"},
		},
	})

	entry := parser.Parse(`{"severity":"critical","message":"boom"}`)
	assert.Equal(t, levelError, entry.level)
}

func TestLogParserTextCustomLevelField(t *testing.T) {
	parser := newLogParser(ConfigLogs{
		Format:     "text",
		LevelField: "lvl",
		LevelValues: ConfigLogLevels{
			Warn: []string{"warning"},
		},
	})

	entry := parser.Parse(`ts=2026-03-17T10:15:00Z lvl=warning msg="slow db"`)
	assert.Equal(t, levelWarn, entry.level)
}

func TestLogParserAutoFallsBackToText(t *testing.T) {
	parser := newLogParser(DefaultConfig().Logs)

	entry := parser.Parse(`INFO server started`)
	assert.Equal(t, levelInfo, entry.level)
}

func TestLogParserAutoParsesJSONWithCustomAliases(t *testing.T) {
	parser := newLogParser(ConfigLogs{
		Format:     "auto",
		LevelField: "lvl",
		LevelValues: ConfigLogLevels{
			Debug: []string{"trace"},
		},
	})

	entry := parser.Parse(`{"lvl":"trace","msg":"details"}`)
	assert.Equal(t, levelDebug, entry.level)
}

func TestExtractKeyValue(t *testing.T) {
	tests := []struct {
		line, key string
		wantVal   string
		wantOK    bool
	}{
		{"level=error msg=boom", "level", "error", true},
		{"level=\"warn\" msg=hi", "level", "warn", true},
		{"msg=hello", "level", "", false},
		{"lvl=info extra=1", "lvl", "info", true},
	}
	for _, tt := range tests {
		val, ok := extractKeyValue(tt.line, tt.key)
		assert.Equal(t, tt.wantOK, ok, tt.line)

		if ok {
			assert.Equal(t, tt.wantVal, val, tt.line)
		}
	}
}

func TestFormatJSON(t *testing.T) {
	tests := []struct {
		input string
		check func(t *testing.T, out string)
	}{
		{`{"a":1}`, func(t *testing.T, out string) {
			assert.Contains(t, out, "\"a\": 1")
		}},
		{`not json`, func(t *testing.T, out string) {
			assert.Equal(t, "not json", out)
		}},
		{`{"b":[1,2]}`, func(t *testing.T, out string) {
			assert.Contains(t, out, "[\n")
		}},
	}
	for _, tt := range tests {
		tt.check(t, formatJSON(tt.input))
	}
}

func TestHighlightAll(t *testing.T) {
	t.Run("no match returns original", func(t *testing.T) {
		out := highlightAll("no match here", "xyz", "xyz", ansiMatchPre, ansiMatchSuf)
		assert.Equal(t, "no match here", out)
	})

	t.Run("empty input", func(t *testing.T) {
		out := highlightAll("", "test", "test", ansiMatchPre, ansiMatchSuf)
		assert.Equal(t, "", out)
	})

	t.Run("match preserves surrounding text", func(t *testing.T) {
		out := highlightAll("abc hello def", "hello", "hello", ansiMatchPre, ansiMatchSuf)
		assert.True(t, strings.HasPrefix(out, "abc "))
		assert.True(t, strings.HasSuffix(out, " def"))
	})

	t.Run("multiple matches", func(t *testing.T) {
		out := highlightAll("aa bb aa", "aa", "aa", ansiMatchPre, ansiMatchSuf)
		// original "aa" appears twice; output should be longer due to styling
		assert.GreaterOrEqual(t, len(out), len("aa bb aa"))
	})
}

func TestHighlightAllSkipsANSI(t *testing.T) {
	const (
		esc   = "\x1b[91m"
		reset = "\x1b[0m"
	)

	line := esc + "ERROR" + reset + " code 9"

	tests := []struct {
		name   string
		search string
		want   string
	}{
		{"digit inside escape not matched", "9", esc + "ERROR" + reset + " code <9>"},
		{"letter inside escape not matched", "m", esc + "ERROR" + reset + " code 9"},
		{"bracket inside escape not matched", "[", esc + "ERROR" + reset + " code 9"},
		{"visible text still matched", "rror", esc + "E<RROR>" + reset + " code 9"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := highlightAll(line, tt.search, strings.ToLower(tt.search), "<", ">")
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestJumpToMatch(t *testing.T) {
	m := newModel()
	m.height = 20
	m.width = 80
	m.lines = []logEntry{
		parseLine("first line"),
		parseLine("match here"),
		parseLine("another line"),
		parseLine("match again"),
	}
	m.refilter()
	m.search = "match"
	m.searchLower = "match"
	m.cursor = 0

	m.jumpToMatch(1)
	assert.Equal(t, 1, m.cursor)

	m.jumpToMatch(1)
	assert.Equal(t, 3, m.cursor)

	m.jumpToMatch(-1)
	assert.Equal(t, 1, m.cursor)
}

func TestJumpToMatchNoSearch(t *testing.T) {
	m := newModel()
	m.height = 10
	m.width = 80
	m.lines = []logEntry{parseLine("line")}
	m.refilter()
	m.cursor = 0

	m.jumpToMatch(1)
	assert.Equal(t, 0, m.cursor)
}

func TestJumpToMatchNotFound(t *testing.T) {
	m := newModel()
	m.height = 10
	m.width = 80
	m.lines = []logEntry{parseLine("aaa"), parseLine("bbb")}
	m.refilter()
	m.search = "zzz"
	m.searchLower = "zzz"
	m.cursor = 0

	m.jumpToMatch(1)
	assert.Equal(t, 0, m.cursor)
}

func TestMoveBy(t *testing.T) {
	m := newModel()
	m.height = 10
	m.width = 80

	for i := 0; i < 20; i++ {
		m.lines = append(m.lines, parseLine("line"))
	}

	m.refilter()

	m.cursor = 0
	m.follow = true

	m.moveBy(5)
	assert.Equal(t, 5, m.cursor)
	assert.False(t, m.follow)

	m.moveBy(-100)
	assert.Equal(t, 0, m.cursor)

	m.moveBy(100)
	assert.Equal(t, 19, m.cursor)
	assert.True(t, m.follow)
}

func TestUpdateNormalKeys(t *testing.T) {
	m := newModel()
	m.height = 20
	m.width = 80

	for i := 0; i < 10; i++ {
		m.appendLine(parseLine(`{"level":"info","msg":"line"}`))
	}

	m.cursor = 5
	m.follow = false

	tests := []struct {
		key   string
		check func(t *testing.T, got logViewModel)
	}{
		{"g", func(t *testing.T, got logViewModel) {
			assert.Equal(t, 0, got.cursor)
			assert.False(t, got.follow)
		}},
		{"G", func(t *testing.T, got logViewModel) {
			assert.Equal(t, 9, got.cursor)
			assert.True(t, got.follow)
		}},
		{"w", func(t *testing.T, got logViewModel) {
			assert.False(t, got.wrap)
		}},
		{"?", func(t *testing.T, got logViewModel) {
			assert.Equal(t, modeHelp, got.mode)
		}},
		{"/", func(t *testing.T, got logViewModel) {
			assert.Equal(t, modeSearch, got.mode)
		}},
		{"s", func(t *testing.T, got logViewModel) {
			assert.Equal(t, modeHighlightSearch, got.mode)
		}},
		{"1", func(t *testing.T, got logViewModel) {
			assert.True(t, got.hasLevelFilter)
			assert.Equal(t, levelError, got.levelFilter)
			assert.Equal(t, "", got.filter)
		}},
		{"0", func(t *testing.T, got logViewModel) {
			assert.False(t, got.hasLevelFilter)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
			got := updated.(logViewModel)
			tt.check(t, got)
		})
	}
}

func TestUpdateSearchInputHighlightMode(t *testing.T) {
	m := newModel()
	m.height = 10
	m.width = 80
	m.mode = modeHighlightSearch
	m.lines = []logEntry{
		parseLine("hello world"),
		parseLine("goodbye world"),
	}
	m.refilter()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	got := updated.(logViewModel)
	assert.Equal(t, "h", got.searchBuf)
	assert.Equal(t, "h", got.search)
	assert.Equal(t, "", got.filter)
	assert.Len(t, got.filtered, 2)

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got = updated.(logViewModel)
	assert.Equal(t, modeNormal, got.mode)
	assert.Equal(t, "h", got.search)
}

func TestUpdateSearchInputSpace(t *testing.T) {
	m := newModel()
	m.height = 10
	m.width = 80
	m.mode = modeSearch
	m.lines = []logEntry{parseLine("hello world")}
	m.refilter()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")})
	got := updated.(logViewModel)
	assert.Equal(t, " ", got.searchBuf)
}

func TestFilterLabel(t *testing.T) {
	assert.Equal(t, "all", filterLabel(""))
	assert.Equal(t, "error", filterLabel("error"))
}

func TestLogHeight(t *testing.T) {
	tests := []struct {
		name    string
		height  int
		preview bool
		want    int
	}{
		{"normal", 20, false, 17},
		{"with preview", 20, true, 8},
		{"tiny", 2, false, 1},
		{"zero", 0, false, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := logViewModel{height: tt.height, preview: tt.preview}
			assert.Equal(t, tt.want, m.logHeight())
		})
	}
}

func TestColorizeJSON(t *testing.T) {
	lines := colorizeJSON(`{"key":"value"}`)
	assert.Greater(t, len(lines), 1)

	lines = colorizeJSON("not json")
	assert.Equal(t, []string{"not json"}, lines)
}

func TestColorizeJSONValue(t *testing.T) {
	inputs := []string{`"hello"`, `"hello",`, "true", "false", "null", "42", "-1", "42,", "{"}
	for _, in := range inputs {
		t.Run(in, func(t *testing.T) {
			assert.Equal(t, in, ansi.Strip(colorizeJSONValue(in)),
				"colorization must not alter visible text")
		})
	}
}

func TestColorizeJSONLine(t *testing.T) {
	inputs := []string{
		`  "key": "value"`,
		`  {`,
		`  }`,
		`  },`,
		`  [`,
		`  ]`,
		`  ],`,
		`  "key": 42`,
	}
	for _, in := range inputs {
		t.Run(strings.TrimSpace(in), func(t *testing.T) {
			assert.Equal(t, in, ansi.Strip(colorizeJSONLine(in)),
				"colorization must not alter visible text")
		})
	}
}

func TestHighlightJSONField(t *testing.T) {
	line := `{"msg":"hello world","level":"info"}`

	out := highlightJSONField(line, "msg")
	assert.Contains(t, out, "hello world")

	same := highlightJSONField(line, "nonexistent")
	assert.Equal(t, line, same)
}

func TestLogEntryRenderMarker(t *testing.T) {
	entry := logEntry{raw: "── MARKER ──", isMarker: true}

	selected := entry.render(true, "", "", "", "", 80, nil)
	assert.Contains(t, selected, "━")
	assert.Contains(t, selected, ">>")

	normal := entry.render(false, "", "", "", "", 80, nil)
	assert.Contains(t, normal, "━")
	assert.NotContains(t, normal, ">>")
}

func TestLogEntryRenderWithLevelKeywords(t *testing.T) {
	entry := parseLine(`level=ERROR msg="boom"`)
	out := entry.render(false, "", "", "", "", 80, nil)
	assert.Contains(t, out, "ERROR")
}

func TestLevelToken(t *testing.T) {
	tests := []struct {
		name string
		line string
		lvl  level
		want string
	}{
		{"warn line with error in msg colors warn", `{"level":"warn","msg":"client error"}`, levelWarn, "warn"},
		{"error line colors error", `{"level":"error","msg":"internal error"}`, levelError, "error"},
		{"info line with warn in msg colors info", `{"level":"info","msg":"warn user"}`, levelInfo, "info"},
		{"unknown level colors nothing", `{"level":"warn","msg":"client error"}`, levelUnknown, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kw, _ := levelToken(tt.line, tt.lvl)
			assert.Equal(t, tt.want, kw)
		})
	}
}

func TestLogEntryRenderSelected(t *testing.T) {
	entry := logEntry{raw: "test line", lower: "test line"}
	out := entry.render(true, "", "", "", "", 80, nil)
	assert.Contains(t, out, ">>")
}

func TestCountVisualLines(t *testing.T) {
	m := newModel()
	m.width = 80
	m.wrap = false
	m.lines = []logEntry{parseLine("short line")}

	m.refilter()

	assert.Equal(t, 1, m.countVisualLines(0))
	assert.Equal(t, 0, m.countVisualLines(99))
}

func TestEnsureVisibleScrollsDown(t *testing.T) {
	m := newModel()
	m.height = 5
	m.width = 80
	m.wrap = false

	for i := 0; i < 20; i++ {
		m.lines = append(m.lines, parseLine("line"))
	}

	m.refilter()

	m.cursor = 15
	m.offset = 0
	m.ensureVisible()
	assert.Greater(t, m.offset, 0)
}

func TestEnsureVisibleScrollsUp(t *testing.T) {
	m := newModel()
	m.height = 10
	m.width = 80

	for i := 0; i < 20; i++ {
		m.lines = append(m.lines, parseLine("line"))
	}

	m.refilter()

	m.offset = 10
	m.cursor = 5
	m.ensureVisible()
	assert.Equal(t, 5, m.offset)
}

func TestWindowSizeMsg(t *testing.T) {
	m := newModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	got := updated.(logViewModel)
	assert.Equal(t, 120, got.width)
	assert.Equal(t, 40, got.height)
}

func TestTitleMsg(t *testing.T) {
	m := newModel()
	updated, _ := m.Update(titleMsg("WATCH"))
	got := updated.(logViewModel)
	assert.Equal(t, "WATCH", got.title)
}

func TestClearWarningMsg(t *testing.T) {
	m := newModel()
	m.ctrlCOnce = true
	updated, _ := m.Update(clearWarningMsg{})
	got := updated.(logViewModel)
	assert.False(t, got.ctrlCOnce)
}

func TestUpdateSearchEsc(t *testing.T) {
	m := newModel()
	m.height = 10
	m.width = 80
	m.mode = modeSearch
	m.filter = "old"
	m.searchBuf = "new"

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	got := updated.(logViewModel)
	assert.Equal(t, modeNormal, got.mode)
	assert.Equal(t, "old", got.searchBuf)
}

func TestUpdateSearchCtrlU(t *testing.T) {
	m := newModel()
	m.height = 10
	m.width = 80
	m.mode = modeSearch
	m.searchBuf = "test"
	m.lines = []logEntry{parseLine("line")}
	m.refilter()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	got := updated.(logViewModel)
	assert.Equal(t, "", got.searchBuf)
	assert.Equal(t, "", got.filter)
}

func TestUpdateNormalEnterAddsBlankLine(t *testing.T) {
	m := newModel()
	m.height = 20
	m.width = 80
	m.lines = []logEntry{parseLine("existing")}
	m.refilter()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(logViewModel)
	assert.Len(t, got.lines, 2)
	assert.Equal(t, "", got.lines[1].raw)
}

func TestUpdateNormalQDetaches(t *testing.T) {
	m := newModel()
	m.height = 10
	m.width = 80

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	got := updated.(logViewModel)
	assert.Equal(t, exitDetach, got.exit)
	assert.NotNil(t, cmd)
}

func TestStatusFooter(t *testing.T) {
	m := newModel()
	m.filtered = []int{0, 1, 2}
	m.cursor = 1
	m.follow = true
	m.wrap = true
	m.search = "test"
	m.title = "RUN"

	footer := m.statusFooter()
	assert.Contains(t, footer, "RUN")
	assert.Contains(t, footer, "FOLLOW")
	assert.Contains(t, footer, "WRAP")
	assert.Contains(t, footer, "SEARCH")
	assert.Contains(t, footer, "2/3")
}

func TestHighlightSearchEsc(t *testing.T) {
	m := newModel()
	m.height = 10
	m.width = 80
	m.mode = modeHighlightSearch
	m.search = "old"
	m.searchBuf = "new"

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	got := updated.(logViewModel)
	assert.Equal(t, modeNormal, got.mode)
	assert.Equal(t, "old", got.searchBuf)
}

func TestMatchFilter(t *testing.T) {
	m := newModel()

	entry := logEntry{lower: "error: something failed"}
	assert.True(t, m.matchFilter(entry))

	m.filter = "error"
	m.filterLower = "error"
	assert.True(t, m.matchFilter(entry))

	m.filter = "warn"
	m.filterLower = "warn"
	assert.False(t, m.matchFilter(entry))
}

func TestMatchLevelFilter(t *testing.T) {
	warnWithError := parseLine(`{"level":"warn","msg":"client error","status":404}`)
	errLine := parseLine(`{"level":"error","msg":"internal server error"}`)
	infoLine := parseLine(`{"level":"info","msg":"ok"}`)

	tests := []struct {
		name   string
		filter level
		entry  logEntry
		want   bool
	}{
		{"warn-with-error-substring excluded by error filter", levelError, warnWithError, false},
		{"error line matches error filter", levelError, errLine, true},
		{"warn line matches warn filter", levelWarn, warnWithError, true},
		{"info line excluded by error filter", levelError, infoLine, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newModel()
			m.hasLevelFilter = true
			m.levelFilter = tt.filter
			assert.Equal(t, tt.want, m.matchFilter(tt.entry))
		})
	}
}

func TestHeaderTextLevel(t *testing.T) {
	m := newModel()
	m.filtered = []int{0}
	m.hasLevelFilter = true
	m.levelFilter = levelError

	assert.Contains(t, m.headerText(), "level: error")
	assert.Contains(t, m.headerText(), "filter: all")
}

func TestAppendLineFiltered(t *testing.T) {
	m := newModel()
	m.height = 10
	m.width = 80
	m.filter = "error"
	m.filterLower = "error"

	m.appendLine(parseLine(`{"level":"info","msg":"ok"}`))
	assert.Len(t, m.filtered, 0)

	m.appendLine(parseLine(`{"level":"error","msg":"fail"}`))
	assert.Len(t, m.filtered, 1)
}

func makeModelWithLines(n int) logViewModel {
	m := newModel()
	m.height = 20
	m.width = 80

	for i := 0; i < n; i++ {
		m.appendLine(parseLine(`{"level":"info","msg":"line"}`))
	}

	m.follow = false

	return m
}

func keyRunes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestUpdateNormalMovement(t *testing.T) {
	tests := []struct {
		name    string
		key     tea.KeyMsg
		cursor  int
		checkFn func(t *testing.T, got logViewModel)
	}{
		{"up/k", keyRunes("k"), 5, func(t *testing.T, got logViewModel) {
			assert.Equal(t, 4, got.cursor)
			assert.False(t, got.follow)
		}},
		{"up at top", keyRunes("k"), 0, func(t *testing.T, got logViewModel) {
			assert.Equal(t, 0, got.cursor)
		}},
		{"down/j", keyRunes("j"), 5, func(t *testing.T, got logViewModel) {
			assert.Equal(t, 6, got.cursor)
		}},
		{"down at bottom enables follow", keyRunes("j"), 19, func(t *testing.T, got logViewModel) {
			assert.Equal(t, 19, got.cursor)
			assert.True(t, got.follow)
		}},
		{"up arrow", tea.KeyMsg{Type: tea.KeyUp}, 5, func(t *testing.T, got logViewModel) {
			assert.Equal(t, 4, got.cursor)
			assert.False(t, got.follow)
		}},
		{"down arrow", tea.KeyMsg{Type: tea.KeyDown}, 5, func(t *testing.T, got logViewModel) {
			assert.Equal(t, 6, got.cursor)
		}},
		{"ctrl+d", tea.KeyMsg{Type: tea.KeyCtrlD}, 0, func(t *testing.T, got logViewModel) {
			assert.Greater(t, got.cursor, 0)
		}},
		{"ctrl+u", tea.KeyMsg{Type: tea.KeyCtrlU}, 15, func(t *testing.T, got logViewModel) {
			assert.Less(t, got.cursor, 15)
		}},
		{"pgdown", tea.KeyMsg{Type: tea.KeyPgDown}, 0, func(t *testing.T, got logViewModel) {
			assert.Greater(t, got.cursor, 0)
		}},
		{"pgup", tea.KeyMsg{Type: tea.KeyPgUp}, 15, func(t *testing.T, got logViewModel) {
			assert.Less(t, got.cursor, 15)
		}},
		{"ctrl+f", tea.KeyMsg{Type: tea.KeyCtrlF}, 0, func(t *testing.T, got logViewModel) {
			assert.Greater(t, got.cursor, 0)
		}},
		{"ctrl+b", tea.KeyMsg{Type: tea.KeyCtrlB}, 15, func(t *testing.T, got logViewModel) {
			assert.Less(t, got.cursor, 15)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := makeModelWithLines(20)
			m.cursor = tt.cursor
			updated, _ := m.Update(tt.key)
			got := updated.(logViewModel)
			tt.checkFn(t, got)
		})
	}
}

func TestUpdateNormalScreenPositions(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyMsg
		// want computes the expected cursor from the pre-update model (inputs),
		// not from the result, so the assertion isn't circular.
		want func(m logViewModel) int
	}{
		{"H top of screen", keyRunes("H"), func(m logViewModel) int { return m.offset }},
		{"M middle of screen", keyRunes("M"), func(m logViewModel) int { return m.offset + m.logHeight()/2 }},
		{"L bottom of screen", keyRunes("L"), func(m logViewModel) int { return m.offset + m.logHeight() - 1 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := makeModelWithLines(20)
			want := tt.want(m)
			wantOffset := m.offset

			updated, _ := m.Update(tt.key)
			got := updated.(logViewModel)

			assert.Equal(t, want, got.cursor)
			assert.Equal(t, wantOffset, got.offset, "offset must not change")
			assert.False(t, got.follow)
		})
	}
}

func TestUpdateNormalTogglePreview(t *testing.T) {
	m := makeModelWithLines(5)
	m.cursor = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	got := updated.(logViewModel)
	assert.True(t, got.preview)

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyTab})
	got = updated.(logViewModel)
	assert.False(t, got.preview)
}

func TestUpdateNormalPreviewScroll(t *testing.T) {
	m := makeModelWithLines(5)
	m.cursor = 0
	m.preview = true
	m.previewLines = []string{"line1", "line2", "line3"}
	m.previewOffset = 0

	updated, _ := m.Update(keyRunes("J"))
	got := updated.(logViewModel)
	assert.Equal(t, 1, got.previewOffset)

	updated, _ = got.Update(keyRunes("K"))
	got = updated.(logViewModel)
	assert.Equal(t, 0, got.previewOffset)

	updated, _ = got.Update(keyRunes("K"))
	got = updated.(logViewModel)
	assert.Equal(t, 0, got.previewOffset)
}

func TestUpdateNormalPreviewScrollNoPreview(t *testing.T) {
	m := makeModelWithLines(5)
	m.preview = false
	m.previewOffset = 0

	updated, _ := m.Update(keyRunes("J"))
	got := updated.(logViewModel)
	assert.Equal(t, 0, got.previewOffset)
}

func TestUpdateNormalSearchMatch(t *testing.T) {
	m := newModel()
	m.height = 20
	m.width = 80
	m.lines = []logEntry{
		parseLine("aaa"),
		parseLine("match here"),
		parseLine("bbb"),
		parseLine("match again"),
	}
	m.refilter()
	m.follow = false
	m.search = "match"
	m.searchLower = "match"
	m.cursor = 0

	updated, _ := m.Update(keyRunes("n"))
	got := updated.(logViewModel)
	assert.Equal(t, 1, got.cursor)

	updated, _ = got.Update(keyRunes("n"))
	got = updated.(logViewModel)
	assert.Equal(t, 3, got.cursor)

	updated, _ = got.Update(keyRunes("N"))
	got = updated.(logViewModel)
	assert.Equal(t, 1, got.cursor)
}

func TestUpdateNormalFilterByLevel(t *testing.T) {
	tests := []struct {
		key   string
		level level
	}{
		{"1", levelError},
		{"2", levelWarn},
		{"3", levelInfo},
		{"4", levelDebug},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			m := makeModelWithLines(5)
			updated, _ := m.Update(keyRunes(tt.key))
			got := updated.(logViewModel)
			assert.True(t, got.hasLevelFilter)
			assert.Equal(t, tt.level, got.levelFilter)
			assert.Equal(t, "", got.filter)
		})
	}
}

func TestUpdateNormalAltEnterMarker(t *testing.T) {
	m := makeModelWithLines(3)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter, Alt: true})
	got := updated.(logViewModel)
	require.Len(t, got.lines, 4)
	assert.True(t, got.lines[3].isMarker)
	assert.Equal(t, "── MARKER ──", got.lines[3].raw)
}

func TestUpdateNormalCopyY(t *testing.T) {
	m := makeModelWithLines(3)
	m.cursor = 1

	updated, _ := m.Update(keyRunes("y"))
	got := updated.(logViewModel)
	assert.Equal(t, 1, got.cursor)

	// out-of-range cursor must not panic on m.filtered[m.cursor]
	m.cursor = len(m.filtered)

	assert.NotPanics(t, func() { m.Update(keyRunes("y")) })
}

func TestHelpModeReturnsToNormal(t *testing.T) {
	keys := []tea.KeyMsg{
		keyRunes("x"),
		keyRunes("j"),
		{Type: tea.KeyEnter},
		{Type: tea.KeyEscape},
		{Type: tea.KeyUp},
	}

	for _, key := range keys {
		t.Run(key.String(), func(t *testing.T) {
			m := newModel()
			m.height = 10
			m.width = 80
			m.mode = modeHelp

			updated, _ := m.Update(key)
			got := updated.(logViewModel)
			assert.Equal(t, modeNormal, got.mode)
		})
	}
}

func TestView(t *testing.T) {
	tests := []struct {
		name  string
		setup func() logViewModel
		check func(t *testing.T, out string)
	}{
		{"width zero returns loading", newModel, func(t *testing.T, out string) {
			assert.Equal(t, "loading...", out)
		}},
		{"help mode", func() logViewModel {
			m := newModel()
			m.width = 80
			m.height = 30
			m.mode = modeHelp

			return m
		}, func(t *testing.T, out string) {
			assert.Contains(t, out, "Keyboard Shortcuts")
		}},
		{"normal with lines", func() logViewModel {
			m := newModel()
			m.width = 80
			m.height = 20
			m.lines = []logEntry{parseLine(`{"level":"info","msg":"hello"}`), parseLine(`{"level":"error","msg":"boom"}`)}
			m.refilter()

			return m
		}, func(t *testing.T, out string) {
			assert.Contains(t, out, "hello")
			assert.Contains(t, out, "boom")
			assert.Contains(t, out, "2 lines")
		}},
		{"with preview", func() logViewModel {
			m := newModel()
			m.width = 80
			m.height = 20
			m.preview = true
			m.previewLines = []string{"preview line 1", "preview line 2"}
			m.lines = []logEntry{parseLine("test line")}
			m.refilter()

			return m
		}, func(t *testing.T, out string) {
			assert.Contains(t, out, "preview line 1")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setup()
			tt.check(t, m.View())
		})
	}
}

func TestHeaderText(t *testing.T) {
	tests := []struct {
		name   string
		search string
		want   string
	}{
		{"without search", "", "filter: all"},
		{"with search", "foo", "search: foo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newModel()
			m.filtered = []int{0, 1}
			m.search = tt.search
			assert.Contains(t, m.headerText(), tt.want)
			assert.Contains(t, m.headerText(), "2 lines")
		})
	}
}

func TestRenderHelp(t *testing.T) {
	m := newModel()
	m.width = 80
	m.height = 30
	out := m.renderHelp()
	assert.Contains(t, out, "Keyboard Shortcuts")
	assert.Contains(t, out, "Press any key to close")
}

func TestFooterTextHighlightSearch(t *testing.T) {
	m := newModel()
	m.mode = modeHighlightSearch
	m.searchBuf = "test"
	footer := m.footerText()
	assert.Contains(t, footer, "s/")
	assert.Contains(t, footer, "test")
}

func TestAnsiPairDecomposition(t *testing.T) {
	styles := []lipgloss.Style{
		styleJSONKey,
		styleJSONString,
		styleSearchHL,
		styleError,
		styleDim,
		styleMsg,
	}

	texts := []string{"hello", "x", "ERROR", "with spaces"}

	for _, s := range styles {
		pre, suf := ansiPair(s)
		for _, text := range texts {
			assert.Equal(t, s.Render(text), pre+text+suf,
				"ansiPair decomposition broken; if this fails, JSON/level/highlight rendering is silently wrong")
		}
	}
}

func TestAnsiPairEmitsANSIWhenProfileForcesColor(t *testing.T) {
	prev := lipgloss.DefaultRenderer().ColorProfile()

	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })
	lipgloss.SetColorProfile(termenv.ANSI256)

	cases := []struct {
		name  string
		style lipgloss.Style
	}{
		{"foreground only", lipgloss.NewStyle().Foreground(lipgloss.Color("12"))},
		{"foreground + background", lipgloss.NewStyle().Background(lipgloss.Color("5")).Foreground(lipgloss.Color("15"))},
		{"bold", lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			pre, suf := ansiPair(tt.style)
			assert.Contains(t, pre, "\x1b[", "prefix missing ANSI escape — lipgloss stripped sentinel?")
			assert.Contains(t, suf, "\x1b[", "suffix missing ANSI reset")
			assert.Equal(t, tt.style.Render("test"), pre+"test"+suf)
		})
	}
}
