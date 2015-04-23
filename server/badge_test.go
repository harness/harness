package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/drone/drone/common"
	"github.com/drone/drone/datastore/mock"
	"github.com/drone/drone/settings"
	. "github.com/franela/goblin"
	"github.com/gin-gonic/gin"
)

func TestBadage(t *testing.T) {
	g := Goblin(t)
	g.Describe("Badge", func() {
		repo := "Freya/Hello-World"
		token := common.Token{
			Kind:   "github",
			Login:  "Freya",
			Label:  "github",
			Repos:  []string{"Freya/Hello-World"},
			Scopes: []string{},
			Expiry: 0,
			Issued: 0,
		}

		var r *gin.Engine
		g.BeforeEach(func() {
			ds := new(mock.Datastore)
			ds.Tokens = map[string]*common.Token{
				(token.Login + "/" + token.Label): &token,
			}

			settings := &settings.Settings{
				Session: &settings.Session{
					Secret:  "",
					Expires: 3600,
				},
				Service: &settings.Service{
					Name:  "github",
					Base:  "https://github.com",
					OAuth: &settings.OAuth{},
				},
			}

			r = gin.New()
			api := r.Group("/api")
			{
				api.Use(SetHeaders())
				api.Use(SetDatastore(ds))
				api.Use(SetSettings(settings))
				api.Use(SetUser(MockSession{Token: &token}))

				badges := api.Group("/badges/:owner/:name")
				{
					badges.Use(SetRepo())

					badges.GET("/status.svg", GetBadge)
					badges.GET("/cc.xml", GetCC)
				}
			}
		})

		g.AfterEach(func() {
		})

		g.It("should provide SVG", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/badges/"+repo+"/status.svg", nil)
			r.ServeHTTP(w, req)

			// TODO(benschumacher) expand this a lot. ;)
			g.Assert(w.Code).Equal(200)
		})
		g.It("should provide CCTray response")
	})
}
