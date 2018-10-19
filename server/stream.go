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

package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/cncd/logging"
	"github.com/cncd/pubsub"
	"github.com/drone/drone/model"
	"github.com/drone/drone/router/middleware/session"
	"github.com/drone/drone/store"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
)

//
// event source streaming for compatibility with quic and http2
//

func EventStreamSSE(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	rw := c.Writer

	flusher, ok := rw.(http.Flusher)
	if !ok {
		c.String(500, "Streaming not supported")
		return
	}

	// ping the client
	io.WriteString(rw, ": ping\n\n")
	flusher.Flush()

	logrus.Debugf("user feed: connection opened")

	user := session.User(c)
	repo := map[string]bool{}
	if user != nil {
		repos, _ := store.FromContext(c).RepoList(user)
		for _, r := range repos {
			repo[r.FullName] = true
		}
	}

	eventc := make(chan []byte, 10)
	ctx, cancel := context.WithCancel(
		context.Background(),
	)

	defer func() {
		cancel()
		close(eventc)
		logrus.Debugf("user feed: connection closed")
	}()

	go func() {
		// TODO remove this from global config
		Config.Services.Pubsub.Subscribe(ctx, "topic/events", func(m pubsub.Message) {
			name := m.Labels["repo"]
			priv := m.Labels["private"]
			if repo[name] || priv == "false" {
				select {
				case <-ctx.Done():
					return
				default:
					eventc <- m.Data
				}
			}
		})
		cancel()
	}()

	for {
		select {
		case <-rw.CloseNotify():
			return
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 30):
			io.WriteString(rw, ": ping\n\n")
			flusher.Flush()
		case buf, ok := <-eventc:
			if ok {
				io.WriteString(rw, "data: ")
				rw.Write(buf)
				io.WriteString(rw, "\n\n")
				flusher.Flush()
			}
		}
	}
}

func LogStreamSSE(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	rw := c.Writer

	flusher, ok := rw.(http.Flusher)
	if !ok {
		c.String(500, "Streaming not supported")
		return
	}

	io.WriteString(rw, ": ping\n\n")
	flusher.Flush()

	// repo := session.Repo(c)
	//
	// // parse the build number and job sequence number from
	// // the repquest parameter.
	// num, _ := strconv.Atoi(c.Params.ByName("number"))
	// ppid, _ := strconv.Atoi(c.Params.ByName("ppid"))
	// name := c.Params.ByName("proc")
	//
	// build, err := store.GetBuildNumber(c, repo, num)
	// if err != nil {
	// 	c.AbortWithError(404, err)
	// 	return
	// }
	//
	// proc, err := store.FromContext(c).ProcChild(build, ppid, name)
	// if err != nil {
	// 	c.AbortWithError(404, err)
	// 	return
	// }

	repo := session.Repo(c)
	buildn, _ := strconv.Atoi(c.Param("build"))
	jobn, _ := strconv.Atoi(c.Param("number"))

	build, err := store.GetBuildNumber(c, repo, buildn)
	if err != nil {
		logrus.Debugln("stream cannot get build number.", err)
		io.WriteString(rw, "event: error\ndata: build not found\n\n")
		return
	}
	proc, err := store.FromContext(c).ProcFind(build, jobn)
	if err != nil {
		logrus.Debugln("stream cannot get proc number.", err)
		io.WriteString(rw, "event: error\ndata: process not found\n\n")
		return
	}
	if proc.State != model.StatusRunning {
		logrus.Debugln("stream not found.")
		io.WriteString(rw, "event: error\ndata: stream not found\n\n")
		return
	}

	logc := make(chan []byte, 10)
	ctx, cancel := context.WithCancel(
		context.Background(),
	)

	logrus.Debugf("log stream: connection opened")

	defer func() {
		cancel()
		close(logc)
		logrus.Debugf("log stream: connection closed")
	}()

	go func() {
		// TODO remove global variable
		Config.Services.Logs.Tail(ctx, fmt.Sprint(proc.ID), func(entries ...*logging.Entry) {
			defer func() {
				recover() // fix #2480
			}()
			for _, entry := range entries {
				select {
				case <-ctx.Done():
					return
				default:
					logc <- entry.Data
				}
			}
		})

		io.WriteString(rw, "event: error\ndata: eof\n\n")

		cancel()
	}()

	id := 1
	last, _ := strconv.Atoi(
		c.Request.Header.Get("Last-Event-ID"),
	)
	if last != 0 {
		logrus.Debugf("log stream: reconnect: last-event-id: %d", last)
	}

	// retry: 10000\n

	for {
		select {
		// after 1 hour of idle (no response) end the stream.
		// this is more of a safety mechanism than anything,
		// and can be removed once the code is more mature.
		case <-time.After(time.Hour):
			return
		case <-rw.CloseNotify():
			return
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 30):
			io.WriteString(rw, ": ping\n\n")
			flusher.Flush()
		case buf, ok := <-logc:
			if ok {
				if id > last {
					io.WriteString(rw, "id: "+strconv.Itoa(id))
					io.WriteString(rw, "\n")
					io.WriteString(rw, "data: ")
					rw.Write(buf)
					io.WriteString(rw, "\n\n")
					flusher.Flush()
				}
				id++
			}
		}
	}
}
