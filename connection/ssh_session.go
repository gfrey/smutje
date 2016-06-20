package connection

import (
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
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
