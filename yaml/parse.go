package parser

import (
	"github.com/drone/drone/pkg/types"

	"github.com/drone/drone/Godeps/_workspace/src/gopkg.in/yaml.v2"
)

func ParseCondition(raw string) (*types.Condition, error) {
	c := struct {
		Condition *types.Condition `yaml:"when"`
	}{}
	err := yaml.Unmarshal([]byte(raw), c)
	return c.Condition, err
}
