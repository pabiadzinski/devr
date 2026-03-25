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
