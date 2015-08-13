package transform

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	common "github.com/drone/drone/pkg/types"
)

// transformRule applies a check or transformation rule
// to the build configuration.
type transformRule func(*common.Config)

var transformRules = [...]transformRule{
	transformSetup,
	transformClone,
	transformBuild,
	transformImages,
	transformDockerPlugin,
}

var rmPrivilegedRules = [...]transformRule{
	rmPrivileged,
	rmVolumes,
	rmNetwork,
}

// Transform executes the default transformers that
// ensure the minimal Yaml configuration is in place
// and correctly configured.
func Transform(c *common.Config) {
	for _, rule := range transformRules {
		rule(c)
	}
}

// TransformSafe executes all transformers that remove
// privileged options from the Yaml.
func TransformSafe(c *common.Config) {
	for _, rule := range rmPrivilegedRules {
		rule(c)
	}
}

// TransformRemoveNetwork executes all transformers that
// remove network options from the Yaml.
func TransformRemoveNetwork(c *common.Config) {
	rmNetwork(c)
}

// TransformRemoveVolumes executes all transformers that
// remove volume options from the Yaml.
func TransformRemoveVolumes(c *common.Config) {
	rmVolumes(c)
}

// TransformRemovePrivileged executes all transformers that
// remove privileged options from the Yaml.
func TransformRemovePrivileged(c *common.Config) {
	rmPrivileged(c)
}

func TransformRepo(c *common.Config, r *common.Repo) {
  transformWorkspace(c, r)
	transformCache(c, r)
}

// transformSetup is a transformer that adds a default
// setup step if none exists.
func transformSetup(c *common.Config) {
	c.Setup = &common.Step{}
	c.Setup.Image = "plugins/drone-build"
	c.Setup.Config = c.Build.Config
}

// transformClone is a transformer that adds a default
// clone step if none exists.
func transformClone(c *common.Config) {
	if c.Clone == nil {
		c.Clone = &common.Step{}
	}
	if len(c.Clone.Image) == 0 {
		c.Clone.Image = "plugins/drone-git"
		c.Clone.Volumes = nil
		c.Clone.NetworkMode = ""
	}
	if c.Clone.Config == nil {
		c.Clone.Config = map[string]interface{}{}
		c.Clone.Config["depth"] = 50
		c.Clone.Config["recursive"] = true
	}
}

// transformBuild is a transformer that removes the
// build configuration vargs. They should have
// already been transferred to the Setup step.
func transformBuild(c *common.Config) {
	c.Build.Config = nil
	c.Build.Entrypoint = []string{"/bin/bash", "-e"}
	c.Build.Command = []string{"/drone/bin/build.sh"}
}

// transformImages is a transformer that ensures every
// step has an image and uses a fully-qualified
// image name.
func transformImages(c *common.Config) {
	c.Setup.Image = imageName(c.Setup.Image)
	c.Clone.Image = imageName(c.Clone.Image)
	for name, step := range c.Publish {
		step.Image = imageNameDefault(step.Image, name)
	}
	for name, step := range c.Deploy {
		step.Image = imageNameDefault(step.Image, name)
	}
	for name, step := range c.Notify {
		step.Image = imageNameDefault(step.Image, name)
	}
}

// transformDockerPlugin is a transformer that ensures the
// official Docker plugin can run in privileged mode. It
// will disable volumes and network mode for added protection.
func transformDockerPlugin(c *common.Config) {
	for _, step := range c.Publish {
		if step.Image == "plugins/drone-docker" {
			step.Privileged = true
			step.Volumes = nil
			step.NetworkMode = ""
			step.Entrypoint = []string{}
			break
		}
	}
}

