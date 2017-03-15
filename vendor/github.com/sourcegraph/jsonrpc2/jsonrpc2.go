// Package jsonrpc2 provides a client and server implementation of
// [JSON-RPC 2.0](http://www.jsonrpc.org/specification).
package jsonrpc2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"sync"
)

// JSONRPC2 describes an interface for issuing requests that speak the
// JSON-RPC 2 protocol.  It isn't really necessary for this package
// itself, but is useful for external users that use the interface as
// an API boundary.
type JSONRPC2 interface {
	// Call issues a standard request (http://www.jsonrpc.org/specification#request_object).
	Call(ctx context.Context, method string, params, result interface{}, opt ...CallOption) error

	// Notify issues a notification request (http://www.jsonrpc.org/specification#notification).
	Notify(ctx context.Context, method string, params interface{}, opt ...CallOption) error

	// Close closes the underlying connection, if it exists.
	Close() error
}

// Request represents a JSON-RPC request or
// notification. See
// http://www.jsonrpc.org/specification#request_object and
// http://www.jsonrpc.org/specification#notification.
type Request struct {
	Method string           `json:"method"`
	Params *json.RawMessage `json:"params,omitempty"`
	ID     ID               `json:"id"`
	Notif  bool             `json:"-"`

	// Meta optionally provides metadata to include in the request.
	//
	// NOTE: It is not part of spec. However, it is useful for propogating
	// tracing context, etc.
	Meta *json.RawMessage `json:"meta,omitempty"`
}

// MarshalJSON implements json.Marshaler and adds the "jsonrpc":"2.0"
// property.
func (r Request) MarshalJSON() ([]byte, error) {
	r2 := struct {
		Method  string           `json:"method"`
		Params  *json.RawMessage `json:"params,omitempty"`
		ID      *ID              `json:"id,omitempty"`
		Meta    *json.RawMessage `json:"meta,omitempty"`
		JSONRPC string           `json:"jsonrpc"`
	}{
		Method:  r.Method,
		Params:  r.Params,
		Meta:    r.Meta,
		JSONRPC: "2.0",
	}
	if !r.Notif {
		r2.ID = &r.ID
	}
	return json.Marshal(r2)
}

// UnmarshalJSON implements json.Unmarshaler.
func (r *Request) UnmarshalJSON(data []byte) error {
	var r2 struct {
		Method string           `json:"method"`
		Params *json.RawMessage `json:"params,omitempty"`
		Meta   *json.RawMessage `json:"meta,omitempty"`
		ID     *ID              `json:"id"`
	}

	// Detect if the "params" field is JSON "null" or just not present
	// by seeing if the field gets overwritten to nil.
	r2.Params = &json.RawMessage{}

	if err := json.Unmarshal(data, &r2); err != nil {
		return err
	}
	r.Method = r2.Method
	if r2.Params == nil {
		r.Params = &jsonNull
	} else if len(*r2.Params) == 0 {
		r.Params = nil
	} else {
		r.Params = r2.Params
	}
	r.Meta = r2.Meta
	if r2.ID == nil {
		r.ID = ID{}
		r.Notif = true
	} else {
		r.ID = *r2.ID
		r.Notif = false
	}
	return nil
}

// SetParams sets r.Params to the JSON representation of v. If JSON
// marshaling fails, it returns an error.
func (r *Request) SetParams(v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	r.Params = (*json.RawMessage)(&b)
	return nil
}

// SetMeta sets r.Meta to the JSON representation of v. If JSON
// marshaling fails, it returns an error.
func (r *Request) SetMeta(v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	r.Meta = (*json.RawMessage)(&b)
	return nil
}

// Response represents a JSON-RPC response. See
// http://www.jsonrpc.org/specification#response_object.
type Response struct {
	ID     ID               `json:"id"`
	Result *json.RawMessage `json:"result"`
	Error  *Error           `json:"error,omitempty"`

	// Meta optionally provides metadata to include in the response.
	//
	// NOTE: It is not part of spec. However, it is useful for propogating
	// tracing context, etc.
	Meta *json.RawMessage `json:"meta,omitempty"`

	// SPEC NOTE: The spec says "If there was an error in detecting
	// the id in the Request object (e.g. Parse error/Invalid
	// Request), it MUST be Null." If we made the ID field nullable,
	// then we'd have to make it a pointer type. For simplicity, we're
	// ignoring the case where there was an error in detecting the ID
	// in the Request object.
}

