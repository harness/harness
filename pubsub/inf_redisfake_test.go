// Copyright 2026 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pubsub

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/go-redis/redis/v8"
)

// infRedisConn is one client connection served by the fake. Replies are queued and written by a
// dedicated goroutine because net.Pipe is unbuffered: a synchronous write could otherwise deadlock
// with a client that is itself blocked writing a command.
type infRedisConn struct {
	conn net.Conn
	out  chan []byte
	once sync.Once
}

func infNewConn(conn net.Conn) *infRedisConn {
	c := &infRedisConn{conn: conn, out: make(chan []byte, 1024)}
	go func() {
		for b := range c.out {
			if _, err := c.conn.Write(b); err != nil {
				return
			}
		}
	}()
	return c
}

func (c *infRedisConn) write(b []byte) error {
	c.out <- b
	return nil
}

func (c *infRedisConn) close() {
	c.once.Do(func() { close(c.out) })
	_ = c.conn.Close()
}

// infRedisFake is an in-process RESP server supporting PUBLISH / SUBSCRIBE; it is only
// reachable through net.Pipe connections so tests never open real sockets.
type infRedisFake struct {
	mx          sync.Mutex
	cmds        [][]string
	subscribers map[string][]*infRedisConn
	publishErr  string
	subscribed  chan string
}

func infNewRedisFake() *infRedisFake {
	return &infRedisFake{subscribers: map[string][]*infRedisConn{}, subscribed: make(chan string, 64)}
}

func (f *infRedisFake) client() *redis.Client {
	return redis.NewClient(&redis.Options{
		MaxRetries: -1,
		Dialer: func(context.Context, string, string) (net.Conn, error) {
			clientConn, serverConn := net.Pipe()
			go f.serve(infNewConn(serverConn))
			return clientConn, nil
		},
	})
}

func (f *infRedisFake) commands() [][]string {
	f.mx.Lock()
	defer f.mx.Unlock()
	return append([][]string(nil), f.cmds...)
}

func (f *infRedisFake) setPublishErr(msg string) {
	f.mx.Lock()
	defer f.mx.Unlock()
	f.publishErr = msg
}

func (f *infRedisFake) serve(c *infRedisConn) {
	defer c.close()
	r := bufio.NewReader(c.conn)
	for {
		args, err := infReadCommand(r)
		if err != nil {
			f.drop(c)
			return
		}
		f.mx.Lock()
		f.cmds = append(f.cmds, args)
		f.mx.Unlock()

		if err = f.handle(c, args); err != nil {
			f.drop(c)
			return
		}
	}
}

func (f *infRedisFake) drop(c *infRedisConn) {
	f.mx.Lock()
	defer f.mx.Unlock()
	for ch, conns := range f.subscribers {
		kept := conns[:0]
		for _, other := range conns {
			if other != c {
				kept = append(kept, other)
			}
		}
		f.subscribers[ch] = kept
	}
}

func (f *infRedisFake) handle(c *infRedisConn, args []string) error {
	switch strings.ToLower(args[0]) {
	case "subscribe":
		for i, ch := range args[1:] {
			f.mx.Lock()
			f.subscribers[ch] = append(f.subscribers[ch], c)
			f.mx.Unlock()
			reply := fmt.Sprintf("*3\r\n$9\r\nsubscribe\r\n$%d\r\n%s\r\n:%d\r\n", len(ch), ch, i+1)
			if err := c.write([]byte(reply)); err != nil {
				return err
			}
			f.subscribed <- ch
		}
		return nil
	case "unsubscribe":
		for _, ch := range args[1:] {
			if err := c.write([]byte(fmt.Sprintf("*3\r\n$11\r\nunsubscribe\r\n$%d\r\n%s\r\n:0\r\n", len(ch), ch))); err != nil {
				return err
			}
		}
		return nil
	case "ping":
		return c.write([]byte("*2\r\n$4\r\npong\r\n$0\r\n\r\n"))
	case "publish":
		f.mx.Lock()
		errMsg := f.publishErr
		targets := append([]*infRedisConn(nil), f.subscribers[args[1]]...)
		f.mx.Unlock()
		if errMsg != "" {
			return c.write([]byte("-" + errMsg + "\r\n"))
		}
		for _, target := range targets {
			msg := fmt.Sprintf("*3\r\n$7\r\nmessage\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n",
				len(args[1]), args[1], len(args[2]), args[2])
			if err := target.write([]byte(msg)); err != nil {
				return err
			}
		}
		return c.write([]byte(fmt.Sprintf(":%d\r\n", len(targets))))
	default:
		return c.write([]byte("+OK\r\n"))
	}
}

func infReadCommand(r *bufio.Reader) ([]string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(line[1:]))
	if err != nil {
		return nil, err
	}
	args := make([]string, 0, n)
	for i := 0; i < n; i++ {
		hdr, hdrErr := r.ReadString('\n')
		if hdrErr != nil {
			return nil, hdrErr
		}
		size, sizeErr := strconv.Atoi(strings.TrimSpace(hdr[1:]))
		if sizeErr != nil {
			return nil, sizeErr
		}
		buf := make([]byte, size+2)
		if _, readErr := io.ReadFull(r, buf); readErr != nil {
			return nil, readErr
		}
		args = append(args, string(buf[:size]))
	}
	return args, nil
}
