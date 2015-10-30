package datastore

import (
	"database/sql"

	"github.com/drone/drone/model"
	"github.com/russross/meddler"
)

type nodestore struct {
	*sql.DB
}

func (db *nodestore) Get(id int64) (*model.Node, error) {
	var node = new(model.Node)
	var err = meddler.Load(db, nodeTable, node, id)
	return node, err
}

func (db *nodestore) GetList() ([]*model.Node, error) {
	var nodes = []*model.Node{}
	var err = meddler.QueryAll(db, &nodes, rebind(nodeListQuery))
	return nodes, err
}

func (db *nodestore) Count() (int, error) {
	var count int
	var err = db.QueryRow(rebind(nodeCountQuery)).Scan(&count)
	return count, err
}

func (db *nodestore) Create(node *model.Node) error {
	return meddler.Insert(db, nodeTable, node)
}

func (db *nodestore) Update(node *model.Node) error {
	return meddler.Update(db, nodeTable, node)
}

func (db *nodestore) Delete(node *model.Node) error {
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
