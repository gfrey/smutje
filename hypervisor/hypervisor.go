package hypervisor

import (
	"github.com/gfrey/smutje/connection"
	"github.com/gfrey/smutje/logger"
	"github.com/pkg/errors"
)

type Client interface {
	Create(l logger.Logger, blueprint string) (string, error)
	UUID(alias string) (string, error)
	ConnectVRes(uuid string) (connection.Client, error)
}

func New(attributes map[string]string) (Client, error) {
	switch typ := attributes["Hypervisor"]; typ {
	case "smartos":
		address, found := attributes["Host"]
		if !found {
			return nil, errors.Errorf("no address specified for smartos host")
		}
		return NewSmartOSHypervisor(address)
	case "gbox":
		return NewGBoxHypervisor()

	default:
		return nil, errors.Errorf("hypervisor %q not supported", typ)
	}
}
