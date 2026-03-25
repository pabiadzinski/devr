//go:build darwin

package devr

import (
	"os/exec"
	"strings"
)

func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)

	return s
}

func Notify(title, msg string) {
	_ = exec.Command("osascript", "-e",
		`display notification "`+escapeAppleScript(msg)+`" with title "`+escapeAppleScript(title)+`"`).Run()
}
