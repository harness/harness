package build

import "fmt"

const (
	StdoutLine int = iota
	StderrLine
	ExitCodeLine
	MetadataLine
	ProgressLine
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
	case ExitCodeLine:
		return fmt.Sprintf("[%s] exit code %s", l.Proc, l.Out)
	default:
		return fmt.Sprintf("[%s:L%v:%vs] %s", l.Proc, l.Pos, l.Time, l.Out)
	}
}

// State defines the state of the container.
type State struct {
	ExitCode  int  // container exit code
	OOMKilled bool // container exited due to oom error
}
