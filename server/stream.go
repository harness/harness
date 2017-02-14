package server

import (
	"fmt"
	"strconv"
	"time"

	"github.com/drone/drone/cache"
	"github.com/drone/drone/model"
	"github.com/drone/drone/router/middleware/session"
	"github.com/drone/drone/store"
	"github.com/drone/mq/stomp"

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
)

// LogStream streams the build log output to the client.
func LogStream(c *gin.Context) {
	repo := session.Repo(c)
	buildn, _ := strconv.Atoi(c.Param("build"))
	jobn, _ := strconv.Atoi(c.Param("number"))

	c.Writer.Header().Set("Content-Type", "text/event-stream")

	build, err := store.GetBuildNumber(c, repo, buildn)
	if err != nil {
		logrus.Debugln("stream cannot get build number.", err)
		c.AbortWithError(404, err)
		return
	}
	job, err := store.GetJobNumber(c, build, jobn)
	if err != nil {
		logrus.Debugln("stream cannot get job number.", err)
		c.AbortWithError(404, err)
		return
	}
	if job.Status != model.StatusRunning {
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
	defer ticker.Stop()

	logs := make(chan []byte)
	done := make(chan bool)
	var eof bool
	dest := fmt.Sprintf("/topic/logs.%d", job.ID)
	client, _ := stomp.FromContext(c)
	sub, err := client.Subscribe(dest, stomp.HandlerFunc(func(m *stomp.Message) {
		if m.Header.GetBool("eof") {
			eof = true
			done <- true
		} else if eof {
			return
		} else {
			logs <- m.Body
		}
		m.Release()
	}))
	if err != nil {
		logrus.Errorf("Unable to read logs from broker. %s", err)
		return
	}
	defer func() {
		client.Unsubscribe(sub)
		close(done)
		close(logs)
	}()

	for {
		select {
		case buf := <-logs:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			ws.WriteMessage(websocket.TextMessage, buf)
		case <-done:
			return
		case <-ticker.C:
			err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait))
			if err != nil {
				return
			}
		}
	}
}

// EventStream produces the User event stream, sending all repository, build
// and agent events to the client.
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
		repo, _ = cache.GetRepoMap(c, user)
	}

	eventc := make(chan []byte, 10)
	quitc := make(chan bool)
	tick := time.NewTicker(pingPeriod)
	defer func() {
		tick.Stop()
		ws.Close()
		logrus.Debug("Successfully closed websocket")
	}()

	client := stomp.MustFromContext(c)
	sub, err := client.Subscribe("/topic/events", stomp.HandlerFunc(func(m *stomp.Message) {
		name := m.Header.GetString("repo")
		priv := m.Header.GetBool("private")
		if repo[name] || !priv {
			eventc <- m.Body
		}
		m.Release()
	}))
	if err != nil {
		logrus.Errorf("Unable to read logs from broker. %s", err)
		return
	}
	defer func() {
		client.Unsubscribe(sub)
		close(quitc)
		close(eventc)
	}()

	go func() {
		defer func() {
			recover()
		}()
		for {
			select {
			case <-quitc:
				return
			case event, ok := <-eventc:
				if !ok {
					return
				}
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				ws.WriteMessage(websocket.TextMessage, event)
			case <-tick.C:
				err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait))
				if err != nil {
					return
				}
			}
		}
	}()

	reader(ws)
}

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
