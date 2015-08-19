package parser

import (
	common "github.com/drone/drone/pkg/types"
	"github.com/drone/drone/pkg/yaml/inject"
	"github.com/drone/drone/pkg/yaml/matrix"
	"github.com/drone/drone/pkg/yaml/transform"

	"github.com/drone/drone/Godeps/_workspace/src/gopkg.in/yaml.v2"
)

// Opts specifies parser options that will permit
// or deny certain Yaml settings.
type Opts struct {
	Volumes    bool
	Network    bool
	Privileged bool
	Whitelist  []string
}

var DefaultOpts = &Opts{
	Volumes:    false,
	Network:    false,
	Privileged: false,
	Whitelist:  []string{"plugins/*"},
}

// Parse parses a build matrix and returns
// a list of build configurations for each axis
// using the default parsing options.
func Parse(raw string, r *common.Repo) ([]*common.Config, error) {
	return ParseOpts(raw, DefaultOpts, r)
}

// ParseOpts parses a build matrix and returns
// a list of build configurations for each axis
// using the provided parsing options.
func ParseOpts(raw string, opts *Opts, r *common.Repo) ([]*common.Config, error) {
	axis, err := matrix.Parse(raw)
	if err != nil {
		return nil, err
	}
	confs := []*common.Config{}

	// when no matrix values exist we should return
	// a single config value with an empty label.
	if len(axis) == 0 {
		conf, err := ParseSingle(raw, opts, r)
		if err != nil {
			return nil, err
		}
		confs = append(confs, conf)
	}

	for _, ax := range axis {
		// inject the matrix values into the raw script
		injected := inject.Inject(raw, ax)
		conf, err := ParseSingle(injected, opts, r)
		if err != nil {
			return nil, err
		}
		conf.Axis = common.Axis(ax)
		confs = append(confs, conf)
	}

	return confs, nil
}

// helper funtion to parse a yaml configuration file.
func ParseSingle(raw string, opts *Opts, r *common.Repo) (*common.Config, error) {
	conf := &common.Config{}
	err := yaml.Unmarshal([]byte(raw), conf)
	if err != nil {
		return nil, err
	}

	// apply rules / transforms
	transform.Defaults(conf)
	if !opts.Network {
		transform.RemoveNetwork(conf)
	}
	if !opts.Volumes {
		transform.RemoveVolumes(conf)
	}
	if !opts.Privileged {
		transform.RemovePrivileged(conf)
	}
	transform.Repo(conf, r)

	// lint the yaml file
	err = Lint(conf)
	if err != nil {
		return nil, err
	}
	err = LintPlugins(conf, opts)
	if err != nil {
		return nil, err
	}
	return conf, err
}

func ParseCondition(raw string) (*common.Condition, error) {
	c := struct {
		Condition *common.Condition `yaml:"when"`
	}{}
	err := yaml.Unmarshal([]byte(raw), c)
	return c.Condition, err
}
