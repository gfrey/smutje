package connection

import "golang.org/x/crypto/ssh"

type sshSession struct {
	*ssh.Session
	withSudo bool
}

func (s *sshSession) Run(cmd string) error {
	if s.withSudo {
		cmd = "sudo " + cmd
	}
	return s.Session.Run(cmd)
}

func (s *sshSession) Start(cmd string) error {
	if s.withSudo {
		cmd = "sudo " + cmd
	}
	return s.Session.Start(cmd)
}
