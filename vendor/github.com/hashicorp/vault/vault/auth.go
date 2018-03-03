package vault

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/vault/helper/consts"
	"github.com/hashicorp/vault/helper/jsonutil"
	"github.com/hashicorp/vault/logical"
)

const (
	// coreAuthConfigPath is used to store the auth configuration.
	// Auth configuration is protected within the Vault itself, which means it
	// can only be viewed or modified after an unseal.
	coreAuthConfigPath = "core/auth"

	// coreLocalAuthConfigPath is used to store credential configuration for
	// local (non-replicated) mounts
	coreLocalAuthConfigPath = "core/local-auth"

	// credentialBarrierPrefix is the prefix to the UUID used in the
	// barrier view for the credential backends.
	credentialBarrierPrefix = "auth/"

	// credentialRoutePrefix is the mount prefix used for the router
	credentialRoutePrefix = "auth/"

	// credentialTableType is the value we expect to find for the credential
	// table and corresponding entries
	credentialTableType = "auth"
)

var (
	// errLoadAuthFailed if loadCredentials encounters an error
	errLoadAuthFailed = errors.New("failed to setup auth table")

	// credentialAliases maps old backend names to new backend names, allowing us
	// to move/rename backends but maintain backwards compatibility
	credentialAliases = map[string]string{"aws-ec2": "aws"}
)

// enableCredential is used to enable a new credential backend
func (c *Core) enableCredential(ctx context.Context, entry *MountEntry) error {
	// Ensure we end the path in a slash
	if !strings.HasSuffix(entry.Path, "/") {
		entry.Path += "/"
	}

	// Ensure there is a name
	if entry.Path == "/" {
		return fmt.Errorf("backend path must be specified")
	}

	c.authLock.Lock()
	defer c.authLock.Unlock()

	// Look for matching name
	for _, ent := range c.auth.Entries {
		switch {
		// Existing is oauth/github/ new is oauth/ or
		// existing is oauth/ and new is oauth/github/
		case strings.HasPrefix(ent.Path, entry.Path):
			fallthrough
		case strings.HasPrefix(entry.Path, ent.Path):
			return logical.CodedError(409, "path is already in use")
		}
	}

	// Ensure the token backend is a singleton
	if entry.Type == "token" {
		return fmt.Errorf("token credential backend cannot be instantiated")
	}

	if conflict := c.router.MountConflict(credentialRoutePrefix + entry.Path); conflict != "" {
		return logical.CodedError(409, fmt.Sprintf("existing mount at %s", conflict))
	}

	// Generate a new UUID and view
	if entry.UUID == "" {
		entryUUID, err := uuid.GenerateUUID()
		if err != nil {
			return err
		}
		entry.UUID = entryUUID
	}
	if entry.Accessor == "" {
		accessor, err := c.generateMountAccessor("auth_" + entry.Type)
		if err != nil {
			return err
		}
		entry.Accessor = accessor
	}
	// Sync values to the cache
	entry.SyncCache()

	viewPath := credentialBarrierPrefix + entry.UUID + "/"
	view := NewBarrierView(c.barrier, viewPath)
	// Mark the view as read-only until the mounting is complete and
	// ensure that it is reset after. This ensures that there will be no
	// writes during the construction of the backend.
	view.setReadOnlyErr(logical.ErrSetupReadOnly)
	defer view.setReadOnlyErr(nil)

	var err error
	var backend logical.Backend
	sysView := c.mountEntrySysView(entry)
	conf := make(map[string]string)
	if entry.Config.PluginName != "" {
		conf["plugin_name"] = entry.Config.PluginName
	}

	// Create the new backend
	backend, err = c.newCredentialBackend(ctx, entry.Type, sysView, view, conf)
	if err != nil {
		return err
	}
	if backend == nil {
		return fmt.Errorf("nil backend returned from %q factory", entry.Type)
	}

	// Check for the correct backend type
	backendType := backend.Type()
	if entry.Type == "plugin" && backendType != logical.TypeCredential {
		return fmt.Errorf("cannot mount '%s' of type '%s' as an auth method", entry.Config.PluginName, backendType)
	}

	// Update the auth table
	newTable := c.auth.shallowClone()
	newTable.Entries = append(newTable.Entries, entry)
	if err := c.persistAuth(ctx, newTable, entry.Local); err != nil {
		return errors.New("failed to update auth table")
	}

	c.auth = newTable

	path := credentialRoutePrefix + entry.Path
	if err := c.router.Mount(backend, path, entry, view); err != nil {
		return err
	}

	if c.logger.IsInfo() {
		c.logger.Info("core: enabled credential backend", "path", entry.Path, "type", entry.Type)
	}
	return nil
}

