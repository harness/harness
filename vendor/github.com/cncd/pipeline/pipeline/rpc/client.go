package rpc

import (
	"context"
	"io"
	"log"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sourcegraph/jsonrpc2"
	websocketrpc "github.com/sourcegraph/jsonrpc2/websocket"
)

const (
	methodNext   = "next"
	methodWait   = "wait"
	methodInit   = "init"
	methodDone   = "done"
	methodExtend = "extend"
	methodUpdate = "update"
	methodUpload = "upload"
	methodLog    = "log"
)

type (
	uploadReq struct {
		ID   string `json:"id"`
		File *File  `json:"file"`
	}

	updateReq struct {
		ID    string `json:"id"`
		State State  `json:"state"`
	}

	logReq struct {
		ID   string `json:"id"`
		Line *Line  `json:"line"`
	}
)

const (
	defaultRetryClount = math.MaxInt32
	defaultBackoff     = 10 * time.Second
)

// Client represents an rpc client.
type Client struct {
	sync.Mutex

	conn     *jsonrpc2.Conn
	done     bool
	retry    int
	backoff  time.Duration
	endpoint string
	token    string
	headers  map[string][]string
}

// NewClient returns a new Client.
func NewClient(endpoint string, opts ...Option) (*Client, error) {
	cli := &Client{
		endpoint: endpoint,
		retry:    defaultRetryClount,
		backoff:  defaultBackoff,
		headers:  map[string][]string{},
	}
	for _, opt := range opts {
		opt(cli)
	}
	err := cli.openRetry()
	return cli, err
}

// Next returns the next pipeline in the queue.
func (t *Client) Next(c context.Context, f Filter) (*Pipeline, error) {
	res := new(Pipeline)
	err := t.call(c, methodNext, f, res)
	return res, err
}

// Wait blocks until the pipeline is complete.
func (t *Client) Wait(c context.Context, id string) error {
	// err := t.call(c, methodWait, id, nil)
	// if err != nil && err.Error() == ErrCancelled.Error() {
	// 	return ErrCancelled
	// }
	return t.call(c, methodWait, id, nil)
}

// Init signals the pipeline is initialized.
func (t *Client) Init(c context.Context, id string, state State) error {
	params := updateReq{id, state}
	return t.call(c, methodInit, &params, nil)
}

// Done signals the pipeline is complete.
func (t *Client) Done(c context.Context, id string, state State) error {
	params := updateReq{id, state}
	return t.call(c, methodDone, &params, nil)
}

// Extend extends the pipeline deadline.
func (t *Client) Extend(c context.Context, id string) error {
	return t.call(c, methodExtend, id, nil)
}

// Update updates the pipeline state.
func (t *Client) Update(c context.Context, id string, state State) error {
	params := updateReq{id, state}
	return t.call(c, methodUpdate, &params, nil)
}

// Log writes the pipeline log entry.
func (t *Client) Log(c context.Context, id string, line *Line) error {
	params := logReq{id, line}
	return t.call(c, methodLog, &params, nil)
}

// Upload uploads the pipeline artifact.
func (t *Client) Upload(c context.Context, id string, file *File) error {
	params := uploadReq{id, file}
	return t.call(c, methodUpload, params, nil)
}

// Close closes the client connection.
func (t *Client) Close() error {
	t.Lock()
	t.done = true
	t.Unlock()
	return t.conn.Close()
}

// call makes the remote prodedure call. If the call fails due to connectivity
// issues the connection is re-establish and call re-attempted.
func (t *Client) call(ctx context.Context, name string, req, res interface{}) error {
	if err := t.conn.Call(ctx, name, req, res); err == nil {
		return nil
	} else if err != jsonrpc2.ErrClosed && err != io.ErrUnexpectedEOF {
		log.Printf("rpc: error making call: %s", err)
		return err
	} else {
		log.Printf("rpc: error making call: connection closed: %s", err)
	}
	if err := t.openRetry(); err != nil {
		return err
	}
	return t.conn.Call(ctx, name, req, res)
}

// openRetry opens the connection and will retry on failure until
// the connection is successfully open, or the maximum retry count
// is exceeded.
func (t *Client) openRetry() error {
	for i := 0; i < t.retry; i++ {
		err := t.open()
		if err == nil {
			break
		}
		if err == io.EOF {
			return err
		}

		log.Printf("rpc: error re-connecting: %s", err)
		<-time.After(t.backoff)
	}
	return nil
}

// open creates a websocket connection to a peer and establishes a json
// rpc communication stream.
func (t *Client) open() error {
	t.Lock()
	defer t.Unlock()
	if t.done {
		return io.EOF
	}
	header := map[string][]string{
		"Content-Type":  {"application/json-rpc"},
		"Authorization": {"Bearer " + t.token},
	}
	for key, value := range t.headers {
		header[key] = value
	}
	conn, _, err := websocket.DefaultDialer.Dial(t.endpoint, http.Header(header))
	if err != nil {
		return err
	}
	stream := websocketrpc.NewObjectStream(conn)
	t.conn = jsonrpc2.NewConn(context.Background(), stream, nil)
	return nil
}
