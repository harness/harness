package server

import (
	"time"

	"github.com/drone/drone/eventbus"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
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
}

// GetEvents will upgrade the connection to a Websocket and will stream
// event updates to the browser.
func GetEvents(c *gin.Context) {
	bus := ToBus(c)
	user := ToUser(c)
	remote := ToRemote(c)

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
				perms := perms(remote, user, event.Repo)
				if perms != nil && perms.Pull {
					ws.WriteJSON(event)
				}
			case <-ticker.C:
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				err := ws.WriteMessage(websocket.PingMessage, []byte{})
				if err != nil {
					ws.Close()
					return
				}
			}
		}
	}()

	readWebsocket(ws)
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
