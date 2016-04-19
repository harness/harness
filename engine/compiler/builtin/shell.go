package builtin

import (
	"bytes"
	"encoding/base64"
	"fmt"

	"github.com/drone/drone/engine/compiler/parse"
)

const (
	Freebsd_amd64 = "freebsd_amd64"
	Linux_adm64   = "linux_amd64"
	Windows_amd64 = "windows_amd64"
)

type shellOp struct {
	visitor
	platform string
}

// NewShellOp returns a transformer that converts the shell node to
// a runnable container.
func NewShellOp(platform string) Visitor {
	return &shellOp{
		platform: platform,
	}
}

func (v *shellOp) VisitContainer(node *parse.ContainerNode) error {
	if node.NodeType != parse.NodeShell {
		return nil
	}

	node.Container.Entrypoint = []string{
		"/bin/sh", "-c",
	}
	node.Container.Command = []string{
		"echo $CI_CMDS | base64 -d | /bin/sh -e",
	}
	if node.Container.Environment == nil {
		node.Container.Environment = map[string]string{}
	}
	node.Container.Environment["HOME"] = "/root"
	node.Container.Environment["SHELL"] = "/bin/sh"
	node.Container.Environment["CI_CMDS"] = toScript(
		node.Root().Path,
		node.Commands,
	)

	return nil
}

func toScript(base string, commands []string) string {
	var buf bytes.Buffer
	for _, command := range commands {
		buf.WriteString(fmt.Sprintf(
			traceScript,
			"<command>"+command+"</command>",
			command,
		))
	}

	script := fmt.Sprintf(
		setupScript,
		buf.String(),
	)

	return base64.StdEncoding.EncodeToString([]byte(script))
}

// setupScript is a helper script this is added to the build to ensure
// a minimum set of environment variables are set correctly.
const setupScript = `
echo $DRONE_NETRC > $HOME/.netrc

%s
`

// traceScript is a helper script that is added to the build script
// to trace a command.
const traceScript = `
echo %q
%s
`
