package parser

import (
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type Branch struct {
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`
}

// ParseBranch parses the branch section of the Yaml document.
func ParseBranch(in []byte) *Branch {
	return parseBranch(in)
}

// ParseBranchString parses the branch section of the Yaml document.
func ParseBranchString(in string) *Branch {
	return ParseBranch([]byte(in))
}

// Matches returns true if the branch matches the include patterns and
// does not match any of the exclude patterns.
func (b *Branch) Matches(branch string) bool {
	// when no includes or excludes automatically match
	if len(b.Include) == 0 && len(b.Exclude) == 0 {
		return true
	}

	// exclusions are processed first. So we can include everything and
	// then selectively exclude certain sub-patterns.
	for _, pattern := range b.Exclude {
		if pattern == branch {
			return false
		}
		if ok, _ := filepath.Match(pattern, branch); ok {
			return false
		}
	}

	for _, pattern := range b.Include {
		if pattern == branch {
			return true
		}
		if ok, _ := filepath.Match(pattern, branch); ok {
			return true
		}
	}

	return false
}

func parseBranch(in []byte) *Branch {
	out1 := struct {
		Branch struct {
			Include stringOrSlice `yaml:"include"`
			Exclude stringOrSlice `yaml:"exclude"`
		} `yaml:"branches"`
	}{}

	out2 := struct {
		Include stringOrSlice `yaml:"branches"`
	}{}

	yaml.Unmarshal(in, &out1)
	yaml.Unmarshal(in, &out2)

	return &Branch{
		Exclude: out1.Branch.Exclude.Slice(),
		Include: append(
			out1.Branch.Include.Slice(),
			out2.Include.Slice()...,
		),
	}
}
