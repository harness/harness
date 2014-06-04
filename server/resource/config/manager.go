package config

type ConfigManager interface {
	Find() (*Config, error)
	Must() *Config
}

// configManager manages configuration data from a combination
// of command line args and .ini files.
type configManager struct {
	dir  string
	conf *Config
}

// NewManager initiales a new CommitManager intended to
// manage and persist commits.
func NewManager(dir string) ConfigManager {
	conf := Config{}
	//conf.Host = "localhost"
	//conf.Scheme = "http"
	return &configManager{dir, &conf}
}

func (c *configManager) Find() (*Config, error) {
	return nil, nil
}

func (c *configManager) Must() *Config {
	conf, err := c.Find()
	if err != nil {
		panic(err)
	}
	return conf
}
