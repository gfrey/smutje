package smutje

import (
	"testing"

	"io"

	"bytes"

	"github.com/gfrey/smutje/connection"
	"github.com/gfrey/smutje/logger"
	"github.com/pkg/errors"
	"strings"
)

const hA = "4a24a6308d8d208844077a8cf89982f3"
const hB = "2f06ae88c46786997b889c10d9f18695"
const hC = "6c3e28ab2914f4bcff2453ca6162f2af"

const hAE, hAC, hAF = "+"+hA, "."+hA, "-"+hA
const hBE, hBC, hBF = "+"+hB, "."+hB, "-"+hB
const hCE, hCC, hCF = "+"+hC, "."+hC, "-"+hC


func TestProvision(t *testing.T) {
	l := logger.NewDiscard()

	tt := []struct {
		curState []string
		failIdx  int
		expState []string
	}{
		{nil, -1, []string{hAE, hBE, hCE}},
		{[]string{hAE}, -1, []string{hAC, hBE, hCE}},
		{[]string{hAF}, -1, []string{hAE, hBE, hCE}},
		{[]string{hAE, hBE}, -1, []string{hAC, hBC, hCE}},
		{[]string{hAE, hBF}, -1, []string{hAC, hBE, hCE}},
		{[]string{hAE, hBE, hCE}, -1, []string{hAE, hBE, hCE}},
		{[]string{hAE, hBE, hCF}, -1, []string{hAC, hBC, hCE}},

		{[]string{hAC, hBC, hCC}, -1, []string{hAC, hBC, hCC}},

		{nil, 0, []string{hAF}},
		{nil, 1, []string{hAE, hBF}},
		{nil, 2, []string{hAE, hBE, hCF}},

		{[]string{"+a", "+b", "+c"}, 0, []string{hAF}},
		{[]string{"+a", "+b", "+c"}, 1, []string{hAE, hBF}},
		{[]string{"+a", "+b", "+c"}, 2, []string{hAE, hBE, hCF}},

		// consider that cached elements won't result in execution (that is why the fail idx doesn't change
		{[]string{hAE, "+b", "+c"}, 0, []string{hAC, hBF}},
		{[]string{hAC, hBE,  "+c"}, 0, []string{hAC, hBC, hCF}},
	}

	for i, tti := range tt {
		client := new(testClient)
		client.failIdx = -1
		if tti.curState != nil {
			client.expCommand = "cat /var/lib/smutje/foobar.log"
			client.cmdOutput = strings.Join(tti.curState, "\n")
		}

		pkg := new(smPackage)
		pkg.ID = "foobar"
		pkg.Scripts = []smScript{
			&bashScript{Script: "echo foo"},
			&smutjeScript{rawCommand: ":write_file testdata/a b"},
			&bashScript{Script: "echo bar"},
		}

		if err := pkg.Prepare(client, smAttributes{}); err != nil {
			t.Fatalf("didn't expect an error, got: %s", err)
		}

		client.curIdx = 0
		client.failIdx = tti.failIdx
		client.expCommand = ""

		err := pkg.Provision(l, client)
		if tti.failIdx == -1 && err != nil {
			t.Errorf("%d: didn't expect an error, got: %s", i, err)
			continue
		} else if tti.failIdx != -1 && err == nil {
			t.Errorf("%d: expected an error, got none", i)
			continue
		}

		newState := pkg.state

		if len(newState) != len(tti.expState) {
			t.Errorf("%d: expected %d elements in new state, got %d", i, len(tti.expState), len(newState))
			continue
		}

		for j, expState := range tti.expState {
			if expState != newState[j] {
				t.Errorf("%d: expected state %d to be %q, got %q", i, j, expState, newState[j])
			}
		}
	}
}

type testClient struct {
	failIdx    int
	curIdx     int

	expCommand string
	cmdOutput  string
}

func (tc *testClient) Name() string {
	return "testClient"
}

func (tc *testClient) NewSession() (connection.Session, error) {
	s := new(testSession)
	if tc.curIdx == tc.failIdx {
		s.fail = true
	}
	tc.curIdx++
	s.cmdOutput = tc.cmdOutput
	s.expCommand = tc.expCommand
	return s, nil
}

func (tc *testClient) NewLoggedSession(l logger.Logger) (connection.Session, error) {
	return tc.NewSession()
}

func (tc *testClient) Close() error {
	return nil
}

type testSession struct {
	Stdin      *bytes.Buffer
	Stdout     *bytes.Buffer
	Stderr     *bytes.Buffer

	fail       bool

	expCommand string
	cmdOutput  string
}

func (ts *testSession) Close() error {
	return nil
}

func (ts *testSession) StdinPipe() (io.WriteCloser, error) {
	ts.Stdin = bytes.NewBuffer(nil)
	return &nopCloser{ts.Stdin}, nil
}

func (ts *testSession) StdoutPipe() (io.Reader, error) {
	ts.Stdout = bytes.NewBuffer(nil)
	return ts.Stdout, nil
}
func (ts *testSession) StderrPipe() (io.Reader, error) {
	ts.Stderr = bytes.NewBuffer(nil)
	return ts.Stderr, nil
}

func (ts *testSession) Run(cmd string) error {
	return ts.Wait()
}

func (ts *testSession) Start(cmd string) error {
	if ts.expCommand != "" {
		if strings.Contains(cmd, ts.expCommand) {
			ts.Stdout.WriteString(ts.cmdOutput)
		} else {
			return errors.Errorf("expected %q to contain %q, it didn't", cmd, ts.expCommand)
		}
	}
	return nil
}

func (ts *testSession) Wait() error {
	if ts.fail {
		return errors.Errorf("asked to fail")
	}
	return nil
}

type nopCloser struct {
	w io.Writer
}

func (n *nopCloser) Write(d []byte) (int, error) {
	return n.w.Write(d)
}

func (*nopCloser) Close() error {
	return nil
}