// MarshalJSON implements json.Marshaler and adds the "jsonrpc":"2.0"
// property.
func (r Response) MarshalJSON() ([]byte, error) {
	if (r.Result == nil || len(*r.Result) == 0) && r.Error == nil {
		return nil, errors.New("can't marshal *jsonrpc2.Response (must have result or error)")
	}
	type tmpType Response // avoid infinite MarshalJSON recursion
	b, err := json.Marshal(tmpType(r))
	if err != nil {
		return nil, err
	}
	b = append(b[:len(b)-1], []byte(`,"jsonrpc":"2.0"}`)...)
	return b, nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (r *Response) UnmarshalJSON(data []byte) error {
	type tmpType Response

	// Detect if the "result" field is JSON "null" or just not present
	// by seeing if the field gets overwritten to nil.
	*r = Response{Result: &json.RawMessage{}}

	if err := json.Unmarshal(data, (*tmpType)(r)); err != nil {
		return err
	}
	if r.Result == nil { // JSON "null"
		r.Result = &jsonNull
	} else if len(*r.Result) == 0 {
		r.Result = nil
	}
	return nil
}

// SetResult sets r.Result to the JSON representation of v. If JSON
// marshaling fails, it returns an error.
func (r *Response) SetResult(v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	r.Result = (*json.RawMessage)(&b)
	return nil
}

// Error represents a JSON-RPC response error.
type Error struct {
	Code    int64            `json:"code"`
	Message string           `json:"message"`
	Data    *json.RawMessage `json:"data"`
}

// SetError sets e.Error to the JSON representation of v. If JSON
// marshaling fails, it panics.
func (e *Error) SetError(v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		panic("Error.SetData: " + err.Error())
	}
	e.Data = (*json.RawMessage)(&b)
}

// Error implements the Go error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("jsonrpc2: code %v message: %s", e.Code, e.Message)
}

const (
	// Errors defined in the JSON-RPC spec. See
	// http://www.jsonrpc.org/specification#error_object.
	CodeParseError       = -32700
	CodeInvalidRequest   = -32600
	CodeMethodNotFound   = -32601
	CodeInvalidParams    = -32602
	CodeInternalError    = -32603
	codeServerErrorStart = -32099
	codeServerErrorEnd   = -32000
)

// Handler handles JSON-RPC requests and notifications.
type Handler interface {
	// Handle is called to handle a request.
	Handle(context.Context, *Conn, *Request)
}

// ID represents a JSON-RPC 2.0 request ID, which may be either a
// string or number (or null, which is unsupported).
type ID struct {
	// At most one of Num or Str may be nonzero. If both are zero
	// valued, then IsNum specifies which field's value is to be used
	// as the ID.
	Num uint64
	Str string

	// IsString controls whether the Num or Str field's value should be
	// used as the ID, when both are zero valued. It must always be
	// set to true if the request ID is a string.
	IsString bool
}

func (id ID) String() string {
	if id.IsString {
		return strconv.Quote(id.Str)
	}
	return strconv.FormatUint(id.Num, 10)
}

// MarshalJSON implements json.Marshaler.
func (id ID) MarshalJSON() ([]byte, error) {
	if id.IsString {
		return json.Marshal(id.Str)
	}
	return json.Marshal(id.Num)
}

// UnmarshalJSON implements json.Unmarshaler.
func (id *ID) UnmarshalJSON(data []byte) error {
	// Support both uint64 and string IDs.
	var v uint64
	if err := json.Unmarshal(data, &v); err == nil {
		*id = ID{Num: v}
		return nil
	}
	var v2 string
	if err := json.Unmarshal(data, &v2); err != nil {
		return err
	}
	*id = ID{Str: v2, IsString: true}
	return nil
}

// Conn is a JSON-RPC client/server connection. The JSON-RPC protocol
// is symmetric, so a Conn runs on both ends of a client-server
// connection.
type Conn struct {
	stream ObjectStream

	h Handler

	mu       sync.Mutex
	shutdown bool
	closing  bool
	seq      uint64
	pending  map[ID]*call

	sending sync.Mutex

	disconnect chan struct{}

	// Set by ConnOpt funcs.
	onRecv func(*Request, *Response)
	onSend func(*Request, *Response)
}

var _ JSONRPC2 = (*Conn)(nil)

// ErrClosed indicates that the JSON-RPC connection is closed (or in
// the process of closing).
var ErrClosed = errors.New("jsonrpc2: connection is closed")

// NewConn creates a new JSON-RPC client/server connection using the
// given ReadWriteCloser (typically a TCP connection or stdio). The
// JSON-RPC protocol is symmetric, so a Conn runs on both ends of a
// client-server connection.
//
// NewClient consumes conn, so you should call Close on the returned
// client not on the given conn.
func NewConn(ctx context.Context, stream ObjectStream, h Handler, opt ...ConnOpt) *Conn {
	c := &Conn{
		stream:     stream,
		h:          h,
		pending:    map[ID]*call{},
		disconnect: make(chan struct{}),
	}
	for _, opt := range opt {
		opt(c)
	}
	go c.readMessages(ctx)
	return c
}

