package connection

import (
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"io"
)

type sshSession struct {
	*ssh.Session
	withSudo bool
}

func (s *sshSession) Run(cmd string) error {
	if s.withSudo {
		cmd = "sudo " + cmd
	}
	return errors.Wrap(s.Session.Run(cmd), "failed to run command")
}

func (s *sshSession) Start(cmd string) error {
	if s.withSudo {
		cmd = "sudo " + cmd
	}
	return errors.Wrap(s.Session.Start(cmd), "failed to start command")
}

func (s *sshSession) Wait() error {
	return errors.Wrap(s.Session.Wait(), "failed to wait for command to exit")
}

func (s *sshSession) StdinPipe() (io.WriteCloser, error) {
	w, err := s.Session.StdinPipe()
	return w, errors.Wrap(err, "failed to fetch stdin pipe")
}

func (s *sshSession) StdoutPipe() (io.Reader, error) {
	r, err := s.Session.StdoutPipe()
	return r, errors.Wrap(err, "failed to fetch stdout pipe")
}

func (s *sshSession) StderrPipe() (io.Reader, error) {
	r, err := s.Session.StderrPipe()
	return r, errors.Wrap(err, "failed to fetch stderr pipe")
}
