package connection

import (
	"bufio"
	"fmt"
	"io"
	"sync"

	"github.com/gfrey/smutje/logger"
)

type loggedSession struct {
	Session

	wg *sync.WaitGroup
}

func newLoggedSession(l logger.Logger, sess Session) (Session, error) {
	s := &loggedSession{Session: sess, wg: new(sync.WaitGroup)}

	stdout, err := s.Session.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := s.Session.StderrPipe()
	if err != nil {
		return nil, err
	}

	s.wg.Add(2)
	go s.readStream(l.Tag("stdout"), stdout)
	go s.readStream(l.Tag("stderr"), stderr)

	return s, nil
}

func (lsess *loggedSession) Close() error {
	err := lsess.Session.Close()
	lsess.wg.Wait()
	return err
}

func (lsess *loggedSession) StdoutPipe() (io.Reader, error) {
	return nil, fmt.Errorf("logged session has no access to stdout pipe!")
}

func (lsess *loggedSession) StderrPipe() (io.Reader, error) {
	return nil, fmt.Errorf("logged session has no access to stderr pipe!")
}

func (lsess *loggedSession) readStream(l logger.Logger, stream io.Reader) {
	defer lsess.wg.Done()

	sc := bufio.NewScanner(stream)
	for sc.Scan() {
		l.Printf(sc.Text())
	}

	if err := sc.Err(); err != nil {
		l.Printf("failed scanning stderr: %s", err)
	}
}
