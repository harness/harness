package stats

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/drone/drone/router/middleware/location"
	"github.com/drone/drone/store"
)

var once sync.Once

var key string

// Schedule is a middleware function that triggers
// usage statistics.
func Schedule(c *gin.Context) {
	once.Do(func() {
		go schedule(c.Copy())
	})
	c.Next()
}

func schedule(c *gin.Context) {
	for {
		send(c)
		select {
		case <-time.After(time.Hour * 24):
		}
	}
}

func send(c *gin.Context) {
	defer recover()

	// disable statistics for local hosting and test environments
	host := location.Hostname(c)
	switch host {
	case "locahost", "127.0.0.1":
		return
	}

	// disable statistics if no key provided
	if len(key) == 0 {
		return // skip
	}

	// disable statistics if environment variable specified
	if os.Getenv("DISABLE_METRICS") != "" {
		return // skip
	}

	in := struct {
		Version string
		Host    string
		Config  struct {
			Database string
			Remote   string
		}
		Stats struct {
			Users int
			Repos int
			Nodes int
		}
	}{}

	var (
		remote_, _   = c.Get("remote")
		database_, _ = c.Get("database")
		version, _   = c.Get("version")
	)

	in.Version = version.(string)
	in.Config.Database = database_.(fmt.Stringer).String()
	in.Config.Remote = remote_.(fmt.Stringer).String()
	in.Stats.Users, _ = store.CountUsers(c)
	in.Stats.Nodes, _ = store.CountNodes(c)
	in.Stats.Repos, _ = store.CountRepos(c)

	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(&in)

	http.Post("https://stats.drone.io", "application/json", buf)
}
