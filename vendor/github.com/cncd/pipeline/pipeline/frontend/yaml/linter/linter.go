package linter

import (
	"fmt"

	"github.com/cncd/pipeline/pipeline/frontend/yaml"
)

// A Linter lints a pipeline configuration.
type Linter struct {
	trusted bool
}

// New creates a new Linter with options.
func New(opts ...Option) *Linter {
	linter := new(Linter)
	for _, opt := range opts {
		opt(linter)
	}
	return linter
}

// Lint lints the configuration.
func (l *Linter) Lint(c *yaml.Config) error {
	var containers []*yaml.Container
	containers = append(containers, c.Pipeline.Containers...)
	containers = append(containers, c.Services.Containers...)

	for _, container := range containers {
		if err := l.lintImage(container); err != nil {
			return err
		}
		if l.trusted == false {
			if err := l.lintTrusted(container); err != nil {
				return err
			}
		}
		if isService(container) == false {
			if err := l.lintEntrypoint(container); err != nil {
				return err
			}
		}
	}

	if len(c.Pipeline.Containers) == 0 {
		return fmt.Errorf("Invalid or missing pipeline section")
	}
	return nil
}

func (l *Linter) lintImage(c *yaml.Container) error {
	if len(c.Image) == 0 {
		return fmt.Errorf("Invalid or missing image")
	}
	return nil
}

func (l *Linter) lintEntrypoint(c *yaml.Container) error {
	if len(c.Entrypoint) != 0 {
		return fmt.Errorf("Cannot override container entrypoint")
	}
	if len(c.Command) != 0 {
		return fmt.Errorf("Cannot override container command")
	}
	return nil
}

func (l *Linter) lintTrusted(c *yaml.Container) error {
	if c.Privileged {
		return fmt.Errorf("Insufficient privileges to use privileged mode")
	}
	if c.ShmSize != 0 {
		return fmt.Errorf("Insufficient privileges to override shm_size")
	}
	if len(c.DNS) != 0 {
		return fmt.Errorf("Insufficient privileges to use custom dns")
	}
	if len(c.DNSSearch) != 0 {
		return fmt.Errorf("Insufficient privileges to use dns_search")
	}
	if len(c.Devices) != 0 {
		return fmt.Errorf("Insufficient privileges to use devices")
	}
	if len(c.ExtraHosts) != 0 {
		return fmt.Errorf("Insufficient privileges to use extra_hosts")
	}
	if len(c.NetworkMode) != 0 {
		return fmt.Errorf("Insufficient privileges to use network_mode")
	}
	if c.Networks.Networks != nil && len(c.Networks.Networks) != 0 {
		return fmt.Errorf("Insufficient privileges to use networks")
	}
	if c.Volumes.Volumes != nil && len(c.Volumes.Volumes) != 0 {
		return fmt.Errorf("Insufficient privileges to use volumes")
	}
	return nil
}

func isService(c *yaml.Container) bool {
	return !isScript(c) && !isPlugin(c)
}

func isScript(c *yaml.Container) bool {
	return len(c.Commands) != 0
}

func isPlugin(c *yaml.Container) bool {
	return len(c.Vargs) != 0
}
