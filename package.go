package smutje

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gfrey/gconn"
	"github.com/gfrey/smutje/parser"
	"github.com/pkg/errors"
)

type smPackage struct {
	Name string
	ID   string

	Attributes Attributes
	Scripts    []smScript

	state   []string
	isDirty bool
}

func newPackage(parentID, path string, attrs Attributes, n *parser.AstNode) (*smPackage, error) {
	if n.Type != parser.AstPackage {
		return nil, fmt.Errorf("expected package node, got %s", n.Type)
	}

	pkg := new(smPackage)
	pkg.Name = n.Name

	pkg.ID = n.ID
	if parentID != "" {
		pkg.ID = parentID + "." + n.ID
	}

	pkg.Attributes = attrs.Copy()
	for _, child := range n.Children {
		switch child.Type {
		case parser.AstAttributes:
			attrs, err := newAttributes(child)
			if err != nil {
				return nil, err
			}
			pkg.Attributes, err = attrs.Merge(pkg.Attributes)
			if err != nil {
				return nil, err
			}
		case parser.AstScript:
			child.ID = pkg.ID + "_" + strconv.Itoa(len(pkg.Scripts))
			script, err := newScript(path, child)
			if err != nil {
				return nil, err
			}
			pkg.Scripts = append(pkg.Scripts, script)
		case parser.AstText:
		// ignore
		default:
			return nil, errors.Errorf("unexpected node found: %s", n.Type)
		}
	}

	return pkg, nil
}

func (pkg *smPackage) Prepare(client gconn.Client, attrs Attributes) (err error) {
	if client != nil { // If a virtual resource doesn't exist yet, the client is nil!
		pkg.state, err = pkg.readPackageState(client)
		if err != nil {
			return err
		}
	}

	hash := ""
	sattrs, err := attrs.Merge(pkg.Attributes)
	if err != nil {
		return err
	}
	for i, s := range pkg.Scripts {
		hash, err = s.Prepare(sattrs, hash)
		if err != nil {
			return err
		}
		if i >= len(pkg.state) || hash != pkg.state[i] {
			pkg.isDirty = true
		}
	}

	return nil
}

func (pkg *smPackage) firstToExec() int {
	firstToExec := -1
	for i, s := range pkg.Scripts {
		if firstToExec == -1 && s.MustExecute() {
			firstToExec = i
		}

		hash := s.Hash()
		if !(i < len(pkg.state) && pkg.state[i][1:] == hash) {
			if firstToExec == -1 {
				firstToExec = i
			}
			return firstToExec
		}
	}
	return -1 // all hashes valid, so nothing to do
}

func (pkg *smPackage) Provision(l *log.Logger, client gconn.Client) (err error) {
	l = tagLogger(l, pkg.ID)

	firstToExec := pkg.firstToExec()
	if firstToExec == -1 {
		l.Printf("all steps cached")
		return
	}

	defer func() {
		e := pkg.writeTargetState(client)
		if err == nil {
			err = e
		}
	}()

	pkg.state = make([]string, len(pkg.Scripts))
	for i, s := range pkg.Scripts {
		hash := s.Hash()
		if i < firstToExec {
			l.Printf("step %d cached", i)
			pkg.state[i] = "." + hash
			continue
		}

		if err = s.Exec(l, client); err != nil {
			l.Printf("failed in %s", hash)
			pkg.state[i] = "-" + hash
			pkg.state = pkg.state[:i+1]
			return err
		}
		l.Printf("executed %s", hash)
		pkg.state[i] = "+" + hash
	}
	return nil
}

func (pkg *smPackage) readPackageState(client gconn.Client) ([]string, error) {
	fname := fmt.Sprintf("/var/lib/smutje/%s.log", pkg.ID)
	cmd := fmt.Sprintf(`if [[ -f '%[1]s' ]]; then cat %[1]s; else mkdir -p /var/lib/smutje; fi`, fname)

	sess, err := client.NewSession("/usr/bin/env", "bash", "-c", fmt.Sprintf("%q", cmd))
	if err != nil {
		return nil, err
	}
	defer sess.Close()

	stdout, err := sess.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := sess.Start(); err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, stdout); err != nil {
		return nil, errors.Wrap(err, "failed to copy output of command")
	}

	if err := sess.Wait(); err != nil {
		return nil, err
	}

	state := []string{}
	sc := bufio.NewScanner(buf)
	for sc.Scan() {
		l := sc.Text()
		switch l[0] {
		case '+', '.':
			state = append(state, l)
		case '-':
			//ignore
		default:
			return nil, errors.Errorf("invalid token read: %s", l)
		}
	}
	return state, errors.Wrap(sc.Err(), "failed to scan output")
}

func (pkg *smPackage) writeTargetState(client gconn.Client) error {
	tstamp := time.Now().UTC().Format("20060102T150405")
	filename := fmt.Sprintf("/var/lib/smutje/%s.%s.log", pkg.ID, tstamp)
	cmd := fmt.Sprintf(`rm -Rf /tmp/smutje/*; cat - > %[1]s && ln -sf %[1]s /var/lib/smutje/%[2]s.log`, filename, pkg.ID)

	sess, err := client.NewSession("/usr/bin/env", "bash", "-c", fmt.Sprintf("%q", cmd))
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

	if _, err := io.WriteString(stdin, strings.Join(pkg.state, "\n")+"\n"); err != nil {
		return errors.Wrap(err, "failed to send script to target")
	}
	stdin.Close()
	return sess.Wait()
}