// disableCredential is used to disable an existing credential backend; the
// boolean indicates if it existed
func (c *Core) disableCredential(ctx context.Context, path string) error {
	// Ensure we end the path in a slash
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}

	// Ensure the token backend is not affected
	if path == "token/" {
		return fmt.Errorf("token credential backend cannot be disabled")
	}

	// Store the view for this backend
	fullPath := credentialRoutePrefix + path
	view := c.router.MatchingStorageByAPIPath(fullPath)
	if view == nil {
		return fmt.Errorf("no matching backend %s", fullPath)
	}

	// Get the backend/mount entry for this path, used to remove ignored
	// replication prefixes
	backend := c.router.MatchingBackend(fullPath)
	entry := c.router.MatchingMountEntry(fullPath)

	// Mark the entry as tainted
	if err := c.taintCredEntry(ctx, path); err != nil {
		return err
	}

	// Taint the router path to prevent routing
	if err := c.router.Taint(fullPath); err != nil {
		return err
	}

	if backend != nil {
		// Revoke credentials from this path
		if err := c.expiration.RevokePrefix(fullPath); err != nil {
			return err
		}

		// Call cleanup function if it exists
		backend.Cleanup(ctx)
	}

	// Unmount the backend
	if err := c.router.Unmount(ctx, fullPath); err != nil {
		return err
	}

	switch {
	case entry.Local, !c.ReplicationState().HasState(consts.ReplicationPerformanceSecondary):
		// Have writable storage, remove the whole thing
		if err := logical.ClearView(ctx, view); err != nil {
			c.logger.Error("core: failed to clear view for path being unmounted", "error", err, "path", path)
			return err
		}

	}

	// Remove the mount table entry
	if err := c.removeCredEntry(ctx, path); err != nil {
		return err
	}
	if c.logger.IsInfo() {
		c.logger.Info("core: disabled credential backend", "path", path)
	}
	return nil
}

// removeCredEntry is used to remove an entry in the auth table
func (c *Core) removeCredEntry(ctx context.Context, path string) error {
	c.authLock.Lock()
	defer c.authLock.Unlock()

	// Taint the entry from the auth table
	newTable := c.auth.shallowClone()
	entry := newTable.remove(path)
	if entry == nil {
		c.logger.Error("core: nil entry found removing entry in auth table", "path", path)
		return logical.CodedError(500, "failed to remove entry in auth table")
	}

	// Update the auth table
	if err := c.persistAuth(ctx, newTable, entry.Local); err != nil {
		return errors.New("failed to update auth table")
	}

	c.auth = newTable

	return nil
}

// remountCredEntryForce takes a copy of the mount entry for the path and fully
// unmounts and remounts the backend to pick up any changes, such as filtered
// paths
func (c *Core) remountCredEntryForce(ctx context.Context, path string) error {
	fullPath := credentialRoutePrefix + path
	me := c.router.MatchingMountEntry(fullPath)
	if me == nil {
		return fmt.Errorf("cannot find mount for path '%s'", path)
	}

	me, err := me.Clone()
	if err != nil {
		return err
	}

	if err := c.disableCredential(ctx, path); err != nil {
		return err
	}
	return c.enableCredential(ctx, me)
}

// taintCredEntry is used to mark an entry in the auth table as tainted
func (c *Core) taintCredEntry(ctx context.Context, path string) error {
	c.authLock.Lock()
	defer c.authLock.Unlock()

	// Taint the entry from the auth table
	// We do this on the original since setting the taint operates
	// on the entries which a shallow clone shares anyways
	entry := c.auth.setTaint(path, true)

	// Ensure there was a match
	if entry == nil {
		return fmt.Errorf("no matching backend")
	}

	// Update the auth table
	if err := c.persistAuth(ctx, c.auth, entry.Local); err != nil {
		return errors.New("failed to update auth table")
	}

	return nil
}

