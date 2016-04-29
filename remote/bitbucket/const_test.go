package bitbucket

import (
	"testing"

	"github.com/drone/drone/model"

	"github.com/franela/goblin"
)

func Test_status(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Bitbucket status", func() {
		g.It("should return passing", func() {
			g.Assert(getStatus(model.StatusSuccess)).Equal(statusSuccess)
		})
		g.It("should return pending", func() {
			g.Assert(getStatus(model.StatusPending)).Equal(statusPending)
			g.Assert(getStatus(model.StatusRunning)).Equal(statusPending)
		})
		g.It("should return failing", func() {
			g.Assert(getStatus(model.StatusFailure)).Equal(statusFailure)
			g.Assert(getStatus(model.StatusKilled)).Equal(statusFailure)
			g.Assert(getStatus(model.StatusError)).Equal(statusFailure)
		})

		g.It("should return passing desc", func() {
			g.Assert(getDesc(model.StatusSuccess)).Equal(descSuccess)
		})
		g.It("should return pending desc", func() {
			g.Assert(getDesc(model.StatusPending)).Equal(descPending)
			g.Assert(getDesc(model.StatusRunning)).Equal(descPending)
		})
		g.It("should return failing desc", func() {
			g.Assert(getDesc(model.StatusFailure)).Equal(descFailure)
		})
		g.It("should return error desc", func() {
			g.Assert(getDesc(model.StatusKilled)).Equal(descError)
			g.Assert(getDesc(model.StatusError)).Equal(descError)
		})
	})
}
