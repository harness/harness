package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	//"time"
	//"io"
	//"io/ioutil"
	"net/http"
	//"net/url"
	"testing"

	"github.com/drone/drone/pkg/config"
	//"github.com/drone/drone/pkg/remote"
	//"github.com/drone/drone/pkg/oauth2"
	"github.com/drone/drone/pkg/remote/builtin/github"
	"github.com/drone/drone/pkg/server/recorder"
	"github.com/drone/drone/pkg/server/session"
	"github.com/drone/drone/pkg/store/mock"

	//ithub.com/drone/drone/Godeps/_workspace/src/github.com/dgrijalva/jwt-go"
	. "github.com/drone/drone/Godeps/_workspace/src/github.com/franela/goblin"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/gin-gonic/gin"
	//"github.com/drone/drone/Godeps/_workspace/src/github.com/stretchr/testify/mock"
	//"github.com/drone/drone/Godeps/_workspace/src/gopkg.in/yaml.v2"

	//eventbus "github.com/drone/drone/pkg/bus/builtin"
	//queue "github.com/drone/*common.Repodrone/pkg/queue/builtin"
	//runner "github.com/drone/drone/pkg/runner/builtin"
	common "github.com/drone/drone/pkg/types"
)

var putRepoTestTbl = []struct {
	*common.Repo
	repoBranch string
	repoErr    error
	repoReturn int
	//test []byte
}{}

