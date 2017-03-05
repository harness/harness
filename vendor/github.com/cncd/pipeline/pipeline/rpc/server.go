package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/sourcegraph/jsonrpc2"
	websocketrpc "github.com/sourcegraph/jsonrpc2/websocket"
)

// errNoSuchMethod is returned when the name rpc method does not exist.
var errNoSuchMethod = errors.New("No such rpc method")

// noContext is an empty context used when no context is required.
var noContext = context.Background()

// Server represents an rpc server.
type Server struct {
	peer Peer
}

// NewServer returns an rpc Server.
func NewServer(peer Peer) *Server {
	return &Server{peer}
}

// ServeHTTP implements an http.Handler that answers rpc requests.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	conn := jsonrpc2.NewConn(ctx,
		websocketrpc.NewObjectStream(c),
		jsonrpc2.HandlerWithError(s.router),
	)
	defer func() {
		cancel()
		conn.Close()
	}()
	<-conn.DisconnectNotify()
}

// router implements an jsonrpc2.Handler that answers RPC requests.
func (s *Server) router(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (interface{}, error) {
	switch req.Method {
	case methodNext:
		return s.next(ctx, req)
	case methodNotify:
		return s.notify(ctx, req)
	case methodExtend:
		return s.extend(ctx, req)
	case methodUpdate:
		return s.update(req)
	case methodLog:
		return s.log(req)
	case methodSave:
		return s.save(req)
	default:
		return nil, errNoSuchMethod
	}
}

// next unmarshals the rpc request parameters and invokes the peer.Next
// procedure. The results are retuned and written to the rpc response.
func (s *Server) next(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	return s.peer.Next(ctx)
}

// notify unmarshals the rpc request parameters and invokes the peer.Notify
// procedure. The results are retuned and written to the rpc response.
func (s *Server) notify(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	var id string
	err := json.Unmarshal([]byte(*req.Params), &id)
	if err != nil {
		return nil, err
	}
	return s.peer.Notify(ctx, id)
}

// extend unmarshals the rpc request parameters and invokes the peer.Extend
// procedure. The results are retuned and written to the rpc response.
func (s *Server) extend(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	var id string
	err := json.Unmarshal([]byte(*req.Params), &id)
	if err != nil {
		return nil, err
	}
	return nil, s.peer.Extend(ctx, id)
}

// update unmarshals the rpc request parameters and invokes the peer.Update
// procedure. The results are retuned and written to the rpc response.
func (s *Server) update(req *jsonrpc2.Request) (interface{}, error) {
	in := new(updateReq)
	if err := json.Unmarshal([]byte(*req.Params), in); err != nil {
		return nil, err
	}
	return nil, s.peer.Update(noContext, in.ID, in.State)
}

// log unmarshals the rpc request parameters and invokes the peer.Log
// procedure. The results are retuned and written to the rpc response.
func (s *Server) log(req *jsonrpc2.Request) (interface{}, error) {
	in := new(logReq)
	if err := json.Unmarshal([]byte(*req.Params), in); err != nil {
		return nil, err
	}
	return nil, s.peer.Log(noContext, in.ID, in.Line)
}

// save unmarshals the rpc request parameters and invokes the peer.Save
// procedure. The results are retuned and written to the rpc response.
func (s *Server) save(req *jsonrpc2.Request) (interface{}, error) {
	in := new(saveReq)
	if err := json.Unmarshal([]byte(*req.Params), in); err != nil {
		return nil, err
	}
	return nil, s.peer.Save(noContext, in.ID, in.Mime, bytes.NewBuffer(in.Data))
}
