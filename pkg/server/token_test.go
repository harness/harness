package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/dgrijalva/jwt-go"
	. "github.com/drone/drone/Godeps/_workspace/src/github.com/franela/goblin"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/gin-gonic/gin"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/stretchr/testify/mock"
	"github.com/drone/drone/pkg/config"
	"github.com/drone/drone/pkg/server/recorder"
	"github.com/drone/drone/pkg/server/session"
	"github.com/drone/drone/pkg/store/mock"
	"github.com/drone/drone/pkg/types"
)

var createTests = []struct {
	inLabel  string
	inBody   string
	storeErr error
	outCode  int
	outKind  string
}{
	{"", `{}`, sql.ErrNoRows, 500, ""},
	{"app1", `{"label": "app1"}`, nil, 200, types.TokenUser},
	{"app2", `{"label": "app2"}`, nil, 200, types.TokenUser},
}

var deleteTests = []struct {
	inLabel       string
	errTokenLabel error
	errDelToken   error
	outCode       int
	outToken      *types.Token
}{
	{"app1", sql.ErrNoRows, nil, 404, &types.Token{}},
	{"app2", nil, sql.ErrNoRows, 400, &types.Token{Label: "app2"}},
	{"app3", nil, nil, 200, &types.Token{Label: "app2"}},
}

func TestToken(t *testing.T) {
	store := new(mocks.Store)

	g := Goblin(t)
	g.Describe("Token", func() {

		// POST /api/user/tokens
		g.It("should create tokens", func() {
			for _, test := range createTests {
				rw := recorder.New()
				ctx := gin.Context{Engine: gin.Default(), Writer: rw}
				body := bytes.NewBufferString(test.inBody)
				ctx.Request, _ = http.NewRequest("POST", "/api/user/tokens", body)

				ctx.Set("datastore", store)
				ctx.Set("user", &types.User{Login: "Freya"})

				conf := &config.Config{}
				conf.Session.Secret = "Otto"
				ctx.Set("settings", conf)
				ctx.Set("session", session.New(conf))

				// prepare the mock
				store.On("AddToken", mock.AnythingOfType("*types.Token")).Return(test.storeErr).Once()
				PostToken(&ctx)

				g.Assert(rw.Code).Equal(test.outCode)
				if test.outCode != 200 {
					continue
				}

				var respjson map[string]interface{}
				json.Unmarshal(rw.Body.Bytes(), &respjson)
				g.Assert(respjson["kind"]).Equal(types.TokenUser)
				g.Assert(respjson["label"]).Equal(test.inLabel)

				// this is probably going too far... maybe just validate hash is not empty?
				jwt.Parse(respjson["hash"].(string), func(token *jwt.Token) (interface{}, error) {
					_, ok := token.Method.(*jwt.SigningMethodHMAC)
					g.Assert(ok).IsTrue()
					g.Assert(token.Claims["label"]).Equal(test.inLabel)
					return nil, nil
				})
			}
		})

		// DELETE /api/user/tokens/:label
		g.It("should delete tokens", func() {
			for _, test := range deleteTests {
				rw := recorder.New()
				ctx := gin.Context{Engine: gin.Default(), Writer: rw}
				ctx.Params = append(ctx.Params, gin.Param{Key: "label", Value: test.inLabel})

				ctx.Set("datastore", store)
				ctx.Set("user", &types.User{Login: "Freya"})

				conf := &config.Config{}
				conf.Session.Secret = "Otto"
				ctx.Set("settings", conf)
				ctx.Set("session", session.New(conf))

				// prepare the mock
				store.On("TokenLabel", mock.AnythingOfType("*types.User"), test.inLabel).Return(test.outToken, test.errTokenLabel).Once()

				if test.errTokenLabel == nil {
					// we don't need this expectation if we error on our first
					store.On("DelToken", mock.AnythingOfType("*types.Token")).Return(test.errDelToken).Once()
				}
				fmt.Println(test)
				DelToken(&ctx)

				g.Assert(rw.Code).Equal(test.outCode)
				if test.outCode != 200 {
					continue
				}

				var respjson map[string]interface{}
				json.Unmarshal(rw.Body.Bytes(), &respjson)
				fmt.Println(rw.Code, respjson)
			}
		})
	})
}
