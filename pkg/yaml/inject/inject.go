package inject

import (
	"sort"
	"strings"

	"github.com/drone/drone/pkg/types"
	"gopkg.in/yaml.v2"
)

// Inject injects a map of parameters into a raw string and returns
// the resulting string.
//
// Parameters are represented in the string using $$ notation, similar
// to how environment variables are defined in Makefiles.
func Inject(raw string, params map[string]string) string {
	if params == nil {
		return raw
	}
	keys := []string{}
	for k := range params {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	injected := raw
	for _, k := range keys {
		v := params[k]
		injected = strings.Replace(injected, "$$"+k, v, -1)
	}
	return injected
}

// InjectSafe attempts to safely inject parameters without leaking
// parameters in the Build or Compose section of the yaml file.
//
// The intended use case for this function are public pull requests.
// We want to avoid a malicious pull request that allows someone
// to inject and print private variables.
func InjectSafe(raw string, params map[string]string) string {
	before, _ := parse(raw)
	after, _ := parse(Inject(raw, params))
	before.Notify = after.Notify
	before.Publish = after.Publish
	before.Deploy = after.Deploy
	result, _ := yaml.Marshal(before)
	return string(result)
}

// helper funtion to parse a yaml configuration file.
func parse(raw string) (*types.Config, error) {
	cfg := types.Config{}
	err := yaml.Unmarshal([]byte(raw), &cfg)
	return &cfg, err
}
