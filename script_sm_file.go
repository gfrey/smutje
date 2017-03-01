package smutje

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gfrey/gconn"
	"github.com/gfrey/glog"
	"github.com/pkg/errors"
)

type execWriteFileCmd struct {
	Source string
	Target string
	Owner  string
	Umask  string

	Render bool

	attrs Attributes
	hash  string
	size  int64
}

func newExecWriteFileCmd(path string, args []string) (*execWriteFileCmd, error) {
	if len(args) < 2 || len(args) == 3 || len(args) > 4 {
		return nil, errors.Errorf(`syntax error: write file/template usage ":write_file <source> <target> [<user> <umask>]?"`)
	}

	filename := args[0]
	if filename[0] != '/' {
		filename = filepath.Join(path, args[0])
		if _, err := os.Stat(filename); err != nil {
			return nil, err
		}
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

func (a *execWriteFileCmd) Prepare(attrs Attributes, prevHash string) (string, error) {
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

func (a *execWriteFileCmd) Exec(l glog.Logger, clients gconn.Client) error {
	r, err := a.read()
	if err != nil {
		return err
	}
	defer r.Close()

	l.Printf("writing file %q", a.Target)
	setFilePerms := ""
	// TODO is possible to set only one of the both?
	if a.Owner != "" && a.Umask != "" {
		setFilePerms = " && chown " + a.Owner + " %[1]s && chmod " + a.Umask + " %[1]s"
	}

	cmd := fmt.Sprintf("{ dir=$(dirname %[1]s); test -d ${dir} || mkdir -p ${dir}; } && cat - > %[1]s%[2]s", a.Target, setFilePerms)
	sess, err := gconn.NewLoggedClient(l, clients).NewSession("/usr/bin/env", "bash", "-c", cmd)
	if err != nil {
		return err
	}
	defer sess.Close()

	stdin, err := sess.StdinPipe()
	if err != nil {
		return errors.Wrap(err, "failed to receive stdin pipe")
	}

	if err := sess.Start(); err != nil {
		return err
	}

	if _, err := io.Copy(stdin, r); err != nil {
		return errors.Wrap(err, "failed to send script to target")
	}
	stdin.Close()

	// TODO validate all bytes written
	// TODO use compression on the wire

	return sess.Wait()
}

func (*execWriteFileCmd) MustExecute() bool {
	return false
}
