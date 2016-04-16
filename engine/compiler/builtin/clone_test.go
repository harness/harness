package builtin

// import (
// 	"testing"

// 	"github.com/libcd/libcd"
// 	"github.com/libcd/libyaml/parse"

// 	"github.com/franela/goblin"
// )

// func Test_clone(t *testing.T) {
// 	root := parse.NewRootNode()

// 	g := goblin.Goblin(t)
// 	g.Describe("clone", func() {

// 		g.It("should use default when nil", func() {
// 			op := NewCloneOp("plugins/git:latest")

// 			op.VisitRoot(root)
// 			g.Assert(root.Clone.(*parse.ContainerNode).Container.Image).Equal("plugins/git:latest")
// 		})

// 		g.It("should use user-defined clone plugin", func() {
// 			op := NewCloneOp("plugins/git:latest")
// 			clone := root.NewCloneNode()
// 			clone.Container = libcd.Container{}
// 			clone.Container.Image = "custom/hg:latest"
// 			root.Clone = clone

// 			op.VisitRoot(root)
// 			g.Assert(clone.Container.Image).Equal("custom/hg:latest")
// 		})
// 	})
// }
