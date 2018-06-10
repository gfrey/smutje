package smutje

import (
	"io"
	"log"
	"os"
)

var logOutput io.Writer = os.Stdout

func tagLogger(old *log.Logger, tag string) *log.Logger {
	return log.New(logOutput, old.Prefix()+tag+" ", old.Flags())
}
