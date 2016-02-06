package logger

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

type Logger interface {
	Tag(name string) Logger
	Printf(format string, args ...interface{}) (int, error)
}

func New() Logger {
	return &logger{output: os.Stdout, started: time.Now()}
}

func NewDiscard() Logger {
	return &logger{output: ioutil.Discard, started: time.Now()}
}

type logger struct {
	output io.Writer
	tags   []string

	started time.Time
}

func (l *logger) Tag(tag string) Logger {
	nl := new(logger)
	nl.output = l.output
	nl.started = l.started
	nl.tags = append(l.tags, tag)
	return nl
}

func (l *logger) Printf(format string, args ...interface{}) (int, error) {
	return fmt.Fprintf(l.output, fmt.Sprintf("[%.3f] [%s] %s\n", time.Since(l.started).Seconds(), strings.Join(l.tags, " "), format), args...)
}
