package vault

import (
	"context"
	"fmt"
	"strings"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/vault/logical"
)

// reloadPluginMounts reloads provided mounts, regardless of
// plugin name, as long as the backend type is plugin.
func (c *Core) reloadMatchingPluginMounts(ctx context.Context, mounts []string) error {
	c.mountsLock.Lock()
	defer c.mountsLock.Unlock()

	var errors error
	for _, mount := range mounts {
		entry := c.router.MatchingMountEntry(mount)
		if entry == nil {
			errors = multierror.Append(errors, fmt.Errorf("cannot fetch mount entry on %s", mount))
			continue
			// return fmt.Errorf("cannot fetch mount entry on %s", mount)
		}

		var isAuth bool
		fullPath := c.router.MatchingMount(mount)
		if strings.HasPrefix(fullPath, credentialRoutePrefix) {
			isAuth = true
		}

		if entry.Type == "plugin" {
			err := c.reloadPluginCommon(ctx, entry, isAuth)
			if err != nil {
				errors = multierror.Append(errors, fmt.Errorf("cannot reload plugin on %s: %v", mount, err))
				continue
			}
			c.logger.Info("core: successfully reloaded plugin", "plugin", entry.Config.PluginName, "path", entry.Path)
		}
	}
	return errors
}

// reloadPlugin reloads all mounted backends that are of
// plugin pluginName (name of the plugin as registered in
// the plugin catalog).
func (c *Core) reloadMatchingPlugin(ctx context.Context, pluginName string) error {
	c.mountsLock.Lock()
	defer c.mountsLock.Unlock()

	// Filter mount entries that only matches the plugin name
	for _, entry := range c.mounts.Entries {
		if entry.Config.PluginName == pluginName && entry.Type == "plugin" {
			err := c.reloadPluginCommon(ctx, entry, false)
			if err != nil {
				return err
			}
			c.logger.Info("core: successfully reloaded plugin", "plugin", pluginName, "path", entry.Path)
		}
	}

	// Filter auth mount entries that ony matches the plugin name
	for _, entry := range c.auth.Entries {
		if entry.Config.PluginName == pluginName && entry.Type == "plugin" {
			err := c.reloadPluginCommon(ctx, entry, true)
			if err != nil {
				return err
			}
			c.logger.Info("core: successfully reloaded plugin", "plugin", pluginName, "path", entry.Path)
		}
	}

	return nil
}

// reloadPluginCommon is a generic method to reload a backend provided a
// MountEntry. entry.Type should be checked by the caller to ensure that
// it's a "plugin" type.
func (c *Core) reloadPluginCommon(ctx context.Context, entry *MountEntry, isAuth bool) error {
	path := entry.Path

	if isAuth {
		path = credentialRoutePrefix + path
	}

	// Fast-path out if the backend doesn't exist
	raw, ok := c.router.root.Get(path)
	if !ok {
		return nil
	}

	re := raw.(*routeEntry)

	// Only call Cleanup if backend is initialized
	if re.backend != nil {
		// Call backend's Cleanup routine
		re.backend.Cleanup(ctx)
	}

	view := re.storageView

	sysView := c.mountEntrySysView(entry)
	conf := make(map[string]string)
	if entry.Config.PluginName != "" {
		conf["plugin_name"] = entry.Config.PluginName
	}

	var backend logical.Backend
	var err error
	if !isAuth {
		// Dispense a new backend
		backend, err = c.newLogicalBackend(ctx, entry.Type, sysView, view, conf)
	} else {
		backend, err = c.newCredentialBackend(ctx, entry.Type, sysView, view, conf)
	}
	if err != nil {
		return err
	}
	if backend == nil {
		return fmt.Errorf("nil backend of type %q returned from creation function", entry.Type)
	}

	// Set the backend back
	re.backend = backend

	return nil
}
