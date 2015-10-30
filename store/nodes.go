package store

import (
	"github.com/drone/drone/model"
	"golang.org/x/net/context"
)

type NodeStore interface {
	// Get gets a user by unique ID.
	Get(int64) (*model.Node, error)

	// GetList gets a list of all nodes in the system.
	GetList() ([]*model.Node, error)

	// Count gets a count of all nodes in the system.
	Count() (int, error)

	// Create creates a node.
	Create(*model.Node) error

	// Update updates a node.
	Update(*model.Node) error

	// Delete deletes a node.
	Delete(*model.Node) error
}

func GetNode(c context.Context, id int64) (*model.Node, error) {
	return FromContext(c).Nodes().Get(id)
}

func GetNodeList(c context.Context) ([]*model.Node, error) {
	return FromContext(c).Nodes().GetList()
}

func CountNodes(c context.Context) (int, error) {
	return FromContext(c).Nodes().Count()
}

func CreateNode(c context.Context, node *model.Node) error {
	return FromContext(c).Nodes().Create(node)
}

func UpdateNode(c context.Context, node *model.Node) error {
	return FromContext(c).Nodes().Update(node)
}

func DeleteNode(c context.Context, node *model.Node) error {
	return FromContext(c).Nodes().Delete(node)
}
