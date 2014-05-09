package publish

import (
    "fmt"

    "github.com/drone/drone/pkg/build/buildfile"
)

// use npm trick instead of running npm adduser that requires stdin
var npmLoginCmd = `
cat <<EOF > ~/.npmrc
_auth = $(echo "%s:%s" | tr -d "\r\n" | base64)
email = %s
EOF
`

type NPM struct {
    // The Email address used by NPM to connect
    // and publish to a repository
    Email string `yaml:"email,omitempty"`

    // The Username used by NPM to connect
    // and publish to a repository
    Username string `yaml:"username,omitempty"`

    // The Password used by NPM to connect
    // and publish to a repository
    Password string `yaml:"password,omitempty"`

    // Fails if the package name and version combination already
    // exists in the registry. Overwrites when the "--force" flag is set.
    Force bool `yaml:"force"`

    // The registry URL of custom npm repository
    Registry string `yaml:"registry,omitempty"`

    // A folder containing the package.json file
    Folder string `yaml:"folder,omitempty"`

    // Registers the published package with the given tag
    Tag string `yaml:"tag,omitempty"`

    Branch string `yaml:"branch,omitempty"`
}

func (n *NPM) Write(f *buildfile.Buildfile) {

    if len(n.Email) == 0 || len(n.Username) == 0 || len(n.Password) == 0 {
        return
    }

    npmPublishCmd := "npm publish %s"

    if n.Tag != "" {
        npmPublishCmd += fmt.Sprintf(" --tag %s", n.Tag)
    }

    if n.Force {
        npmPublishCmd += " --force"
    }

    f.WriteCmdSilent("echo 'publishing to NPM ...'")

    // Login to registry
    f.WriteCmdSilent(fmt.Sprintf(npmLoginCmd, n.Username, n.Password, n.Email))

    // Setup custom npm registry
    if n.Registry != "" {
        f.WriteCmdSilent(fmt.Sprintf("npm config set registry %s", n.Registry))
    }

    f.WriteCmd(fmt.Sprintf(npmPublishCmd, n.Folder))
}