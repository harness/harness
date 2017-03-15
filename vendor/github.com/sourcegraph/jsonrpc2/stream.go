package jsonrpc2

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

// An ObjectStream is a bidirectional stream of JSON-RPC 2.0 objects.
type ObjectStream interface {
	// WriteObject writes a JSON-RPC 2.0 object to the stream.
	WriteObject(obj interface{}) error

	// ReadObject reads the next JSON-RPC 2.0 object from the stream
	// and stores it in the value pointed to by v.
	ReadObject(v interface{}) error

	io.Closer
}

// A bufferedObjectStream is an ObjectStream that uses a buffered
// io.ReadWriteCloser to send and receive objects.
type bufferedObjectStream struct {
	conn io.Closer // all writes should go through w, all reads through r
	w    *bufio.Writer
	r    *bufio.Reader

	codec ObjectCodec

	mu sync.Mutex
}

// NewBufferedStream creates a buffered stream from a network
// connection (or other similar interface). The underlying
// objectStream is used to produce the bytes to write to the stream
// for the JSON-RPC 2.0 objects.
func NewBufferedStream(conn io.ReadWriteCloser, codec ObjectCodec) ObjectStream {
	return &bufferedObjectStream{
		conn:  conn,
		w:     bufio.NewWriter(conn),
		r:     bufio.NewReader(conn),
		codec: codec,
	}
}

// WriteObject implements ObjectStream.
func (t *bufferedObjectStream) WriteObject(obj interface{}) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if err := t.codec.WriteObject(t.w, obj); err != nil {
		return err
	}
	return t.w.Flush()
}

// ReadObject implements ObjectStream.
func (t *bufferedObjectStream) ReadObject(v interface{}) error {
	return t.codec.ReadObject(t.r, v)
}

// Close implements ObjectStream.
func (t *bufferedObjectStream) Close() error {
	return t.conn.Close()
}

// An ObjectCodec specifies how to encoed and decode a JSON-RPC 2.0
// object in a stream.
type ObjectCodec interface {
	// WriteObject writes a JSON-RPC 2.0 object to the stream.
	WriteObject(stream io.Writer, obj interface{}) error

	// ReadObject reads the next JSON-RPC 2.0 object from the stream
	// and stores it in the value pointed to by v.
	ReadObject(stream *bufio.Reader, v interface{}) error
}

// VarintObjectCodec reads/writes JSON-RPC 2.0 objects with a varint
// header that encodes the byte length.
type VarintObjectCodec struct{}

// WriteObject implements ObjectCodec.
func (VarintObjectCodec) WriteObject(stream io.Writer, obj interface{}) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	var buf [binary.MaxVarintLen64]byte
	b := binary.PutUvarint(buf[:], uint64(len(data)))
	if _, err := stream.Write(buf[:b]); err != nil {
		return err
	}
	if _, err := stream.Write(data); err != nil {
		return err
	}
	return nil
}

// ReadObject implements ObjectCodec.
func (VarintObjectCodec) ReadObject(stream *bufio.Reader, v interface{}) error {
	b, err := binary.ReadUvarint(stream)
	if err != nil {
		return err
	}
	return json.NewDecoder(io.LimitReader(stream, int64(b))).Decode(v)
}

// VSCodeObjectCodec reads/writes JSON-RPC 2.0 objects with
// Content-Length and Content-Type headers, as specified by
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#base-protocol.
type VSCodeObjectCodec struct{}

// WriteObject implements ObjectCodec.
func (VSCodeObjectCodec) WriteObject(stream io.Writer, obj interface{}) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stream, "Content-Length: %d\r\n", len(data)); err != nil {
		return err
	}
	if _, err := fmt.Fprint(stream, "Content-Type: application/vscode-jsonrpc; charset=utf8\r\n\r\n"); err != nil {
		return err
	}
	if _, err := stream.Write(data); err != nil {
		return err
	}
	return nil
}

// ReadObject implements ObjectCodec.
func (VSCodeObjectCodec) ReadObject(stream *bufio.Reader, v interface{}) error {
	var contentLength uint64
	for {
		line, err := stream.ReadString('\r')
		if err != nil {
			return err
		}
		b, err := stream.ReadByte()
		if err != nil {
			return err
		}
		if b != '\n' {
			return fmt.Errorf(`jsonrpc2: line endings must be \r\n`)
		}
		if line == "\r" {
			break
		}
		if strings.HasPrefix(line, "Content-Length: ") {
			line = strings.TrimPrefix(line, "Content-Length: ")
			line = strings.TrimSpace(line)
			var err error
			contentLength, err = strconv.ParseUint(line, 10, 32)
			if err != nil {
				return err
			}
		}
	}
	if contentLength == 0 {
		return fmt.Errorf("jsonrpc2: no Content-Length header found")
	}
	return json.NewDecoder(io.LimitReader(stream, int64(contentLength))).Decode(v)
}
