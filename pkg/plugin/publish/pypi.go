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
    %s

[%s]
username:%s
password:%s
%s
EOF`

var deployCmd = `
if [ -n "$_PYPI_SETUP_PY" ]
then
    python $_PYPI_SETUP_PY sdist %s upload -r %s
    if [ $? -ne 0 ]
    then
        echo "Deploy to PyPI failed - perhaps due to the version number not being incremented. Continuing..."
    fi
else
    echo "Failed to find setup.py file"
fi
`

type PyPI struct {
	Username   string   `yaml:"username,omitempty"`
	Password   string   `yaml:"password,omitempty"`
	Formats    []string `yaml:"formats,omitempty"`
	Repository string   `yaml:"repository,omitempty"`
	Branch     string   `yaml:"branch,omitempty"`
}

func (p *PyPI) Write(f *buildfile.Buildfile) {
	var indexServer string
	var repository string

	if len(p.Username) == 0 || len(p.Password) == 0 {
		// nothing to do if the config is fundamentally flawed
		return
	}

	// Handle the setting a custom pypi server/repository
	if len(p.Repository) == 0 {
		indexServer = "pypi"
		repository = ""
	} else {
		indexServer = "custom"
		repository = fmt.Sprintf("repository:%s", p.Repository)
	}

	f.WriteCmdSilent("echo 'publishing to PyPI...'")

	// find the setup.py file
	f.WriteCmdSilent("_PYPI_SETUP_PY=$(find . -name 'setup.py')")

	// build the .pypirc file that pypi expects
	f.WriteCmdSilent(fmt.Sprintf(pypirc, indexServer, indexServer, p.Username, p.Password, repository))
	formatStr := p.BuildFormatStr()

	// if we found the setup.py file use it to deploy
	f.WriteCmdSilent(fmt.Sprintf(deployCmd, formatStr, indexServer))
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
