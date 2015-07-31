package docker

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/docker/pkg/term"
)

const (
	APIVERSION        = 1.12
	DEFAULTHTTPPORT   = 2375
	DEFAULTUNIXSOCKET = "/var/run/docker.sock"
	DEFAULTPROTOCOL   = "unix"
	DEFAULTTAG        = "latest"
	VERSION           = "0.8.0"
)

// Enables verbose logging to the Terminal window
var Logging = true

// New creates an instance of the Docker Client
func New() *Client {
	return NewHost("")
}

func NewHost(uri string) *Client {
	var cli, _ = NewHostCert(uri, nil, nil)
	return cli
}

func NewHostCertFile(uri, cert, key string) (*Client, error) {
	if len(key) == 0 || len(cert) == 0 {
		return NewHostCert(uri, nil, nil)
	}
	certfile, err := ioutil.ReadFile(cert)
	if err != nil {
		return nil, err
	}
	keyfile, err := ioutil.ReadFile(key)
	if err != nil {
		return nil, err
	}
	return NewHostCert(uri, certfile, keyfile)
}

func NewHostCert(uri string, cert, key []byte) (*Client, error) {
	var host = GetHost(uri)
	var proto, addr = SplitProtoAddr(host)

	var cli = new(Client)
	cli.proto = proto
	cli.addr = addr
	cli.scheme = "http"
	cli.Images = &ImageService{cli}
	cli.Containers = &ContainerService{cli}

	// if no certificate is provided returns the
	// client with no TLS configured.
	if cert == nil || key == nil || len(cert) == 0 || len(key) == 0 {
		cli.trans = &http.Transport{
			Dial: func(dial_network, dial_addr string) (net.Conn, error) {
				return net.DialTimeout(cli.proto, cli.addr, 32*time.Second)
			},
		}
		return cli, nil
	}

	// loads the key value pair in pem format
	pem, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}

	// setup the client TLS and store the certificate.
	// also skip verification since we are (typically)
	// going to be using certs for IP addresses.
	cli.scheme = "https"
	cli.tls = new(tls.Config)
	cli.tls.InsecureSkipVerify = true
	cli.tls.Certificates = []tls.Certificate{pem}

	// disable compression for local socket communication.
	if cli.proto == DEFAULTPROTOCOL {
		cli.trans.DisableCompression = true
	}

	// creates a transport that uses the custom tls configuration
	// to securely connect to remote Docker clients.
	cli.trans = &http.Transport{
		TLSClientConfig: cli.tls,
		Dial: func(dial_network, dial_addr string) (net.Conn, error) {
			return net.DialTimeout(cli.proto, cli.addr, 32*time.Second)
		},
	}

	return cli, nil
}

// GetHost returns the Docker Host address in order to
// connect to the Docker Daemon. It implements a very
// simple set of fallthrough logic to determine which
// address to use.
func GetHost(host string) string {
	// if a default value was provided this
	// shoudl be used
	if len(host) != 0 {
		return host
	}
	// else attempt to use the DOCKER_HOST
	// environment variable
	var env = os.Getenv("DOCKER_HOST")
	if len(env) != 0 {
		return env
	}
	// else check to see if the default unix
	// socket exists and return
	_, err := os.Stat(DEFAULTUNIXSOCKET)
	if err == nil {
		return fmt.Sprintf("%s://%s", DEFAULTPROTOCOL, DEFAULTUNIXSOCKET)
	}
	// else return the standard TCP address
	return fmt.Sprintf("tcp://0.0.0.0:%d", DEFAULTHTTPPORT)
}

// SplitProtoAddr is a helper function that splits
// a host into Protocol and Address.
func SplitProtoAddr(host string) (string, string) {
	var parts = strings.Split(host, "://")
	var proto, addr string
	switch {
	case len(parts) == 2:
		proto = parts[0]
		addr = parts[1]
	default:
		proto = "tcp"
		addr = parts[0]
	}
	return proto, addr
}

type Client struct {
	tls    *tls.Config
	trans  *http.Transport
	scheme string
	proto  string
	addr   string

	Images     *ImageService
	Containers *ContainerService
}

var (
	// Returned if the specified resource does not exist.
	ErrNotFound = errors.New("Not Found")

	// Return if something going wrong
	ErrInternalServer = errors.New("Internal Server Error")

	// Returned if the caller attempts to make a call or modify a resource
	// for which the caller is not authorized.
	//
	// The request was a valid request, the caller's authentication credentials
	// succeeded but those credentials do not grant the caller permission to
	// access the resource.
	ErrForbidden = errors.New("Forbidden")

	// Returned if the call requires authentication and either the credentials
	// provided failed or no credentials were provided.
	ErrNotAuthorized = errors.New("Unauthorized")

	// Returned if the caller submits a badly formed request. For example,
	// the caller can receive this return if you forget a required parameter.
	ErrBadRequest = errors.New("Bad Request")
)

