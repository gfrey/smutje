package hypervisor

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	"github.com/gfrey/smutje/connection"
	"github.com/gfrey/smutje/logger"
)

type smartOS struct {
	addr   string
	user   string
	client connection.Client
}

func NewSmartOSHypervisor(addr, user string) (Client, error) {
	var err error
	hp := new(smartOS)
	hp.addr, hp.user = addr, user
	hp.client, err = connection.NewSSHClient(addr, user)
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
	return "", sc.Err()
}

func (hp *smartOS) Create(l logger.Logger, blueprint string) (string, error) {
	m := map[string]interface{}{}
	if err := json.Unmarshal([]byte(blueprint), &m); err != nil {
		return "", err
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
		return "", err
	}
	go func() {
		defer wg.Done()
		_, _ = io.Copy(outBuf, stderr)
	}()

	stdin, err := sess.StdinPipe()
	if err != nil {
		return "", err
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
		return "", fmt.Errorf("wrong response received: %s", output)
	}

	return strings.TrimPrefix(output, expResponsePrefix), nil
}