package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/drone/drone/common"
	"github.com/drone/drone/eventbus"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/koding/websocketproxy"
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

// GetEvents will upgrade the connection to a Websocket and will stream
// event updates to the browser.
func GetEvents(c *gin.Context) {
	bus := ToBus(c)
	user := ToUser(c)
	remote := ToRemote(c)

	// TODO (bradrydzewski) revisit this approach at some point.
	//
	// instead of constantly checking for remote permissions, we will
	// cache them for the lifecycle of this websocket. The pro here is
	// that we are making way less external calls (good). The con is that
	// if a ton of developers conntect to websockets for long periods of
	// time with heavy build traffic (not super likely, but possible) this
	// caching strategy could take up a lot of memory.
	perms_ := map[string]*common.Perm{}

	// upgrade the websocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.Fail(400, err)
		return
	}

	ticker := time.NewTicker(pingPeriod)
	eventc := make(chan *eventbus.Event, 1)
	bus.Subscribe(eventc)
	defer func() {
		bus.Unsubscribe(eventc)
		ticker.Stop()
		ws.Close()
		close(eventc)
	}()

	go func() {
		for {
			select {
			case event := <-eventc:
				if event == nil {
					return // why would this ever happen?
				}
				perm, ok := perms_[event.Repo.FullName]
				if !ok {
					perm = perms(remote, user, event.Repo)
					perms_[event.Repo.FullName] = perm
				}

				if perm != nil && perm.Pull {
					err := ws.WriteJSON(event)
					if err != nil {
						log.Errorln(err, event)
					}
				}
			case <-ticker.C:
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				err := ws.WriteMessage(websocket.PingMessage, []byte{})
				if err != nil {
					ws.Close()
					log.Debugf("closed websocket")
					return
				}
			}
		}
	}()

	readWebsocket(ws)
	log.Debugf("closed websocket")
}

func GetStream(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	build, _ := strconv.Atoi(c.Params.ByName("build"))
	task, _ := strconv.Atoi(c.Params.ByName("number"))

	agent, err := store.BuildAgent(repo.FullName, build)
	if err != nil {
		c.Fail(404, err)
		return
	}

	url_, err := url.Parse("ws://" + agent.Addr)
	if err != nil {
		c.Fail(500, err)
		return
	}
	url_.Path = fmt.Sprintf("/stream/%s/%v/%v", repo.FullName, build, task)
	proxy := websocketproxy.NewProxy(url_)
	proxy.ServeHTTP(c.Writer, c.Request)

	log.Debugf("closed websocket")
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
