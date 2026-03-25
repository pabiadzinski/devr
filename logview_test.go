package devr

import (
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
