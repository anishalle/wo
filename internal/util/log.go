package util

import (
	"fmt"
	"os"
)

func Debugf(format string, args ...any) {
	if os.Getenv("WO_DEBUG") == "" {
		return
	}
	fmt.Fprintf(os.Stderr, "wo debug: "+format+"\n", args...)
}
