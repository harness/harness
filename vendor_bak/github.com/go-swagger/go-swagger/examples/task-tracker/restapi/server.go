package restapi

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	flags "github.com/jessevdk/go-flags"
	graceful "github.com/tylerb/graceful"

	"github.com/go-swagger/go-swagger/examples/task-tracker/restapi/operations"
)

//go:generate swagger generate server -t ../.. -A TaskTracker -f ./swagger.yml

// NewServer creates a new api task tracker server
func NewServer(api *operations.TaskTrackerAPI) *Server {
	s := new(Server)
	s.api = api
	if api != nil {
		s.handler = configureAPI(api)
	}
	return s
}

// Server for the task tracker API
type Server struct {
	Host string `long:"host" description:"the IP to listen on" default:"localhost" env:"HOST"`
	Port int    `long:"port" description:"the port to listen on for insecure connections, defaults to a random value" env:"PORT"`

	TLSHost           string         `long:"tls-host" description:"the IP to listen on for tls, when not specified it's the same as --host" env:"TLS_HOST"`
	TLSPort           int            `long:"tls-port" description:"the port to listen on for secure connections, defaults to a random value" env:"TLS_PORT"`
	TLSCertificate    flags.Filename `long:"tls-certificate" description:"the certificate to use for secure connections" required:"true" env:"TLS_CERTIFICATE"`
	TLSCertificateKey flags.Filename `long:"tls-key" description:"the private key to use for secure conections" required:"true" env:"TLS_PRIVATE_KEY"`

	api     *operations.TaskTrackerAPI
	handler http.Handler
}

// SetAPI configures the server with the specified API. Needs to be called before Serve
func (s *Server) SetAPI(api *operations.TaskTrackerAPI) {
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

	fmt.Printf("serving task tracker at http://%s\n", listener.Addr())
	go func() {
		if err := httpServer.Serve(tcpKeepAliveListener{listener.(*net.TCPListener)}); err != nil {
			log.Fatalln(err)
		}
	}()

	httpsServer := &graceful.Server{Server: new(http.Server)}
	httpsServer.Handler = s.handler
	httpsServer.TLSConfig = new(tls.Config)
	httpsServer.TLSConfig.NextProtos = []string{"http/1.1"}
	// https://www.owasp.org/index.php/Transport_Layer_Protection_Cheat_Sheet#Rule_-_Only_Support_Strong_Protocols
	httpsServer.TLSConfig.MinVersion = tls.VersionTLS11
	httpsServer.TLSConfig.Certificates = make([]tls.Certificate, 1)
	httpsServer.TLSConfig.Certificates[0], err = tls.LoadX509KeyPair(string(s.TLSCertificate), string(s.TLSCertificateKey))
	if err != nil {
		return err
	}

	if s.TLSHost == "" {
		s.TLSHost = s.Host
	}
	tlsListener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.TLSHost, s.TLSPort))
	if err != nil {
		return err
	}

	fmt.Printf("serving task tracker at https://%s\n", tlsListener.Addr())
	wrapped := tls.NewListener(tcpKeepAliveListener{tlsListener.(*net.TCPListener)}, httpsServer.TLSConfig)
	if err := httpsServer.Serve(wrapped); err != nil {
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
