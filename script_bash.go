package smutje

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"

	"strings"

	"github.com/gfrey/gconn"
	"github.com/pkg/errors"
)

type bashScript struct {
	ID     string
	Script string
	hash   string
}

func (bashScript) MustExecute() bool {
	return false
}

func (s *bashScript) Hash() string {
	return s.hash
}

func (s *bashScript) Prepare(attrs Attributes, prevHash string) (string, error) {
	script, err := renderString(s.ID, "set -e\n"+s.Script+"\n", attrs)
	if err != nil {
		return "", err
	}
	s.Script = script
	s.hash = fmt.Sprintf("%x", md5.Sum([]byte(prevHash+s.Script)))
	return s.hash, nil
}

func (s *bashScript) Exec(l *log.Logger, client gconn.Client) error {
	fname := fmt.Sprintf("/var/lib/smutje/%s.sh", s.hash)
	cmd := fmt.Sprintf("cat - > %[1]s && bash -l %[1]s", fname)

	sess, err := gconn.NewLoggedClient(l, client).NewSession("/usr/bin/env", "bash", "-c", fmt.Sprintf("%q", cmd))
	if err != nil {
		return err
	}
	defer sess.Close()

	l.Printf("%s", strings.TrimSpace(s.Script[7:]))

	stdin, err := sess.StdinPipe()
	if err != nil {
		return errors.Wrap(err, "failed to receive stdin pipe")
	}
	defer stdin.Close()

	if err := sess.Start(); err != nil {
		return err
	}

	switch n, err := io.WriteString(stdin, s.Script); {
	case err != nil:
		return errors.Wrap(err, "failed to send script to target")
	case n != len(s.Script):
		return errors.Errorf("expected to send %d bytes, sent %d", len(s.Script), n)
	default:
		stdin.Close()
		return sess.Wait()
	}
}
