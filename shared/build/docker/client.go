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

	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/docker/pkg/term"
	"github.com/docker/docker/utils"
)

const (
	APIVERSION        = 1.9
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
	c := &Client{}

	c.setHost(DEFAULTUNIXSOCKET)

	c.Images = &ImageService{c}
	c.Containers = &ContainerService{c}
	return c
}

func NewHost(address string) *Client {
	c := &Client{}

	// parse the address and split
	pieces := strings.Split(address, "://")
	if len(pieces) == 2 {
		c.proto = pieces[0]
		c.addr = pieces[1]
	} else if len(pieces) == 1 {
		c.addr = pieces[0]
	}

	c.Images = &ImageService{c}
	c.Containers = &ContainerService{c}
	return c
}

func NewClient(addr, cert, key string) (*Client, error) {
	// generate a new Client
	var cli = NewHost(addr)
	cli.tls = new(tls.Config)

	// this is required in order for Docker to connect
	// to a certificate generated for an IP address and
	// not a Domain name
	cli.tls.InsecureSkipVerify = true

	// loads the keyvalue pair and stores the
	// cert (pem) in a certificate store (array)
	pem, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}
	cli.tls.Certificates = []tls.Certificate{pem}

	// creates a transport that uses the custom tls
	// configuration to securely connect to remote
	// Docker clients.
	cli.trans = &http.Transport{
		TLSClientConfig: cli.tls,
		Dial: func(dial_network, dial_addr string) (net.Conn, error) {
			return net.DialTimeout(cli.proto, cli.addr, 32*time.Second)
		},
	}

	if cli.proto == "unix" {
		// no need in compressing for local communications
		cli.trans.DisableCompression = true
	}

	return cli, nil
}

type Client struct {
	tls   *tls.Config
	trans *http.Transport
	proto string
	addr  string

	Images     *ImageService
	Containers *ContainerService
}

var (
	// Returned if the specified resource does not exist.
	ErrNotFound = errors.New("Not Found")

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

func (c *Client) setHost(defaultUnixSocket string) {
	c.proto = DEFAULTPROTOCOL
	c.addr = defaultUnixSocket

	if os.Getenv("DOCKER_HOST") != "" {
		pieces := strings.Split(os.Getenv("DOCKER_HOST"), "://")
		if len(pieces) == 2 {
			c.proto = pieces[0]
			c.addr = pieces[1]
		} else if len(pieces) == 1 {
			c.addr = pieces[0]
		}
	} else {
		// if the default socket doesn't exist then
		// we'll try to connect to the default tcp address
		if _, err := os.Stat(defaultUnixSocket); err != nil {
			c.proto = "tcp"
			c.addr = "0.0.0.0:2375"
		}
	}
}

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

	// Read the bytes from the body (make sure we defer close the body)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Check for an http error status (ie not 200 StatusOK)
	switch resp.StatusCode {
	case 404:
		return ErrNotFound
	case 403:
		return ErrForbidden
	case 401:
		return ErrNotAuthorized
	case 400:
		return ErrBadRequest
	}

	// Unmarshall the JSON response
	if out != nil {
		return json.Unmarshal(body, out)
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
		return utils.DisplayJSONMessagesStream(resp.Body, out, terminalFd, isTerminal)
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
