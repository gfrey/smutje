package connection

import (
	"io"

	"github.com/gfrey/smutje/logger"
)

type Session interface {
	Close() error

	StdinPipe() (io.WriteCloser, error)
	StdoutPipe() (io.Reader, error)
	StderrPipe() (io.Reader, error)

	Run(cmd string) error
	Start(cmd string) error
	Wait() error
}

type Client interface {
	Name() string
	NewSession() (Session, error)
	NewLoggedSession(l logger.Logger) (Session, error)
	Close() error
}
