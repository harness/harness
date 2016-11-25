package transform

import (
	"fmt"

	"github.com/drone/drone/yaml"
)

func Check(c *yaml.Config, trusted bool) error {
	var images []*yaml.Container
	images = append(images, c.Pipeline...)
	images = append(images, c.Services...)

	for _, image := range c.Pipeline {
		if err := CheckEntrypoint(image); err != nil {
			return err
		}
		if trusted {
			continue
		}
		if err := CheckTrusted(image); err != nil {
			return err
		}
	}
	for _, image := range c.Services {
		if trusted {
			continue
		}
		if err := CheckTrusted(image); err != nil {
			return err
		}
	}
	return nil
}

// validate the plugin command and entrypoint and return an error
// the user attempts to set or override these values.
func CheckEntrypoint(c *yaml.Container) error {
	if c.Detached {
		return nil
	}
	if len(c.Entrypoint) != 0 {
		return fmt.Errorf("Cannot set plugin Entrypoint")
	}
	if len(c.Command) != 0 {
		return fmt.Errorf("Cannot set plugin Command")
	}
	return nil
}

// validate the container configuration and return an error if restricted
// configurations are used.
func CheckTrusted(c *yaml.Container) error {
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
	if len(c.Network) != 0 {
		return fmt.Errorf("Insufficient privileges to override the network")
	}
	if c.OomKillDisable {
		return fmt.Errorf("Insufficient privileges to disable oom_kill")
	}
	if len(c.Volumes) != 0 {
		return fmt.Errorf("Insufficient privileges to use volumes")
	}
	if len(c.VolumesFrom) != 0 {
		return fmt.Errorf("Insufficient privileges to use volumes_from")
	}
	return nil
}
