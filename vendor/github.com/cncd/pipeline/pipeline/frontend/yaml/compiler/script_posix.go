package compiler

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
)

// generateScriptPosix is a helper function that generates a build script
// for a linux container using the given
func generateScriptPosix(commands []string) string {
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
// TODO: Unsetting DRONE_* is present for backward compatibility and should
// be removed in a future version.
const setupScript = `
if [ -n "$CI_NETRC_MACHINE" ]; then
cat <<EOF > $HOME/.netrc
machine $CI_NETRC_MACHINE
login $CI_NETRC_USERNAME
password $CI_NETRC_PASSWORD
EOF
chmod 0600 $HOME/.netrc
fi
unset CI_NETRC_USERNAME
unset CI_NETRC_PASSWORD
unset CI_SCRIPT
unset DRONE_NETRC_USERNAME
unset DRONE_NETRC_PASSWORD
%s
`

// traceScript is a helper script that is added to the build script
// to trace a command.
const traceScript = `
echo + %s
%s
`
