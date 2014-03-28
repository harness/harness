package publish

import (
	"fmt"
	"github.com/drone/drone/pkg/build/buildfile"
)

// set up the .pypirc file
var pypirc = `
cat <<EOF > $HOME/.pypirc
[pypirc]
servers = pypi
[server-login]
username:%s
password:%s
EOF`

type PyPI struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

func (p *PyPI) Write(f *buildfile.Buildfile) {
	if len(p.Username) == 0 || len(p.Password) == 0 {
		// nothing to do if the config is bad
		return
	}
	f.WriteCmdSilent("echo 'publishing to PyPI...'")

	// find the setup.py file
	f.WriteCmdSilent("_PYPI_SETUP_PY=$(find . -name 'setup.py')")

	f.WriteCmdSilent(fmt.Sprintf(pypirc, p.Username, p.Password))

	// if we found the setup.py file use it to deploy
	f.WriteCmd("[ -z $_PYPI_SETUP_PY ] || python $_PYPI_SETUP_PY sdist --formats gztar,zip upload")
	f.WriteCmd("[ -z $_PYPI_SETUP_PY ] && echo 'Failed to find setup.py file'")
}
