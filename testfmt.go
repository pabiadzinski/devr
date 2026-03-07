package devr

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	fmtDots     = "dots"
	fmtShort    = "short"
	fmtTestname = "testname"
	fmtVerbose  = "verbose"
)

type testEvent struct {
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Output  string  `json:"Output"`
	Elapsed float64 `json:"Elapsed"`
}

type testFormatter struct {
	w      io.Writer
	format string

	failOutput map[string][]string
	failOrder  []string
	pkgFailed  map[string]bool
	passed     int
	failed     int
	skipped    int
	dotsCol    int
}

func newTestFormatter(w io.Writer, format string) *testFormatter {
	return &testFormatter{
		w:          w,
		format:     format,
		failOutput: make(map[string][]string),
		pkgFailed:  make(map[string]bool),
	}
}

func (f *testFormatter) put(format string, args ...any) {
	_, _ = fmt.Fprintf(f.w, format, args...)
}

func (f *testFormatter) Process(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var ev testEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			f.put("%s\n", scanner.Text())
			continue
		}

		f.handle(ev)
	}
}

func (f *testFormatter) handle(ev testEvent) {
	switch f.format {
	case fmtDots:
		f.handleDots(ev)
	case fmtShort:
		f.handleShort(ev)
	case fmtVerbose:
		f.handleVerbose(ev)
	default:
		f.handleTestname(ev)
	}
}

func (f *testFormatter) handleDots(ev testEvent) {
	if ev.Test == "" {
		f.collectFailOutput(ev)

		if ev.Action == "fail" {
			f.pkgFailed[ev.Package] = true
		}

		return
	}

	switch ev.Action {
	case "pass":
		f.passed++
		f.dot(green + "·")
	case "fail":
		f.failed++
		f.dot(red + "✗")
	case "skip":
		f.skipped++
		f.dot(yellow + "-")
	}

	f.collectFailOutput(ev)
}

func (f *testFormatter) dot(s string) {
	f.put("%s%s", s, reset)

	f.dotsCol++
	if f.dotsCol >= 80 {
		f.put("\n")
		f.dotsCol = 0
	}
}

func (f *testFormatter) handleShort(ev testEvent) {
	if ev.Test != "" {
		switch ev.Action {
		case "pass":
			f.passed++
		case "fail":
			f.failed++
		case "skip":
			f.skipped++
		}

		f.collectFailOutput(ev)

		return
	}

	switch ev.Action {
	case "pass":
		f.put("%s✓%s  %s%s%s (%s)\n", green, reset, dim, shortPkg(ev.Package), reset, formatElapsed(ev.Elapsed))
	case "fail":
		f.pkgFailed[ev.Package] = true
		f.put("%s✗%s  %s (%s)\n", red, reset, shortPkg(ev.Package), formatElapsed(ev.Elapsed))
	case "output":
		f.collectFailOutput(ev)
	}
}

func (f *testFormatter) handleTestname(ev testEvent) {
	if ev.Test == "" {
		f.collectFailOutput(ev)

		switch ev.Action {
		case "pass":
			f.put("%s✓%s  %s%s%s (%s)\n", green, reset, dim, shortPkg(ev.Package), reset, formatElapsed(ev.Elapsed))
		case "fail":
			f.pkgFailed[ev.Package] = true
			f.put("%s✗%s  %s (%s)\n", red, reset, shortPkg(ev.Package), formatElapsed(ev.Elapsed))
		}

		return
	}

	switch ev.Action {
	case "pass":
		f.passed++
		f.put("  %s✓%s  %s %s(%s)%s\n", green, reset, ev.Test, dim, formatElapsed(ev.Elapsed), reset)
	case "fail":
		f.failed++
		f.put("  %s✗%s  %s\n", red, reset, ev.Test)
	case "skip":
		f.skipped++
		f.put("  %s-%s  %s%s%s\n", yellow, reset, dim, ev.Test, reset)
	}

	f.collectFailOutput(ev)
}

