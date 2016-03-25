package datastore

import (
	"github.com/drone/drone/model"
	"github.com/russross/meddler"
)

func (db *datastore) GetNode(id int64) (*model.Node, error) {
	var node = new(model.Node)
	var err = meddler.Load(db, nodeTable, node, id)
	return node, err
}

func (db *datastore) GetNodeList() ([]*model.Node, error) {
	var nodes = []*model.Node{}
	var err = meddler.QueryAll(db, &nodes, rebind(nodeListQuery))
	return nodes, err
}

func (db *datastore) CreateNode(node *model.Node) error {
	return meddler.Insert(db, nodeTable, node)
}

func (db *datastore) UpdateNode(node *model.Node) error {
	return meddler.Update(db, nodeTable, node)
}

func (db *datastore) DeleteNode(node *model.Node) error {
	var _, err = db.Exec(rebind(nodeDeleteStmt), node.ID)
	return err
}

const nodeTable = "nodes"

const nodeListQuery = `
SELECT *
FROM nodes
ORDER BY node_addr
`

const nodeCountQuery = `
SELECT COUNT(*) FROM nodes
`

const nodeDeleteStmt = `
DELETE FROM nodes
WHERE node_id=?
`