// helper function used to make HTTP requests to the Docker daemon.
func (c *Client) do(method, path string, in, out interface{}) error {
	// if data input is provided, serialize to JSON
	var payload io.Reader
	if in != nil {
		buf, err := json.Marshal(in)
		if err != nil {
			return err
		}
		payload = bytes.NewBuffer(buf)
	}

	// create the request
	req, err := http.NewRequest(method, fmt.Sprintf("/v%g%s", APIVERSION, path), payload)
	if err != nil {
		return err
	}

	// set the appropariate headers
	req.Header = http.Header{}
	req.Header.Set("User-Agent", "Docker-Client/"+VERSION)
	req.Header.Set("Content-Type", "application/json")

	// dial the host server
	req.URL.Host = c.addr
	req.URL.Scheme = "http"
	if c.tls != nil {
		req.URL.Scheme = "https"
	}

	resp, err := c.HTTPClient().Do(req)
	if err != nil {
		return err
	}

	// make sure we defer close the body
	defer resp.Body.Close()

	// Check for an http error status (ie not 200 StatusOK)
	switch resp.StatusCode {
	case 500:
		return ErrInternalServer
	case 404:
		return ErrNotFound
	case 403:
		return ErrForbidden
	case 401:
		return ErrNotAuthorized
	case 400:
		return ErrBadRequest
	}

	// Decode the JSON response
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}

	return nil
}

func (c *Client) hijack(method, path string, setRawTerminal bool, out io.Writer) error {
	req, err := http.NewRequest(method, fmt.Sprintf("/v%g%s", APIVERSION, path), nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Docker-Client/"+VERSION)
	req.Header.Set("Content-Type", "plain/text")
	req.Host = c.addr

	dial, err := c.Dial()
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			return fmt.Errorf("Can't connect to docker daemon. Is 'docker -d' running on this host?")
		}
		return err
	}
	clientconn := httputil.NewClientConn(dial, nil)
	defer clientconn.Close()

	// Server hijacks the connection, error 'connection closed' expected
	clientconn.Do(req)

	// Hijack the connection to read / write
	rwc, br := clientconn.Hijack()
	defer rwc.Close()

	// launch a goroutine to copy the stream
	// of build output to the writer.
	errStdout := make(chan error, 1)
	go func() {
		var err error
		if setRawTerminal {
			_, err = io.Copy(out, br)
		} else {
			_, err = stdcopy.StdCopy(out, out, br)
		}

		errStdout <- err
	}()

	// wait for a response
	if err := <-errStdout; err != nil {
		return err
	}
	return nil
}

func (c *Client) stream(method, path string, in io.Reader, out io.Writer, headers http.Header) error {
	if (method == "POST" || method == "PUT") && in == nil {
		in = bytes.NewReader(nil)
	}

	// setup the request
	req, err := http.NewRequest(method, fmt.Sprintf("/v%g%s", APIVERSION, path), in)
	if err != nil {
		return err
	}

	// set default headers
	req.Header = headers
	req.Header.Set("User-Agent", "Docker-Client/0.6.4")
	req.Header.Set("Content-Type", "plain/text")

	// dial the host server
	req.URL.Host = c.addr
	req.URL.Scheme = "http"
	if c.tls != nil {
		req.URL.Scheme = "https"
	}

	resp, err := c.HTTPClient().Do(req)
	if err != nil {
		return err
	}

	// make sure we defer close the body
	defer resp.Body.Close()

	// Check for an http error status (ie not 200 StatusOK)
	switch resp.StatusCode {
	case 500:
		return ErrInternalServer
	case 404:
		return ErrNotFound
	case 403:
		return ErrForbidden
	case 401:
		return ErrNotAuthorized
	case 400:
		return ErrBadRequest
	}

	// If no output we exit now with no errors
	if out == nil {
		io.Copy(ioutil.Discard, resp.Body)
		return nil
	}

	// copy the output stream to the writer
	if resp.Header.Get("Content-Type") == "application/json" {
		var terminalFd = os.Stdin.Fd()
		var isTerminal = term.IsTerminal(terminalFd)

		// it may not make sense to put this code here, but it works for
		// us at the moment, and I don't feel like refactoring
		return jsonmessage.DisplayJSONMessagesStream(resp.Body, out, terminalFd, isTerminal)
	}
	// otherwise plain text
	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}

	return nil
}

func (c *Client) HTTPClient() *http.Client {
	if c.trans != nil {
		return &http.Client{Transport: c.trans}
	}
	return &http.Client{
		// WARN Leak Transport's Pooling Connection
		Transport: &http.Transport{
			Dial: func(dial_network, dial_addr string) (net.Conn, error) {
				return net.DialTimeout(c.proto, c.addr, 32*time.Second)
			},
		},
	}
}

func (c *Client) Dial() (net.Conn, error) {
	if c.tls != nil && c.proto != "unix" {
		return tls.Dial(c.proto, c.addr, c.tls)
	}
	return net.Dial(c.proto, c.addr)
}

func (c *Client) CloseIdleConnections() {
	if c.trans != nil {
		c.trans.CloseIdleConnections()
	}
}
