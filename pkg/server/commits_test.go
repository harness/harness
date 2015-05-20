package server

import (
	"bytes"
	//"database/sql"
	"encoding/json"
	//"encoding/xml"
	//"errors"
	"net/http"
	//"net/url"
	"io/ioutil"
	"testing"

	//	"github.com/drone/drone/common"
	//"github.com/drone/drone/pkg/ccmenu"
	"github.com/drone/drone/pkg/server/recorder"
	"github.com/drone/drone/pkg/store/mock"
	common "github.com/drone/drone/pkg/types"
	. "github.com/franela/goblin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
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
			ctx.Params = append(ctx.Params, gin.Param{Key: "owner", Value: repo1.Owner})
			ctx.Params = append(ctx.Params, gin.Param{Key: "name", Value: repo1.Name})
			ctx.Params = append(ctx.Params, gin.Param{Key: "number", Value: "1"})

			//body := bytes.NewBufferString("/:" + repo1.Owner + "/:" + repo1.Name + "/:" + "1")
			//body := bytes.NewBufferString("{}")
			var buf bytes.Buffer
			json.NewEncoder(&buf).Encode(commit1)
			//ctx.Request, _ = http.NewRequest("POST", "/api/repos", body)
			ctx.Request, _ = http.NewRequest("POST", "/api/repos", ioutil.NopCloser(&buf))
			//ctx.Request = &http.Request{Body: ioutil.NopCloser(&buf)}

			//
			ctx.Set("datastore", store)
			ctx.Set("repo", repo1)
			ctx.Set("commit", commit1)
			// Start mock
			getCommit1 := &common.Commit{}
			getBuildList1 := &[]common.Build{}
			err := 0
			//mock.AnythingOfType("int")
			//store.On("CommitSeq", repo1, "1").Return(getCommit1, 200).Once()
			//store.On("CommitSeq", repo1, mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(getCommit1, nil).Once()
			store.On("CommitSeq", repo1, mock.AnythingOfType("int")).Return(getCommit1, err).Once()
			store.On("BuildList", getCommit1).Return(getBuildList1, err).Once()
			GetCommit(ctx)
			//
			json.NewDecoder(rw.Body).Decode(getCommit1)
			g.Assert(rw.Code).Equal(200)
			g.Assert(getCommit1).Equal(commit1)
		})

	})
}
