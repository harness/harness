// Copyright 2018 Drone.IO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/drone/drone/version"
	"github.com/urfave/cli"
)

// the file implements some basic healthcheck logic based on the
// following specification:
//   https://github.com/mozilla-services/Dockerflow

func init() {
	http.HandleFunc("/varz", handleStats)
	http.HandleFunc("/healthz", handleHeartbeat)
	http.HandleFunc("/version", handleVersion)
}

func handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if counter.Healthy() {
		w.WriteHeader(200)
	} else {
		w.WriteHeader(500)
	}
}

func handleVersion(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Header().Add("Content-Type", "text/json")
	json.NewEncoder(w).Encode(versionResp{
		Source:  "https://github.com/drone/drone",
		Version: version.Version.String(),
	})
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	if counter.Healthy() {
		w.WriteHeader(200)
	} else {
		w.WriteHeader(500)
	}
	w.Header().Add("Content-Type", "text/json")
	counter.writeTo(w)
}

type versionResp struct {
	Version string `json:"version"`
	Source  string `json:"source"`
}

// default statistics counter
var counter = &state{
	Metadata: map[string]info{},
}

type state struct {
	sync.Mutex `json:"-"`
	Polling    int             `json:"polling_count"`
	Running    int             `json:"running_count"`
	Metadata   map[string]info `json:"running"`
}

type info struct {
	ID      string        `json:"id"`
	Repo    string        `json:"repository"`
	Build   string        `json:"build_number"`
	Started time.Time     `json:"build_started"`
	Timeout time.Duration `json:"build_timeout"`
}

func (s *state) Add(id string, timeout time.Duration, repo, build string) {
	s.Lock()
	s.Polling--
	s.Running++
	s.Metadata[id] = info{
		ID:      id,
		Repo:    repo,
		Build:   build,
		Timeout: timeout,
		Started: time.Now().UTC(),
	}
	s.Unlock()
}

func (s *state) Done(id string) {
	s.Lock()
	s.Polling++
	s.Running--
	delete(s.Metadata, id)
	s.Unlock()
}

func (s *state) Healthy() bool {
	s.Lock()
	defer s.Unlock()
	now := time.Now()
	buf := time.Hour // 1 hour buffer
	for _, item := range s.Metadata {
		if now.After(item.Started.Add(item.Timeout).Add(buf)) {
			return false
		}
	}
	return true
}

func (s *state) writeTo(w io.Writer) (int, error) {
	s.Lock()
	out, _ := json.Marshal(s)
	s.Unlock()
	return w.Write(out)
}

// handles pinging the endpoint and returns an error if the
// agent is in an unhealthy state.
func pinger(c *cli.Context) error {
	resp, err := http.Get("http://localhost:3000/healthz")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("agent returned non-200 status code")
	}
	return nil
}
