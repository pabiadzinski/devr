package devr

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func jsonEvent(action, pkg, test, output string, elapsed float64) string {
	parts := []string{
		`"Action":"` + action + `"`,
		`"Package":"` + pkg + `"`,
	}

	if test != "" {
		parts = append(parts, `"Test":"`+test+`"`)
	}

	if output != "" {
		parts = append(parts, `"Output":"`+output+`"`)
	}

	if elapsed > 0 {
		parts = append(parts, fmt.Sprintf(`"Elapsed":%g`, elapsed))
	}

	return "{" + strings.Join(parts, ",") + "}"
}

func TestShortPkg(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"github.com/user/repo/pkg", "pkg"},
		{"github.com/user/repo", "repo"},
		{"mypackage", "mypackage"},
		{"", ""},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, shortPkg(tt.input))
	}
}

func TestFormatElapsed(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0.005, "5ms"},
		{0.100, "100ms"},
		{0.999, "999ms"},
		{1.0, "1.0s"},
		{1.5, "1.5s"},
		{10.23, "10.2s"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, formatElapsed(tt.input))
	}
}

func TestTestFormatterDots(t *testing.T) {
	input := strings.Join([]string{
		jsonEvent("pass", "pkg/a", "TestOne", "", 0),
		jsonEvent("fail", "pkg/a", "TestTwo", "", 0),
		jsonEvent("skip", "pkg/a", "TestThree", "", 0),
		jsonEvent("pass", "pkg/a", "", "", 0.5),
	}, "\n")

	var buf bytes.Buffer

	f := newTestFormatter(&buf, fmtDots)
	f.Process(strings.NewReader(input))

	assert.Equal(t, 1, f.passed)
	assert.Equal(t, 1, f.failed)
	assert.Equal(t, 1, f.skipped)
	assert.Contains(t, buf.String(), "·")
	assert.Contains(t, buf.String(), "✗")
	assert.Contains(t, buf.String(), "-")
}

func TestTestFormatterShort(t *testing.T) {
	input := strings.Join([]string{
		jsonEvent("pass", "github.com/user/repo/api", "TestA", "", 0),
		jsonEvent("pass", "github.com/user/repo/api", "", "", 0.2),
	}, "\n")

	var buf bytes.Buffer

	f := newTestFormatter(&buf, fmtShort)
	f.Process(strings.NewReader(input))

	assert.Equal(t, 1, f.passed)
	assert.Contains(t, buf.String(), "api")
	assert.Contains(t, buf.String(), "✓")
}

func TestTestFormatterTestname(t *testing.T) {
	input := strings.Join([]string{
		jsonEvent("pass", "pkg/a", "TestFoo", "", 0.01),
		jsonEvent("skip", "pkg/a", "TestBar", "", 0),
		jsonEvent("pass", "pkg/a", "", "", 0.1),
	}, "\n")

	var buf bytes.Buffer

	f := newTestFormatter(&buf, fmtTestname)
	f.Process(strings.NewReader(input))

	assert.Equal(t, 1, f.passed)
	assert.Equal(t, 1, f.skipped)
	assert.Contains(t, buf.String(), "TestFoo")
	assert.Contains(t, buf.String(), "TestBar")
}

func TestTestFormatterVerbose(t *testing.T) {
	input := strings.Join([]string{
		jsonEvent("run", "pkg", "TestX", "", 0),
		jsonEvent("output", "pkg", "TestX", "--- PASS: TestX (0.00s)\n", 0),
		jsonEvent("pass", "pkg", "TestX", "", 0.001),
	}, "\n")

	var buf bytes.Buffer

	f := newTestFormatter(&buf, fmtVerbose)
	f.Process(strings.NewReader(input))

	assert.Equal(t, 1, f.passed)
	assert.Contains(t, buf.String(), "PASS")
}

func TestTestFormatterSummary(t *testing.T) {
	var buf bytes.Buffer

	f := newTestFormatter(&buf, fmtDots)
	f.passed = 5
	f.failed = 1
	f.skipped = 2

	f.Summary()

	out := buf.String()
	assert.Contains(t, out, "5 passed")
	assert.Contains(t, out, "1 failed")
	assert.Contains(t, out, "2 skipped")
}

func TestTestFormatterSummaryNoFailures(t *testing.T) {
	var buf bytes.Buffer

	f := newTestFormatter(&buf, fmtDots)
	f.passed = 3

	f.Summary()

	out := buf.String()
	assert.Contains(t, out, "3 passed")
	assert.NotContains(t, out, "failed")
	assert.NotContains(t, out, "skipped")
}

