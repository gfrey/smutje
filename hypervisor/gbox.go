package hypervisor

import (
	"encoding/json"
	"log"

	"github.com/gfrey/gbox"
	"github.com/gfrey/gconn"
	"github.com/pkg/errors"
)

type gboxClient struct {
}

type gboxBlueprint struct {
	Template string `json:"template"`
	Name     string `json:"name"`
}

func NewGBoxHypervisor() (Client, error) {
	return new(gboxClient), nil
}

func (hp *gboxClient) ConnectVRes(name string) (gconn.Client, error) {
	addr, err := gbox.ReadProperty(name, "/VirtualBox/GuestInfo/Net/1/V4/IP")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get IP of vbox")
	}

	return gconn.NewSSHClient(addr, "ubuntu")
}

func (hp *gboxClient) UUID(name string) (string, error) {
	switch {
	case !gbox.ExistsVM(name):
		return "", nil
	case !gbox.RunningVM(name):
		return name, gbox.StartVM(name)
	default:
		return name, nil
	}

}

func (hp *gboxClient) Create(l *log.Logger, blueprint string) (string, error) {
	bp := new(gboxBlueprint)
	if err := json.Unmarshal([]byte(blueprint), &bp); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal the blueprint")
	}

	if bp.Name == "" {
		return "", errors.Errorf("no name for VM specified in blueprint")
	}

	switch {
	case bp.Template == "":
		return "", errors.Errorf("no template specified in blueprint")
	case !gbox.ExistsTemplate(bp.Template):
		return "", errors.Errorf("template %q specified in blueprint, does not exist", bp.Template)
	}

	err := gbox.CreateVM(bp.Name, bp.Template)
	if err != nil {
		return "", errors.Wrap(err, "failed to create VM")
	}

	return bp.Name, errors.Wrap(gbox.StartVM(bp.Name), "failed to start VM")
}
