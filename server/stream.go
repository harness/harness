package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/cncd/logging"
	"github.com/cncd/pubsub"
	"github.com/drone/drone/cache"
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
		repo, _ = cache.GetRepoMap(c, user)
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
