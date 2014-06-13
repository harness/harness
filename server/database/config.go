package database

import (
	"github.com/BurntSushi/toml"
	"github.com/drone/drone/shared/model"
)

type ConfigManager interface {
	Find() *model.Config
}

// configManager manages configuration data from a
// configuration file using .toml format
type configManager struct {
	conf *model.Config
}

// NewConfigManager initiales a new CommitManager intended to
// manage and persist commits.
func NewConfigManager(filename string) ConfigManager {
	c := configManager{}
	c.conf = &model.Config{}

	// load the configuration file and parse
	_, err := toml.DecodeFile(filename, c.conf)
	if err != nil {
		panic(err)
	}

	return &c
}

func (c *configManager) Find() *model.Config {
	return c.conf
}
