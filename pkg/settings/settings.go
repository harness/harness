package settings

import "github.com/BurntSushi/toml"

// Service represents the configuration details required
// to connect to the revision control system (ie GitHub, Bitbucket)
type Service struct {
	// Name defines the name of the plugin. Possible values
	// may be github, gitlab, bitbucket, or gogs.
	Name string `toml:"name"`

	// Address defines the address (uri) of the plugin for
	// communication via the net/rpc package.
	Address string `toml:"address"`

	// Base defines the base URL for the service. For example:
	// https://github.com
	// https://bitbucket.org
	// https://gitlab.drone.io
	Base string `toml:"base"`

	// Indicates registration is open. If true any user
	// will be able to setup an account. If false, the
	// system administrator will need to provision accounts.
	Open bool `toml:"open"`

	// Orgs defines a list of organizations the user
	// must belong to in order to register. This will
	// take precedence over the `Open` paramter.
	Orgs []string `toml:"orgs"`

	// PrivateMode should be set to true if the
	// remote system requires authentication for
	// cloning public (open source) repositories.
	PrivateMode bool `toml:"private_mode"`

	// SkipVerify instructs the client to skip SSL verification.
	// This may be used with self-signed certificates, however,
	// is not recommended for security reasons.
	SkipVerify bool `toml:"skip_verify"`

	// OAuth configuration data. If nil or empty, Drone may
	// assume basic authentication via username and password.
	OAuth *OAuth `toml:"oauth"`
}

// OAuth defines how a user should autheticate with the service.
// This supports OAuth2 and OAuth1 protocols.
type OAuth struct {
	Client       string   `toml:"client"`
	Secret       string   `toml:"secret"`
	Authorize    string   `toml:"authorize"`
	AccessToken  string   `toml:"access_token"`
	RequestToken string   `toml:"request_token"`
	Scope        []string `toml:"scope"`
}

// Server represents the web server configuration details
// used to server HTTP requests.
type Server struct {
	Base string `toml:"base"`
	Addr string `toml:"addr"`
	Cert string `toml:"cert"`
	Key  string `toml:"key"`

	Scheme   string `toml:"scheme"`
	Hostname string `toml:"hostname"`
}

// Session represents the session configuration details
// used to generate, validate and expire authentication
// sessions.
type Session struct {
	Secret  string `toml:"secret"`
	Expires int64  `toml:"expires"`
}

// Docker represents the configuration details used
// to connect to the Docker daemon when scheduling
// and executing builds in containers.
type Docker struct {
	Cert  string   `toml:"cert"`
	Key   string   `toml:"key"`
	Nodes []string `toml:"nodes"`
}

// Database represents the configuration details used
// to connect to the embedded Bolt database.
type Database struct {
	Driver     string `toml:"driver"`
	Datasource string `toml:"datasource"`
}

type Agents struct {
	Secret string `toml:"secret"`
}

// Settings defines global settings for the Drone system.
type Settings struct {
	Database *Database `toml:"database"`
	Docker   *Docker   `toml:"docker"`
	Service  *Service  `toml:"service"`
	Server   *Server   `toml:"server"`
	Session  *Session  `toml:"session"`
	Agents   *Agents   `toml:"agents"`

	Plugins map[string]interface{} `toml:"plugins"`
}

// Parse parses the Drone settings file at the specified path
// and unmarshals to a Settings structure.
func Parse(path string) (*Settings, error) {
	s := &Settings{}
	_, err := toml.DecodeFile(path, s)
	return applyDefaults(s), err
}

// ParseString parses the Drone settings string and unmarshals
// to a Settings structure.
func ParseString(data string) (*Settings, error) {
	s := &Settings{}
	_, err := toml.Decode(data, s)
	return applyDefaults(s), err
}

func applyDefaults(s *Settings) *Settings {
	if s.Session == nil {
		s.Session = &Session{}
	}
	// if no session token is provided we can
	// instead use the client secret to sign
	// our sessions and tokens.
	if len(s.Session.Secret) == 0 {
		s.Session.Secret = s.Service.OAuth.Secret
	}
	return s
}
