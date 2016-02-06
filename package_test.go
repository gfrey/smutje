package smutje

import (
	"fmt"
	"testing"

	"io"

	"bytes"

	"github.com/gfrey/smutje/connection"
	"github.com/gfrey/smutje/logger"
)

func TestProvision(t *testing.T) {
	pkg := new(smPackage)
	pkg.Scripts = append(pkg.Scripts, &bashScript{Script: "echo foo"})
	pkg.Scripts = append(pkg.Scripts, &smutjeScript{rawCommand: ":write_file testdata/a b"})
	pkg.Scripts = append(pkg.Scripts, &bashScript{Script: "echo bar"})

	l := logger.NewDiscard()

	if err := pkg.Prepare(nil, smAttributes{}); err != nil {
		t.Fatalf("didn't expect an error, got: %s", err)
	}

	hA := "4a24a6308d8d208844077a8cf89982f3"
	hB := "2f06ae88c46786997b889c10d9f18695"
	hC := "6c3e28ab2914f4bcff2453ca6162f2af"

	hAE, hAC, hAF := "+"+hA, "."+hA, "-"+hA
	hBE, hBC, hBF := "+"+hB, "."+hB, "-"+hB
	hCE, hCC, hCF := "+"+hC, "."+hC, "-"+hC

	tt := []struct {
		curState []string
		failIdx  int
		expState []string
	}{
		{nil, -1, []string{hAE, hBE, hCE}},
		{[]string{hAE}, -1, []string{hAC, hBE, hCE}},
		{[]string{hAE, hBE}, -1, []string{hAC, hBC, hCE}},
		{[]string{hAC, hBC, hCC}, -1, []string{hAC, hBC, hCC}},

		{nil, 0, []string{hAF}},
		{nil, 1, []string{hAE, hBF}},
		{nil, 2, []string{hAE, hBE, hCF}},

		{[]string{"+a", "+b", "+c"}, 0, []string{hAF}},
		{[]string{"+a", "+b", "+c"}, 1, []string{hAE, hBF}},
		{[]string{"+a", "+b", "+c"}, 2, []string{hAE, hBE, hCF}},
	}

	for i, tti := range tt {
		client := new(testClient)
		client.failIdx = tti.failIdx

		pkg.state = tti.curState

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
	failIdx int
	curIdx  int
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
	return s, nil
}

func (tc *testClient) NewLoggedSession(l logger.Logger) (connection.Session, error) {
	return tc.NewSession()
}

func (tc *testClient) Close() error {
	return nil
}

type testSession struct {
	Stdin  *bytes.Buffer
	Stdout *bytes.Buffer
	Stderr *bytes.Buffer

	fail bool
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
	return nil
}

func (ts *testSession) Wait() error {
	if ts.fail {
		return fmt.Errorf("asked to fail")
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
