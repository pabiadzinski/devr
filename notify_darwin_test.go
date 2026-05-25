//go:build darwin

package devr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkEscapeAppleScript(b *testing.B) {
	in := `Build "failed" at line 42` + "\n" + `error: cannot find symbol` + "\r\n" + `\path\to\file`
	for i := 0; i < b.N; i++ {
		_ = escapeAppleScript(in)
	}
}

func TestEscapeAppleScript(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "hello", "hello"},
		{"backslash", `a\b`, `a\\b`},
		{"quote", `a"b`, `a\"b`},
		{"newline", "a\nb", "a b"},
		{"carriage return", "a\rb", "a b"},
		{"crlf", "a\r\nb", "a  b"},
		{"mixed", "a\\\"b\nc", `a\\\"b c`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, escapeAppleScript(tt.in))
		})
	}
}
