package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	//"net/url"
	"testing"
	//"time"

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
	"github.com/drone/drone/Godeps/_workspace/src/gopkg.in/yaml.v2"

	//eventbus "github.com/drone/drone/pkg/bus/builtin"
	//queue "github.com/drone/drone/pkg/queue/builtin"
	//runner "github.com/drone/drone/pkg/runner/builtin"
	common "github.com/drone/drone/pkg/types"
)

func TestLogin(t *testing.T) {
	store := new(mocks.Store)
	//
	g := Goblin(t)
	g.Describe("Login", func() {

		g.It("Should get login", func() {
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
			config1.Auth.RequestToken = "TESTING" //Which will fall into getLoginOauth2
			remote1 := github.New(config1)
			//hook1 := &common.Hook{
			//	Repo:   repo1,
			//	Commit: commit1,
			//}
			config1.Session.Secret = "Oliv"
			session1 := session.New(config1)
			//token1 := &common.Token{
			//	Kind:   common.TokenUser, //.TokenSess, //.TokenHook,
			//	Login:  user1.Login,
			//	Label:  hook1.Repo.FullName,
			//	UserID: user1.ID,
			//	Issued: time.Now().UTC().Unix(),
			//}
			tokenstr1 := "0123456789ABCDEF"
			//getUrl1, _ := url.Parse("https://github.com")
			//netrc1 := &common.Netrc{
			//	Login:    user1.Token,
			//	Password: "x-oauth-basic",
			//	Machine:  getUrl1.Host,
			//}
			fakeYMLFile := fmt.Sprintf(`[{"type": "file",
"encoding": "base64",
"size": 5362,
"name": "README.md",
"path": "README.md",
"content": "encoded content ...",
"sha": "3d21ec53a331a6f037a91c368710b99387d012c1",
"url": "https://api.github.com/repos/octokit/octokit.rb/contents/README.md",
"git_url": "https://api.github.com/repos/octokit/octokit.rb/git/blobs/3d21ec53a331a6f037a91c368710b99387d012c1",
"html_url": "https://github.com/octokit/octokit.rb/blob/master/README.md",
"download_url": "https://raw.githubusercontent.com/octokit/octokit.rb/master/README.md",
"_links": {
"git": "https://api.github.com/repos/octokit/octokit.rb/git/blobs/3d21ec53a331a6f037a91c368710b99387d012c1",
"self": "https://api.github.com/repos/octokit/octokit.rb/contents/README.md",
"html": "https://github.com/octokit/octokit.rb/blob/master/README.md",
"owner": "oliveiradan",
"Name":  "drone-test1"
}
}]`)
			bufYMLFile, _ := json.Marshal(&fakeYMLFile)
			//
			type Matrix map[string][]string
			mtxData1 := struct {
				Matrix map[string][]string
			}{}
			//err :=
			yaml.Unmarshal([]byte(bufYMLFile), &mtxData1)
			parseCond1 := struct {
				Condition *common.Condition `yaml:"when"`
			}{}
			//err =
			yaml.Unmarshal([]byte(bufYMLFile), &parseCond1)

			// GET /authorize
			rw := recorder.New()
			ctx := &gin.Context{Engine: gin.Default(), Writer: rw}
			//
			urlBase := "/authorize"
			//urlString := (repo1.Owner + "/" + repo1.Name + "/builds" + "/1")
			urlFull := urlBase //(urlBase + urlString)
			//
			buf, _ := json.Marshal(&user1)
			ctx.Request, _ = http.NewRequest("GET", urlFull, bytes.NewBuffer(buf))
			ctx.Request.Header.Set("Content-Type", "application/json")
			//
			ctx.Set("datastore", store)
			ctx.Set("settings", config1)
			ctx.Set("session", session1)
			ctx.Set("remote", remote1)
			ctx.Set("login", user1)
			// Start mock
			store.On("UserLogin", user1.Login).Return(user1, nil).Once()
			store.On("SetUser", user1).Return(nil).Once()
			fmt.Println("file: ", bufYMLFile)
			fmt.Println("tokenstr1: ", tokenstr1)
			//GetLogin(ctx)
			fmt.Println("commit1: ", commit1, "repo1: ", repo1)
			//
			//var readerOut []byte
			//json.Unmarshal(rw.Body.Bytes(), &readerOut)
			//fmt.Println("Reader: ", readerOut)
			g.Assert(rw.Code).Equal(200)
			//fmt.Println("tokenstr1: ", tokenstr1)
			////var respjson map[string]interface{}
			////json.Unmarshal(rw.Body.Bytes(), &respjson)
			////g.Assert(respjson["kind"]).Equal(types.TokenUser)
			////g.Assert(respjson["label"]).Equal(test.inLabel)
		})
	})
}
