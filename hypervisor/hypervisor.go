package hypervisor

import (
	"fmt"

	"github.com/gfrey/smutje/connection"
	"github.com/gfrey/smutje/logger"
)

type Client interface {
	Create(l logger.Logger, blueprint string) (string, error)
	UUID(alias string) (string, error)
	ConnectVRes(uuid string) (connection.Client, error)
}

func New(typ, address, username string) (Client, error) {
	switch typ {
	case "smartos":
		return NewSmartOSHypervisor(address, username)

	default:
		return nil, fmt.Errorf("hypervisor %q not supported", typ)
	}
}
