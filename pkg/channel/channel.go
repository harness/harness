package channel

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"code.google.com/p/go.net/websocket"
	"github.com/dchest/authcookie"
)

// secret key used to generate tokens
var secret = make([]byte, 32)

func init() {
	// generate the secret key by reading
	// from crypto/random
	if _, err := io.ReadFull(rand.Reader, secret); err != nil {
		panic(err)
	}
}

// Create will generate a token and create a new
// channel over which messages will be sent.
func Create(name string) string {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := hubs[name]; !ok {
		hub := newHub(false, true)
		hubs[name] = hub
		go hub.run()
	}
	return authcookie.NewSinceNow(name, 24*time.Hour, secret)
}

// CreateStream will generate a token and create a new
// channel over which messages streams (ie build output)
// are sent.
func CreateStream(name string) string {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := hubs[name]; !ok {
		hub := newHub(true, false)
		hubs[name] = hub
		go hub.run()
	}
	return authcookie.NewSinceNow(name, 24*time.Hour, secret)
}

// Token will generate a token, but will not create
// a new channel.
func Token(name string) string {
	return authcookie.NewSinceNow(name, 24*time.Hour, secret)
}

// Send sends a message on the named channel.
func Send(name string, message string) error {
	return SendBytes(name, []byte(message))
}

// SendJSON sends a JSON-encoded value on
// the named channel.
func SendJSON(name string, value interface{}) error {
	m, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return SendBytes(name, m)
}

// SendBytes send a message in byte format on
// the named channel.
func SendBytes(name string, value []byte) error {
	// get the hub for the specified channel name
	mu.RLock()
	hub, ok := hubs[name]
	mu.RUnlock()

	if !ok {
		return fmt.Errorf("channel does not exist")
	}

	go hub.Write(value)
	return nil
}

func Read(ws *websocket.Conn) {

	// get the name from the request
	hash := ws.Request().FormValue("token")

	// get the hash of the token
	name := authcookie.Login(hash, secret)

	// get the hub for the specified channel name
	mu.RLock()
	hub, ok := hubs[name]
	mu.RUnlock()

	// if hub not found, exit
	if !ok {
		ws.Close()
		return
	}

	// internal representation of a connection
	// maximum queue of 100000 messages
	conn := &connection{
		send: make(chan string, 100000),
		ws:   ws,
	}

	// register the connection with the hub
	hub.register <- conn

	defer func() {
		go func() {
			hub.unregister <- conn
		}()
		closed := <-hub.closed

		// this will remove the hub when the connection is
		// closed if the
		if hub.autoClose && closed {
			mu.Lock()
			delete(hubs, name)
			mu.Unlock()
		}
	}()

	go conn.writer()
	conn.reader()
}

func Close(name string) {
	// get the hub for the specified channel name
	mu.RLock()
	hub, ok := hubs[name]
	mu.RUnlock()

	if !ok {
		return
	}

	// close hub connections
	hub.Close()

	// remove the hub
	mu.Lock()
	delete(hubs, name)
	mu.Unlock()
}
