package engine

import (
	"testing"

	"github.com/drone/drone/model"
	"github.com/franela/goblin"
)

func TestPool(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Pool", func() {

		g.It("Should allocate nodes", func() {
			n := &model.Node{Addr: "unix:///var/run/docker.sock"}
			pool := newPool()
			pool.allocate(n)
			g.Assert(len(pool.nodes)).Equal(1)
			g.Assert(len(pool.nodec)).Equal(1)
			g.Assert(pool.nodes[n]).Equal(true)
		})

		g.It("Should not re-allocate an allocated node", func() {
			n := &model.Node{Addr: "unix:///var/run/docker.sock"}
			pool := newPool()
			g.Assert(pool.allocate(n)).Equal(true)
			g.Assert(pool.allocate(n)).Equal(false)
		})

		g.It("Should reserve a node", func() {
			n := &model.Node{Addr: "unix:///var/run/docker.sock"}
			pool := newPool()
			pool.allocate(n)
			g.Assert(<-pool.reserve()).Equal(n)
		})

		g.It("Should release a node", func() {
			n := &model.Node{Addr: "unix:///var/run/docker.sock"}
			pool := newPool()
			pool.allocate(n)
			g.Assert(len(pool.nodec)).Equal(1)
			g.Assert(<-pool.reserve()).Equal(n)
			g.Assert(len(pool.nodec)).Equal(0)
			pool.release(n)
			g.Assert(len(pool.nodec)).Equal(1)
			g.Assert(<-pool.reserve()).Equal(n)
			g.Assert(len(pool.nodec)).Equal(0)
		})

		g.It("Should not release an unallocated node", func() {
			n := &model.Node{Addr: "unix:///var/run/docker.sock"}
			pool := newPool()
			g.Assert(len(pool.nodes)).Equal(0)
			g.Assert(len(pool.nodec)).Equal(0)
			pool.release(n)
			g.Assert(len(pool.nodes)).Equal(0)
			g.Assert(len(pool.nodec)).Equal(0)
			pool.release(nil)
			g.Assert(len(pool.nodes)).Equal(0)
			g.Assert(len(pool.nodec)).Equal(0)
		})

		g.It("Should list all allocated nodes", func() {
			n1 := &model.Node{Addr: "unix:///var/run/docker.sock"}
			n2 := &model.Node{Addr: "unix:///var/run/docker.sock"}
			pool := newPool()
			pool.allocate(n1)
			pool.allocate(n2)
			g.Assert(len(pool.nodes)).Equal(2)
			g.Assert(len(pool.nodec)).Equal(2)
			g.Assert(len(pool.list())).Equal(2)
		})

		g.It("Should remove a node", func() {
			n1 := &model.Node{Addr: "unix:///var/run/docker.sock"}
			n2 := &model.Node{Addr: "unix:///var/run/docker.sock"}
			pool := newPool()
			pool.allocate(n1)
			pool.allocate(n2)
			g.Assert(len(pool.nodes)).Equal(2)
			pool.deallocate(n1)
			pool.deallocate(n2)
			g.Assert(len(pool.nodes)).Equal(0)
			g.Assert(len(pool.list())).Equal(0)
		})

	})
}
