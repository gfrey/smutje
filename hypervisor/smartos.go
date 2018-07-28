package hypervisor

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"strings"
	"sync"

	"github.com/gfrey/gconn"
	"github.com/pkg/errors"
)

type smartOS struct {
	addr   string
	user   string
	client gconn.Client
}

func SmartOS(addr string) (Client, error) {
	var err error
	hp := new(smartOS)
	hp.addr = addr
	hp.user = "root"
	hp.client, err = gconn.NewSSHClient(hp.addr, hp.user)
	return hp, err
}

func (hp *smartOS) ConnectVRes(uuid string) (gconn.Client, error) {
	// determine the vm brand
	brand, err := hp.Brand(uuid)
	if err != nil {
		return nil, err
	}

	switch brand {
	case "kvm":
		ip, err := hp.KVMIP(uuid)
		if err != nil {
			return nil, err
		}
		return gconn.NewSSHProxyClient(hp.client, ip, "root")
	case "joyent", "lx":
		return gconn.NewWrappedClient(hp.client, "zlogin "+uuid), nil
	default:
		return nil, errors.Errorf("unknown VM brand: %s", brand)
	}
}

func (hp *smartOS) KVMIP(uuid string) (string, error) {
	sess, err := hp.client.NewSession("vmadm get " + uuid + " | json nics[0].ip")
	if err != nil {
		return "", err
	}
	defer sess.Close()

	stdout, err := sess.StdoutPipe()
	if err != nil {
		return "", err
	}

	if err := sess.Start(); err != nil {
		return "", err
	}

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, stdout); err != nil {
		return "", err
	}

	if err := sess.Wait(); err != nil {
		return "", err
	}

	return strings.TrimSpace(buf.String()), nil
}

func (hp *smartOS) Brand(uuid string) (string, error) {
	// determine whether the VM in question already exists
	sess, err := hp.client.NewSession("vmadm get " + uuid + " | json brand")
	if err != nil {
		return "", err
	}
	defer sess.Close()

	stdout, err := sess.StdoutPipe()
	if err != nil {
		return "", err
	}

	if err := sess.Start(); err != nil {
		return "", err
	}

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, stdout); err != nil {
		return "", err
	}

	if err := sess.Wait(); err != nil {
		return "", err
	}

	return strings.TrimSpace(buf.String()), nil
}

func (hp *smartOS) UUID(alias string) (string, error) {
	// determine whether the VM in question already exists
	sess, err := hp.client.NewSession("vmadm", "list", "-p")
	if err != nil {
		return "", err
	}
	defer sess.Close()

	stdout, err := sess.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err := sess.Start(); err != nil {
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

func image_uuid(m map[string]interface{}) (string, error) {
	u, found := m["image_uuid"]
	if found {
		uuid, ok := u.(string)
		if !ok {
			return "", errors.Errorf("image_uuid not a string")
		}
		return uuid, nil
	}
	return "", nil
}

func image_uuids(m map[string]interface{}) ([]string, error) {
	uuids := []string{}
	switch u, err := image_uuid(m); {
	case err != nil:
		return nil, err
	case u != "":
		uuids = append(uuids, u)
	}

	disksR, found := m["disks"]
	if found {
		disks, ok := disksR.([]map[string]interface{})
		if !ok {
			return uuids, nil
		}

		for _, disk := range disks {
			switch u, err := image_uuid(disk); {
			case err != nil:
				return nil, err
			case u != "":
				uuids = append(uuids, u)
			}
		}
	}

	return uuids, nil
}

func (hp *smartOS) Create(l *log.Logger, blueprint string) (string, error) {
	m := map[string]interface{}{}
	if err := json.Unmarshal([]byte(blueprint), &m); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal the blueprint")
	}

	l.Printf("updating the image database")
	if err := runCommand(hp.client, "imgadm update"); err != nil {
		return "", err
	}

	imgUUIDs, err := image_uuids(m)
	if err != nil {
		return "", err
	}

	for _, imgUUID := range imgUUIDs {
		l.Printf("importing image %s", imgUUID)
		if err := runCommand(hp.client, "imgadm import -q "+imgUUID); err != nil {
			return "", err
		}
	}

	// determine whether the VM in question already exists
	sess, err := hp.client.NewSession("vmadm", "create")
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
	if err := sess.Run(); err != nil {
		wg.Wait()
		log.Printf("failed: %s", outBuf.String())
		return "", err
	}
	wg.Wait()

	output := strings.TrimSpace(outBuf.String())
	expResponsePrefix := "Successfully created VM "
	if !strings.HasPrefix(output, expResponsePrefix) {
		return "", errors.Errorf("wrong response received: %s", output)
	}

	vmID := strings.TrimPrefix(output, expResponsePrefix)

	if autostart, ok := m["autostart"].(bool); ok && !autostart {
		sess, err := hp.client.NewSession("vmadm", "start", vmID)
		if err != nil {
			return "", err
		}
		defer sess.Close()

		l.Printf("starting the virtual resource")
		if err := sess.Run(); err != nil {
			log.Printf("failed to start VM %s", vmID)
			return "", err
		}
	}

	return vmID, nil
}
