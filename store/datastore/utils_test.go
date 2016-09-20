package datastore

import (
	"testing"

	"github.com/drone/drone/model"
	"github.com/franela/goblin"
)

func TestUtils(t *testing.T) {
	g := goblin.Goblin(t)
	g.Describe("Utils", func() {

		g.It("Should Calculate Pagination", func() {
			g.Assert(calculatePagination(10, 100)).Equal(1)
			g.Assert(calculatePagination(10, 5)).Equal(2)
			g.Assert(calculatePagination(10, 3)).Equal(4)
		})

		g.It("Should Resize a RepoLite List", func() {
			repoLiteList := []*model.RepoLite{
				{FullName: "bradrydzewski/drone"},
				{FullName: "drone/drone"},
				{FullName: "octocat/hello-world-1"},
				{FullName: "octocat/hello-world-2"},
				{FullName: "octocat/hello-world-3"},
			}
			notResized := resizeList(repoLiteList, 0, 10)
			page1 := resizeList(repoLiteList, 0, 2)
			page2 := resizeList(repoLiteList, 1, 2)
			page3 := resizeList(repoLiteList, 2, 2)

			g.Assert(len(notResized)).Equal(5)

			g.Assert(len(page1)).Equal(2)
			g.Assert(page1[0].FullName).Equal("bradrydzewski/drone")
			g.Assert(page1[1].FullName).Equal("drone/drone")

			g.Assert(len(page2)).Equal(2)
			g.Assert(page2[0].FullName).Equal("octocat/hello-world-1")
			g.Assert(page2[1].FullName).Equal("octocat/hello-world-2")

			g.Assert(len(page3)).Equal(1)
			g.Assert(page3[0].FullName).Equal("octocat/hello-world-3")
		})
	})
}