// Close closes the JSON-RPC connection. The connection may not be
// used after it has been closed.
func (c *Conn) Close() error {
	c.mu.Lock()
	if c.shutdown || c.closing {
		c.mu.Unlock()
		return ErrClosed
	}
	c.closing = true
	c.mu.Unlock()
	return c.stream.Close()
}

func (c *Conn) send(ctx context.Context, m *anyMessage, wait bool) (cc *call, err error) {
	c.sending.Lock()
	defer c.sending.Unlock()

	// m.request.ID could be changed, so we store a copy to correctly
	// clean up pending
	var id ID

	c.mu.Lock()
	if c.shutdown || c.closing {
		c.mu.Unlock()
		return nil, ErrClosed
	}

	// Store requests so we can later associate them with incoming
	// responses.
	if m.request != nil && wait {
		cc = &call{request: m.request, seq: c.seq, done: make(chan error, 1)}
		if !m.request.ID.IsString && m.request.ID.Num == 0 {
			// unset, use next seq as call ID
			m.request.ID.Num = c.seq
		}
		id = m.request.ID
		c.pending[id] = cc
		c.seq++
	}
	c.mu.Unlock()

	if c.onSend != nil {
		switch {
		case m.request != nil:
			c.onSend(m.request, nil)
		case m.response != nil:
			c.onSend(nil, m.response)
		}
	}

	// From here on, if we fail to send this, then we need to remove
	// this from the pending map so we don't block on it or pile up
	// pending entries for unsent messages.
	defer func() {
		if err != nil {
			if cc != nil {
				c.mu.Lock()
				delete(c.pending, id)
				c.mu.Unlock()
			}
		}
	}()

	if err := c.stream.WriteObject(m); err != nil {
		return nil, err
	}
	return cc, nil
}

