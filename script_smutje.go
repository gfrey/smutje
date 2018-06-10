package smutje

import (
	"log"
	"strings"

	"github.com/gfrey/gconn"
	"github.com/pkg/errors"
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

func (s *smutjeScript) Prepare(attrs Attributes, prevHash string) (string, error) {
	if err := s.initCommands(attrs); err != nil {
		return "", err
	}

	return s.Command.Prepare(attrs, prevHash)
}

func (s *smutjeScript) Exec(l *log.Logger, client gconn.Client) error {
	return s.Command.Exec(l, client)
}

func (s *smutjeScript) MustExecute() bool {
	return s.Command.MustExecute()
}

func (s *smutjeScript) initCommands(attrs Attributes) error {
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
