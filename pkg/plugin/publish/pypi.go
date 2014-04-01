package publish

import (
	"fmt"
	"github.com/drone/drone/pkg/build/buildfile"
)

// set up the .pypirc file
var pypirc = `
cat <<EOF > $HOME/.pypirc
[distutils]
index-servers = 
    pypi

[pypi]
username:%s
password:%s
EOF`

var deployCmd = `
if [ -z $_PYPI_SETUP_PY ]
then
    python $_PYPI_SETUP_PY sdist %s upload
    if [ $? -ne 0 ]
    then
        echo "Deploy to PyPI failed - perhaps due to the version number not being incremented. Continuing..."
    fi
else
    echo "Failed to find setup.py file"
fi
`

type PyPI struct {
	Username string   `yaml:"username,omitempty"`
	Password string   `yaml:"password,omitempty"`
	Formats  []string `yaml:"formats,omitempty"`
}

func (p *PyPI) Write(f *buildfile.Buildfile) {
	if len(p.Username) == 0 || len(p.Password) == 0 {
		// nothing to do if the config is fundamentally flawed
		return
	}
	f.WriteCmdSilent("echo 'publishing to PyPI...'")

	// find the setup.py file
	f.WriteCmdSilent("_PYPI_SETUP_PY=$(find . -name 'setup.py')")

	// build the .pypirc file that pypi expects
	f.WriteCmdSilent(fmt.Sprintf(pypirc, p.Username, p.Password))
	formatStr := p.BuildFormatStr()

	// if we found the setup.py file use it to deploy
	f.WriteCmdSilent(fmt.Sprintf(deployCmd, formatStr))
}

func (p *PyPI) BuildFormatStr() string {
	if len(p.Formats) == 0 {
		// the format parameter is optional - if it's not here,
		// omit the format string completely.
		return ""
	}
	fmtStr := "--formats "
	for i := range p.Formats {
		fmtStr += p.Formats[i] + ","
	}
	return fmtStr[:len(fmtStr)-1]
}
