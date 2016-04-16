package builtin

// import (
// 	"testing"

// 	"github.com/libcd/libcd"
// 	"github.com/libcd/libyaml/parse"

// 	"github.com/franela/goblin"
// )

// func Test_cache(t *testing.T) {
// 	root := parse.NewRootNode()

// 	g := goblin.Goblin(t)
// 	g.Describe("cache", func() {

// 		g.It("should use default when nil", func() {
// 			op := NewCacheOp("plugins/cache:latest", "/tmp/cache")

// 			op.VisitRoot(root)
// 			g.Assert(root.Cache.(*parse.ContainerNode).Container.Image).Equal("plugins/cache:latest")
// 			g.Assert(root.Cache.(*parse.ContainerNode).Container.Volumes[0]).Equal("/tmp/cache:/cache")
// 		})

// 		g.It("should use user-defined cache plugin", func() {
// 			op := NewCacheOp("plugins/cache:latest", "/tmp/cache")
// 			cache := root.NewCacheNode()
// 			cache.Container = libcd.Container{}
// 			cache.Container.Image = "custom/cacher:latest"
// 			root.Cache = cache

// 			op.VisitRoot(root)
// 			g.Assert(cache.Container.Image).Equal("custom/cacher:latest")
// 		})
// 	})
// }