// Call initiates a JSON-RPC call using the specified method and
// params, and waits for the response. If the response is successful,
// its result is stored in result (a pointer to a value that can be
// JSON-unmarshaled into); otherwise, a non-nil error is returned.
func (c *Conn) Call(ctx context.Context, method string, params, result interface{}, opts ...CallOption) error {
	req := &Request{Method: method}
	if err := req.SetParams(params); err != nil {
		return err
	}
	for _, opt := range opts {
		if err := opt.apply(req); err != nil {
			return err
		}
	}
	call, err := c.send(ctx, &anyMessage{request: req}, true)
	if err != nil {
		return err
	}
	select {
	case err, ok := <-call.done:
		if !ok {
			err = ErrClosed
		}
		if err != nil {
			return err
		}
		if result != nil {
			if call.response.Result == nil {
				call.response.Result = &jsonNull
			}
			// TODO(sqs): error handling
			if err := json.Unmarshal(*call.response.Result, result); err != nil {
				return err
			}
		}
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

var jsonNull = json.RawMessage("null")

// Notify is like Call, but it returns when the notification request
// is sent (without waiting for a response, because JSON-RPC
// notifications do not have responses).
func (c *Conn) Notify(ctx context.Context, method string, params interface{}, opts ...CallOption) error {
	req := &Request{Method: method, Notif: true}
	if err := req.SetParams(params); err != nil {
		return err
	}
	for _, opt := range opts {
		if err := opt.apply(req); err != nil {
			return err
		}
	}
	_, err := c.send(ctx, &anyMessage{request: req}, false)
	return err
}

// Reply sends a successful response with a result.
func (c *Conn) Reply(ctx context.Context, id ID, result interface{}) error {
	resp := &Response{ID: id}
	if err := resp.SetResult(result); err != nil {
		return err
	}
	_, err := c.send(ctx, &anyMessage{response: resp}, false)
	return err
}

// ReplyWithError sends a response with an error.
func (c *Conn) ReplyWithError(ctx context.Context, id ID, respErr *Error) error {
	_, err := c.send(ctx, &anyMessage{response: &Response{ID: id, Error: respErr}}, false)
	return err
}

// SendResponse sends resp to the peer. It is lower level than (*Conn).Reply.
func (c *Conn) SendResponse(ctx context.Context, resp *Response) error {
	_, err := c.send(ctx, &anyMessage{response: resp}, false)
	return err
}

// DisconnectNotify returns a channel that is closed when the
// underlying connection is disconnected.
func (c *Conn) DisconnectNotify() <-chan struct{} {
	return c.disconnect
}

func (c *Conn) readMessages(ctx context.Context) {
	var err error
	for err == nil {
		var m anyMessage
		err = c.stream.ReadObject(&m)
		if err != nil {
			break
		}

		switch {
		case m.request != nil:
			if c.onRecv != nil {
				c.onRecv(m.request, nil)
			}
			go c.h.Handle(ctx, c, m.request)

		case m.response != nil:
			resp := m.response
			if resp != nil {
				id := resp.ID
				c.mu.Lock()
				call := c.pending[id]
				delete(c.pending, id)
				c.mu.Unlock()

				if call != nil {
					call.response = resp
				}

				if c.onRecv != nil {
					var req *Request

					if call != nil {
						req = call.request
					}
					c.onRecv(req, resp)
				}

				switch {
				case call == nil:
					log.Printf("jsonrpc2: ignoring response #%s with no corresponding request", id)

				case resp.Error != nil:
					call.done <- resp.Error
					close(call.done)

				default:
					call.done <- nil
					close(call.done)
				}
			}
		}
	}

	c.sending.Lock()
	c.mu.Lock()
	c.shutdown = true
	closing := c.closing
	if err == io.EOF {
		if closing {
			err = ErrClosed
		} else {
			err = io.ErrUnexpectedEOF
		}
	}
	for _, call := range c.pending {
		call.done <- err
		close(call.done)
	}
	c.mu.Unlock()
	c.sending.Unlock()
	if err != io.ErrUnexpectedEOF && !closing {
		log.Println("jsonrpc2: protocol error:", err)
	}
	close(c.disconnect)
}

// call represents a JSON-RPC call over its entire lifecycle.
type call struct {
	request  *Request
	response *Response
	seq      uint64 // the seq of the request
	done     chan error
}

// anyMessage represents either a JSON Request or Response.
type anyMessage struct {
	request  *Request
	response *Response
}

func (m anyMessage) MarshalJSON() ([]byte, error) {
	var v interface{}
	switch {
	case m.request != nil && m.response == nil:
		v = m.request
	case m.request == nil && m.response != nil:
		v = m.response
	}
	if v != nil {
		return json.Marshal(v)
	}
	return nil, errors.New("jsonrpc2: message must have exactly one of the request or response fields set")
}

func (m *anyMessage) UnmarshalJSON(data []byte) error {
	// The presence of these fields distinguishes between the 2
	// message types.
	type msg struct {
		ID     interface{}              `json:"id"`
		Method *string                  `json:"method"`
		Result anyValueWithExplicitNull `json:"result"`
		Error  interface{}              `json:"error"`
	}

	var isRequest, isResponse bool
	checkType := func(m *msg) error {
		mIsRequest := m.Method != nil
		mIsResponse := m.Result.null || m.Result.value != nil || m.Error != nil
		if (!mIsRequest && !mIsResponse) || (mIsRequest && mIsResponse) {
			return errors.New("jsonrpc2: unable to determine message type (request or response)")
		}
		if (mIsRequest && isResponse) || (mIsResponse && isRequest) {
			return errors.New("jsonrpc2: batch message type mismatch (must be all requests or all responses)")
		}
		isRequest = mIsRequest
		isResponse = mIsResponse
		return nil
	}

	if isArray := len(data) > 0 && data[0] == '['; isArray {
		var msgs []msg
		if err := json.Unmarshal(data, &msgs); err != nil {
			return err
		}
		if len(msgs) == 0 {
			return errors.New("jsonrpc2: invalid empty batch")
		}
		for _, msg := range msgs {
			if err := checkType(&msg); err != nil {
				return err
			}
		}
	} else {
		var msg msg
		if err := json.Unmarshal(data, &msg); err != nil {
			return err
		}
		if err := checkType(&msg); err != nil {
			return err
		}
	}

	var v interface{}
	switch {
	case isRequest && !isResponse:
		v = &m.request
	case !isRequest && isResponse:
		v = &m.response
	}
	if err := json.Unmarshal(data, v); err != nil {
		return err
	}
	if !isRequest && isResponse && m.response.Error == nil && m.response.Result == nil {
		m.response.Result = &jsonNull
	}
	return nil
}

// anyValueWithExplicitNull is used to distinguish {} from
// {"result":null} by anyMessage's JSON unmarshaler.
type anyValueWithExplicitNull struct {
	null  bool // JSON "null"
	value interface{}
}

func (v anyValueWithExplicitNull) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *anyValueWithExplicitNull) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if string(data) == "null" {
		*v = anyValueWithExplicitNull{null: true}
		return nil
	}
	*v = anyValueWithExplicitNull{}
	return json.Unmarshal(data, &v.value)
}

var (
	errInvalidRequestJSON  = errors.New("jsonrpc2: request must be either a JSON object or JSON array")
	errInvalidResponseJSON = errors.New("jsonrpc2: response must be either a JSON object or JSON array")
)
