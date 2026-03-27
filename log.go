package devr

import (
	"fmt"
	"io"
	"os"
)

const (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	dim    = "\033[2m"
	bold   = "\033[1m"
)

var (
	LogDebug bool
	logOut   io.Writer = os.Stderr
)

func Info(msg string, args ...any) {
	_, _ = fmt.Fprintf(logOut, green+"INFO"+reset+"  "+msg+"\n", args...)
}

func Warn(msg string, args ...any) {
	_, _ = fmt.Fprintf(logOut, yellow+"WARN"+reset+"  "+msg+"\n", args...)
}

func Error(msg string, args ...any) {
	_, _ = fmt.Fprintf(logOut, red+"ERROR"+reset+" "+msg+"\n", args...)
}

func Dbg(msg string, args ...any) {
	if !LogDebug {
		return
	}

	_, _ = fmt.Fprintf(logOut, dim+"DEBUG"+reset+" "+msg+"\n", args...)
}
