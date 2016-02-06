package smutje

import (
	"fmt"

	"net"

	"github.com/gfrey/smutje/connection"
	"github.com/gfrey/smutje/hypervisor"
	"github.com/gfrey/smutje/logger"
	"github.com/gfrey/smutje/parser"
)

type smResource struct {
	ID        string
	Name      string
	Blueprint string

	Attributes smAttributes
	Packages   []*smPackage

	client     connection.Client
	hypervisor hypervisor.Client
	uuid       string

	address  string
	username string

	isVirtual bool
}

func newResource(n *parser.AstNode) (*smResource, error) {
	res := new(smResource)
	res.Attributes = smAttributes{}
	res.ID = n.ID
	res.Name = n.Name

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
			pkgs, attrs, err := handleChild("", child)
			if err != nil {
				return nil, err
			}
			res.Packages = append(res.Packages, pkgs...)
			res.Attributes = res.Attributes.Merge(attrs)
		}
	}

	return res, nil
}

func (res *smResource) Prepare(l logger.Logger) error {
	l = l.Tag(res.ID)

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

func (res *smResource) Generate(l logger.Logger) (err error) {
	if res.isVirtual && res.client == nil {
		if res.uuid == "" {
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

	sess, err := res.client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	return sess.Run("mkdir -p /var/lib/smutje")
}

func (res *smResource) Provision(l logger.Logger) (err error) {
	l = l.Tag(res.ID)

	for _, pkg := range res.Packages {
		if err := pkg.Provision(l, res.client); err != nil {
			return err
		}
	}
	return nil
}

func (res *smResource) initializeClient() (err error) {
	_, ok := res.Attributes["Hypervisor"]
	if ok {
		res.isVirtual = true
	} else if res.Blueprint != "" {
		return fmt.Errorf("hypervisor attribute required for blueprint to be supported!")
	}

	res.address, ok = res.Attributes["Address"]
	if !ok {
		for _, cand := range []string{res.Name, res.ID} {
			ips, err := net.LookupIP(cand)
			if err == nil && len(ips) > 0 {
				res.address = ips[0].String()
				break
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
		res.hypervisor, err = hypervisor.New(res.Attributes["Hypervisor"], res.address, res.username)
		if err != nil {
			return err
		}

		res.uuid, err = res.hypervisor.UUID(res.ID)
		if err == nil && res.uuid != "" {
			res.client, err = res.hypervisor.ConnectVRes(res.uuid)
		}
		return err
	default:
		res.client, err = connection.NewSSHClient(res.address, res.username)
		return err
	}
}

func handleChild(parentID string, node *parser.AstNode) ([]*smPackage, smAttributes, error) {
	attrs := smAttributes{}
	pkgs := []*smPackage{}
	switch node.Type {
	case parser.AstAttributes:
		newAttrs, err := newAttributes(node)
		if err != nil {
			return nil, nil, err
		}
		attrs.MergeInplace(newAttrs)
	case parser.AstPackage:
		pkg, err := newPackage(parentID, node)
		if err != nil {
			return nil, nil, err
		}
		pkgs = append(pkgs, pkg)
	case parser.AstInclude:
		newPkgs, err := newInclude(parentID, node)
		if err != nil {
			return nil, nil, err
		}
		pkgs = append(pkgs, newPkgs...)
	case parser.AstText:
	// ignore
	default:
		return nil, nil, fmt.Errorf("unexpected node seen: %s", node.Type)
	}

	return pkgs, attrs, nil
}
