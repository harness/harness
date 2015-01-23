package npm

import (
	"fmt"

	"github.com/drone/config"
	"github.com/drone/drone/plugin/condition"
	"github.com/drone/drone/shared/build/buildfile"
)

// command to create the .npmrc file that stores
// the login credentials, as opposed to npm login
// which requires stdin.
const CmdLogin = `
cat <<EOF > ~/.npmrc
_auth = $(echo "%s:%s" | tr -d "\r\n" | base64)
email = %s
EOF
`

// command to publish npm package if not published
const CmdPublish = `
_NPM_PACKAGE_NAME=$(cd %s && npm list | head -n 1 | cut -d ' ' -f1)
_NPM_PACKAGE_TAG="%s"
if [ -z "$(npm info ${_NPM_PACKAGE_NAME} 2> /dev/null)" ]
then
	npm publish %s
	[ -n ${_NPM_PACKAGE_TAG} ] && npm tag ${_NPM_PACKAGE_NAME} ${_NPM_PACKAGE_TAG}
else
	echo "skipping publish, package ${_NPM_PACKAGE_NAME} already published"
fi
unset _NPM_PACKAGE_NAME
unset _NPM_PACKAGE_TAG
`

const (
	CmdAlwaysAuth  = "npm set always-auth true"
	CmdSetRegistry = "npm config set registry %s"
)

var (
	DefaultUser  = config.String("npm-user", "")
	DefaultPass  = config.String("npm-pass", "")
	DefaultEmail = config.String("npm-email", "")
)

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

	// The registry URL of custom npm repository
	Registry string `yaml:"registry,omitempty"`

	// A folder containing the package.json file
	Folder string `yaml:"folder,omitempty"`

	// Registers the published package with the given tag
	Tag string `yaml:"tag,omitempty"`

	// Force npm to always require authentication when accessing the registry.
	AlwaysAuth bool `yaml:"always_auth"`

	Condition *condition.Condition `yaml:"when,omitempty"`
}

func (n *NPM) Write(f *buildfile.Buildfile) {
	// If the yaml doesn't provide a username or password
	// we should attempt to use the global defaults.
	if len(n.Email) == 0 ||
		len(n.Username) == 0 ||
		len(n.Password) == 0 {
		n.Username = *DefaultUser
		n.Password = *DefaultPass
		n.Email = *DefaultEmail
	}

	// If the yaml doesn't provide a username or password,
	// and there was not global configuration defined, EXIT.
	if len(n.Email) == 0 ||
		len(n.Username) == 0 ||
		len(n.Password) == 0 {
		return
	}

	// Setup the npm credentials
	f.WriteCmdSilent(fmt.Sprintf(CmdLogin, n.Username, n.Password, n.Email))

	// Setup custom npm registry
	if len(n.Registry) != 0 {
		f.WriteCmd(fmt.Sprintf(CmdSetRegistry, n.Registry))
	}

	// Set npm to always authenticate
	if n.AlwaysAuth {
		f.WriteCmd(CmdAlwaysAuth)
	}

	if len(n.Folder) == 0 {
		n.Folder = "."
	}

	f.WriteString(fmt.Sprintf(CmdPublish, n.Folder, n.Tag, n.Folder))
}

func (n *NPM) GetCondition() *condition.Condition {
	return n.Condition
}
