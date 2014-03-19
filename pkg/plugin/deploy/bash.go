package deploy

import (
	"github.com/drone/drone/pkg/build/buildfile"
)

type Bash struct {
	Command string `yaml:"command,omitempty"`
}

func (g *Bash) Write(f *buildfile.Buildfile) {
	f.WriteCmd(g.Command)
}
