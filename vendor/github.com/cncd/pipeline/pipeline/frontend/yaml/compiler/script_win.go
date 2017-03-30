package compiler

import (
	"bytes"
	"fmt"
	"regexp"
)

// generateScriptWindows is a helper function that generates a build script
// for a Windows
func generateScriptWindows(commands []string) string {
	var buf bytes.Buffer
	var re = regexp.MustCompile(`("|>|<|\*|%|&|\||!|=|\(|\)|;|^|,|` + "`" + `)`)

	for _, command := range commands {
		escaped := re.ReplaceAllString(command, `^$1`)
		buf.WriteString(fmt.Sprintf(
			traceScriptWin,
			escaped,
			command,
		))
	}
	script := fmt.Sprintf("%s%s",
		setupScriptWin,
		buf.String(),
	)
	return script
}

// setupScript is a helper script this is added to the build to ensure
// a minimum set of environment variables are set correctly.
const setupScriptWin = `
@ECHO OFF
IF defined CI_NETRC_MACHINE (
ECHO machine %CI_NETRC_MACHINE% >%HOMEPATH%/.netrc
ECHO login %CI_NETRC_USERNAME% >>%HOMEPATH%/.netrc
ECHO password %CI_NETRC_PASSWORD% >>%HOMEPATH%/.netrc
)

SET CI_NETRC_USERNAME=
SET CI_NETRC_PASSWORD=
SET CI_SCRIPT=
`

// traceScript is a helper script that is added to the build script
// to trace a command.
const traceScriptWin = `
echo + %s
%s || exit /b 1
`
