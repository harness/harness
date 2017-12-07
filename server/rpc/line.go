package rpc

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Identifies the type of line in the logs.
const (
	LineStdout int = iota
	LineStderr
	LineExitCode
	LineMetadata
	LineProgress
)

// Line is a line of console output.
type Line struct {
	Proc string `json:"proc,omitempty"`
	Time int64  `json:"time,omitempty"`
	Type int    `json:"type,omitempty"`
	Pos  int    `json:"pos,omityempty"`
	Out  string `json:"out,omitempty"`
}

func (l *Line) String() string {
	switch l.Type {
	case LineExitCode:
		return fmt.Sprintf("[%s] exit code %s", l.Proc, l.Out)
	default:
		return fmt.Sprintf("[%s:L%v:%vs] %s", l.Proc, l.Pos, l.Time, l.Out)
	}
}

// LineWriter sends logs to the client.
type LineWriter struct {
	peer  Peer
	id    string
	name  string
	num   int
	now   time.Time
	rep   *strings.Replacer
	lines []*Line
}

// NewLineWriter returns a new line reader.
func NewLineWriter(peer Peer, id, name string, secret ...string) *LineWriter {
	w := new(LineWriter)
	w.peer = peer
	w.id = id
	w.name = name
	w.num = 0
	w.now = time.Now().UTC()

	var oldnew []string
	for _, old := range secret {
		oldnew = append(oldnew, old)
		oldnew = append(oldnew, "********")
	}
	if len(oldnew) != 0 {
		w.rep = strings.NewReplacer(oldnew...)
	}
	return w
}

func (w *LineWriter) Write(p []byte) (n int, err error) {
	out := string(p)
	if w.rep != nil {
		out = w.rep.Replace(out)
	}

	line := &Line{
		Out:  out,
		Proc: w.name,
		Pos:  w.num,
		Time: int64(time.Since(w.now).Seconds()),
		Type: LineStdout,
	}
	w.peer.Log(context.Background(), w.id, line)
	w.num++

	// for _, part := range bytes.Split(p, []byte{'\n'}) {
	// 	line := &Line{
	// 		Out:  string(part),
	// 		Proc: w.name,
	// 		Pos:  w.num,
	// 		Time: int64(time.Since(w.now).Seconds()),
	// 		Type: LineStdout,
	// 	}
	// 	w.peer.Log(context.Background(), w.id, line)
	// 	w.num++
	// }
	w.lines = append(w.lines, line)
	return len(p), nil
}

// Lines returns the line history
func (w *LineWriter) Lines() []*Line {
	return w.lines
}

// Clear clears the line history
func (w *LineWriter) Clear() {
	w.lines = w.lines[:0]
}