func TestRepos(t *testing.T) {
	store := new(mocks.Store)
	//
	g := Goblin(t)
	g.Describe("Repos", func() {

		g.It("Should get repo", func() {
			//
			user1 := &common.User{
				ID:    1,
				Login: "oliveiradan",
				Name:  "Daniel Oliveira",
				Email: "doliveirabrz@gmail.com",
				Token: "e1c372bc477d38972c54b1794bdf3932",
			}
			repo1 := &common.Repo{
				ID:       1,
				UserID:   1,
				Owner:    "oliveiradan",
				Name:     "drone-test1",
				FullName: "oliveiradan/drone-test1",
			}
			perm1 := &common.Perm{
				Admin: true,
				Pull:  true,
				Push:  true,
			}
			type repoResp struct {
				*common.Repo
				Perms *common.Perm `json:"permissions,omitempty"`

				Keypair *common.Keypair   `json:"keypair,omitempty"`
				Params  map[string]string `json:"params,omitempty"`
				Starred bool              `json:"starred,omitempty"`
			}
			data1 := repoResp{repo1, perm1, nil, nil, true}

			// GET /api/repos/:owner/:name
			rw := recorder.New()
			ctx := &gin.Context{Engine: gin.Default(), Writer: rw}
			//ctx.Params = append(ctx.Params, gin.Param{Key: "number", Value: "1"})
			//
			urlBase := "api/repos/"
			urlString := (repo1.Owner + "/" + repo1.Name)
			urlFull := (urlBase + urlString)
			//
			buf, _ := json.Marshal(&data1)
			ctx.Request, _ = http.NewRequest("GET", urlFull, bytes.NewBuffer(buf))
			ctx.Request.Header.Set("Content-Type", "application/json")
			//
			ctx.Set("datastore", store)
			ctx.Set("repo", repo1)
			ctx.Set("user", user1)
			ctx.Set("perm", perm1)

			// Start mock
			store.On("Starred", user1, repo1).Return(data1.Starred, nil).Once()
			GetRepo(ctx)

			//
			var readerOut repoResp //[]byte
			json.Unmarshal(rw.Body.Bytes(), &readerOut)
			fmt.Println("Reader: ", readerOut)
			g.Assert(rw.Code).Equal(200)
			g.Assert(readerOut.Repo.ID).Equal(repo1.ID)
			g.Assert(readerOut.Perms.Admin).Equal(perm1.Admin)
			g.Assert(readerOut.Perms.Pull).Equal(perm1.Pull)
		})

		g.It("Should put repo", func() {
			//
			user1 := &common.User{
				ID:    1,
				Login: "oliveiradan",
				Name:  "Daniel Oliveira",
				Email: "doliveirabrz@gmail.com",
				Token: "e1c372bc477d38972c54b1794bdf3932",
			}
			repo1 := &common.Repo{
				UserID:   1,
				Owner:    "oliveiradan",
				Name:     "drone-test1",
				FullName: "oliveiradan/drone-test1",
			}
			perm1 := &common.Perm{
				Admin: true,
				Pull:  true,
				Push:  true,
			}
			type repoResp struct {
				*common.Repo
				Perms   *common.Perm      `json:"permissions,omitempty"`
				Keypair *common.Keypair   `json:"keypair,omitempty"`
				Params  map[string]string `json:"params,omitempty"`
				Starred bool              `json:"starred,omitempty"`
			}
			data1 := repoResp{repo1, perm1, nil, nil, true}

			// PUT /api/repos/:owner/:name
			rw := recorder.New()
			ctx := &gin.Context{Engine: gin.Default(), Writer: rw}
			//ctx.Params = append(ctx.Params, gin.Param{Key: "number", Value: "1"})
			//
			urlBase := "api/repos/"
			urlString := (repo1.Owner + "/" + repo1.Name)
			urlFull := (urlBase + urlString)
			//
			buf, _ := json.Marshal(&data1)
			ctx.Request, _ = http.NewRequest("PUT", urlFull, bytes.NewBuffer(buf))
			ctx.Request.Header.Set("Content-Type", "application/json")
			//
			ctx.Set("datastore", store)
			ctx.Set("repo", repo1)
			ctx.Set("user", user1)
			ctx.Set("perm", perm1)

			// Start mock
			store.On("SetRepo", repo1).Return(nil).Once()
			store.On("Starred", user1, repo1).Return(data1.Starred, nil).Once()
			PutRepo(ctx)

			//
			var readerOut repoResp //[]byte
			json.Unmarshal(rw.Body.Bytes(), &readerOut)
			fmt.Println("Reader: ", readerOut)
			g.Assert(rw.Code).Equal(200)
			g.Assert(readerOut.Repo.ID).Equal(repo1.ID)
			g.Assert(readerOut.Perms.Admin).Equal(perm1.Admin)
			g.Assert(readerOut.Perms.Pull).Equal(perm1.Pull)
		})

		g.It("Should delete repo", func() {
			//
			user1 := &common.User{
				ID:    1,
				Login: "oliveiradan",
				Name:  "Daniel Oliveira",
				Email: "doliveirabrz@gmail.com",
				Token: "e1c372bc477d38972c54b1794bdf3932",
			}
			repo1 := &common.Repo{
				UserID:   1,
				Owner:    "oliveiradan",
				Name:     "drone-test1",
				FullName: "oliveiradan/drone-test1",
			}
			config1 := &config.Config{}
			config1.Auth.Client = "87e2bdf99eece72c73c1"
			config1.Auth.Secret = "6b4031674ace18723ac41f58d63bff69276e5d1b"
			remote1 := github.New(config1)

			// DEL /api/repos/:owner/:name
			rw := recorder.New()
			ctx := &gin.Context{Engine: gin.Default(), Writer: rw}
			//ctx.Params = append(ctx.Params, gin.Param{Key: "number", Value: "1"})
			//
			urlBase := "api/repos/"
			urlString := (repo1.Owner + "/" + repo1.Name)
			urlFull := (urlBase + urlString)
			//
			buf, _ := json.Marshal(&repo1)
			ctx.Request, _ = http.NewRequest("DELETE", urlFull, bytes.NewBuffer(buf))
			ctx.Request.Header.Set("Content-Type", "application/json")
			//
			ctx.Set("datastore", store)
			ctx.Set("repo", repo1)
			ctx.Set("user", user1)
			ctx.Set("remote", remote1)
			// Start mock
			store.On("DelRepo", repo1).Return(nil).Once()
			DeleteRepo(ctx)

			// as we don't an existing repo, we should have an error.
			if (rw.Status()) != 0 {
				g.Assert(rw.Code).Equal(400)
			} else {
				g.Assert(rw.Code).Equal(200)
			}
			//
			//var readerOut []byte
			readerOut := &common.Repo{} //[]byte
			json.Unmarshal(rw.Body.Bytes(), &readerOut)
			//fmt.Println("Reader: ", readerOut)
		})

		g.It("Should post repo", func() {
			//
			user1 := &common.User{
				ID:    1,
				Login: "oliveiradan",
				Name:  "Daniel Oliveira",
				Email: "doliveirabrz@gmail.com",
				Token: "e1c372bc477d38972c54b1794bdf3932",
			}
			repo1 := &common.Repo{
				UserID:   1,
				Owner:    "oliveiradan",
				Name:     "drone-test1",
				FullName: "oliveiradan/drone-test1",
			}
			config1 := &config.Config{}
			config1.Auth.Client = "87e2bdf99eece72c73c1"
			config1.Auth.Secret = "6b4031674ace18723ac41f58d63bff69276e5d1b"
			remote1 := github.New(config1)
			config1.Session.Secret = "Otto"
			session1 := session.New(config1)

			// POST /api/repos/:owner/:name
			rw := recorder.New()
			ctx := &gin.Context{Engine: gin.Default(), Writer: rw}
			ctx.Params = append(ctx.Params, gin.Param{Key: "owner", Value: repo1.Owner})
			ctx.Params = append(ctx.Params, gin.Param{Key: "name", Value: repo1.Name})

			//
			urlBase := "api/repos/"
			urlString := (repo1.Owner + "/" + repo1.Name)
			urlFull := (urlBase + urlString)
			//
			buf, _ := json.Marshal(&repo1)
			ctx.Request, _ = http.NewRequest("POST", urlFull, bytes.NewBuffer(buf))
			ctx.Request.Header.Set("Content-Type", "application/json")
			//
			ctx.Set("datastore", store)
			ctx.Set("repo", repo1)
			ctx.Set("user", user1)
			ctx.Set("session", session1)
			ctx.Set("remote", remote1)
			// Start mock
			store.On("RepoName", repo1.Owner, repo1.Owner).Return(nil, nil).Once()
			store.On("AddRepo", repo1).Return(nil).Once()
			store.On("AddStar", user1, repo1).Return(nil).Once()
			PostRepo(ctx)

			// as we don't an existing repo, we should have an error.
			if (rw.Status()) != 0 {
				g.Assert(rw.Code).Equal(400)
			} else {
				g.Assert(rw.Code).Equal(200)
			}
			//var readerOut []byte
			readerOut := &common.Repo{} //[]byte
			json.Unmarshal(rw.Body.Bytes(), &readerOut)
			//fmt.Println("Reader: ", readerOut)
		})

		g.It("Should unsubscribe", func() {
			//
			user1 := &common.User{
				ID:    1,
				Login: "oliveiradan",
				Name:  "Daniel Oliveira",
				Email: "doliveirabrz@gmail.com",
				Token: "e1c372bc477d38972c54b1794bdf3932",
			}
			repo1 := &common.Repo{
				UserID:   1,
				Owner:    "oliveiradan",
				Name:     "drone-test1",
				FullName: "oliveiradan/drone-test1",
			}

			// DEL /api/subscribers/:owner/:name
			rw := recorder.New()
			ctx := &gin.Context{Engine: gin.Default(), Writer: rw}

			//
			urlBase := "api/subscribers/"
			urlString := (repo1.Owner + "/" + repo1.Name)
			urlFull := (urlBase + urlString)
			//
			buf, _ := json.Marshal(&repo1)
			ctx.Request, _ = http.NewRequest("DELETE", urlFull, bytes.NewBuffer(buf))
			ctx.Request.Header.Set("Content-Type", "application/json")
			//
			ctx.Set("datastore", store)
			ctx.Set("repo", repo1)
			ctx.Set("user", user1)
			// Start mock
			store.On("DelStar", user1, repo1).Return(nil, nil).Once()
			Unsubscribe(ctx)

			//--
			g.Assert(rw.Code).Equal(200)
			//var readerOut []byte
			readerOut := &common.Repo{}
			json.Unmarshal(rw.Body.Bytes(), &readerOut)
			//fmt.Println("Reader: ", readerOut)
		})

		g.It("Should subscribe", func() {
			//
			user1 := &common.User{
				ID:    1,
				Login: "oliveiradan",
				Name:  "Daniel Oliveira",
				Email: "doliveirabrz@gmail.com",
				Token: "e1c372bc477d38972c54b1794bdf3932",
			}
			repo1 := &common.Repo{
				UserID:   1,
				Owner:    "oliveiradan",
				Name:     "drone-test1",
				FullName: "oliveiradan/drone-test1",
			}

			// DEL /api/subscribers/:owner/:name
			rw := recorder.New()
			ctx := &gin.Context{Engine: gin.Default(), Writer: rw}

			//
			urlBase := "api/subscribers/"
			urlString := (repo1.Owner + "/" + repo1.Name)
			urlFull := (urlBase + urlString)
			//
			buf, _ := json.Marshal(&repo1)
			ctx.Request, _ = http.NewRequest("DELETE", urlFull, bytes.NewBuffer(buf))
			ctx.Request.Header.Set("Content-Type", "application/json")
			//
			ctx.Set("datastore", store)
			ctx.Set("repo", repo1)
			ctx.Set("user", user1)
			// Start mock
			store.On("AddStar", user1, repo1).Return(nil, nil).Once()
			Subscribe(ctx)

			//
			g.Assert(rw.Code).Equal(200)
			//var readerOut []byte
			readerOut := &common.Repo{}
			json.Unmarshal(rw.Body.Bytes(), &readerOut)
			//fmt.Println("Reader: ", readerOut)
		})

	})
}
