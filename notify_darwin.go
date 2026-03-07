//go:build darwin

package devr

import "os/exec"

func Notify(title, msg string) {
	_ = exec.Command("osascript", "-e",
		`display notification "`+msg+`" with title "`+title+`"`).Run()
}
