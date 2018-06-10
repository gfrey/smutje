package smutje

import (
	"fmt"
	"log"

	"net"

	"github.com/gfrey/gconn"
	"github.com/gfrey/smutje/hypervisor"
	"github.com/gfrey/smutje/parser"
	"github.com/pkg/errors"
)

type Resource struct {
	ID        string
	Name      string
	Blueprint string

	Attributes Attributes
	Packages   []*smPackage

	client     gconn.Client
	hypervisor hypervisor.Client
	uuid       string

	address  string
	username string

	isVirtual bool
}

func NewResource(path string, n *parser.AstNode) (*Resource, error) {
	res := new(Resource)
	res.ID = n.ID
	res.Name = n.Name

	res.Attributes = Attributes{}
	res.Attributes["Hostname"] = n.ID

	for _, child := range n.Children {
		switch child.Type {
		case parser.AstBlueprint:
			blueprint, err := newBlueprint(child)
			if err != nil {
				return nil, err
			}
			res.Blueprint = blueprint
		default:
			pkgs, err := handleChild("", path, res.Attributes, child)
			if err != nil {
				return nil, err
			}
			res.Packages = append(res.Packages, pkgs...)
		}
	}

	return res, nil
}

func (res *Resource) Prepare(l *log.Logger) error {
	l = tagLogger(l, res.ID)

	if err := res.initializeClient(); err != nil {
		return err
	}

	for _, pkg := range res.Packages {
		if err := pkg.Prepare(res.client, res.Attributes); err != nil {
			return err
		}
	}
	return nil
}

func (res *Resource) Generate(l *log.Logger) (err error) {
	if res.isVirtual && res.client == nil {
		if res.uuid == "" {
			res.Blueprint, err = renderString(res.ID+"/blueprint", res.Blueprint, res.Attributes)
			if err != nil {
				return err
			}

			res.uuid, err = res.hypervisor.Create(l, res.Blueprint)
			if err != nil {
				return err
			}
		}

		res.client, err = res.hypervisor.ConnectVRes(res.uuid)
		if err != nil {
			return err
		}
	}

	sess, err := res.client.NewSession("/usr/bin/env", "bash", "-c", `"mkdir -p /tmp/smutje && mkdir -p /var/lib/smutje"`)
	if err != nil {
		return err
	}
	defer sess.Close()

	return sess.Run()
}

func (res *Resource) Provision(l *log.Logger) (err error) {
	l = tagLogger(l, res.ID)

	for _, pkg := range res.Packages {
		if err := pkg.Provision(l, res.client); err != nil {
			return err
		}
	}
	return nil
}

func (res *Resource) initializeClient() (err error) {
	var ok bool
	switch _, ok = res.Attributes["Hypervisor"]; {
	case ok:
		res.isVirtual = true
	case res.Blueprint != "":
		return errors.Errorf("hypervisor attribute required for blueprint to be supported!")
	default:
		res.address, ok = res.Attributes["Address"]
		if !ok {
			for _, cand := range []string{res.Name, res.ID} {
				ips, err := net.LookupIP(cand)
				if err == nil && len(ips) > 0 {
					res.address = ips[0].String()
					break
				}
			}
		}
		if res.address == "" {
			return fmt.Errorf("Host address attribute not specified!")
		}
	}

	res.username, ok = res.Attributes["Username"]
	if !ok {
		res.username = "root"
	}

	switch {
	case res.isVirtual:
		res.hypervisor, err = hypervisor.New(res.Attributes)
		if err != nil {
			return err
		}

		res.uuid, err = res.hypervisor.UUID(res.ID)
		if err == nil && res.uuid != "" {
			res.client, err = res.hypervisor.ConnectVRes(res.uuid)
		}
		return err
	default:
		res.client, err = gconn.NewSSHClient(res.address, res.username)
		return err
	}
}

func handleChild(parentID, path string, attrs Attributes, node *parser.AstNode) ([]*smPackage, error) {
	pkgs := []*smPackage{}
	switch node.Type {
	case parser.AstAttributes:
		newAttrs, err := newAttributes(node)
		if err != nil {
			return nil, err
		}
		if err := attrs.MergeInplace(newAttrs); err != nil {
			return nil, err
		}
	case parser.AstPackage:
		pkg, err := newPackage(parentID, path, attrs, node)
		if err != nil {
			return nil, err
		}
		pkgs = append(pkgs, pkg)
	case parser.AstInclude:
		newPkgs, err := newInclude(parentID, path, attrs, node)
		if err != nil {
			return nil, err
		}
		pkgs = append(pkgs, newPkgs...)
	case parser.AstText:
	// ignore
	default:
		return nil, errors.Errorf("unexpected node seen: %s", node.Type)
	}

	return pkgs, nil
}