func TestTestFormatterPrintFailures(t *testing.T) {
	var buf bytes.Buffer

	f := newTestFormatter(&buf, fmtShort)

	input := strings.Join([]string{
		jsonEvent("output", "pkg/x", "TestBad", "    expected: true\n", 0),
		jsonEvent("fail", "pkg/x", "TestBad", "", 0),
		jsonEvent("fail", "pkg/x", "", "", 0.1),
	}, "\n")
	f.Process(strings.NewReader(input))
	f.Summary()

	out := buf.String()
	assert.Contains(t, out, "FAILURES")
	assert.Contains(t, out, "expected: true")
}

func TestIsFailKey(t *testing.T) {
	f := &testFormatter{
		pkgFailed: map[string]bool{
			"github.com/user/repo/pkg": true,
		},
	}

	tests := []struct {
		key  string
		want bool
	}{
		{"github.com/user/repo/pkg/TestFoo", true},
		{"github.com/user/repo/other/TestBar", false},
		{"github.com/user/repo/pkg", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, f.isFailKey(tt.key), tt.key)
	}
}

func TestTestFormatterVerboseFullCoverage(t *testing.T) {
	input := strings.Join([]string{
		jsonEvent("run", "pkg", "TestA", "", 0),
		jsonEvent("output", "pkg", "TestA", "--- PASS: TestA (0.00s)\n", 0),
		jsonEvent("pass", "pkg", "TestA", "", 0.001),
		jsonEvent("run", "pkg", "TestB", "", 0),
		jsonEvent("output", "pkg", "TestB", "--- FAIL: TestB (0.00s)\n", 0),
		jsonEvent("fail", "pkg", "TestB", "", 0.001),
		jsonEvent("run", "pkg", "TestC", "", 0),
		jsonEvent("output", "pkg", "TestC", "--- SKIP: TestC (0.00s)\n", 0),
		jsonEvent("skip", "pkg", "TestC", "", 0),
		jsonEvent("output", "pkg", "", "PASS\n", 0),
		jsonEvent("output", "pkg", "", "FAIL\n", 0),
		jsonEvent("output", "pkg", "", "ok  pkg 0.1s\n", 0),
		jsonEvent("output", "pkg", "", "=== RUN TestD\n", 0),
		jsonEvent("output", "pkg", "", "plain output\n", 0),
		jsonEvent("fail", "pkg", "", "", 0.1),
	}, "\n")

	var buf bytes.Buffer

	f := newTestFormatter(&buf, fmtVerbose)
	f.Process(strings.NewReader(input))

	assert.Equal(t, 1, f.passed)
	assert.Equal(t, 1, f.failed)
	assert.Equal(t, 1, f.skipped)
	assert.True(t, f.pkgFailed["pkg"])

	out := buf.String()
	assert.Contains(t, out, "PASS")
	assert.Contains(t, out, "FAIL")
	assert.Contains(t, out, "SKIP")
	assert.Contains(t, out, "plain output")
}

func TestTestFormatterDotsLineWrap(t *testing.T) {
	var events []string

	for i := 0; i < 85; i++ {
		events = append(events, jsonEvent("pass", "pkg", fmt.Sprintf("Test%d", i), "", 0))
	}

	events = append(events, jsonEvent("pass", "pkg", "", "", 0.5))

	var buf bytes.Buffer

	f := newTestFormatter(&buf, fmtDots)
	f.Process(strings.NewReader(strings.Join(events, "\n")))

	assert.Equal(t, 85, f.passed)
	assert.Contains(t, buf.String(), "\n")
}

func TestTestFormatterProcessInvalidJSON(t *testing.T) {
	var buf bytes.Buffer

	f := newTestFormatter(&buf, fmtDots)
	f.Process(strings.NewReader("not json at all"))

	assert.Contains(t, buf.String(), "not json at all")
}

func TestTestFormatterTestnameFailedPkg(t *testing.T) {
	input := strings.Join([]string{
		jsonEvent("fail", "pkg/a", "TestX", "", 0),
		jsonEvent("fail", "pkg/a", "", "", 0.1),
	}, "\n")

	var buf bytes.Buffer

	f := newTestFormatter(&buf, fmtTestname)
	f.Process(strings.NewReader(input))

	assert.Equal(t, 1, f.failed)
	assert.Contains(t, buf.String(), "✗")
}

func TestTestFormatterShortFailedPkg(t *testing.T) {
	input := strings.Join([]string{
		jsonEvent("fail", "github.com/user/repo/api", "TestA", "", 0),
		jsonEvent("output", "github.com/user/repo/api", "", "build error\n", 0),
		jsonEvent("fail", "github.com/user/repo/api", "", "", 0.2),
	}, "\n")

	var buf bytes.Buffer

	f := newTestFormatter(&buf, fmtShort)
	f.Process(strings.NewReader(input))

	assert.Equal(t, 1, f.failed)
	assert.Contains(t, buf.String(), "✗")
	assert.Contains(t, buf.String(), "api")
}