// rmPrivileged is a transformer that ensures every
// step is executed in non-privileged mode.
func rmPrivileged(c *common.Config) {
	c.Setup.Privileged = false
	c.Clone.Privileged = false
	c.Build.Privileged = false
	for _, step := range c.Publish {
		if step.Image == "plugins/drone-docker" {
			continue // the official docker plugin is the only exception here
		}
		step.Privileged = false
	}
	for _, step := range c.Deploy {
		step.Privileged = false
	}
	for _, step := range c.Notify {
		step.Privileged = false
	}
	for _, step := range c.Compose {
		step.Privileged = false
	}
}

// rmVolumes is a transformer that ensures every
// step is executed without volumes.
func rmVolumes(c *common.Config) {
	c.Setup.Volumes = nil
	c.Clone.Volumes = nil
	c.Build.Volumes = nil
	for _, step := range c.Publish {
		step.Volumes = nil
	}
	for _, step := range c.Deploy {
		step.Volumes = nil
	}
	for _, step := range c.Notify {
		step.Volumes = nil
	}
	for _, step := range c.Compose {
		step.Volumes = nil
	}
}

// rmNetwork is a transformer that ensures every
// step is executed with default bridge networking.
func rmNetwork(c *common.Config) {
	c.Setup.NetworkMode = ""
	c.Clone.NetworkMode = ""
	c.Build.NetworkMode = ""
	for _, step := range c.Publish {
		step.NetworkMode = ""
	}
	for _, step := range c.Deploy {
		step.NetworkMode = ""
	}
	for _, step := range c.Notify {
		step.NetworkMode = ""
	}
	for _, step := range c.Compose {
		step.NetworkMode = ""
	}
}

// transformWorkspace is a transformer that adds the workspace
// directory to the configuration based on the repository
// information.
func transformWorkspace(c *common.Config, r *common.Repo) {
	//c.Clone.Dir = workspaceRoot(r)
}

// transformCache is a transformer that adds volumes
// to the configuration based on the cache.
func transformCache(c *common.Config, r *common.Repo) {
	cacheCount := len(c.Build.Cache)

	if cacheCount != 0 {
		volumes := make([]string, cacheCount)

		cache := cacheRoot(r)
		workspace := workspaceRoot(r)

		for i, dir := range c.Build.Cache {
			cacheDir := filepath.Join(cache, dir)
			workspaceDir := filepath.Join(workspace, dir)

			volumes[i] = fmt.Sprintf("%s:%s", cacheDir, workspaceDir)
		}

		c.Setup.Volumes = append(c.Setup.Volumes, volumes...)
		c.Clone.Volumes = append(c.Clone.Volumes, volumes...)
		c.Build.Volumes = append(c.Build.Volumes, volumes...)
		for _, step := range c.Publish {
			step.Volumes = append(step.Volumes, volumes...)
		}
		for _, step := range c.Deploy {
			step.Volumes = append(step.Volumes, volumes...)
		}
		for _, step := range c.Notify {
			step.Volumes = append(step.Volumes, volumes...)
		}
		for _, step := range c.Compose {
			step.Volumes = append(step.Volumes, volumes...)
		}
	}
}

// imageName is a helper function that resolves the
// image name. When using official drone plugins it
// is possible to use an alias name. This converts to
// the fully qualified name.
func imageName(name string) string {
	if strings.Contains(name, "/") {
		return name
	}
	name = strings.Replace(name, "_", "-", -1)
	name = "plugins/drone-" + name
	return name
}

// imageNameDefault is a helper function that resolves
// the image name. If the image name is blank the
// default name is used instead.
func imageNameDefault(name, defaultName string) string {
	if len(name) == 0 {
		name = defaultName
	}
	return imageName(name)
}

func workspaceRoot(r *common.Repo) string {
  return filepath.Join("/drone/src", repoPath(r))
}

func cacheRoot(r *common.Repo) string {
	return filepath.Join("/tmp/drone/cache", repoPath(r))
}

func repoPath(r *common.Repo) string {
	parsed, _ := url.Parse(r.Link)
	return filepath.Join(parsed.Host, r.FullName)
}
