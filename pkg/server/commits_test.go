package server

import (
	"bytes"
	//"strings"
	//"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	//"net/url"
	"fmt"
	"github.com/drone/drone/pkg/server/recorder"
	"github.com/drone/drone/pkg/store/mock"
	common "github.com/drone/drone/pkg/types"
	. "github.com/franela/goblin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestCommits(t *testing.T) {
	store := new(mocks.Store)
	//_url, _ := url.Parse("http://localhost:8080")

	g := Goblin(t)
	g.Describe("Commits", func() {

		g.It("Should get commit", func() {
			//
			buildList := []*common.Build{
				&common.Build{
					CommitID: 1,
					State:    "success",
					ExitCode: 0,
					Sequence: 1,
				},
				&common.Build{
					CommitID: 3,
					State:    "error",
					ExitCode: 1,
					Sequence: 2,
				},
			}
			commit1 := &common.Commit{
				RepoID: 1,
				State:  common.StateSuccess,
				Ref:    "refs/heads/master",
				Sha:    "14710626f22791619d3b7e9ccf58b10374e5b76d",
				Builds: buildList,
			}
			repo1 := &common.Repo{
				UserID:   1,
				Owner:    "oliveiradan",
				Name:     "drone-test1",
				FullName: "oliveiradan/drone-test1",
			}
			//  GET /api/repos/:owner/:name/:number
			rw := recorder.New()
			ctx := &gin.Context{Engine: gin.Default(), Writer: rw}
			//ctx.Params = append(ctx.Params, gin.Param{Key: "owner", Value: repo1.Owner})
			//ctx.Params = append(ctx.Params, gin.Param{Key: "name", Value: repo1.Name})
			ctx.Params = append(ctx.Params, gin.Param{Key: "number", Value: "1"})
			//
			var err error
			var buf bytes.Buffer
			urlBase := "/api/repos/"
			urlString := (repo1.Owner + "/" + repo1.Name + "/" + "1")
			urlFull := (urlBase + urlString)
			//
			json.NewEncoder(&buf).Encode(commit1)
			httpRequest, _ := http.NewRequest("GET", urlFull, ioutil.NopCloser(&buf))
			httpRequest.Header.Set("Content-Type", "application/json")
			ctx.Request = httpRequest
			//
			ctx.Set("datastore", store)
			ctx.Set("repo", repo1)
			ctx.Set("commit", commit1)
			// Start mock
			getCommit1 := &common.Commit{}
			store.On("CommitSeq", repo1, mock.AnythingOfType("int")).Return(getCommit1, err).Once()
			fmt.Println("commit1: ", getCommit1, " error: ", err)
			g.Assert(err).Equal(nil)
			store.On("BuildList", getCommit1).Return(getCommit1.Builds, err).Once()
			fmt.Println("commit1 builds: ", getCommit1.Builds, " error: ", err)
			g.Assert(err).Equal(nil)
			GetCommit(ctx)
			//
			json.NewDecoder(rw.Body).Decode(getCommit1)
			g.Assert(rw.Code).Equal(200)
			//g.Assert(getCommit1).Equal(commit1)
		})

		g.It("Should get commits", func() {
			//
			buildList := []*common.Build{
				&common.Build{
					CommitID: 1,
					State:    "success",
					ExitCode: 0,
					Sequence: 1,
				},
				&common.Build{
					CommitID: 3,
					State:    "error",
					ExitCode: 1,
					Sequence: 2,
				},
			}
			commit1 := &common.Commit{
				RepoID: 1,
				State:  common.StateSuccess,
				Ref:    "refs/heads/master",
				Sha:    "14710626f22791619d3b7e9ccf58b10374e5b76d",
				Builds: buildList,
			}
			repo1 := &common.Repo{
				UserID:   1,
				Owner:    "oliveiradan",
				Name:     "drone-test1",
				FullName: "oliveiradan/drone-test1",
			}
			// GET /api/repos/:owner/:name/builds
			rw := recorder.New()
			ctx := &gin.Context{Engine: gin.Default(), Writer: rw}
			//
			var err error
			var buf bytes.Buffer
			urlBase := "/api/repos/"
			urlString := (repo1.Owner + "/" + repo1.Name + "/builds")
			urlFull := (urlBase + urlString)
			//
			json.NewEncoder(&buf).Encode(commit1)
			//bodyReader := strings.NewReader(`{}`)
			//httpRequest, _ := http.NewRequest("GET", urlFull, bodyReader)
			httpRequest, _ := http.NewRequest("GET", urlFull, ioutil.NopCloser(&buf))
			httpRequest.Header.Set("Content-Type", "application/json")
			ctx.Request = httpRequest
			//
			ctx.Set("datastore", store)
			ctx.Set("repo", repo1)
			ctx.Set("commit", commit1)
			// Start mock
			getCommits1 := []*common.Commit{}
			store.On("CommitList", repo1, 20, 0).Return(getCommits1, err).Once()
			GetCommits(ctx)
			//
			json.NewDecoder(rw.Body).Decode(getCommits1)
			g.Assert(rw.Code).Equal(200)
			g.Assert(len(getCommits1) != 0)
			//g.Assert(getCommits1).Equal(commit1)
		})

	})
}
