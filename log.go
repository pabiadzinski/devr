package devr

import (
	"fmt"
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

func Info(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, green+"INFO"+reset+"  "+msg+"\n", args...)
}

func Warn(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, yellow+"WARN"+reset+"  "+msg+"\n", args...)
}

func Error(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, red+"ERROR"+reset+" "+msg+"\n", args...)
}

var LogDebug bool

func Dbg(msg string, args ...any) {
	if !LogDebug {
		return
	}

	fmt.Fprintf(os.Stderr, dim+"DEBUG"+reset+" "+msg+"\n", args...)
}
