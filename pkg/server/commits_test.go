package server

import (
	"bytes"
	//"strings"
	//"database/sql"
	"encoding/json"
	"io"
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
			//getCommit1 := &common.Commit{}
			fmt.Println("TEST: ", err)
			store.On("CommitSeq", repo1, mock.AnythingOfType("int")).Return(commit1, nil).Once()
			store.On("BuildList", commit1).Return(commit1.Builds, nil).Once()
			GetCommit(ctx)
			//
			commitOut := &common.Commit{}
			json.NewDecoder(rw.Body).Decode(commitOut)
			g.Assert(rw.Code).Equal(200)
			//g.Assert(getCommit1).Equal(commit1)
		})

		g.It("Should get commits", func() {
			//
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
			commit1 := &common.Commit{
				RepoID: 1,
				State:  common.StateSuccess,
				Ref:    "refs/heads/master",
				Sha:    "14710626f22791619d3b7e9ccf58b10374e5b76d",
				Builds: buildList1,
			}
			commit2 := &common.Commit{
				RepoID: 1,
				State:  common.StatePending,
				Ref:    "refs/heads/master",
				Sha:    "7d6621222626298aeb03892d1a40a64b69070e66",
				Builds: buildList2,
			}
			repo1 := &common.Repo{
				ID:       1,
				UserID:   1,
				Owner:    "oliveiradan",
				Name:     "drone-test1",
				FullName: "oliveiradan/drone-test1",
			}
			commitList1 := make([]*common.Commit, 2)
			commitList1[0] = commit1
			commitList1[1] = commit2

			// GET /api/repos/:owner/:name/builds
			rw := recorder.New()
			ctx := &gin.Context{Engine: gin.Default(), Writer: rw}
			//
			//var err error
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
			store.On("CommitList", repo1, 20, 0).Return(commitList1, nil).Once()
			GetCommits(ctx)
			//
			commitListOut := []*common.Commit{}
			json.NewDecoder(rw.Body).Decode(commitListOut)
			g.Assert(rw.Code).Equal(200)
			//g.Assert(len(getCommits1) != 0)
			//g.Assert(getCommits1).Equal(commit1)
		})

		g.It("Should get logs", func() {
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
			// GET /api/repos/:owner/:name/logs/:number/:task
			rw := recorder.New()
			ctx := &gin.Context{Engine: gin.Default(), Writer: rw}
			ctx.Params = append(ctx.Params, gin.Param{Key: "full", Value: "true"})
			ctx.Params = append(ctx.Params, gin.Param{Key: "number", Value: "1"})
			ctx.Params = append(ctx.Params, gin.Param{Key: "task", Value: "1"})
			//
			var err error
			var buf bytes.Buffer
			urlBase := "/api/repos/"
			urlString := (repo1.Owner + "/" + repo1.Name + "/logs" + "/1" + "/1")
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
			path := fmt.Sprintf("/logs/%s/%v/%v", repo1.FullName, "1", "1")
			//store.SetBlob(path, []byte("foobar"))
			var getRC io.ReadCloser = ioutil.NopCloser(bytes.NewBufferString("foobar"))
			store.On("GetBlobReader", path).Return(getRC, nil).Once()
			GetLogs(ctx)
			fmt.Println("err: ", err)
			//
			//json.NewDecoder(rw.Body).Decode(getReader)
			//g.Assert(rw.Code).Equal(200)
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
			//service2 := settings.Settings{Service: &settings.Service{OAuth.Client: "87e2bdf99eece72c73c1"},
			//	&settings.Service{OAuth.Secret: "6b4031674ace18723ac41f58d63bff69276e5d1b"},
			//}
			service1 := &settings.Service{
				OAuth.Client: "87e2bdf99eece72c73c1",
				OAuth.Secret: "6b4031674ace18723ac41f58d63bff69276e5d1b",
			}
			remote1 := github.New(service1)
			queue1 := queue.New()

			// POST /api/builds/:owner/:name/builds/:number
			rw := recorder.New()
			ctx := &gin.Context{Engine: gin.Default(), Writer: rw}
			ctx.Params = append(ctx.Params, gin.Param{Key: "number", Value: "1"})
			//
			var err error
			var buf bytes.Buffer
			urlBase := "/api/builds/"
			urlString := (repo1.Owner + "/" + repo1.Name + "/builds" + "/1")
			urlFull := (urlBase + urlString)
			//
			json.NewEncoder(&buf).Encode(commit1)
			//bodyReader := strings.NewReader(`{}`)
			//httpRequest, _ := http.NewRequest("GET", urlFull, bodyReader)
			httpRequest, _ := http.NewRequest("POST", urlFull, ioutil.NopCloser(&buf))
			httpRequest.Header.Set("Content-Type", "application/json")
			ctx.Request = httpRequest
			//
			ctx.Set("datastore", store)
			ctx.Set("repo", repo1)
			ctx.Set("remote", remote1)
			ctx.Set("queue", queue1)

			//ctx.Set("settings", &config)
			//ctx.Set("session", session.New(config.Session))
			//ctx.Set("queue", queue1)
			//ctx.Set("remote" remote1)
			// Start mock
			////path := fmt.Sprintf("/logs/%s/%v/%v", repo1.FullName, "1", "1")
			//store.SetBlob(path, []byte("foobar"))
			//var getRC io.ReadCloser = ioutil.NopCloser(bytes.NewBufferString("foobar"))
			//store.On("GetBlobReader", path).Return(getRC, nil).Once()
			//GetLogs(ctx)
			fmt.Println("err: ", err)
			//
			//json.NewDecoder(rw.Body).Decode(getReader)
			//g.Assert(rw.Code).Equal(200)

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
				OAuth.Client: "87e2bdf99eece72c73c1",
				OAuth.Secret: "6b4031674ace18723ac41f58d63bff69276e5d1b",
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
			var err error
			var buf bytes.Buffer
			urlBase := "/api/builds/"
			urlString := (repo1.Owner + "/" + repo1.Name + "/builds" + "/1")
			urlFull := (urlBase + urlString)
			//
			json.NewEncoder(&buf).Encode(commit1)
			//bodyReader := strings.NewReader(`{}`)
			httpRequest, _ := http.NewRequest("DELETE", urlFull, ioutil.NopCloser(&buf))
			httpRequest.Header.Set("Content-Type", "application/json")
			ctx.Request = httpRequest
			//
			ctx.Set("datastore", store)
			ctx.Set("repo", repo1)
			ctx.Set("remote", remote1)
			ctx.Set("queue", queue1)
			ctx.Set("runner", runner1)

			// Start mock
			//path := fmt.Sprintf("/logs/%s/%v/%v", repo1.FullName, "1", "1")
			//store.SetBlob(path, []byte("foobar"))
			//var getRC io.ReadCloser = ioutil.NopCloser(bytes.NewBufferString("foobar"))
			//store.On("GetBlobReader", path).Return(getRC, nil).Once()
			//GetLogs(ctx)
			fmt.Println("err: ", err)
			//
			//json.NewDecoder(rw.Body).Decode(getReader)
			//g.Assert(rw.Code).Equal(200)
		})
	})

}