func TestHandleVerboseOutputBranches(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		contains string
	}{
		{"pass line", "--- PASS: TestX (0.00s)", green},
		{"fail line", "--- FAIL: TestX (0.00s)", red},
		{"skip line", "--- SKIP: TestX (0.00s)", yellow},
		{"PASS", "PASS", green},
		{"FAIL", "FAIL", red},
		{"ok line", "ok  pkg 0.1s", green},
		{"RUN line", "=== RUN TestX", dim},
		{"default", "some normal output", "some normal output"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			f := newTestFormatter(&buf, fmtVerbose)
			f.handleVerbose(testEvent{Action: "output", Package: "pkg", Output: tt.output + "\n"})
			assert.Contains(t, buf.String(), tt.contains)
		})
	}
}

func TestHandleVerboseCounters(t *testing.T) {
	tests := []struct {
		name    string
		action  string
		test    string
		wantP   int
		wantF   int
		wantS   int
		wantPkg bool
	}{
		{"test pass", "pass", "TestA", 1, 0, 0, false},
		{"test fail", "fail", "TestA", 0, 1, 0, false},
		{"test skip", "skip", "TestA", 0, 0, 1, false},
		{"pkg fail", "fail", "", 0, 0, 0, true},
		{"pkg pass", "pass", "", 0, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			f := newTestFormatter(&buf, fmtVerbose)
			f.handleVerbose(testEvent{Action: tt.action, Package: "pkg", Test: tt.test})
			assert.Equal(t, tt.wantP, f.passed)
			assert.Equal(t, tt.wantF, f.failed)
			assert.Equal(t, tt.wantS, f.skipped)
			assert.Equal(t, tt.wantPkg, f.pkgFailed["pkg"])
		})
	}
}

func TestHandleVerboseNonOutputReturnsEarly(t *testing.T) {
	var buf bytes.Buffer

	f := newTestFormatter(&buf, fmtVerbose)
	f.handleVerbose(testEvent{Action: "run", Package: "pkg", Test: "TestX"})
	assert.Empty(t, buf.String())
}

func TestCollectFailOutput(t *testing.T) {
	f := newTestFormatter(nil, fmtShort)

	f.collectFailOutput(testEvent{Action: "run", Package: "pkg", Test: "TestA"})
	assert.Empty(t, f.failOrder)

	f.collectFailOutput(testEvent{Action: "output", Package: "pkg", Test: "TestA", Output: "line1\n"})
	assert.Equal(t, []string{"pkg/TestA"}, f.failOrder)
	assert.Equal(t, []string{"line1\n"}, f.failOutput["pkg/TestA"])

	f.collectFailOutput(testEvent{Action: "output", Package: "pkg", Test: "TestA", Output: "line2\n"})
	assert.Equal(t, []string{"pkg/TestA"}, f.failOrder)
	assert.Equal(t, []string{"line1\n", "line2\n"}, f.failOutput["pkg/TestA"])

	f.collectFailOutput(testEvent{Action: "output", Package: "pkg", Output: "pkg-level\n"})
	assert.Equal(t, []string{"pkg/TestA", "pkg"}, f.failOrder)
	assert.Equal(t, []string{"pkg-level\n"}, f.failOutput["pkg"])
}

func TestPrintFailuresNoFails(t *testing.T) {
	var buf bytes.Buffer

	f := newTestFormatter(&buf, fmtShort)
	f.failed = 0
	f.printFailures()
	assert.Empty(t, buf.String())
}

func TestPrintFailuresFiltering(t *testing.T) {
	var buf bytes.Buffer

	f := newTestFormatter(&buf, fmtShort)
	f.failed = 1
	f.pkgFailed["pkg"] = true
	f.failOrder = []string{"pkg/TestBad"}
	f.failOutput["pkg/TestBad"] = []string{
		"=== RUN TestBad\n",
		"--- FAIL: TestBad (0.00s)\n",
		"FAIL pkg\n",
		"\n",
		"    expected: 42\n",
		"    got: 0\n",
	}

	f.printFailures()

	out := buf.String()

	assert.Contains(t, out, "FAILURES")
	assert.Contains(t, out, "pkg/TestBad")
	assert.Contains(t, out, "expected: 42")
	assert.Contains(t, out, "got: 0")
	assert.NotContains(t, out, "=== RUN")
	assert.NotContains(t, out, "--- FAIL")
	assert.Equal(t, 1, strings.Count(out, "expected"))
}
