package restapi

import (
	"fmt"
	"net"
	"net/http"
	"time"

	graceful "github.com/tylerb/graceful"

	"github.com/go-swagger/go-swagger/examples/tutorials/todo-list/server-2/restapi/operations"
)

//go:generate swagger generate server -t ../.. -A TodoList -f ./swagger.yml

// NewServer creates a new api todo list server
func NewServer(api *operations.TodoListAPI) *Server {
	s := new(Server)
	s.api = api
	if api != nil {
		s.handler = configureAPI(api)
	}
	return s
}

// Server for the todo list API
type Server struct {
	Host string `long:"host" description:"the IP to listen on" default:"localhost" env:"HOST"`
	Port int    `long:"port" description:"the port to listen on for insecure connections, defaults to a random value" env:"PORT"`

	api     *operations.TodoListAPI
	handler http.Handler
}

// SetAPI configures the server with the specified API. Needs to be called before Serve
func (s *Server) SetAPI(api *operations.TodoListAPI) {
	if api == nil {
		s.api = nil
		s.handler = nil
		return
	}

	s.api = api
	s.handler = configureAPI(api)
}

// Serve the api
func (s *Server) Serve() (err error) {

	httpServer := &graceful.Server{Server: new(http.Server)}
	httpServer.Handler = s.handler

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.Host, s.Port))
	if err != nil {
		return err
	}

	fmt.Printf("serving todo list at http://%s\n", listener.Addr())
	if err := httpServer.Serve(tcpKeepAliveListener{listener.(*net.TCPListener)}); err != nil {
		return err
	}

	return nil
}

// Shutdown server and clean up resources
func (s *Server) Shutdown() error {
	s.api.ServerShutdown()
	return nil
}

// tcpKeepAliveListener is copied from the stdlib net/http package

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}