// loadCredentials is invoked as part of postUnseal to load the auth table
func (c *Core) loadCredentials(ctx context.Context) error {
	authTable := &MountTable{}
	localAuthTable := &MountTable{}

	// Load the existing mount table
	raw, err := c.barrier.Get(ctx, coreAuthConfigPath)
	if err != nil {
		c.logger.Error("core: failed to read auth table", "error", err)
		return errLoadAuthFailed
	}
	rawLocal, err := c.barrier.Get(ctx, coreLocalAuthConfigPath)
	if err != nil {
		c.logger.Error("core: failed to read local auth table", "error", err)
		return errLoadAuthFailed
	}

	c.authLock.Lock()
	defer c.authLock.Unlock()

	if raw != nil {
		if err := jsonutil.DecodeJSON(raw.Value, authTable); err != nil {
			c.logger.Error("core: failed to decode auth table", "error", err)
			return errLoadAuthFailed
		}
		c.auth = authTable
	}

	var needPersist bool
	if c.auth == nil {
		c.auth = c.defaultAuthTable()
		needPersist = true
	}

	if rawLocal != nil {
		if err := jsonutil.DecodeJSON(rawLocal.Value, localAuthTable); err != nil {
			c.logger.Error("core: failed to decode local auth table", "error", err)
			return errLoadAuthFailed
		}
		if localAuthTable != nil && len(localAuthTable.Entries) > 0 {
			c.auth.Entries = append(c.auth.Entries, localAuthTable.Entries...)
		}
	}

	// Upgrade to typed auth table
	if c.auth.Type == "" {
		c.auth.Type = credentialTableType
		needPersist = true
	}

	// Upgrade to table-scoped entries
	for _, entry := range c.auth.Entries {
		if entry.Table == "" {
			entry.Table = c.auth.Type
			needPersist = true
		}
		if entry.Accessor == "" {
			accessor, err := c.generateMountAccessor("auth_" + entry.Type)
			if err != nil {
				return err
			}
			entry.Accessor = accessor
			needPersist = true
		}

		// Sync values to the cache
		entry.SyncCache()
	}

	if !needPersist {
		return nil
	}

	if err := c.persistAuth(ctx, c.auth, false); err != nil {
		c.logger.Error("core: failed to persist auth table", "error", err)
		return errLoadAuthFailed
	}
	return nil
}

// persistAuth is used to persist the auth table after modification
func (c *Core) persistAuth(ctx context.Context, table *MountTable, localOnly bool) error {
	if table.Type != credentialTableType {
		c.logger.Error("core: given table to persist has wrong type", "actual_type", table.Type, "expected_type", credentialTableType)
		return fmt.Errorf("invalid table type given, not persisting")
	}

	for _, entry := range table.Entries {
		if entry.Table != table.Type {
			c.logger.Error("core: given entry to persist in auth table has wrong table value", "path", entry.Path, "entry_table_type", entry.Table, "actual_type", table.Type)
			return fmt.Errorf("invalid auth entry found, not persisting")
		}
	}

	nonLocalAuth := &MountTable{
		Type: credentialTableType,
	}

	localAuth := &MountTable{
		Type: credentialTableType,
	}

	for _, entry := range table.Entries {
		if entry.Local {
			localAuth.Entries = append(localAuth.Entries, entry)
		} else {
			nonLocalAuth.Entries = append(nonLocalAuth.Entries, entry)
		}
	}

	if !localOnly {
		// Marshal the table
		compressedBytes, err := jsonutil.EncodeJSONAndCompress(nonLocalAuth, nil)
		if err != nil {
			c.logger.Error("core: failed to encode and/or compress auth table", "error", err)
			return err
		}

		// Create an entry
		entry := &Entry{
			Key:   coreAuthConfigPath,
			Value: compressedBytes,
		}

		// Write to the physical backend
		if err := c.barrier.Put(ctx, entry); err != nil {
			c.logger.Error("core: failed to persist auth table", "error", err)
			return err
		}
	}

	// Repeat with local auth
	compressedBytes, err := jsonutil.EncodeJSONAndCompress(localAuth, nil)
	if err != nil {
		c.logger.Error("core: failed to encode and/or compress local auth table", "error", err)
		return err
	}

	entry := &Entry{
		Key:   coreLocalAuthConfigPath,
		Value: compressedBytes,
	}

	if err := c.barrier.Put(ctx, entry); err != nil {
		c.logger.Error("core: failed to persist local auth table", "error", err)
		return err
	}

	return nil
}

