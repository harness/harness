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
	"github.com/gorilla/websocket"
)

var (
	// Time allowed to write the file to the client.
	writeWait = 5 * time.Second

	// Time allowed to read the next pong message from the client.
	pongWait = 60 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = 30 * time.Second

	// upgrader defines the default behavior for upgrading the websocket.
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func reader(ws *websocket.Conn) {
	defer ws.Close()
	ws.SetReadLimit(512)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}
}

func LogStream(c *gin.Context) {
	repo := session.Repo(c)
	buildn, _ := strconv.Atoi(c.Param("build"))
	jobn, _ := strconv.Atoi(c.Param("number"))

	build, err := store.GetBuildNumber(c, repo, buildn)
	if err != nil {
		logrus.Debugln("stream cannot get build number.", err)
		c.AbortWithError(404, err)
		return
	}
	proc, err := store.FromContext(c).ProcFind(build, jobn)
	if err != nil {
		logrus.Debugln("stream cannot get proc number.", err)
		c.AbortWithError(404, err)
		return
	}
	if proc.State != model.StatusRunning {
		logrus.Debugln("stream not found.")
		c.AbortWithStatus(404)
		return
	}

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			logrus.Errorf("Cannot upgrade websocket. %s", err)
		}
		return
	}
	logrus.Debugf("Successfull upgraded websocket")

	ticker := time.NewTicker(pingPeriod)
	logc := make(chan []byte, 10)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer func() {
		cancel()
		ticker.Stop()
		close(logc)
		logrus.Debugf("Successfully closing websocket")
	}()

	go func() {
		// TODO remove global variable
		Config.Services.Logs.Tail(ctx, fmt.Sprint(proc.ID), func(entries ...*logging.Entry) {
			for _, entry := range entries {
				select {
				case <-ctx.Done():
					return
				default:
					logc <- entry.Data
				}
			}
		})
		cancel()
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case buf, ok := <-logc:
				if ok {
					ws.SetWriteDeadline(time.Now().Add(writeWait))
					ws.WriteMessage(websocket.TextMessage, buf)
				}
			case <-ticker.C:
				err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait))
				if err != nil {
					return
				}
			}
		}
	}()
	reader(ws)
}

func EventStream(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			logrus.Errorf("Cannot upgrade websocket. %s", err)
		}
		return
	}
	logrus.Debugf("Successfull upgraded websocket")

	user := session.User(c)
	repo := map[string]bool{}
	if user != nil {
		repos, _ := store.FromContext(c).RepoList(user)
		for _, r := range repos {
			repo[r.FullName] = true
		}
	}

	ticker := time.NewTicker(pingPeriod)
	eventc := make(chan []byte, 10)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer func() {
		cancel()
		ticker.Stop()
		close(eventc)
		logrus.Debugf("Successfully closing websocket")
	}()

	go func() {
		// TODO remove this from global config
		Config.Services.Pubsub.Subscribe(c, "topic/events", func(m pubsub.Message) {
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

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case buf, ok := <-eventc:
				if ok {
					ws.SetWriteDeadline(time.Now().Add(writeWait))
					ws.WriteMessage(websocket.TextMessage, buf)
				}
			case <-ticker.C:
				err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait))
				if err != nil {
					return
				}
			}
		}
	}()
	reader(ws)
}

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
		Config.Services.Pubsub.Subscribe(c, "topic/events", func(m pubsub.Message) {
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
