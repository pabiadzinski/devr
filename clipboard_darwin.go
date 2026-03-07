package devr

import (
	"os/exec"
	"strings"
)

func copyToClipboard(s string) {
	c := exec.Command("pbcopy")
	c.Stdin = strings.NewReader(s)
	_ = c.Run()
}
