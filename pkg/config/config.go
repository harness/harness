package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	log "github.com/drone/drone/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/naoina/toml"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/vrischmann/envconfig"
)

type Config struct {
	Remote struct {
		Driver string `envconfig:"optional"`
	}

	Auth struct {
		Client       string   `envconfig:"optional"`
		Secret       string   `envconfig:"optional"`
		Authorize    string   `envconfig:"optional"`
		AccessToken  string   `envconfig:"optional"`
		RequestToken string   `envconfig:"optional"`
		Scope        []string `envconfig:"optional"`
	}

	Server struct {
		Base     string `envconfig:"optional"`
		Addr     string `envconfig:"optional"`
		Cert     string `envconfig:"optional"`
		Key      string `envconfig:"optional"`
		Scheme   string `envconfig:"optional"`
		Hostname string `envconfig:"optional"`
	}

	Session struct {
		Secret  string `envconfig:"optional"`
		Expires int64  `envconfig:"optional"`
	}

	Agents struct {
		Secret string `envconfig:"optional"`
	}

	Database struct {
		Driver     string `envconfig:"optional"`
		Datasource string `envconfig:"optional"`
	}

	Docker struct {
		Cert  string `envconfig:"optional"`
		Key   string `envconfig:"optional"`
		Addr  string `envconfig:"optional"`
		Swarm bool   `envconfig:"optional"`
	}

	// Environment represents a set of global environment
	// variable declarations that can be injected into
	// build plugins. An example use case might be SMTP
	// configuration.
	Environment []string `envconfig:"optional"`

	// Plugins represents a white-list of plugins
	// that the system is authorized to load.
	Plugins []string `envconfig:"optional"`

	Github struct {
		API         string   `envconfig:"optional"`
		Host        string   `envconfig:"optional"`
		Client      string   `envconfig:"optional"`
		Secret      string   `envconfig:"optional"`
		PrivateMode bool     `envconfig:"optional"`
		SkipVerify  bool     `envconfig:"optional"`
		Open        bool     `envconfig:"optional"`
		Orgs        []string `envconfig:"optional"`
	}

	Bitbucket struct {
		Client string   `envconfig:"optional"`
		Secret string   `envconfig:"optional"`
		Open   bool     `envconfig:"optional"`
		Orgs   []string `envconfig:"optional"`
	}

	Gitlab struct {
		Host       string   `envconfig:"optional"`
		Client     string   `envconfig:"optional"`
		Secret     string   `envconfig:"optional"`
		SkipVerify bool     `envconfig:"optional"`
		Open       bool     `envconfig:"optional"`
		Orgs       []string `envconfig:"optional"`
		Search     bool     `envconfig:"optional"`
	}
}

// Load loads the configuration file and reads
// parameters from environment variables.
func Load(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadBytes(data)
}

// LoadBytes reads the configuration file and
// reads parameters from environment variables.
func LoadBytes(data []byte) (*Config, error) {
	conf := &Config{}
	err := toml.Unmarshal(data, conf)
	if err != nil {
		return nil, err
	}
	err = envconfig.InitWithPrefix(conf, "DRONE")
	if err != nil {
		return nil, err
	}
	return applyDefaults(conf), nil
}

func applyDefaults(c *Config) *Config {
	// if no session token is provided we can
	// instead use the client secret to sign
	// our sessions and tokens.
	if len(c.Session.Secret) == 0 {
		c.Session.Secret = c.Auth.Secret
	}

	// Prevent crash on start, use sqlite3
	// driver as default if DRONE_DATABASE_DRIVER and
	// DRONE_DATABASE_DATASOURCE not specifed
	if len(c.Database.Driver) == 0 && len(c.Database.Datasource) == 0 {
		c.Database.Driver = "sqlite3"

		pwd, err := os.Getwd()
		if err != nil {
			panic(err)
		}

		c.Database.Datasource = path.Join(pwd, "drone.sqlite3")
		log.Warnf("Use default database settings, driver: %q, config: %q", c.Database.Driver, c.Database.Datasource)
	}

	// Set default settings for remotes
	switch strings.ToLower(c.Remote.Driver) {
	case "github":
		if len(c.Github.API) == 0 && len(c.Github.Host) == 0 {
			c.Github.API = "https://api.github.com/"
			c.Github.Host = "https://github.com"
			log.Warnf("Use default github settings, host: %q, api: %q", c.Github.Host, c.Github.API)
		} else if len(c.Github.API) == 0 && len(c.Github.Host) != 0 {
			c.Github.API = fmt.Sprintf("%s/api/v3/", c.Github.Host)
			log.Warnf("Github API not specified, use: %q", c.Github.API)
		}
	}

	return c
}
