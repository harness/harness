package linter

import (
	"fmt"

	"github.com/cncd/pipeline/pipeline/frontend/yaml"
)

const (
	blockClone uint8 = iota
	blockPipeline
	blockServices
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
	if len(c.Pipeline.Containers) == 0 {
		return fmt.Errorf("Invalid or missing pipeline section")
	}
	if err := l.lint(c.Clone.Containers, blockClone); err != nil {
		return err
	}
	if err := l.lint(c.Pipeline.Containers, blockPipeline); err != nil {
		return err
	}
	if err := l.lint(c.Services.Containers, blockServices); err != nil {
		return err
	}
	return nil
}

func (l *Linter) lint(containers []*yaml.Container, block uint8) error {
	for _, container := range containers {
		if err := l.lintImage(container); err != nil {
			return err
		}
		if l.trusted == false {
			if err := l.lintTrusted(container); err != nil {
				return err
			}
		}
		if block != blockServices && !container.Detached {
			if err := l.lintEntrypoint(container); err != nil {
				return err
			}
		}
		if err := l.lintCommands(container); err != nil {
			return err
		}
	}
	return nil
}

func (l *Linter) lintImage(c *yaml.Container) error {
	if len(c.Image) == 0 {
		return fmt.Errorf("Invalid or missing image")
	}
	return nil
}

func (l *Linter) lintCommands(c *yaml.Container) error {
	if len(c.Commands) == 0 {
		return nil
	}
	if len(c.Vargs) != 0 {
		var keys []string
		for key := range c.Vargs {
			keys = append(keys, key)
		}
		return fmt.Errorf("Cannot configure both commands and custom attributes %v", keys)
	}
	if len(c.Entrypoint) != 0 {
		return fmt.Errorf("Cannot configure both commands and entrypoint attributes")
	}
	if len(c.Command) != 0 {
		return fmt.Errorf("Cannot configure both commands and command attributes")
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
