package smutje

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gfrey/gconn"
	"github.com/pkg/errors"
)

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

func (a *execJenkinsArtifactCmd) Prepare(attrs Attributes, prevHash string) (string, error) {
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

func (a *execJenkinsArtifactCmd) Exec(l *log.Logger, client gconn.Client) error {
	l.Printf("downloading file %q from %q", a.Target, a.url)
	rawCmd := "{ dir=$(dirname %[1]s); test -d ${dir} || mkdir -p ${dir}; } && curl -sSL %[2]s -o %[1]s"
	// TODO is possible to set only one of the both?
	if a.Owner != "" && a.Umask != "" {
		rawCmd += " && chown " + a.Owner + " %[1]s && chmod " + a.Umask + " %[1]s"
	}

	cmd := fmt.Sprintf("'"+rawCmd+"'", a.Target, a.url)
	sess, err := gconn.NewLoggedClient(l, client).NewSession("/usr/bin/env", "bash", "-c", cmd)
	if err != nil {
		return err
	}
	defer sess.Close()

	if err := sess.Start(); err != nil {
		return err
	}

	return sess.Wait()
}

func (*execJenkinsArtifactCmd) MustExecute() bool {
	return false
}
