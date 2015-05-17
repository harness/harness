package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/drone/drone/pkg/server/recorder"
	"github.com/drone/drone/pkg/store/mock"
	common "github.com/drone/drone/pkg/types"
	. "github.com/franela/goblin"
	"github.com/gin-gonic/gin"
)

func TestUser(t *testing.T) {
	store := new(mocks.Store)

	g := Goblin(t)
	g.Describe("User", func() {

		g.It("should get", func() {
			rw := recorder.New()
			ctx := &gin.Context{Engine: gin.Default(), Writer: rw}

			user := &common.User{Login: "octocat"}
			ctx.Set("user", user)

			GetUserCurr(ctx)

			out := &common.User{}
			json.NewDecoder(rw.Body).Decode(out)
			g.Assert(rw.Code).Equal(200)
			g.Assert(out).Equal(user)
		})

		g.It("should put", func() {
			var buf bytes.Buffer
			in := &common.User{Email: "octocat@github.com"}
			json.NewEncoder(&buf).Encode(in)

			rw := recorder.New()
			ctx := &gin.Context{Engine: gin.Default(), Writer: rw}
			ctx.Request = &http.Request{Body: ioutil.NopCloser(&buf)}
			ctx.Request.Header = http.Header{}
			ctx.Request.Header.Set("Content-Type", "application/json")

			user := &common.User{Login: "octocat"}
			ctx.Set("user", user)
			ctx.Set("datastore", store)
			store.On("SetUser", user).Return(nil).Once()

			PutUserCurr(ctx)

			out := &common.User{}
			json.NewDecoder(rw.Body).Decode(out)
			g.Assert(rw.Code).Equal(200)
			g.Assert(out.Login).Equal(user.Login)
			g.Assert(out.Email).Equal(in.Email)
			g.Assert(out.Gravatar).Equal("7194e8d48fa1d2b689f99443b767316c")
		})

		g.It("should put, error", func() {
			var buf bytes.Buffer
			in := &common.User{Email: "octocat@github.com"}
			json.NewEncoder(&buf).Encode(in)

			rw := recorder.New()
			ctx := &gin.Context{Engine: gin.Default(), Writer: rw}
			ctx.Request = &http.Request{Body: ioutil.NopCloser(&buf)}
			ctx.Request.Header = http.Header{}
			ctx.Request.Header.Set("Content-Type", "application/json")

			user := &common.User{Login: "octocat"}
			ctx.Set("user", user)
			ctx.Set("datastore", store)
			store.On("SetUser", user).Return(errors.New("error")).Once()

			PutUserCurr(ctx)

			out := &common.User{}
			json.NewDecoder(rw.Body).Decode(out)
			g.Assert(rw.Code).Equal(400)
		})
	})
}
