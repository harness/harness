package model

import (
	"testing"

	"github.com/drone/drone/shared/database"
	"github.com/franela/goblin"
)

func TestNode(t *testing.T) {
	db := database.OpenTest()
	defer db.Close()

	g := goblin.Goblin(t)
	g.Describe("Nodes", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM nodes")
		})

		g.It("Should create a node", func() {
			node := Node{
				Addr: "unix:///var/run/docker/docker.sock",
				Arch: "linux_amd64",
			}
			err := InsertNode(db, &node)
			g.Assert(err == nil).IsTrue()
			g.Assert(node.ID != 0).IsTrue()
		})

		g.It("Should update a node", func() {
			node := Node{
				Addr: "unix:///var/run/docker/docker.sock",
				Arch: "linux_amd64",
			}
			err := InsertNode(db, &node)
			g.Assert(err == nil).IsTrue()
			g.Assert(node.ID != 0).IsTrue()

			node.Addr = "unix:///var/run/docker.sock"

			err1 := UpdateNode(db, &node)
			getnode, err2 := GetNode(db, node.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(node.ID).Equal(getnode.ID)
			g.Assert(node.Addr).Equal(getnode.Addr)
			g.Assert(node.Arch).Equal(getnode.Arch)
		})

		g.It("Should get a node", func() {
			node := Node{
				Addr: "unix:///var/run/docker/docker.sock",
				Arch: "linux_amd64",
			}
			err := InsertNode(db, &node)
			g.Assert(err == nil).IsTrue()
			g.Assert(node.ID != 0).IsTrue()

			getnode, err := GetNode(db, node.ID)
			g.Assert(err == nil).IsTrue()
			g.Assert(node.ID).Equal(getnode.ID)
			g.Assert(node.Addr).Equal(getnode.Addr)
			g.Assert(node.Arch).Equal(getnode.Arch)
		})

		g.It("Should get a node list", func() {
			node1 := Node{
				Addr: "unix:///var/run/docker/docker.sock",
				Arch: "linux_amd64",
			}
			node2 := Node{
				Addr: "unix:///var/run/docker.sock",
				Arch: "linux_386",
			}
			InsertNode(db, &node1)
			InsertNode(db, &node2)

			nodes, err := GetNodeList(db)
			g.Assert(err == nil).IsTrue()
			g.Assert(len(nodes)).Equal(2)
		})

		g.It("Should delete a node", func() {
			node := Node{
				Addr: "unix:///var/run/docker/docker.sock",
				Arch: "linux_amd64",
			}
			err1 := InsertNode(db, &node)
			err2 := DeleteNode(db, &node)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()

			_, err := GetNode(db, node.ID)
			g.Assert(err == nil).IsFalse()
		})
	})
}
