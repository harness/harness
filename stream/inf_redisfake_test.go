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

package stream

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// infRedisErr is replied as a RESP error.
type infRedisErr string

// infRedisSimple is replied as a RESP simple string.
type infRedisSimple string

// infRedisHandler produces the reply for a command. Supported reply types:
// nil (null), infRedisErr, infRedisSimple, string (bulk), int/int64 and []any (array).
type infRedisHandler func(args []string) any

// infRedisFake is a tiny in-process RESP server reachable only through net.Pipe connections,
// so tests never open real sockets.
type infRedisFake struct {
	handler infRedisHandler

	mx   sync.Mutex
	cmds [][]string
}

func infNewRedisFake(h infRedisHandler) *infRedisFake {
	return &infRedisFake{handler: h}
}

// client returns a redis client whose connections are served by the fake.
func (f *infRedisFake) client() *redis.Client {
	return redis.NewClient(&redis.Options{
		MaxRetries: -1,
		PoolSize:   16,
		Dialer: func(context.Context, string, string) (net.Conn, error) {
			clientConn, serverConn := net.Pipe()
			go f.serve(serverConn)
			return clientConn, nil
		},
	})
}

// commands returns a copy of all commands received so far.
func (f *infRedisFake) commands() [][]string {
	f.mx.Lock()
	defer f.mx.Unlock()
	out := make([][]string, len(f.cmds))
	copy(out, f.cmds)
	return out
}

// count returns how many received commands start with the given (case-insensitive) words.
func (f *infRedisFake) count(words ...string) int {
	n := 0
	for _, cmd := range f.commands() {
		if len(cmd) < len(words) {
			continue
		}
		match := true
		for i, w := range words {
			if !strings.EqualFold(cmd[i], w) {
				match = false
				break
			}
		}
		if match {
			n++
		}
	}
	return n
}

func (f *infRedisFake) serve(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	for {
		args, err := infReadCommand(r)
		if err != nil {
			return
		}
		f.mx.Lock()
		f.cmds = append(f.cmds, args)
		f.mx.Unlock()

		reply := f.handler(args)
		if _, err = conn.Write(infEncodeReply(reply)); err != nil {
			return
		}
	}
}

func infReadCommand(r *bufio.Reader) ([]string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(line, "*") {
		return nil, fmt.Errorf("unexpected line %q", line)
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

func infEncodeReply(v any) []byte {
	switch t := v.(type) {
	case nil:
		return []byte("$-1\r\n")
	case infRedisErr:
		return []byte("-" + string(t) + "\r\n")
	case infRedisSimple:
		return []byte("+" + string(t) + "\r\n")
	case string:
		return []byte(fmt.Sprintf("$%d\r\n%s\r\n", len(t), t))
	case int:
		return []byte(fmt.Sprintf(":%d\r\n", t))
	case int64:
		return []byte(fmt.Sprintf(":%d\r\n", t))
	case []any:
		out := []byte(fmt.Sprintf("*%d\r\n", len(t)))
		for _, e := range t {
			out = append(out, infEncodeReply(e)...)
		}
		return out
	default:
		panic(fmt.Sprintf("unsupported reply type %T", v))
	}
}

// infStreamMsg builds a stream entry reply: [id, [k1, v1, ...]].
func infStreamMsg(id string, kv ...string) []any {
	fields := make([]any, len(kv))
	for i, s := range kv {
		fields[i] = s
	}
	return []any{id, fields}
}

const (
	infCmdXGroup     = "xgroup"
	infCmdXReadGroup = "xreadgroup"
	infStreamKey     = "ns:s"
)

// infStreamReply builds an XREADGROUP reply for the single stream used by the tests.
func infStreamReply(msgs ...[]any) []any {
	entries := make([]any, len(msgs))
	for i, m := range msgs {
		entries[i] = m
	}
	return []any{[]any{infStreamKey, entries}}
}

// infSlowNil sleeps briefly before replying null so polling loops don't spin.
func infSlowNil() any {
	time.Sleep(5 * time.Millisecond)
	return nil
}
