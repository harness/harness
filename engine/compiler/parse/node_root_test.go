package parse

import (
	"testing"

	"github.com/franela/goblin"
)

func TestRootNode(t *testing.T) {
	g := goblin.Goblin(t)
	r := &RootNode{}

	g.Describe("Root Node", func() {

		g.It("should return self as root", func() {
			g.Assert(r).Equal(r.Root())
		})

		g.It("should create a Volume Node", func() {
			n := r.NewVolumeNode("foo")
			g.Assert(n.Root()).Equal(r)
			g.Assert(n.Name).Equal("foo")
			g.Assert(n.String()).Equal(NodeVolume)
			g.Assert(n.Type()).Equal(NodeType(NodeVolume))
		})

		g.It("should create a Network Node", func() {
			n := r.NewNetworkNode("foo")
			g.Assert(n.Root()).Equal(r)
			g.Assert(n.Name).Equal("foo")
			g.Assert(n.String()).Equal(NodeNetwork)
			g.Assert(n.Type()).Equal(NodeType(NodeNetwork))
		})

		g.It("should create a Plugin Node", func() {
			n := r.NewPluginNode()
			g.Assert(n.Root()).Equal(r)
			g.Assert(n.String()).Equal(NodePlugin)
			g.Assert(n.Type()).Equal(NodeType(NodePlugin))
		})

		g.It("should create a Shell Node", func() {
			n := r.NewShellNode()
			g.Assert(n.Root()).Equal(r)
			g.Assert(n.String()).Equal(NodeShell)
			g.Assert(n.Type()).Equal(NodeType(NodeShell))
		})

		g.It("should create a Service Node", func() {
			n := r.NewServiceNode()
			g.Assert(n.Root()).Equal(r)
			g.Assert(n.String()).Equal(NodeService)
			g.Assert(n.Type()).Equal(NodeType(NodeService))
		})

		g.It("should create a Build Node", func() {
			n := r.NewBuildNode(".")
			g.Assert(n.Root()).Equal(r)
			g.Assert(n.Context).Equal(".")
			g.Assert(n.String()).Equal(NodeBuild)
			g.Assert(n.Type()).Equal(NodeType(NodeBuild))
		})

		g.It("should create a Cache Node", func() {
			n := r.NewCacheNode()
			g.Assert(n.Root()).Equal(r)
			g.Assert(n.String()).Equal(NodeCache)
			g.Assert(n.Type()).Equal(NodeType(NodeCache))
		})

		g.It("should create a Clone Node", func() {
			n := r.NewCloneNode()
			g.Assert(n.Root()).Equal(r)
			g.Assert(n.String()).Equal(NodeClone)
			g.Assert(n.Type()).Equal(NodeType(NodeClone))
		})

		g.It("should create a Container Node", func() {
			n := r.NewContainerNode()
			g.Assert(n.Root()).Equal(r)
			g.Assert(n.String()).Equal(NodeContainer)
			g.Assert(n.Type()).Equal(NodeType(NodeContainer))
		})
	})
}
