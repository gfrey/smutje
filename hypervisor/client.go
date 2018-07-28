package hypervisor

import (
	"log"

	"github.com/gfrey/gconn"
)

type Client interface {
	Create(l *log.Logger, blueprint string) (string, error)
	UUID(alias string) (string, error)
	ConnectVRes(uuid string) (gconn.Client, error)
}
