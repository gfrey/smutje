package smutje

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/gfrey/smutje/connection"
	"github.com/gfrey/smutje/logger"
	"github.com/gfrey/smutje/parser"
	"github.com/pkg/errors"
)

type smPackage struct {
	Name string
	ID   string

	Attributes smAttributes
	Scripts    []smScript

	state   []string
	isDirty bool
}

func newPackage(parentID, path string, attrs smAttributes, n *parser.AstNode) (*smPackage, error) {
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

func (pkg *smPackage) Prepare(client connection.Client, attrs smAttributes) (err error) {
	if client != nil { // If a virtual resource doesn't exist yet, the client is nil!
		pkg.state, err = pkg.readPackageState(client)
		if err != nil {
			return errors.Errorf("failed to read target state: %s", err)
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

func (pkg *smPackage) Provision(l logger.Logger, client connection.Client) (err error) {
	l = l.Tag(pkg.ID)

	oldState := pkg.state
	pkg.state = make([]string, len(pkg.Scripts))

	allCached := true

	defer func() {
		if allCached {
			return
		}

		e := pkg.writeTargetState(client)
		if err == nil {
			err = e
		}
	}()

	for i, s := range pkg.Scripts {
		hash := s.Hash()

		if allCached && i < len(oldState) && oldState[i][1:] == hash {
			l.Printf("step %d cached", i)
			pkg.state[i] = "." + hash
			continue
		}

		allCached = false
		if err = s.Exec(l, client); err != nil {
			l.Printf("failed in %s", hash)
			pkg.state[i] = "-" + hash
			pkg.state = pkg.state[:i+1]
			return err
		}
		pkg.state[i] = "+" + hash
	}
	return nil
}

func (pkg *smPackage) readPackageState(client connection.Client) ([]string, error) {
	sess, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer sess.Close()

	stdout, err := sess.StdoutPipe()
	if err != nil {
		return nil, err
	}

	fname := fmt.Sprintf("/var/lib/smutje/%s.log", pkg.ID)
	cmd := fmt.Sprintf(`bash -c "if [[ -f %[1]q ]]; then cat %[1]s; else mkdir -p /var/lib/smutje; fi"`, fname)
	if err := sess.Start(cmd); err != nil {
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

func (pkg *smPackage) writeTargetState(client connection.Client) error {
	sess, err := client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	stdin, err := sess.StdinPipe()
	if err != nil {
		return errors.Wrap(err, "failed to receive stdin pipe")
	}

	tstamp := time.Now().UTC().Format("20060102T150405")
	filename := fmt.Sprintf("/var/lib/smutje/%s.%s.log", pkg.ID, tstamp)
	cmd := fmt.Sprintf(`bash -c "cat - > %[1]s && ln -sf %[1]s /var/lib/smutje/%[2]s.log"`, filename, pkg.ID)
	if err := sess.Start(cmd); err != nil {
		return err
	}

	if _, err := io.WriteString(stdin, strings.Join(pkg.state, "\n")+"\n"); err != nil {
		return errors.Wrap(err, "failed to send script to target")
	}
	stdin.Close()
	return sess.Wait()
}
