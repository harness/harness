package server

import (
	"bufio"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/drone/drone/eventbus"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	// "github.com/koding/websocketproxy"
)

const (
	// Time allowed to write the message to the client.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the client.
	pongWait = 60 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// GetRepoEvents will upgrade the connection to a Websocket and will stream
// event updates to the browser.
func GetRepoEvents(c *gin.Context) {
	bus := ToBus(c)
	repo := ToRepo(c)
	c.Writer.Header().Set("Content-Type", "text/event-stream")

	eventc := make(chan *eventbus.Event, 1)
	bus.Subscribe(eventc)
	defer func() {
		bus.Unsubscribe(eventc)
		log.Infof("closed event stream")
	}()

	c.Stream(func(w io.Writer) bool {
		event := <-eventc
		if event == nil {
			return false
		}
		if event.Kind == eventbus.EventRepo &&
			event.Name == repo.FullName {
			c.SSEvent("message", event.Msg)
		}

		return true
	})
}

func GetStream(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	runner := ToRunner(c)
	commitseq, _ := strconv.Atoi(c.Params.ByName("build"))
	buildseq, _ := strconv.Atoi(c.Params.ByName("number"))

	// agent, err := store.BuildAgent(repo.FullName, build)
	// if err != nil {
	// 	c.Fail(404, err)
	// 	return
	// }

	commit, err := store.CommitSeq(repo, commitseq)
	if err != nil {
		c.Fail(404, err)
		return
	}
	build, err := store.BuildSeq(commit, buildseq)
	if err != nil {
		c.Fail(404, err)
		return
	}

	rc, err := runner.Logs(build)
	if err != nil {
		c.Fail(404, err)
		return
	}

	// upgrade the websocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.Fail(400, err)
		return
	}

	var ticker = time.NewTicker(pingPeriod)
	var out = make(chan []byte)
	defer func() {
		log.Infof("closed stdout websocket")
		ticker.Stop()
		rc.Close()
		ws.Close()
	}()

	go func() {
		for {
			select {
			case <-c.Writer.CloseNotify():
				rc.Close()
				ws.Close()
				return
			case line := <-out:
				ws.WriteMessage(websocket.TextMessage, line)
			case <-ticker.C:
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				err := ws.WriteMessage(websocket.PingMessage, []byte{})
				if err != nil {
					rc.Close()
					ws.Close()
					return
				}
			}
		}
	}()

	go func() {
		rd := bufio.NewReader(rc)
		for {
			str, err := rd.ReadBytes('\n')

			if err != nil {
				break
			}
			if len(str) == 0 {
				break
			}

			out <- str
		}
		rc.Close()
		ws.Close()
	}()

	readWebsocket(ws)

	// url_, err := url.Parse("ws://" + agent.Addr)
	// if err != nil {
	// 	c.Fail(500, err)
	// 	return
	// }
	// url_.Path = fmt.Sprintf("/stream/%s/%v/%v", repo.FullName, build, task)
	// proxy := websocketproxy.NewProxy(url_)
	// proxy.ServeHTTP(c.Writer, c.Request)

	// log.Debugf("closed websocket")
}

// readWebsocket will block while reading the websocket data
func readWebsocket(ws *websocket.Conn) {
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
