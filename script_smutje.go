package smutje

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gfrey/smutje/connection"
	"github.com/gfrey/smutje/logger"
	"github.com/pkg/errors"
	"bufio"
	"strconv"
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

func (s *smutjeScript) MustExecute() bool {
	return s.Command.MustExecute()
}

func (s *smutjeScript) initCommands(attrs smAttributes) error {
	raw, err := renderString(s.ID, s.rawCommand, attrs)
	if err != nil {
		return err
	}

	args := strings.Fields(raw)
	if len(args) == 0 {
		return errors.Errorf("empty command received")
	}
	switch strings.ToLower(args[0]) {
	case ":write_file":
		s.Command, err = newExecWriteFileCmd(s.Path, args[1:])
	case ":write_template":
		s.Command, err = newExecWriteTemplateCmd(s.Path, args[1:])
	case ":jenkins_artifact":
		s.Command, err = newJenkinsArtifactCmd(args[1:])
	case ":inject_passwords":
		s.Command, err = newInjectPasswordsCmd(args[1:])
	default:
		return errors.Errorf("command %s unknown", args[0])
	}
	return err
}

type execInjectPasswordsCmd struct {
	Passwords []string

	values map[string]string
	hash string

	cache map[string]string
}

func newInjectPasswordsCmd(args []string) (*execInjectPasswordsCmd, error) {
	if len(args) == 0 {
		return nil, errors.Errorf(`syntax error: password injector usage ":inject_password [<password_name>]+"`)
	}

	cmd := new(execInjectPasswordsCmd)
	cmd.Passwords = args
	cmd.values = make(map[string]string, len(args))
	return cmd, nil
}

func (a *execInjectPasswordsCmd) Hash() string {
	return a.hash
}

func (a *execInjectPasswordsCmd) Prepare(attrs smAttributes, prevHash string) (string, error) {
	hash := md5.New()
	if _, err := hash.Write([]byte(prevHash)); err != nil {
		return "", errors.Wrap(err, "failed to write hash")
	}

	for _, pwdName := range a.Passwords {
		pwd, err := a.getPassword(pwdName)
		if err != nil {
			return "", err
		}
		a.values[pwdName] = pwd
		if _, err := hash.Write([]byte(pwd)); err != nil {
			return "", errors.Wrap(err, "failed to write hash")
		}

		attrs["PASSWORD_"+pwdName] = fmt.Sprintf(`$(grep %s /tmp/smutje/passwords | cut -f2)`, pwdName)
		attrs["PASSWORD_"+pwdName+"_RAW"] = pwd
		attrs["PASSWORD_"+pwdName+"_QUOTED"] = strconv.Quote(pwd)
	}


	a.hash = fmt.Sprintf("%x", hash.Sum(nil))
	return a.hash, nil
}

func (a *execInjectPasswordsCmd) Exec(l logger.Logger, clients connection.Client) error {
	sess, err := clients.NewLoggedSession(l)
	if err != nil {
		return err
	}
	defer sess.Close()

	l.Printf("injecting passwords")

	stdin, err := sess.StdinPipe()
	if err != nil {
		return errors.Wrap(err, "failed to receive stdin pipe")
	}

	cmd := fmt.Sprintf(`/usr/bin/env bash -c "cat - > /tmp/smutje/passwords"`)
	if err := sess.Start(cmd); err != nil {
		return err
	}

	for k, v := range a.values {
		line := fmt.Sprintf("%s\t%s\n", k, v)
		_, err := stdin.Write([]byte(line))
		if err != nil {
			stdin.Close()
			return errors.Wrap(err, "failed to write password to target")
		}
	}
	stdin.Close()

	return sess.Wait()
}

func (a *execInjectPasswordsCmd) MustExecute() bool {
	return true
}

func (a *execInjectPasswordsCmd) getPassword(name string) (string, error) {
	if a.cache == nil {
		cache := map[string]string{}

		fh, err := os.Open(".passwords")
		if err != nil {
			return "", errors.Wrap(err, "failed to read passwords")
		}

		sc := bufio.NewScanner(fh)
		for i := 0; sc.Scan(); i++ {
			line := sc.Text()
			if strings.TrimSpace(line) == "" {
				continue
			}
			parts := strings.Split(line, ":")
			if len(parts) != 2 {
				return "", errors.Errorf("invalid syntax in passwords line %d: expected 2 parts, got %d", i, len(parts))
			}
			name, pwd := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
			cache[name] = pwd
		}

		if err := sc.Err(); err != nil {
			return "", errors.Wrap(err, "failed to parse passwords file")
		}

		a.cache = cache
	}
	pwd, found := a.cache[name]
	if ! found {
		return "", errors.Errorf("password %q not found", name)
	}
	return pwd, nil
}

type execJenkinsArtifactCmd struct {
	Host     string
	Job      string
	Artifact string
	Target   string
	Owner    string
	Umask    string

	hash string
	url  string
}

func newJenkinsArtifactCmd(args []string) (*execJenkinsArtifactCmd, error) {
	if len(args) < 4 || len(args) > 6 {
		return nil, errors.Errorf(`syntax error: jenkins artifact usage ":jenkins_artifact <host> <job> <artifact> <target> [<user> <umask>]?"`)
	}

	cmd := new(execJenkinsArtifactCmd)
	cmd.Host, cmd.Job, cmd.Artifact, cmd.Target = args[0], args[1], args[2], args[3]
	cmd.Owner, cmd.Umask = "root", "0644"
	if len(args) > 4 {
		cmd.Owner = args[4]
	}
	if len(args) == 6 {
		cmd.Umask = args[5]
	}

	return cmd, nil
}

func (a *execJenkinsArtifactCmd) Hash() string {
	return a.hash
}

func (a *execJenkinsArtifactCmd) Prepare(attrs smAttributes, prevHash string) (string, error) {
	a.url = fmt.Sprintf("http://%s/job/%s/lastSuccessfulBuild/artifact/%s", a.Host, a.Job, a.Artifact)
	resp, err := http.Get(a.url + "/*fingerprint*/")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return "", err
	}

	parts := strings.Split(buf.String(), "MD5: ")
	if len(parts) != 2 {
		return "", errors.New("failed to read artifact fingerprint")
	}
	fingerprint := strings.SplitN(parts[1], " ", 2)[0]

	hash := md5.New()
	if _, err := hash.Write([]byte(prevHash + a.Host + a.Job + a.Artifact + fingerprint)); err != nil {
		return "", errors.Wrap(err, "failed to create command hash")
	}
	a.hash = fmt.Sprintf("%x", hash.Sum(nil))

	return a.hash, nil
}

func (a *execJenkinsArtifactCmd) Exec(l logger.Logger, client connection.Client) error {
	sess, err := client.NewLoggedSession(l)
	if err != nil {
		return err
	}
	defer sess.Close()

	l.Printf("downloading file %q from %q", a.Target, a.url)
	setFilePerms := ""
	// TODO is possible to set only one of the both?
	if a.Owner != "" && a.Umask != "" {
		setFilePerms = " && chown " + a.Owner + " %[1]s && chmod " + a.Umask + " %[1]s"
	}

	cmd := fmt.Sprintf(`bash -c "{ dir=$(dirname %[1]s); test -d \${dir} || mkdir -p \${dir}; } && curl -sSL %[2]s -o %[1]s`+setFilePerms+`"`, a.Target, a.url)
	if err := sess.Start(cmd); err != nil {
		return err
	}

	return sess.Wait()
}

func (*execJenkinsArtifactCmd) MustExecute() bool {
	return false
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
		return errors.Wrap(err, "failed to receive stdin pipe")
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
