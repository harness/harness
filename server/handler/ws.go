package handler

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/drone/drone/server/datastore"
	"github.com/drone/drone/server/pubsub"
	"github.com/drone/drone/server/worker"

	"github.com/goji/context"
	"github.com/gorilla/websocket"
	"github.com/zenazn/goji/web"
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

// WsUser will upgrade the connection to a Websocket and will stream
// all events to the browser pertinent to the authenticated user. If the user
// is not authenticated, only public events are streamed.
func WsUser(c web.C, w http.ResponseWriter, r *http.Request) {
	var ctx = context.FromC(c)
	var user = ToUser(c)

	// upgrade the websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// register a channel for global events
	channel := pubsub.Register(ctx, "_global")
	sub := channel.Subscribe()

	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		sub.Close()
		ws.Close()
	}()

	go func() {
		for {
			select {
			case msg := <-sub.Read():
				work, ok := msg.(*worker.Work)
				if !ok {
					break
				}

				role, permerr := datastore.GetPerm(ctx, user, work.Repo)
				if permerr != nil && permerr != sql.ErrNoRows {
					// for debugging
					log.Printf("WS: Error getting permissions for repository %s. Error: %s\n", work.Repo.Name, permerr)
				}

				// user must have read access to private the repository
				// in order to pass this message along
				if work.Repo.Private == true && role.Read == false {
					break
				}

				ws.SetWriteDeadline(time.Now().Add(writeWait))
				err := ws.WriteJSON(work)
				if err != nil {
					ws.Close()
					return
				}
			case <-sub.CloseNotify():
				ws.Close()
				return
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

// WsConsole will upgrade the connection to a Websocket and will stream
// the build output to the browser.
func WsConsole(c web.C, w http.ResponseWriter, r *http.Request) {
	var commitID, _ = strconv.Atoi(c.URLParams["id"])
	var ctx = context.FromC(c)
	var user = ToUser(c)

	commit, err := datastore.GetCommit(ctx, int64(commitID))
	if err != nil {
		log.Printf("WS: Error retrieving commit by ID. %s\n", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	repo, err := datastore.GetRepo(ctx, commit.RepoID)
	if err != nil {
		log.Printf("WS: Error retrieving repo by ID. %s\n", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	role, err := datastore.GetPerm(ctx, user, repo)
	if err != nil || role.Read == false {
		if user == nil {
			log.Println("WS: Error getting User session")
		}
		log.Println("WS: Error retrieving Read permission.", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// find a channel that we can subscribe to
	// and listen for stream updates.
	channel := pubsub.Lookup(ctx, commit.ID)
	if channel == nil {
		log.Println("WS: Error getting build stream from channel")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	sub := channel.Subscribe()
	defer sub.Close()

	// upgrade the websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		ws.Close()
	}()

	go func() {
		for {
			select {
			case msg := <-sub.Read():
				data, ok := msg.([]byte)
				if !ok {
					break
				}
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				err := ws.WriteMessage(websocket.TextMessage, data)
				if err != nil {
					ws.Close()
					return
				}
			case <-sub.CloseNotify():
				ws.Close()
				return
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
