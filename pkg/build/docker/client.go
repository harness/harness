package docker

import (
	"bytes"
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

	"github.com/dotcloud/docker/pkg/term"
	"github.com/dotcloud/docker/utils"
)

const (
	APIVERSION        = 1.9
	DEFAULTHTTPPORT   = 4243
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

type Client struct {
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
			c.addr = "0.0.0.0:4243"
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
	req.Host = c.addr
	dial, err := net.Dial(c.proto, c.addr)
	if err != nil {
		return err
	}

	// make the request
	conn := httputil.NewClientConn(dial, nil)
	resp, err := conn.Do(req)
	defer conn.Close()
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

	dial, err := net.Dial(c.proto, c.addr)
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
			_, err = utils.StdCopy(out, out, br)
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
	req.Host = c.addr
	dial, err := net.Dial(c.proto, c.addr)
	if err != nil {
		return err
	}

	// make the request
	conn := httputil.NewClientConn(dial, nil)
	resp, err := conn.Do(req)
	defer conn.Close()
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
