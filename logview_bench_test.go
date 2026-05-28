package devr

import (
	"strings"
	"testing"
)

var benchJSONSmall = `{"level":"info","msg":"hello","ts":"2026-04-28T10:00:00Z"}`

var benchJSONMedium = `{"level":"error","ts":"2026-04-28T10:00:00Z","caller":"app/server.go:142","msg":"request failed","method":"GET","path":"/api/users/42","status":500,"duration_ms":123,"error":"connection refused","trace_id":"abc-123-def-456"}`

var benchJSONLarge = func() string {
	var b strings.Builder

	b.WriteString(`{"level":"info","msg":"batch","items":[`)

	for i := 0; i < 100; i++ {
		if i > 0 {
			b.WriteString(",")
		}

		b.WriteString(`{"id":`)
		b.WriteString(`12345`)
		b.WriteString(`,"name":"item-with-some-data","active":true,"score":0.5}`)
	}

	b.WriteString(`]}`)

	return b.String()
}()

var benchPlain = "this is a plain log line, not JSON at all"

func BenchmarkFormatJSON(b *testing.B) {
	cases := []struct {
		name string
		in   string
	}{
		{"small", benchJSONSmall},
		{"medium", benchJSONMedium},
		{"large", benchJSONLarge},
		{"plain", benchPlain},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = formatJSON(c.in)
			}
		})
	}
}

func BenchmarkColorizeJSON(b *testing.B) {
	cases := []struct {
		name string
		in   string
	}{
		{"small", benchJSONSmall},
		{"medium", benchJSONMedium},
		{"large", benchJSONLarge},
		{"plain", benchPlain},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = colorizeJSON(c.in)
			}
		})
	}
}

func BenchmarkUpdatePreview(b *testing.B) {
	m := newModel()
	m.height = 30
	m.width = 120
	m.preview = true

	for i := 0; i < 1000; i++ {
		m.appendLine(parseLine(benchJSONMedium))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		m.cursor = i % len(m.filtered)
		m.updatePreview()
	}
}

func BenchmarkLogEntryRender(b *testing.B) {
	cases := []struct {
		name   string
		entry  logEntry
		filter string
		search string
		fields []string
	}{
		{"plain", parseLine(benchPlain), "", "", nil},
		{"json-level", parseLine(benchJSONMedium), "", "", nil},
		{"filter-hit", parseLine(benchJSONMedium), "request", "request", nil},
		{"search-hit", parseLine(benchJSONMedium), "", "", nil},
		{"field-hl", parseLine(benchJSONMedium), "", "", []string{"msg"}},
		{"filter+search", parseLine(benchJSONMedium), "request", "request", nil},
	}

	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			search, searchLower := "", ""
			if c.name == "search-hit" {
				search, searchLower = "request", "request"
			}

			for i := 0; i < b.N; i++ {
				_ = c.entry.render(false, c.filter, strings.ToLower(c.filter), search, searchLower, 120, c.fields)
			}
		})
	}
}

func BenchmarkView(b *testing.B) {
	cases := []struct {
		name    string
		input   string
		filter  string
		search  string
		wrap    bool
		preview bool
	}{
		{"plain-nowrap", benchPlain, "", "", false, false},
		{"json-nowrap", benchJSONMedium, "", "", false, false},
		{"json-wrap", benchJSONMedium, "", "", true, false},
		{"json-search", benchJSONMedium, "", "request", false, false},
		{"json-preview", benchJSONMedium, "", "", false, true},
	}

	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			m := newModel()
			m.height = 30
			m.width = 120
			m.wrap = c.wrap
			m.preview = c.preview

			if c.search != "" {
				m.search = c.search
				m.searchLower = c.search
			}

			if c.filter != "" {
				m.filter = c.filter
				m.filterLower = c.filter
			}

			for i := 0; i < 1000; i++ {
				m.appendLine(parseLine(c.input))
			}

			if c.preview {
				m.updatePreview()
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = m.View()
			}
		})
	}
}
