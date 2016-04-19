package runner

import (
	"sync"
	"testing"

	"github.com/franela/goblin"
)

func TestPipe(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("Pipe", func() {
		g.It("should get next line from buffer", func() {
			line := &Line{
				Proc: "redis",
				Pos:  1,
				Out:  "starting redis server",
			}
			pipe := newPipe(10)
			pipe.lines <- line
			next := pipe.Next()
			g.Assert(next).Equal(line)
		})

		g.It("should get null line on buffer closed", func() {
			pipe := newPipe(10)

			var wg sync.WaitGroup
			wg.Add(1)

			go func() {
				next := pipe.Next()
				g.Assert(next == nil).IsTrue("line should be nil")
				wg.Done()
			}()

			pipe.Close()
			wg.Wait()
		})

		g.Describe("Line output", func() {
			g.It("should prefix string() with metadata", func() {
				line := Line{
					Proc: "redis",
					Time: 60,
					Pos:  1,
					Out:  "starting redis server",
				}
				g.Assert(line.String()).Equal("[redis:L1:60s] starting redis server")
			})
		})
	})
}
