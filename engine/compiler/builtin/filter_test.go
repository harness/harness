package builtin

// import (
// 	"testing"

// 	"github.com/franela/goblin"
// )

// func TestFilter(t *testing.T) {
// 	g := goblin.Goblin(t)
// 	g.Describe("Filters", func() {

// 		g.It("Should match no branch filter", func() {
// 			c := &Container{}
// 			FilterBranch("feature/foo")(nil, c)
// 			g.Assert(c.Disabled).IsFalse()
// 		})

// 		g.It("Should match branch", func() {
// 			c := &Container{}
// 			c.Conditions.Branch.parts = []string{"feature/*"}
// 			FilterBranch("feature/foo")(nil, c)
// 			g.Assert(c.Disabled).IsFalse()
// 		})

// 		g.It("Should match branch wildcard", func() {
// 			c := &Container{}
// 			c.Conditions.Branch.parts = []string{"feature/*"}
// 			FilterBranch("feature/foo")(nil, c)
// 			g.Assert(c.Disabled).IsFalse()
// 		})

// 		g.It("Should disable when branch filter doesn't match", func() {
// 			c := &Container{}
// 			c.Conditions.Branch.parts = []string{"feature/*", "develop"}
// 			FilterBranch("master")(nil, c)
// 			g.Assert(c.Disabled).IsTrue()
// 		})

// 		g.It("Should match no platform filter", func() {
// 			c := &Container{}
// 			FilterPlatform("linux_amd64")(nil, c)
// 			g.Assert(c.Disabled).IsFalse()
// 		})

// 		g.It("Should match platform", func() {
// 			c := &Container{}
// 			c.Conditions.Platform.parts = []string{"linux_amd64"}
// 			FilterPlatform("linux_amd64")(nil, c)
// 			g.Assert(c.Disabled).IsFalse()
// 		})

// 		g.It("Should disable when platform filter doesn't match", func() {
// 			c := &Container{}
// 			c.Conditions.Platform.parts = []string{"linux_arm", "linux_arm64"}
// 			FilterPlatform("linux_amd64")(nil, c)
// 			g.Assert(c.Disabled).IsTrue()
// 		})

// 		g.It("Should match no environment filter", func() {
// 			c := &Container{}
// 			FilterEnvironment("production")(nil, c)
// 			g.Assert(c.Disabled).IsFalse()
// 		})

// 		g.It("Should match environment", func() {
// 			c := &Container{}
// 			c.Conditions.Environment.parts = []string{"production"}
// 			FilterEnvironment("production")(nil, c)
// 			g.Assert(c.Disabled).IsFalse()
// 		})

// 		g.It("Should disable when environment filter doesn't match", func() {
// 			c := &Container{}
// 			c.Conditions.Environment.parts = []string{"develop", "staging"}
// 			FilterEnvironment("production")(nil, c)
// 			g.Assert(c.Disabled).IsTrue()
// 		})

// 		g.It("Should match no event filter", func() {
// 			c := &Container{}
// 			FilterEvent("push")(nil, c)
// 			g.Assert(c.Disabled).IsFalse()
// 		})

// 		g.It("Should match event", func() {
// 			c := &Container{}
// 			c.Conditions.Event.parts = []string{"push"}
// 			FilterEvent("push")(nil, c)
// 			g.Assert(c.Disabled).IsFalse()
// 		})

// 		g.It("Should disable when event filter doesn't match", func() {
// 			c := &Container{}
// 			c.Conditions.Event.parts = []string{"push", "tag"}
// 			FilterEvent("pull_request")(nil, c)
// 			g.Assert(c.Disabled).IsTrue()
// 		})

// 		g.It("Should match matrix", func() {
// 			c := &Container{}
// 			c.Conditions.Matrix = map[string]string{
// 				"go":    "1.5",
// 				"redis": "3.0",
// 			}
// 			matrix := map[string]string{
// 				"go":    "1.5",
// 				"redis": "3.0",
// 				"node":  "5.0.0",
// 			}
// 			FilterMatrix(matrix)(nil, c)
// 			g.Assert(c.Disabled).IsFalse()
// 		})

// 		g.It("Should disable when event filter doesn't match", func() {
// 			c := &Container{}
// 			c.Conditions.Matrix = map[string]string{
// 				"go":    "1.5",
// 				"redis": "3.0",
// 			}
// 			matrix := map[string]string{
// 				"go":    "1.4.2",
// 				"redis": "3.0",
// 				"node":  "5.0.0",
// 			}
// 			FilterMatrix(matrix)(nil, c)
// 			g.Assert(c.Disabled).IsTrue()
// 		})
// 	})
// }