// setupCredentials is invoked after we've loaded the auth table to
// initialize the credential backends and setup the router
func (c *Core) setupCredentials(ctx context.Context) error {
	var err error
	var persistNeeded bool
	var backendType logical.BackendType

	c.authLock.Lock()
	defer c.authLock.Unlock()

	for _, entry := range c.auth.Entries {
		var backend logical.Backend
		// Work around some problematic code that existed in master for a while
		if strings.HasPrefix(entry.Path, credentialRoutePrefix) {
			entry.Path = strings.TrimPrefix(entry.Path, credentialRoutePrefix)
			persistNeeded = true
		}

		// Create a barrier view using the UUID
		viewPath := credentialBarrierPrefix + entry.UUID + "/"
		view := NewBarrierView(c.barrier, viewPath)

		// Mark the view as read-only until the mounting is complete and
		// ensure that it is reset after. This ensures that there will be no
		// writes during the construction of the backend.
		view.setReadOnlyErr(logical.ErrSetupReadOnly)
		defer view.setReadOnlyErr(nil)

		// Initialize the backend
		sysView := c.mountEntrySysView(entry)
		conf := make(map[string]string)
		if entry.Config.PluginName != "" {
			conf["plugin_name"] = entry.Config.PluginName
		}

		backend, err = c.newCredentialBackend(ctx, entry.Type, sysView, view, conf)
		if err != nil {
			c.logger.Error("core: failed to create credential entry", "path", entry.Path, "error", err)
			if entry.Type == "plugin" {
				// If we encounter an error instantiating the backend due to an error,
				// skip backend initialization but register the entry to the mount table
				// to preserve storage and path.
				c.logger.Warn("core: skipping plugin-based credential entry", "path", entry.Path)
				goto ROUTER_MOUNT
			}
			return errLoadAuthFailed
		}
		if backend == nil {
			return fmt.Errorf("nil backend returned from %q factory", entry.Type)
		}

		// Check for the correct backend type
		backendType = backend.Type()
		if entry.Type == "plugin" && backendType != logical.TypeCredential {
			return fmt.Errorf("cannot mount '%s' of type '%s' as an auth backend", entry.Config.PluginName, backendType)
		}

	ROUTER_MOUNT:
		// Mount the backend
		path := credentialRoutePrefix + entry.Path
		err = c.router.Mount(backend, path, entry, view)
		if err != nil {
			c.logger.Error("core: failed to mount auth entry", "path", entry.Path, "error", err)
			return errLoadAuthFailed
		}

		// Ensure the path is tainted if set in the mount table
		if entry.Tainted {
			c.router.Taint(path)
		}

		// Check if this is the token store
		if entry.Type == "token" {
			c.tokenStore = backend.(*TokenStore)

			// this is loaded *after* the normal mounts, including cubbyhole
			c.router.tokenStoreSaltFunc = c.tokenStore.Salt
			c.tokenStore.cubbyholeBackend = c.router.MatchingBackend("cubbyhole/").(*CubbyholeBackend)
		}
	}

	if persistNeeded {
		return c.persistAuth(ctx, c.auth, false)
	}

	return nil
}

// teardownCredentials is used before we seal the vault to reset the credential
// backends to their unloaded state. This is reversed by loadCredentials.
func (c *Core) teardownCredentials(ctx context.Context) error {
	c.authLock.Lock()
	defer c.authLock.Unlock()

	if c.auth != nil {
		authTable := c.auth.shallowClone()
		for _, e := range authTable.Entries {
			backend := c.router.MatchingBackend(credentialRoutePrefix + e.Path)
			if backend != nil {
				backend.Cleanup(ctx)
			}
		}
	}

	c.auth = nil
	c.tokenStore = nil
	return nil
}

// newCredentialBackend is used to create and configure a new credential backend by name
func (c *Core) newCredentialBackend(
	ctx context.Context,
	t string,
	sysView logical.SystemView,
	view logical.Storage,
	conf map[string]string) (logical.Backend, error) {
	if alias, ok := credentialAliases[t]; ok {
		t = alias
	}
	f, ok := c.credentialBackends[t]
	if !ok {
		return nil, fmt.Errorf("unknown backend type: %s", t)
	}

	config := &logical.BackendConfig{
		StorageView: view,
		Logger:      c.logger,
		Config:      conf,
		System:      sysView,
	}

	b, err := f(ctx, config)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// defaultAuthTable creates a default auth table
func (c *Core) defaultAuthTable() *MountTable {
	table := &MountTable{
		Type: credentialTableType,
	}
	tokenUUID, err := uuid.GenerateUUID()
	if err != nil {
		panic(fmt.Sprintf("could not generate UUID for default auth table token entry: %v", err))
	}
	tokenAccessor, err := c.generateMountAccessor("auth_token")
	if err != nil {
		panic(fmt.Sprintf("could not generate accessor for default auth table token entry: %v", err))
	}
	tokenAuth := &MountEntry{
		Table:       credentialTableType,
		Path:        "token/",
		Type:        "token",
		Description: "token based credentials",
		UUID:        tokenUUID,
		Accessor:    tokenAccessor,
	}
	table.Entries = append(table.Entries, tokenAuth)
	return table
}
