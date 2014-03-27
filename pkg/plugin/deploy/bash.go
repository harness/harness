package deploy

import (
	"github.com/drone/drone/pkg/build/buildfile"
)

type Bash struct {
	Script  []string `yaml:"script,omitempty"`
	Command string   `yaml:"command,omitempty"`
}

func (g *Bash) Write(f *buildfile.Buildfile) {
	g.Script = append(g.Script, g.Command)

	for _, cmd := range g.Script {
		f.WriteCmd(cmd)
	}
}
