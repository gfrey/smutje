package hypervisor

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"strings"
	"sync"

	"github.com/gfrey/smutje/connection"
	"github.com/gfrey/smutje/logger"
	"github.com/pkg/errors"
)

type smartOS struct {
	addr   string
	user   string
	client connection.Client
}

func NewSmartOSHypervisor(addr string) (Client, error) {
	var err error
	hp := new(smartOS)
	hp.addr = addr
	hp.user = "root"
	hp.client, err = connection.NewSSHClient(hp.addr, hp.user)
	return hp, err
}

func (hp *smartOS) ConnectVRes(uuid string) (connection.Client, error) {
	sshClient, err := connection.NewSSHClient(hp.addr, hp.user)
	if err != nil {
		return nil, err
	}
	return connection.NewWrappedClient(sshClient, "zlogin "+uuid), nil
}

func (hp *smartOS) UUID(alias string) (string, error) {
	// determine whether the VM in question already exists
	sess, err := hp.client.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()

	stdout, err := sess.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err := sess.Start("vmadm list -p"); err != nil {
		return "", err
	}

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, stdout); err != nil {
		return "", err
	}

	if err := sess.Wait(); err != nil {
		return "", err
	}

	sc := bufio.NewScanner(buf)
	for sc.Scan() {
		fields := strings.Split(sc.Text(), ":")
		if fields[4] == alias {
			return fields[0], nil
		}
	}

	if err := sc.Err(); err != nil {
		return "", errors.Wrap(err, "failed to scan output")
	}

	return "", nil
}

func (hp *smartOS) Create(l logger.Logger, blueprint string) (string, error) {
	m := map[string]interface{}{}
	if err := json.Unmarshal([]byte(blueprint), &m); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal the blueprint")
	}

	l.Printf("updating the image database")
	if err := runCommand(hp.client, "imgadm update"); err != nil {
		return "", err
	}

	imgUUID := m["image_uuid"].(string)
	l.Printf("importing image %s", imgUUID)
	if err := runCommand(hp.client, "imgadm import -q "+imgUUID); err != nil {
		return "", err
	}

	// determine whether the VM in question already exists
	sess, err := hp.client.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()

	wg := new(sync.WaitGroup)
	wg.Add(2)

	outBuf := bytes.NewBuffer(nil)
	stderr, err := sess.StderrPipe()
	if err != nil {
		return "", errors.Wrap(err, "failed to retrieve stderr pipe")
	}
	go func() {
		defer wg.Done()
		_, _ = io.Copy(outBuf, stderr)
	}()

	stdin, err := sess.StdinPipe()
	if err != nil {
		return "", errors.Wrap(err, "failed to retrieve stdin pipe")
	}
	go func() {
		defer wg.Done()
		_, _ = io.WriteString(stdin, blueprint)
		stdin.Close()
	}()

	l.Printf("creating the virtual resource")
	if err := sess.Run("vmadm create"); err != nil {
		log.Printf("failed: %s", outBuf.String())
		return "", err
	}

	wg.Wait()

	output := strings.TrimSpace(outBuf.String())
	expResponsePrefix := "Successfully created VM "
	if !strings.HasPrefix(output, expResponsePrefix) {
		return "", errors.Errorf("wrong response received: %s", output)
	}

	return strings.TrimPrefix(output, expResponsePrefix), nil
}
