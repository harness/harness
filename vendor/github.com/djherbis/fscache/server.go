package fscache

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
)

// ListenAndServe hosts a Cache for access via NewRemote
func ListenAndServe(c Cache, addr string) error {
	return (&server{c: c}).ListenAndServe(addr)
}

// NewRemote returns a Cache run via ListenAndServe
func NewRemote(raddr string) Cache {
	return &remote{raddr: raddr}
}

type server struct {
	c Cache
}

func (s *server) ListenAndServe(addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}

		go s.Serve(c)
	}
}

const (
	actionGet    = iota
	actionRemove = iota
	actionExists = iota
	actionClean  = iota
)

func getKey(r io.Reader) string {
	dec := newDecoder(r)
	buf := bytes.NewBufferString("")
	io.Copy(buf, dec)
	return buf.String()
}

func sendKey(w io.Writer, key string) {
	enc := newEncoder(w)
	enc.Write([]byte(key))
	enc.Close()
}

func (s *server) Serve(c net.Conn) {
	var action int
	fmt.Fscanf(c, "%d\n", &action)

	switch action {
	case actionGet:
		s.get(c, getKey(c))
	case actionRemove:
		s.c.Remove(getKey(c))
	case actionExists:
		s.exists(c, getKey(c))
	case actionClean:
		s.c.Clean()
	}
}

func (s *server) exists(c net.Conn, key string) {
	if s.c.Exists(key) {
		fmt.Fprintf(c, "%d\n", 1)
	} else {
		fmt.Fprintf(c, "%d\n", 0)
	}
}

func (s *server) get(c net.Conn, key string) {
	r, w, err := s.c.Get(key)
	if err != nil {
		return // handle this better
	}
	defer r.Close()

	if w != nil {
		go func() {
			fmt.Fprintf(c, "%d\n", 1)
			io.Copy(w, newDecoder(c))
			w.Close()
		}()
	} else {
		fmt.Fprintf(c, "%d\n", 0)
	}

	enc := newEncoder(c)
	io.Copy(enc, r)
	enc.Close()
}

type remote struct {
	raddr string
}

func (rmt *remote) Get(key string) (r ReadAtCloser, w io.WriteCloser, err error) {
	c, err := net.Dial("tcp", rmt.raddr)
	if err != nil {
		return nil, nil, err
	}
	fmt.Fprintf(c, "%d\n", actionGet)
	sendKey(c, key)

	var i int
	fmt.Fscanf(c, "%d\n", &i)

	var ch chan struct{}

	switch i {
	case 0:
		ch = make(chan struct{}) // close net.Conn on reader close
	case 1:
		ch = make(chan struct{}, 1) // two closes before net.Conn close

		w = &safeCloser{
			c:  c,
			ch: ch,
			w:  newEncoder(c),
		}
	default:
		return nil, nil, errors.New("bad bad bad")
	}

	r = &safeCloser{
		c:  c,
		ch: ch,
		r:  newDecoder(c),
	}

	return r, w, nil
}

type safeCloser struct {
	c  net.Conn
	ch chan<- struct{}
	r  ReadAtCloser
	w  io.WriteCloser
}

func (s *safeCloser) ReadAt(p []byte, off int64) (int, error) {
	return s.r.ReadAt(p, off)
}
func (s *safeCloser) Read(p []byte) (int, error)  { return s.r.Read(p) }
func (s *safeCloser) Write(p []byte) (int, error) { return s.w.Write(p) }

// Close only closes the underlying connection when ch is full.
func (s *safeCloser) Close() (err error) {
	if s.r != nil {
		err = s.r.Close()
	} else if s.w != nil {
		err = s.w.Close()
	}

	select {
	case s.ch <- struct{}{}:
		return err
	default:
		return s.c.Close()
	}
}

func (rmt *remote) Exists(key string) bool {
	c, err := net.Dial("tcp", rmt.raddr)
	if err != nil {
		return false
	}
	fmt.Fprintf(c, "%d\n", actionExists)
	sendKey(c, key)
	var i int
	fmt.Fscanf(c, "%d\n", &i)
	return i == 1
}

func (rmt *remote) Remove(key string) error {
	c, err := net.Dial("tcp", rmt.raddr)
	if err != nil {
		return err
	}
	fmt.Fprintf(c, "%d\n", actionRemove)
	sendKey(c, key)
	return nil
}

func (rmt *remote) Clean() error {
	c, err := net.Dial("tcp", rmt.raddr)
	if err != nil {
		return err
	}
	fmt.Fprintf(c, "%d\n", actionClean)
	return nil
}
