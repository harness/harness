package transform

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/drone/drone/yaml"
)

// CommandTransform transforms the custom shell commands in the Yaml pipeline
// into a container ENTRYPOINT and and CMD for execution.
func CommandTransform(c *yaml.Config) error {
	for _, p := range c.Pipeline {

		if isPlugin(p) {
			continue
		}

		p.Entrypoint = []string{
			"/bin/sh", "-c",
		}
		p.Command = []string{
			"echo $DRONE_SCRIPT | base64 -d | /bin/sh -e",
		}
		if p.Environment == nil {
			p.Environment = map[string]string{}
		}
		p.Environment["HOME"] = "/root"
		p.Environment["SHELL"] = "/bin/sh"
		p.Environment["DRONE_SCRIPT"] = toScript(
			p.Commands,
		)
	}
	return nil
}

func toScript(commands []string) string {
	var buf bytes.Buffer
	for _, command := range commands {
		escaped := fmt.Sprintf("%q", command)
		escaped = strings.Replace(escaped, "$", `\$`, -1)
		buf.WriteString(fmt.Sprintf(
			traceScript,
			escaped,
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
if [ -n "$DRONE_NETRC_MACHINE" ]; then
cat <<EOF > $HOME/.netrc
machine $DRONE_NETRC_MACHINE
login $DRONE_NETRC_USERNAME
password $DRONE_NETRC_PASSWORD
EOF
chmod 0600 $HOME/.netrc
fi

unset DRONE_NETRC_USERNAME
unset DRONE_NETRC_PASSWORD
unset DRONE_SCRIPT

%s
`

// traceScript is a helper script that is added to the build script
// to trace a command.
const traceScript = `
echo + %s
%s
`
