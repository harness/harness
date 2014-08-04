package github

import (
	"testing"

	"github.com/drone/drone/shared/model"
	"github.com/franela/goblin"
)

func Test_Client(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Github Status", func() {

		g.It("Should get a status", func() {
			g.Assert(getStatus(model.StatusEnqueue)).Equal(StatusPending)
			g.Assert(getStatus(model.StatusStarted)).Equal(StatusPending)
			g.Assert(getStatus(model.StatusSuccess)).Equal(StatusSuccess)
			g.Assert(getStatus(model.StatusFailure)).Equal(StatusFailure)
			g.Assert(getStatus(model.StatusError)).Equal(StatusError)
			g.Assert(getStatus(model.StatusKilled)).Equal(StatusError)
			g.Assert(getStatus(model.StatusNone)).Equal(StatusError)
		})

		g.It("Should get a description", func() {
			g.Assert(getDesc(model.StatusEnqueue)).Equal(DescPending)
			g.Assert(getDesc(model.StatusStarted)).Equal(DescPending)
			g.Assert(getDesc(model.StatusSuccess)).Equal(DescSuccess)
			g.Assert(getDesc(model.StatusFailure)).Equal(DescFailure)
			g.Assert(getDesc(model.StatusError)).Equal(DescError)
			g.Assert(getDesc(model.StatusKilled)).Equal(DescError)
			g.Assert(getDesc(model.StatusNone)).Equal(DescError)
		})

		g.It("Should get a target url", func() {
			var (
				url    = "https://drone.io"
				host   = "github.com"
				owner  = "drone"
				repo   = "go-bitbucket"
				branch = "master"
				commit = "0c0cf4ece975efdfcf6daa78b03d4e84dd257da7"
			)

			var got = getTarget(url, host, owner, repo, branch, commit)
			var want = "https://drone.io/github.com/drone/go-bitbucket/master/0c0cf4ece975efdfcf6daa78b03d4e84dd257da7"
			g.Assert(got).Equal(want)
		})
	})
}
