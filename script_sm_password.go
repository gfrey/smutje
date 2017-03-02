package smutje

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gfrey/gconn"
	"github.com/gfrey/glog"
	"github.com/pkg/errors"
)

type execInjectPasswordsCmd struct {
	Passwords []string

	values map[string]string
	hash   string

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

func (a *execInjectPasswordsCmd) Prepare(attrs Attributes, prevHash string) (string, error) {
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

func (a *execInjectPasswordsCmd) Exec(l glog.Logger, clients gconn.Client) error {
	sess, err := gconn.NewLoggedClient(l, clients).NewSession("/usr/bin/env", "bash", "-c", `"cat - >> /tmp/smutje/passwords"`)
	if err != nil {
		return err
	}
	defer sess.Close()

	l.Printf("injecting passwords")

	stdin, err := sess.StdinPipe()
	if err != nil {
		return errors.Wrap(err, "failed to receive stdin pipe")
	}

	if err := sess.Start(); err != nil {
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
	if !found {
		return "", errors.Errorf("password %q not found", name)
	}
	return pwd, nil
}
