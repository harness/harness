package channel

import (
	"sync"
)

// mutex to lock access to the
// internal map of hubs.
var mu sync.RWMutex

// a map of hubs. each hub represents a different
// channel that a set of users can listen on. For
// example, we may have a hub to stream build output
// for github.com/foo/bar or a channel to post
// updates for user octocat.
var hubs = map[string]*hub{}

type hub struct {
	// Registered connections
	connections map[*connection]bool

	// Inbound messages from the connections.
	broadcast chan string

	// Register requests from the connections.
	register chan *connection

	// Unregister requests from connections.
	unregister chan *connection

	// Buffer of sent data. This is used mostly
	// for build output. A client may connect after
	// the build has already started, in which case
	// we need to stream them the build history.
	history []string

	// Send a "shutdown" signal
	close chan bool

	// Hub responds on this channel letting you know
	// if it's active
	closed chan bool

	// Auto shutdown when last connection removed
	autoClose bool

	// Send history
	sendHistory bool
}

func newHub(sendHistory, autoClose bool) *hub {
	h := hub{
		broadcast:   make(chan string),
		register:    make(chan *connection),
		unregister:  make(chan *connection),
		connections: make(map[*connection]bool),
		history:     make([]string, 0), // This should be pre-allocated, but it's not
		close:       make(chan bool),
		autoClose:   autoClose,
		closed:      make(chan bool),
		sendHistory: sendHistory,
	}

	return &h
}

func sendHistory(c *connection, history []string) {
	if len(history) > 0 {
		for i := range history {
			c.send <- history[i]
		}
	}
}

func (h *hub) run() {
	// make sure we don't bring down the application
	// if somehow we encounter a nil pointer or some
	// other unexpected behavior.
	defer func() {
		recover()
	}()

	for {
		select {
		case c := <-h.register:
			h.connections[c] = true
			if len(h.history) > 0 {
				b := make([]string, len(h.history))
				copy(b, h.history)
				go sendHistory(c, b)
			}
		case c := <-h.unregister:
			delete(h.connections, c)
			close(c.send)
			shutdown := h.autoClose && (len(h.connections) == 0)
			if shutdown {
				h.closed <- shutdown
				return
			}
			h.closed <- shutdown
		case m := <-h.broadcast:
			if h.sendHistory {
				h.history = append(h.history, m)
			}
			for c := range h.connections {
				select {
				case c.send <- m:
					// do nothing
				default:
					delete(h.connections, c)
					go c.ws.Close()
				}
			}
		case <-h.close:
			for c := range h.connections {
				delete(h.connections, c)
				close(c.send)
			}
			h.closed <- true
			return
		}

	}
}

func (h *hub) Close() {
	h.close <- true
}

func (h *hub) Write(p []byte) (n int, err error) {
	h.broadcast <- string(p)
	return len(p), nil
}
