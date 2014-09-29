package director

import (
	"testing"

	"code.google.com/p/go.net/context"
	"github.com/drone/drone/server/worker"
	"github.com/drone/drone/server/worker/pool"
	"github.com/franela/goblin"
)

func TestDirector(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Director", func() {

		g.It("Should mark work as pending", func() {
			d := New()
			d.markPending(&worker.Work{})
			d.markPending(&worker.Work{})
			g.Assert(len(d.GetPending())).Equal(2)
		})

		g.It("Should mark work as started", func() {
			d := New()
			w1 := worker.Work{}
			w2 := worker.Work{}
			d.markPending(&w1)
			d.markPending(&w2)
			g.Assert(len(d.GetPending())).Equal(2)
			d.markStarted(&w1, &mockWorker{})
			g.Assert(len(d.GetStarted())).Equal(1)
			g.Assert(len(d.GetPending())).Equal(1)
			d.markStarted(&w2, &mockWorker{})
			g.Assert(len(d.GetStarted())).Equal(2)
			g.Assert(len(d.GetPending())).Equal(0)
		})

		g.It("Should mark work as complete", func() {
			d := New()
			w1 := worker.Work{}
			w2 := worker.Work{}
			d.markStarted(&w1, &mockWorker{})
			d.markStarted(&w2, &mockWorker{})
			g.Assert(len(d.GetStarted())).Equal(2)
			d.markComplete(&w1)
			g.Assert(len(d.GetStarted())).Equal(1)
			d.markComplete(&w2)
			g.Assert(len(d.GetStarted())).Equal(0)
		})

		g.It("Should get work assignments", func() {
			d := New()
			w1 := worker.Work{}
			w2 := worker.Work{}
			d.markStarted(&w1, &mockWorker{})
			d.markStarted(&w2, &mockWorker{})
			g.Assert(len(d.GetAssignemnts())).Equal(2)
		})

		g.It("Should recover from a panic", func() {
			d := New()
			d.Do(nil, nil)
			g.Assert(true).Equal(true)
		})

		g.It("Should distribute work to worker", func() {
			work := &worker.Work{}
			workr := &mockWorker{}
			c := context.Background()
			p := pool.New()
			p.Allocate(workr)
			c = pool.NewContext(c, p)

			d := New()
			d.do(c, work)
			g.Assert(workr.work).Equal(work) // verify mock worker gets work
		})

		g.It("Should add director to context", func() {
			d := New()
			c := context.Background()
			c = NewContext(c, d)
			g.Assert(worker.FromContext(c)).Equal(d)
		})
	})
}

// fake worker for testing purpose only
type mockWorker struct {
	name string
	work *worker.Work
}

func (m *mockWorker) Do(c context.Context, w *worker.Work) {
	m.work = w
}
