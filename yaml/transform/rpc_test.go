package transform

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/drone/drone/yaml"
	"github.com/franela/goblin"
)

func handleNetrcRemoval(w http.ResponseWriter, r *http.Request) {
	c := new(yaml.Config)
	err := json.NewDecoder(r.Body).Decode(c)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	for _, container := range c.Pipeline {
		if strings.HasPrefix(container.Image, "plugins/git") {
			continue
		}
		container.Environment["DRONE_NETRC_USERNAME"] = ""
		container.Environment["DRONE_NETRC_PASSWORD"] = ""
		container.Environment["DRONE_NETRC_MACHINE"] = ""
	}
	json.NewEncoder(w).Encode(c)
}

func Test_rpc_transform(t *testing.T) {
	g := goblin.Goblin(t)
	g.Describe("rpc transform", func() {

		g.It("should mutate the yaml", func() {
			c := newConfig(&yaml.Container{
				Image: "golang",
				Environment: map[string]string{
					"DRONE_NETRC_USERNAME": "foo",
					"DRONE_NETRC_PASSWORD": "bar",
					"DRONE_BRANCH":         "master",
				},
				Commands: []string{
					"go build",
					"go test",
				},
			})

			server := httptest.NewServer(http.HandlerFunc(handleNetrcRemoval))
			defer server.Close()

			err := RemoteTransform(c, server.URL)
			g.Assert(err == nil).IsTrue()
			g.Assert(c.Pipeline[0].Image).Equal("golang")
			g.Assert(c.Pipeline[0].Environment["DRONE_BRANCH"]).Equal("master")
			g.Assert(c.Pipeline[0].Environment["DRONE_NETRC_USERNAME"]).Equal("")
			g.Assert(c.Pipeline[0].Environment["DRONE_NETRC_PASSWORD"]).Equal("")
			g.Assert(c.Pipeline[0].Commands[0]).Equal("go build")
			g.Assert(c.Pipeline[0].Commands[1]).Equal("go test")
		})
	})
}
