package model

import (
	"github.com/drone/drone/shared/database"
	"github.com/russross/meddler"
)

type Node struct {
	ID   int64  `meddler:"node_id,pk" json:"id"`
	Addr string `meddler:"node_addr"  json:"address"`
	Arch string `meddler:"node_arch"  json:"architecture"`
	Cert string `meddler:"node_cert"  json:"-"`
	Key  string `meddler:"node_key"   json:"-"`
	CA   string `meddler:"node_ca"    json:"-"`
}

func GetNode(db meddler.DB, id int64) (*Node, error) {
	var node = new(Node)
	var err = meddler.Load(db, nodeTable, node, id)
	return node, err
}

func GetNodeList(db meddler.DB) ([]*Node, error) {
	var nodes = []*Node{}
	var err = meddler.QueryAll(db, &nodes, database.Rebind(nodeListQuery))
	return nodes, err
}

func InsertNode(db meddler.DB, node *Node) error {
	return meddler.Insert(db, nodeTable, node)
}

func UpdateNode(db meddler.DB, node *Node) error {
	return meddler.Update(db, nodeTable, node)
}

func DeleteNode(db meddler.DB, node *Node) error {
	var _, err = db.Exec(database.Rebind(nodeDeleteStmt), node.ID)
	return err
}

const nodeTable = "nodes"

const nodeListQuery = `
SELECT *
FROM nodes
ORDER BY node_addr
`

const nodeDeleteStmt = `
DELETE FROM nodes
WHERE node_id=?
`

const (
	Freebsd_386 uint = iota
	Freebsd_amd64
	Freebsd_arm
	Linux_386
	Linux_amd64
	Linux_arm
	Linux_arm64
	Solaris_amd64
	Windows_386
	Windows_amd64
)

var Archs = map[string]uint{
	"freebsd_386":   Freebsd_386,
	"freebsd_amd64": Freebsd_amd64,
	"freebsd_arm":   Freebsd_arm,
	"linux_386":     Linux_386,
	"linux_amd64":   Linux_amd64,
	"linux_arm":     Linux_arm,
	"linux_arm64":   Linux_arm64,
	"solaris_amd64": Solaris_amd64,
	"windows_386":   Windows_386,
	"windows_amd64": Windows_amd64,
}
