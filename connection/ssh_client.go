package connection

import (
	"fmt"
	"net"
	"os"

	"github.com/gfrey/smutje/logger"
	"github.com/pkg/errors"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/terminal"
)

type sshClient struct {
	addr   string
	client *ssh.Client
	config *ssh.ClientConfig
	agent  agent.Agent

	smutjeScript string
}

func NewSSHClient(addr, user string) (Client, error) {
	sc := new(sshClient)
	sc.config = new(ssh.ClientConfig)
	sc.addr = addr
	sc.config.User = user

	sc.agent = sshAgent()

	if sc.agent != nil {
		signers, err := sc.agent.Signers()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to get signers from SSH agent: %s", err)
		} else if len(signers) > 0 {
			sc.config.Auth = []ssh.AuthMethod{ssh.PublicKeys(signers...)}
		}
	}
	sc.config.Auth = append(sc.config.Auth, ssh.PasswordCallback(sc.askForPassword))

	var err error
	addr = fmt.Sprintf("%s:%d", sc.addr, 22)
	sc.client, err = ssh.Dial("tcp", addr, sc.config)
	return sc, errors.Wrap(err, "failed to connect to SSH host")
}

func (sc *sshClient) Name() string {
	return sc.addr
}

func (sc *sshClient) NewSession() (Session, error) {
	s, err := sc.client.NewSession()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new SSH session")
	}

	return &sshSession{Session: s, withSudo: sc.config.User != "root"}, nil
}

func (sc *sshClient) NewLoggedSession(l logger.Logger) (Session, error) {
	sess, err := sc.NewSession()
	if err != nil {
		return nil, err
	}

	return newLoggedSession(l, sess)
}

func (sc *sshClient) Close() error {
	return sc.client.Close()
}

func sshAgent() agent.Agent {
	sshAuthSocket := os.Getenv("SSH_AUTH_SOCK")
	if sshAuthSocket == "" {
		return nil
	}

	agentConn, err := net.Dial("unix", sshAuthSocket)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to SSH agent: %s", err)
		return nil
	}

	return agent.NewClient(agentConn)
}

func (sc *sshClient) askForPassword() (string, error) {
	fmt.Printf("Password for %s@%s: ", sc.config.User, sc.addr)
	buf, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	fmt.Printf("\n")
	return string(buf), errors.Wrap(err, "failed to read password")
}
