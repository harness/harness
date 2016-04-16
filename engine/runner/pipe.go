package runner

import "fmt"

// Pipe returns a buffered pipe that is connected to the console output.
type Pipe struct {
	lines chan *Line
	eof   chan bool
}

// Next returns the next Line of console output.
func (p *Pipe) Next() *Line {
	select {
	case line := <-p.lines:
		return line
	case <-p.eof:
		return nil
	}
}

// Close closes the pipe of console output.
func (p *Pipe) Close() {
	go func() {
		p.eof <- true
	}()
}

func newPipe(buffer int) *Pipe {
	return &Pipe{
		lines: make(chan *Line, buffer),
		eof:   make(chan bool),
	}
}

// Line is a line of console output.
type Line struct {
	Proc string `json:"proc,omitempty"`
	Time int64  `json:"time,omitempty"`
	Type int    `json:"type,omitempty"`
	Pos  int    `json:"pos,omityempty"`
	Out  string `json:"out,omitempty"`
}

func (l *Line) String() string {
	return fmt.Sprintf("[%s:L%v:%vs] %s", l.Proc, l.Pos, l.Time, l.Out)
}

// TODO(bradrydzewski) consider an alternate buffer impelmentation based on the
// x.crypto ssh buffer https://github.com/golang/crypto/blob/master/ssh/buffer.go
