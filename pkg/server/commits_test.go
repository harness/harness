package server

import (
	"bytes"
	//"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/drone/drone/pkg/server/recorder"
	"github.com/drone/drone/pkg/store/mock"
	common "github.com/drone/drone/pkg/types"
	. "github.com/franela/goblin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
	//
	eventbus "github.com/drone/drone/pkg/bus/builtin"
	queue "github.com/drone/drone/pkg/queue/builtin"
	"github.com/drone/drone/pkg/remote/github"
	runner "github.com/drone/drone/pkg/runner/builtin"
	"github.com/drone/drone/pkg/settings"
)

func TestCommits(t *testing.T) {
	store := new(mocks.Store)
	//_url, _ := url.Parse("http://localhost:8080")

	g := Goblin(t)
	g.Describe("Commits", func() {

		g.It("Should get commit", func() {
			//
			repo1 := &common.Repo{
				UserID:   1,
				Owner:    "oliveiradan",
				Name:     "drone-test1",
				FullName: "oliveiradan/drone-test1",
			}
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
				RepoID:   repo1.ID, //1,
				Sequence: 1,
				State:    common.StateSuccess,
				Ref:      "refs/heads/master",
				Sha:      "14710626f22791619d3b7e9ccf58b10374e5b76d",
				Builds:   buildList,
			}
			//  GET /api/repos/:owner/:name/:number
			rw := recorder.New()
			ctx := &gin.Context{Engine: gin.Default(), Writer: rw}
			ctx.Params = append(ctx.Params, gin.Param{Key: "number", Value: "1"})
			//
			urlBase := "/api/repos/"
			urlString := (repo1.Owner + "/" + repo1.Name + "/" + "1")
			urlFull := (urlBase + urlString)
			//
			buf, _ := json.Marshal(&commit1)
			ctx.Request, _ = http.NewRequest("GET", urlFull, bytes.NewBuffer(buf))
			ctx.Request.Header.Set("Content-Type", "application/json")
			//
			ctx.Set("datastore", store)
			ctx.Set("repo", repo1)
			ctx.Set("commit", commit1)
			// Start mock
			store.On("CommitSeq", repo1, mock.AnythingOfType("int")).Return(commit1, nil).Once()
			store.On("BuildList", commit1).Return(commit1.Builds, nil).Once()
			GetCommit(ctx)
			//
			commitOut := &common.Commit{}
			json.Unmarshal(rw.Body.Bytes(), &commitOut)
			g.Assert(rw.Code).Equal(200)
			g.Assert(commitOut.RepoID).Equal(commit1.RepoID)
			g.Assert(commitOut.Sequence).Equal(commit1.Sequence)
			g.Assert(commitOut.Sha).Equal(commit1.Sha)
			g.Assert(len(commitOut.Builds)).Equal(len(commit1.Builds))
		})

		g.It("Should get commits", func() {
			//
			repo1 := &common.Repo{
				ID:       1,
				UserID:   1,
				Owner:    "oliveiradan",
				Name:     "drone-test1",
				FullName: "oliveiradan/drone-test1",
			}
			buildList1 := []*common.Build{
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
			buildList2 := []*common.Build{
				&common.Build{
					CommitID: 5,
					State:    "pending",
					ExitCode: 0,
					Sequence: 3,
				},
				&common.Build{
					CommitID: 7,
					State:    "success",
					ExitCode: 1,
					Sequence: 4,
				},
			}
			commitList1 := []*common.Commit{
				&common.Commit{
					RepoID: repo1.ID,
					State:  common.StateSuccess,
					Ref:    "refs/heads/master",
					Sha:    "14710626f22791619d3b7e9ccf58b10374e5b76d",
					Builds: buildList1,
				},
				&common.Commit{
					RepoID: repo1.ID,
					State:  common.StatePending,
					Ref:    "refs/heads/master",
					Sha:    "7d6621222626298aeb03892d1a40a64b69070e66",
					Builds: buildList2,
				},
			}
			// GET /api/repos/:owner/:name/builds
			rw := recorder.New()
			ctx := &gin.Context{Engine: gin.Default(), Writer: rw}
			//
			urlBase := "/api/repos/"
			urlString := (repo1.Owner + "/" + repo1.Name + "/builds")
			urlFull := (urlBase + urlString)
			//
			buf, _ := json.Marshal(&commitList1)
			ctx.Request, _ = http.NewRequest("GET", urlFull, bytes.NewBuffer(buf))
			ctx.Request.Header.Set("Content-Type", "application/json")
			//
			ctx.Set("datastore", store)
			ctx.Set("repo", repo1)
			// Start mock
			store.On("CommitList", repo1, 20, 0).Return(commitList1, nil).Once()
			GetCommits(ctx)
			//
			commitListOut := []*common.Commit{}
			json.Unmarshal(rw.Body.Bytes(), &commitListOut)
			g.Assert(rw.Code).Equal(200)
			g.Assert(len(commitListOut)).Equal(len(commitList1))
			g.Assert(commitListOut[0].Sha).Equal(commitList1[0].Sha)
			g.Assert(commitListOut[0].Ref).Equal(commitList1[0].Ref)
		})

		g.It("Should get logs", func() {
			//
			repo1 := &common.Repo{
				UserID:   1,
				Owner:    "oliveiradan",
				Name:     "drone-test1",
				FullName: "oliveiradan/drone-test1",
			}
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
				RepoID: repo1.ID, //1,
				State:  common.StateSuccess,
				Ref:    "refs/heads/master",
				Sha:    "14710626f22791619d3b7e9ccf58b10374e5b76d",
				Builds: buildList,
			}
			// GET /api/repos/:owner/:name/logs/:number/:task
			rw := recorder.New()
			ctx := &gin.Context{Engine: gin.Default(), Writer: rw}
			ctx.Params = append(ctx.Params, gin.Param{Key: "full", Value: "true"})
			ctx.Params = append(ctx.Params, gin.Param{Key: "number", Value: "1"})
			ctx.Params = append(ctx.Params, gin.Param{Key: "task", Value: "1"})
			//
			urlBase := "/api/repos/"
			urlString := (repo1.Owner + "/" + repo1.Name + "/logs" + "/1" + "/1")
			urlFull := (urlBase + urlString)
			//
			buf, _ := json.Marshal(&commit1)
			ctx.Request, _ = http.NewRequest("GET", urlFull, bytes.NewBuffer(buf))
			ctx.Request.Header.Set("Content-Type", "application/json")
			//
			ctx.Set("datastore", store)
			ctx.Set("repo", repo1)
			// Start mock
			path := fmt.Sprintf("/logs/%s/%v/%v", repo1.FullName, "1", "1")
			getRC := ioutil.NopCloser(bytes.NewBuffer([]byte("foobar")))
			store.On("GetBlobReader", path).Return(getRC, nil).Once()
			GetLogs(ctx)
			//
			var readerOut io.ReadCloser
			//var readerOut []byte
			json.Unmarshal(rw.Body.Bytes(), &readerOut)
			g.Assert(rw.Code).Equal(200)
			//g.Assert(readerOut).Equal(getRC)
		})

		g.It("Should run build", func() {
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
			service1 := &settings.Service{
				OAuth: &settings.OAuth{
					Client: "87e2bdf99eece72c73c1",
					Secret: "6b4031674ace18723ac41f58d63bff69276e5d1b",
				},
			}
			remote1 := github.New(service1)
			queue1 := queue.New()
			user1 := &common.User{
				Login: "octocat",
				Name:  "octocat octocat",
				Email: "foo@bar.com",
				Token: "b31191cccb023d627d367eb272c10bc4",
			}
			getUrl1, _ := url.Parse("https://github.com")
			netrc1 := &common.Netrc{
				Login:    user1.Token,
				Password: "x-oauth-basic",
				Machine:  getUrl1.Host,
			}
			// POST /api/builds/:owner/:name/builds/:number
			rw := recorder.New()
			ctx := &gin.Context{Engine: gin.Default(), Writer: rw}
			ctx.Params = append(ctx.Params, gin.Param{Key: "number", Value: "1"})
			//
			urlBase := "/api/builds/"
			urlString := (repo1.Owner + "/" + repo1.Name + "/builds" + "/1")
			urlFull := (urlBase + urlString)
			//
			buf, _ := json.Marshal(&commit1)
			ctx.Request, _ = http.NewRequest("POST", urlFull, bytes.NewBuffer(buf))
			ctx.Request.Header.Set("Content-Type", "application/json")
			//
			ctx.Set("datastore", store)
			ctx.Set("repo", repo1)
			ctx.Set("remote", remote1)
			ctx.Set("queue", queue1)
			// Start mock
			store.On("CommitSeq", repo1, mock.AnythingOfType("int")).Return(commit1, nil).Once()
			store.On("BuildList", commit1).Return(commit1.Builds, nil).Once()
			store.On("User", repo1.UserID).Return(user1, nil).Once()
			store.On("SetCommit", commit1).Return(nil).Once()
			store.On("Netrc", user1).Return(netrc1, nil).Once()
			store.On("Script", user1, repo1, commit1).Return([]byte("foo"), nil)
			RunBuild(ctx)
			//
			//var readerOut io.ReadCloser
			var readerOut []byte
			json.Unmarshal(rw.Body.Bytes(), &readerOut)
			g.Assert(rw.Code).Equal(200)

		})

		g.It("Should kill build", func() {
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
			service1 := &settings.Service{
				OAuth: &settings.OAuth{
					Client: "87e2bdf99eece72c73c1",
					Secret: "6b4031674ace18723ac41f58d63bff69276e5d1b",
				},
			}
			remote1 := github.New(service1)
			queue1 := queue.New()
			eventbus1 := eventbus.New()
			updater1 := runner.NewUpdater(eventbus1, store, remote1)
			runner1 := runner.Runner{Updater: updater1}
			// DELETE /api/builds/:owner/:name/builds/:number
			rw := recorder.New()
			ctx := &gin.Context{Engine: gin.Default(), Writer: rw}
			ctx.Params = append(ctx.Params, gin.Param{Key: "number", Value: "1"})
			//
			urlBase := "/api/builds/"
			urlString := (repo1.Owner + "/" + repo1.Name + "/builds" + "/1")
			urlFull := (urlBase + urlString)
			//
			buf, _ := json.Marshal(&commit1)
			ctx.Request, _ = http.NewRequest("DELETE", urlFull, bytes.NewBuffer(buf))
			ctx.Request.Header.Set("Content-Type", "application/json")
			//
			ctx.Set("datastore", store)
			ctx.Set("repo", repo1)
			ctx.Set("remote", remote1)
			ctx.Set("queue", queue1)
			ctx.Set("runner", &runner1)
			// Start mock
			store.On("CommitSeq", repo1, mock.AnythingOfType("int")).Return(commit1, nil).Once()
			store.On("BuildList", commit1).Return(commit1.Builds, nil).Once()
			store.On("SetCommit", commit1).Return(nil).Once()
			KillBuild(ctx)
			//
			var readerOut []byte
			json.Unmarshal(rw.Body.Bytes(), &readerOut)
			g.Assert(rw.Code).Equal(200)
			//g.Assert(src)
		})
	})

}
