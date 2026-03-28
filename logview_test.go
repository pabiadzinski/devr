package devr

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
		out := highlightAll("no match here", "xyz", "xyz", styleMatch)
		assert.Equal(t, "no match here", out)
	})

	t.Run("empty input", func(t *testing.T) {
		out := highlightAll("", "test", "test", styleMatch)
		assert.Equal(t, "", out)
	})

	t.Run("match preserves surrounding text", func(t *testing.T) {
		out := highlightAll("abc hello def", "hello", "hello", styleMatch)
		assert.True(t, strings.HasPrefix(out, "abc "))
		assert.True(t, strings.HasSuffix(out, " def"))
	})

	t.Run("multiple matches", func(t *testing.T) {
		out := highlightAll("aa bb aa", "aa", "aa", styleMatch)
		// original "aa" appears twice; output should be longer due to styling
		assert.GreaterOrEqual(t, len(out), len("aa bb aa"))
	})
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
			assert.Equal(t, "error", got.filter)
		}},
		{"0", func(t *testing.T, got logViewModel) {
			assert.Equal(t, "", got.filter)
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
	tests := []struct {
		input string
	}{
		{`"hello"`},
		{`"hello",`},
		{"true"},
		{"false"},
		{"null"},
		{"42"},
		{"-1"},
		{"42,"},
		{"{"},
	}
	for _, tt := range tests {
		out := colorizeJSONValue(tt.input)
		assert.NotEmpty(t, out, tt.input)
	}
}

func TestColorizeJSONLine(t *testing.T) {
	assert.NotEmpty(t, colorizeJSONLine(`  "key": "value"`))
	assert.NotEmpty(t, colorizeJSONLine(`  {`))
	assert.NotEmpty(t, colorizeJSONLine(`  }`))
	assert.NotEmpty(t, colorizeJSONLine(`  },`))
	assert.NotEmpty(t, colorizeJSONLine(`  [`))
	assert.NotEmpty(t, colorizeJSONLine(`  ]`))
	assert.NotEmpty(t, colorizeJSONLine(`  ],`))
	assert.NotEmpty(t, colorizeJSONLine(`  "key": 42`))
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

	normal := entry.render(false, "", "", "", "", 80, nil)
	assert.Contains(t, normal, "━")
}

func TestLogEntryRenderWithLevelKeywords(t *testing.T) {
	entry := logEntry{raw: `level=ERROR msg="boom"`, lower: `level=error msg="boom"`}
	out := entry.render(false, "", "", "", "", 80, nil)
	assert.Contains(t, out, "ERROR")
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
