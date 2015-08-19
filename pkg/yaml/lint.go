package parser

import (
	"fmt"
	"path/filepath"
	"strings"

	common "github.com/drone/drone/pkg/types"
)

// lintRule defines a function that runs lint
// checks against a Yaml Config file. If the rule
// fails it should return an error message.
type lintRule func(*common.Config) error

var lintRules = []lintRule{
	expectBuild,
	expectImage,
	expectCommand,
	expectCloneInWorkspace,
	expectCacheInWorkspace,
}

// Lint runs all lint rules against the Yaml Config.
func Lint(c *common.Config) error {
	for _, rule := range lintRules {
		err := rule(c)
		if err != nil {
			return err
		}
	}
	return nil
}

// lint rule that fails when no build is defined
func expectBuild(c *common.Config) error {
	if c.Build == nil {
		return fmt.Errorf("Yaml must define a build section")
	}
	return nil
}

// lint rule that fails when no build image is defined
func expectImage(c *common.Config) error {
	if len(c.Build.Image) == 0 {
		return fmt.Errorf("Yaml must define a build image")
	}
	return nil
}

// lint rule that fails when no build commands are defined
func expectCommand(c *common.Config) error {
	if c.Setup.Config == nil || c.Setup.Config["commands"] == nil {
		return fmt.Errorf("Yaml must define build / setup commands")
	}
	return nil
}

// lint rule that fails if the clone directory is not contained
// in the root workspace.
func expectCloneInWorkspace(c *common.Config) error {
	pathv, ok := c.Clone.Config["path"]
	var path string

	if ok {
		path, _ = pathv.(string)
	}
	if len(path) == 0 {
		// This should only happen if the transformer was not run
		return fmt.Errorf("No workspace specified")
	}

	relative, relOk := filepath.Rel("/drone/src", path)
	if relOk != nil {
		return fmt.Errorf("Path is not relative to root")
	}

	cleaned := filepath.Clean(relative)
	if strings.Index(cleaned, "../") != -1 {
		return fmt.Errorf("Cannot clone above the root")
	}

	return nil
}

// lint rule that fails if the cache directories are not contained
// in the workspace.
func expectCacheInWorkspace(c *common.Config) error {
	for _, step := range c.Build.Cache {
		if strings.Index(step, ":") != -1 {
			return fmt.Errorf("Cache cannot contain : in the path")
		}

		cleaned := filepath.Clean(step)

		if strings.Index(cleaned, "../") != -1 {
			return fmt.Errorf("Cache must point to a path in the workspace")
		} else if cleaned == "." {
			return fmt.Errorf("Cannot cache the workspace")
		}
	}

	return nil
}

func LintPlugins(c *common.Config, opts *Opts) error {
	if len(opts.Whitelist) == 0 {
		return nil
	}

	var images []string
	images = append(images, c.Setup.Image)
	images = append(images, c.Clone.Image)
	for _, step := range c.Publish {
		images = append(images, step.Image)
	}
	for _, step := range c.Deploy {
		images = append(images, step.Image)
	}
	for _, step := range c.Notify {
		images = append(images, step.Image)
	}

	for _, image := range images {
		match := false
		for _, pattern := range opts.Whitelist {
			if pattern == image {
				match = true
				break
			}
			ok, err := filepath.Match(pattern, image)
			if ok && err == nil {
				match = true
				break
			}
		}
		if !match {
			return fmt.Errorf("Cannot use un-trusted image %s", image)
		}
	}
	return nil
}
