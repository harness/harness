package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/drone/drone/version"
	"github.com/urfave/cli"
)

// the file implements some basic healthcheck logic based on the
// following specification:
//   https://github.com/mozilla-services/Dockerflow

func init() {
	http.HandleFunc("/__heartbeat__", handleHeartbeat)
	http.HandleFunc("/__version__", handleVersion)
}

func handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}

func handleVersion(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Header().Add("Content-Type", "text/json")
	json.NewEncoder(w).Encode(versionResp{
		Source:  "https://github.com/drone/drone",
		Version: version.Version.String(),
	})
}

type versionResp struct {
	Version string `json:"version"`
	Source  string `json:"source"`
}

// handles pinging the endpoint and returns an error if the
// agent is in an unhealthy state.
func pinger(c *cli.Context) error {
	resp, err := http.Get("http://localhost:3000/__heartbeat__")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("agent returned non-200 status code")
	}
	return nil
}
