package smutje

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gfrey/smutje/connection"
	"github.com/gfrey/smutje/logger"
)

type smutjeScript struct {
	ID         string
	Path       string
	rawCommand string
	Command    smScript
}

func (s *smutjeScript) Hash() string {
	return s.Command.Hash()
}

func (s *smutjeScript) Prepare(attrs smAttributes, prevHash string) (string, error) {
	if err := s.initCommands(attrs); err != nil {
		return "", err
	}

	return s.Command.Prepare(attrs, prevHash)
}

func (s *smutjeScript) Exec(l logger.Logger, client connection.Client) error {
	return s.Command.Exec(l, client)
}

func (s *smutjeScript) initCommands(attrs smAttributes) error {
	raw, err := renderString(s.ID, s.rawCommand, attrs)
	if err != nil {
		return err
	}

	args := strings.Fields(raw)
	if len(args) == 0 {
		return fmt.Errorf("empty command received")
	}
	switch strings.ToLower(args[0]) {
	case ":write_file":
		s.Command, err = newExecWriteFileCmd(s.Path, args[1:])
	case ":write_template":
		s.Command, err = newExecWriteTemplateCmd(s.Path, args[1:])
	default:
		return fmt.Errorf("command %s unknown", args[0])
	}
	return err
}

type execWriteFileCmd struct {
	Source string
	Target string
	Owner  string
	Umask  string

	Render bool

	attrs smAttributes
	hash  string
	size  int64
}

func newExecWriteFileCmd(path string, args []string) (*execWriteFileCmd, error) {
	if len(args) < 2 || len(args) == 3 || len(args) > 4 {
		return nil, fmt.Errorf(`syntax error: write file/template usage ":write_file <source> <target> [<user> <umask>]?"`)
	}

	filename := filepath.Join(path, args[0])
	if _, err := os.Stat(filename); err != nil {
		return nil, err
	}

	cmd := &execWriteFileCmd{Source: filename, Target: args[1], Render: false}

	if len(args) > 2 {
		cmd.Owner = args[2]
		cmd.Umask = args[3]
	}

	return cmd, nil
}

func newExecWriteTemplateCmd(path string, args []string) (*execWriteFileCmd, error) {
	cmd, err := newExecWriteFileCmd(path, args)
	if err != nil {
		return nil, err
	}
	cmd.Render = true

	return cmd, nil
}

func (a *execWriteFileCmd) Hash() string {
	return a.hash
}

func (a *execWriteFileCmd) Prepare(attrs smAttributes, prevHash string) (string, error) {
	a.attrs = attrs
	r, err := a.read()
	if err != nil {
		return "", err
	}
	defer r.Close()

	hash := md5.New()
	if _, err := hash.Write([]byte(prevHash + a.Target + a.Owner + a.Umask)); err != nil {
		return "", err
	}
	size, err := io.Copy(hash, r)
	if err != nil {
		return "", err
	}

	a.size = size
	a.hash = fmt.Sprintf("%x", hash.Sum(nil))
	return a.hash, nil
}

func (a *execWriteFileCmd) read() (io.ReadCloser, error) {
	if a.Render {
		return renderFile(a.Source, a.attrs)
	}
	return os.Open(a.Source)
}

func (a *execWriteFileCmd) Exec(l logger.Logger, clients connection.Client) error {
	r, err := a.read()
	if err != nil {
		return err
	}
	defer r.Close()

	sess, err := clients.NewLoggedSession(l)
	if err != nil {
		return err
	}
	defer sess.Close()

	stdin, err := sess.StdinPipe()
	if err != nil {
		return err
	}

	l.Printf("writing file %q", a.Target)
	setFilePerms := ""
	// TODO is possible to set only one of the both?
	if a.Owner != "" && a.Umask != "" {
		setFilePerms = " && chown " + a.Owner + " %[1]s && chmod " + a.Umask + " %[1]s"
	}

	cmd := fmt.Sprintf(`bash -c "{ dir=$(dirname %[1]s); test -d \${dir} || mkdir -p \${dir}; } && cat - > %[1]s`+setFilePerms+`"`, a.Target)
	if err := sess.Start(cmd); err != nil {
		return err
	}

	if _, err := io.Copy(stdin, r); err != nil {
		return err
	}
	stdin.Close()

	// TODO validate all bytes written
	// TODO use compression on the wire

	return sess.Wait()
}
