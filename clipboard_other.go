//go:build !darwin

package devr

import (
	"os/exec"
	"strings"
)

func copyToClipboard(s string) {
	if path, err := exec.LookPath("xclip"); err == nil {
		c := exec.Command(path, "-selection", "clipboard")
		c.Stdin = strings.NewReader(s)
		_ = c.Run()

		return
	}

	if path, err := exec.LookPath("xsel"); err == nil {
		c := exec.Command(path, "--clipboard", "--input")
		c.Stdin = strings.NewReader(s)
		_ = c.Run()
	}
}