func (f *testFormatter) handleVerbose(ev testEvent) {
	if ev.Test != "" {
		switch ev.Action {
		case "pass":
			f.passed++
		case "fail":
			f.failed++
		case "skip":
			f.skipped++
		}
	}

	if ev.Action == "fail" && ev.Test == "" {
		f.pkgFailed[ev.Package] = true
	}

	if ev.Action != "output" {
		return
	}

	line := strings.TrimRight(ev.Output, "\n")
	trimmed := strings.TrimSpace(line)

	switch {
	case strings.HasPrefix(trimmed, "--- PASS:"):
		f.put("%s%s%s\n", green, line, reset)
	case strings.HasPrefix(trimmed, "--- FAIL:"):
		f.put("%s%s%s\n", red, line, reset)
	case strings.HasPrefix(trimmed, "--- SKIP:"):
		f.put("%s%s%s\n", yellow, line, reset)
	case strings.HasPrefix(trimmed, "PASS"):
		f.put("%s%s%s\n", green, line, reset)
	case strings.HasPrefix(trimmed, "FAIL"):
		f.put("%s%s%s\n", red, line, reset)
	case strings.HasPrefix(trimmed, "ok "):
		f.put("%s%s%s\n", green, line, reset)
	case strings.HasPrefix(trimmed, "=== RUN"):
		f.put("%s%s%s\n", dim, line, reset)
	default:
		f.put("%s\n", line)
	}
}

func (f *testFormatter) collectFailOutput(ev testEvent) {
	if ev.Action != "output" {
		return
	}

	key := ev.Package
	if ev.Test != "" {
		key = ev.Package + "/" + ev.Test
	}

	if _, exists := f.failOutput[key]; !exists {
		f.failOrder = append(f.failOrder, key)
	}

	f.failOutput[key] = append(f.failOutput[key], ev.Output)
}

func (f *testFormatter) Summary() {
	if f.format == fmtDots && f.dotsCol > 0 {
		f.put("\n")
	}

	f.printFailures()

	f.put("\n%s%s%s", bold, strings.Repeat("─", 40), reset)

	parts := []string{fmt.Sprintf("%s%d passed%s", green, f.passed, reset)}
	if f.failed > 0 {
		parts = append(parts, fmt.Sprintf("%s%d failed%s", red, f.failed, reset))
	}

	if f.skipped > 0 {
		parts = append(parts, fmt.Sprintf("%s%d skipped%s", yellow, f.skipped, reset))
	}

	f.put("\n%s\n", strings.Join(parts, ", "))
}

func (f *testFormatter) printFailures() {
	if f.failed == 0 {
		return
	}

	f.put("\n%s%sFAILURES:%s\n", bold, red, reset)

	for _, key := range f.failOrder {
		if !f.isFailKey(key) {
			continue
		}

		lines := f.failOutput[key]
		f.put("\n%s── %s%s\n", red, key, reset)

		for _, line := range lines {
			out := strings.TrimRight(line, "\n")

			trimmed := strings.TrimSpace(out)
			if trimmed == "" || strings.HasPrefix(trimmed, "=== RUN") ||
				strings.HasPrefix(trimmed, "--- FAIL") ||
				strings.HasPrefix(trimmed, "FAIL") {
				continue
			}

			f.put("  %s\n", out)
		}
	}
}

func (f *testFormatter) isFailKey(key string) bool {
	for pkg := range f.pkgFailed {
		if strings.HasPrefix(key, pkg+"/") {
			return true
		}
	}

	return false
}

func shortPkg(pkg string) string {
	if i := strings.LastIndex(pkg, "/"); i >= 0 {
		return pkg[i+1:]
	}

	return pkg
}

func formatElapsed(s float64) string {
	d := time.Duration(s * float64(time.Second))
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}

	return fmt.Sprintf("%.1fs", d.Seconds())
}
