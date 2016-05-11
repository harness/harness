package build

import "fmt"

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

// State defines the state of the container.
type State struct {
	ExitCode  int  // container exit code
	OOMKilled bool // container exited due to oom error
}
