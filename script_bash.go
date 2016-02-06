package smutje

import (
	"crypto/md5"
	"fmt"
	"io"

	"github.com/gfrey/smutje/connection"
	"github.com/gfrey/smutje/logger"
)

type bashScript struct {
	ID     string
	Script string
	hash   string
}

func (s *bashScript) Hash() string {
	return s.hash
}

func (s *bashScript) Prepare(attrs smAttributes, prevHash string) (string, error) {
	script, err := renderString(s.ID, "set -ex\n"+s.Script+"\n", attrs)
	if err != nil {
		return "", err
	}
	s.Script = script
	s.hash = fmt.Sprintf("%x", md5.Sum([]byte(prevHash+s.Script)))
	return s.hash, nil
}

func (s *bashScript) Exec(l logger.Logger, client connection.Client) error {
	sess, err := client.NewLoggedSession(l)
	if err != nil {
		return err
	}
	defer sess.Close()

	stdin, err := sess.StdinPipe()
	if err != nil {
		return err
	}
	defer stdin.Close()

	fname := fmt.Sprintf("/var/lib/smutje/%s.sh", s.hash)
	cmd := fmt.Sprintf(`bash -c "cat - > %[1]s && bash -l %[1]s"`, fname)
	if err := sess.Start(cmd); err != nil {
		return err
	}

	switch n, err := io.WriteString(stdin, s.Script); {
	case err != nil:
		return err
	case n != len(s.Script):
		return fmt.Errorf("expected to send %d bytes, sent %d", len(s.Script), n)
	default:
		stdin.Close()
		return sess.Wait()
	}
}
