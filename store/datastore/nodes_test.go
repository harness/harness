package datastore

import (
	"testing"

	"github.com/drone/drone/model"
	"github.com/franela/goblin"
)

func TestNodes(t *testing.T) {
	db := openTest()
	defer db.Close()

	s := From(db)
	g := goblin.Goblin(t)
	g.Describe("Nodes", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM nodes")
		})

		g.It("Should create a node", func() {
			node := model.Node{
				Addr: "unix:///var/run/docker/docker.sock",
				Arch: "linux_amd64",
			}
			err := s.CreateNode(&node)
			g.Assert(err == nil).IsTrue()
			g.Assert(node.ID != 0).IsTrue()
		})

		g.It("Should update a node", func() {
			node := model.Node{
				Addr: "unix:///var/run/docker/docker.sock",
				Arch: "linux_amd64",
			}
			err := s.CreateNode(&node)
			g.Assert(err == nil).IsTrue()
			g.Assert(node.ID != 0).IsTrue()

			node.Addr = "unix:///var/run/docker.sock"

			err1 := s.UpdateNode(&node)
			getnode, err2 := s.GetNode(node.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(node.ID).Equal(getnode.ID)
			g.Assert(node.Addr).Equal(getnode.Addr)
			g.Assert(node.Arch).Equal(getnode.Arch)
		})

		g.It("Should get a node", func() {
			node := model.Node{
				Addr: "unix:///var/run/docker/docker.sock",
				Arch: "linux_amd64",
			}
			err := s.CreateNode(&node)
			g.Assert(err == nil).IsTrue()
			g.Assert(node.ID != 0).IsTrue()

			getnode, err := s.GetNode(node.ID)
			g.Assert(err == nil).IsTrue()
			g.Assert(node.ID).Equal(getnode.ID)
			g.Assert(node.Addr).Equal(getnode.Addr)
			g.Assert(node.Arch).Equal(getnode.Arch)
		})

		g.It("Should get a node list", func() {
			node1 := model.Node{
				Addr: "unix:///var/run/docker/docker.sock",
				Arch: "linux_amd64",
			}
			node2 := model.Node{
				Addr: "unix:///var/run/docker.sock",
				Arch: "linux_386",
			}
			s.CreateNode(&node1)
			s.CreateNode(&node2)

			nodes, err := s.GetNodeList()
			g.Assert(err == nil).IsTrue()
			g.Assert(len(nodes)).Equal(2)
		})

		g.It("Should delete a node", func() {
			node := model.Node{
				Addr: "unix:///var/run/docker/docker.sock",
				Arch: "linux_amd64",
			}
			err1 := s.CreateNode(&node)
			err2 := s.DeleteNode(&node)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()

			_, err := s.GetNode(node.ID)
			g.Assert(err == nil).IsFalse()
		})
	})
}
