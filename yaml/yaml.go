package yaml

import (
	"gopkg.in/yaml.v2"
)

type Config struct {
	Debug    bool     `yaml:"debug"`
	Branches []string `yaml:"branches"`
}

func Parse(raw string) (*Config, error) {
	c := &Config{}
	err := yaml.Unmarshal([]byte(raw), c)
	return c, err
}
