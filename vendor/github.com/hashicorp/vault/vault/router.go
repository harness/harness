package vault

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/armon/go-metrics"
	"github.com/armon/go-radix"
	"github.com/hashicorp/vault/helper/salt"
	"github.com/hashicorp/vault/logical"
)

// Router is used to do prefix based routing of a request to a logical backend
type Router struct {
	l                  sync.RWMutex
	root               *radix.Tree
	mountUUIDCache     *radix.Tree
	mountAccessorCache *radix.Tree
	tokenStoreSaltFunc func() (*salt.Salt, error)
	// storagePrefix maps the prefix used for storage (ala the BarrierView)
	// to the backend. This is used to map a key back into the backend that owns it.
	// For example, logical/uuid1/foobar -> secrets/ (kv backend) + foobar
	storagePrefix *radix.Tree
}

// NewRouter returns a new router
func NewRouter() *Router {
	r := &Router{
		root:               radix.New(),
		storagePrefix:      radix.New(),
		mountUUIDCache:     radix.New(),
		mountAccessorCache: radix.New(),
	}
	return r
}

// routeEntry is used to represent a mount point in the router
type routeEntry struct {
	tainted       bool
	backend       logical.Backend
	mountEntry    *MountEntry
	storageView   logical.Storage
	storagePrefix string
	rootPaths     *radix.Tree
	loginPaths    *radix.Tree
}

type validateMountResponse struct {
	MountType     string `json:"mount_type" structs:"mount_type" mapstructure:"mount_type"`
	MountAccessor string `json:"mount_accessor" structs:"mount_accessor" mapstructure:"mount_accessor"`
	MountPath     string `json:"mount_path" structs:"mount_path" mapstructure:"mount_path"`
}

// validateMountByAccessor returns the mount type and ID for a given mount
// accessor
func (r *Router) validateMountByAccessor(accessor string) *validateMountResponse {
	if accessor == "" {
		return nil
	}

	mountEntry := r.MatchingMountByAccessor(accessor)
	if mountEntry == nil {
		return nil
	}

	mountPath := mountEntry.Path
	if mountEntry.Table == credentialTableType {
		mountPath = credentialRoutePrefix + mountPath
	}

	return &validateMountResponse{
		MountAccessor: mountEntry.Accessor,
		MountType:     mountEntry.Type,
		MountPath:     mountPath,
	}
}

// SaltID is used to apply a salt and hash to an ID to make sure its not reversible
func (re *routeEntry) SaltID(id string) string {
	return salt.SaltID(re.mountEntry.UUID, id, salt.SHA1Hash)
}

// Mount is used to expose a logical backend at a given prefix, using a unique salt,
// and the barrier view for that path.
func (r *Router) Mount(backend logical.Backend, prefix string, mountEntry *MountEntry, storageView *BarrierView) error {
	r.l.Lock()
	defer r.l.Unlock()

	// Check if this is a nested mount
	if existing, _, ok := r.root.LongestPrefix(prefix); ok && existing != "" {
		return fmt.Errorf("cannot mount under existing mount '%s'", existing)
	}

	// Build the paths
	var localView logical.Storage = storageView
	paths := new(logical.Paths)
	if backend != nil {
		specialPaths := backend.SpecialPaths()
		if specialPaths != nil {
			paths = specialPaths
		}
	}

	// Create a mount entry
	re := &routeEntry{
		tainted:       false,
		backend:       backend,
		mountEntry:    mountEntry,
		storagePrefix: storageView.prefix,
		storageView:   localView,
		rootPaths:     pathsToRadix(paths.Root),
		loginPaths:    pathsToRadix(paths.Unauthenticated),
	}

	switch {
	case prefix == "":
		return fmt.Errorf("missing prefix to be used for router entry; mount_path: %q, mount_type: %q", re.mountEntry.Path, re.mountEntry.Type)
	case re.storagePrefix == "":
		return fmt.Errorf("missing storage view prefix; mount_path: %q, mount_type: %q", re.mountEntry.Path, re.mountEntry.Type)
	case re.mountEntry.UUID == "":
		return fmt.Errorf("missing mount identifier; mount_path: %q, mount_type: %q", re.mountEntry.Path, re.mountEntry.Type)
	case re.mountEntry.Accessor == "":
		return fmt.Errorf("missing mount accessor; mount_path: %q, mount_type: %q", re.mountEntry.Path, re.mountEntry.Type)
	}

	r.root.Insert(prefix, re)
	r.storagePrefix.Insert(re.storagePrefix, re)
	r.mountUUIDCache.Insert(re.mountEntry.UUID, re.mountEntry)
	r.mountAccessorCache.Insert(re.mountEntry.Accessor, re.mountEntry)

	return nil
}

// Unmount is used to remove a logical backend from a given prefix
func (r *Router) Unmount(ctx context.Context, prefix string) error {
	r.l.Lock()
	defer r.l.Unlock()

	// Fast-path out if the backend doesn't exist
	raw, ok := r.root.Get(prefix)
	if !ok {
		return nil
	}

	// Call backend's Cleanup routine
	re := raw.(*routeEntry)
	if re.backend != nil {
		re.backend.Cleanup(ctx)
	}

	// Purge from the radix trees
	r.root.Delete(prefix)
	r.storagePrefix.Delete(re.storagePrefix)
	r.mountUUIDCache.Delete(re.mountEntry.UUID)
	r.mountAccessorCache.Delete(re.mountEntry.Accessor)

	return nil
}

// Remount is used to change the mount location of a logical backend
func (r *Router) Remount(src, dst string) error {
	r.l.Lock()
	defer r.l.Unlock()

	// Check for existing mount
	raw, ok := r.root.Get(src)
	if !ok {
		return fmt.Errorf("no mount at '%s'", src)
	}

	// Update the mount point
	r.root.Delete(src)
	r.root.Insert(dst, raw)
	return nil
}

// Taint is used to mark a path as tainted. This means only RollbackOperation
// RevokeOperation requests are allowed to proceed
func (r *Router) Taint(path string) error {
	r.l.Lock()
	defer r.l.Unlock()
	_, raw, ok := r.root.LongestPrefix(path)
	if ok {
		raw.(*routeEntry).tainted = true
	}
	return nil
}

// Untaint is used to unmark a path as tainted.
func (r *Router) Untaint(path string) error {
	r.l.Lock()
	defer r.l.Unlock()
	_, raw, ok := r.root.LongestPrefix(path)
	if ok {
		raw.(*routeEntry).tainted = false
	}
	return nil
}

func (r *Router) MatchingMountByUUID(mountID string) *MountEntry {
	if mountID == "" {
		return nil
	}

	r.l.RLock()
	defer r.l.RUnlock()

	_, raw, ok := r.mountUUIDCache.LongestPrefix(mountID)
	if !ok {
		return nil
	}

	return raw.(*MountEntry)
}

// MatchingMountByAccessor returns the MountEntry by accessor lookup
func (r *Router) MatchingMountByAccessor(mountAccessor string) *MountEntry {
	if mountAccessor == "" {
		return nil
	}

	r.l.RLock()
	defer r.l.RUnlock()

	_, raw, ok := r.mountAccessorCache.LongestPrefix(mountAccessor)
	if !ok {
		return nil
	}

	return raw.(*MountEntry)
}

// MatchingMount returns the mount prefix that would be used for a path
func (r *Router) MatchingMount(path string) string {
	r.l.RLock()
	defer r.l.RUnlock()
	var mount = r.matchingMountInternal(path)
	return mount
}

func (r *Router) matchingMountInternal(path string) string {
	mount, _, ok := r.root.LongestPrefix(path)
	if !ok {
		return ""
	}
	return mount
}

// matchingPrefixInternal returns a mount prefix that a path may be a part of
func (r *Router) matchingPrefixInternal(path string) string {
	var existing string = ""
	fn := func(existing_path string, _v interface{}) bool {
		if strings.HasPrefix(existing_path, path) {
			existing = existing_path
			return true
		}
		return false
	}
	r.root.WalkPrefix(path, fn)
	return existing
}

// MountConflict determines if there are potential path conflicts
func (r *Router) MountConflict(path string) string {
	r.l.RLock()
	defer r.l.RUnlock()
	if exact_match := r.matchingMountInternal(path); exact_match != "" {
		return exact_match
	}
	if prefix_match := r.matchingPrefixInternal(path); prefix_match != "" {
		return prefix_match
	}
	return ""
}

// MatchingStorageByAPIPath/StoragePath returns the storage used for
// API/Storage paths respectively
func (r *Router) MatchingStorageByAPIPath(path string) logical.Storage {
	return r.matchingStorage(path, true)
}
func (r *Router) MatchingStorageByStoragePath(path string) logical.Storage {
	return r.matchingStorage(path, false)
}
func (r *Router) matchingStorage(path string, apiPath bool) logical.Storage {
	var raw interface{}
	var ok bool
	r.l.RLock()
	if apiPath {
		_, raw, ok = r.root.LongestPrefix(path)
	} else {
		_, raw, ok = r.storagePrefix.LongestPrefix(path)
	}
	r.l.RUnlock()
	if !ok {
		return nil
	}
	return raw.(*routeEntry).storageView
}

// MatchingMountEntry returns the MountEntry used for a path
func (r *Router) MatchingMountEntry(path string) *MountEntry {
	r.l.RLock()
	_, raw, ok := r.root.LongestPrefix(path)
	r.l.RUnlock()
	if !ok {
		return nil
	}
	return raw.(*routeEntry).mountEntry
}

// MatchingBackend returns the backend used for a path
func (r *Router) MatchingBackend(path string) logical.Backend {
	r.l.RLock()
	_, raw, ok := r.root.LongestPrefix(path)
	r.l.RUnlock()
	if !ok {
		return nil
	}
	return raw.(*routeEntry).backend
}

// MatchingSystemView returns the SystemView used for a path
func (r *Router) MatchingSystemView(path string) logical.SystemView {
	r.l.RLock()
	_, raw, ok := r.root.LongestPrefix(path)
	r.l.RUnlock()
	if !ok {
		return nil
	}
	return raw.(*routeEntry).backend.System()
}

// MatchingStoragePrefixByAPIPath/StoragePath returns the mount path matching
// and storage prefix matching the given API/Storage path respectively
func (r *Router) MatchingStoragePrefixByAPIPath(path string) (string, string, bool) {
	return r.matchingStoragePrefix(path, true)
}
func (r *Router) MatchingStoragePrefixByStoragePath(path string) (string, string, bool) {
	return r.matchingStoragePrefix(path, false)
}
func (r *Router) matchingStoragePrefix(path string, apiPath bool) (string, string, bool) {
	var raw interface{}
	var ok bool
	r.l.RLock()
	if apiPath {
		_, raw, ok = r.root.LongestPrefix(path)
	} else {
		_, raw, ok = r.storagePrefix.LongestPrefix(path)
	}
	r.l.RUnlock()
	if !ok {
		return "", "", false
	}

	// Extract the mount path and storage prefix
	re := raw.(*routeEntry)
	mountPath := re.mountEntry.Path
	prefix := re.storagePrefix

	// Add back the prefix for credential backends
	if !apiPath && strings.HasPrefix(path, credentialBarrierPrefix) {
		mountPath = credentialRoutePrefix + mountPath
	}

	return mountPath, prefix, true
}

// Route is used to route a given request
func (r *Router) Route(ctx context.Context, req *logical.Request) (*logical.Response, error) {
	resp, _, _, err := r.routeCommon(ctx, req, false)
	return resp, err
}

// Route is used to route a given existence check request
func (r *Router) RouteExistenceCheck(ctx context.Context, req *logical.Request) (bool, bool, error) {
	_, ok, exists, err := r.routeCommon(ctx, req, true)
	return ok, exists, err
}

func (r *Router) routeCommon(ctx context.Context, req *logical.Request, existenceCheck bool) (*logical.Response, bool, bool, error) {
	// Find the mount point
	r.l.RLock()
	adjustedPath := req.Path
	mount, raw, ok := r.root.LongestPrefix(adjustedPath)
	if !ok && !strings.HasSuffix(adjustedPath, "/") {
		// Re-check for a backend by appending a slash. This lets "foo" mean
		// "foo/" at the root level which is almost always what we want.
		adjustedPath += "/"
		mount, raw, ok = r.root.LongestPrefix(adjustedPath)
	}
	r.l.RUnlock()
	if !ok {
		return logical.ErrorResponse(fmt.Sprintf("no handler for route '%s'", req.Path)), false, false, logical.ErrUnsupportedPath
	}
	req.Path = adjustedPath
	defer metrics.MeasureSince([]string{"route", string(req.Operation),
		strings.Replace(mount, "/", "-", -1)}, time.Now())
	re := raw.(*routeEntry)

	// Filtered mounts will have a nil backend
	if re.backend == nil {
		return logical.ErrorResponse(fmt.Sprintf("no handler for route '%s'", req.Path)), false, false, logical.ErrUnsupportedPath
	}

	// If the path is tainted, we reject any operation except for
	// Rollback and Revoke
	if re.tainted {
		switch req.Operation {
		case logical.RevokeOperation, logical.RollbackOperation:
		default:
			return logical.ErrorResponse(fmt.Sprintf("no handler for route '%s'", req.Path)), false, false, logical.ErrUnsupportedPath
		}
	}

	// Adjust the path to exclude the routing prefix
	originalPath := req.Path
	req.Path = strings.TrimPrefix(req.Path, mount)
	req.MountPoint = mount
	req.MountType = re.mountEntry.Type
	if req.Path == "/" {
		req.Path = ""
	}

	// Attach the storage view for the request
	req.Storage = re.storageView

	originalEntityID := req.EntityID

	// Allow EntityID to passthrough to the system backend. This is required to
	// allow clients to generate MFA credentials in respective entity objects
	// in identity store via the system backend.
	switch {
	case strings.HasPrefix(originalPath, "sys/"):
	default:
		req.EntityID = ""
	}

	// Hash the request token unless the request is being routed to the token
	// or system backend.
	clientToken := req.ClientToken
	switch {
	case strings.HasPrefix(originalPath, "auth/token/"):
	case strings.HasPrefix(originalPath, "sys/"):
	case strings.HasPrefix(originalPath, "cubbyhole/"):
		// In order for the token store to revoke later, we need to have the same
		// salted ID, so we double-salt what's going to the cubbyhole backend
		salt, err := r.tokenStoreSaltFunc()
		if err != nil {
			return nil, false, false, err
		}
		req.ClientToken = re.SaltID(salt.SaltID(req.ClientToken))
	default:
		req.ClientToken = re.SaltID(req.ClientToken)
	}

	// Cache the pointer to the original connection object
	originalConn := req.Connection

	// Cache the identifier of the request
	originalReqID := req.ID

	// Cache the client token's number of uses in the request
	originalClientTokenRemainingUses := req.ClientTokenRemainingUses
	req.ClientTokenRemainingUses = 0

	// Cache the headers and hide them from backends
	headers := req.Headers
	req.Headers = nil

	// Cache the wrap info of the request
	var wrapInfo *logical.RequestWrapInfo
	if req.WrapInfo != nil {
		wrapInfo = &logical.RequestWrapInfo{
			TTL:      req.WrapInfo.TTL,
			Format:   req.WrapInfo.Format,
			SealWrap: req.WrapInfo.SealWrap,
		}
	}

	// Reset the request before returning
	defer func() {
		req.Path = originalPath
		req.MountPoint = mount
		req.MountType = re.mountEntry.Type
		req.Connection = originalConn
		req.ID = originalReqID
		req.Storage = nil
		req.ClientToken = clientToken
		req.ClientTokenRemainingUses = originalClientTokenRemainingUses
		req.WrapInfo = wrapInfo
		req.Headers = headers
		// This is only set in one place, after routing, so should never be set
		// by a backend
		req.SetLastRemoteWAL(0)

		// This will be used for attaching the mount accessor for the identities
		// returned by the authentication backends
		req.MountAccessor = re.mountEntry.Accessor

		req.EntityID = originalEntityID
	}()

	// Invoke the backend
	if existenceCheck {
		ok, exists, err := re.backend.HandleExistenceCheck(ctx, req)
		return nil, ok, exists, err
	} else {
		resp, err := re.backend.HandleRequest(ctx, req)
		// When a token gets renewed, the request hits this path and reaches
		// token store. Token store delegates the renewal to the expiration
		// manager. Expiration manager in-turn creates a different logical
		// request and forwards the request to the auth backend that had
		// initially authenticated the login request. The forwarding to auth
		// backend will make this code path hit for the second time for the
		// same renewal request. The accessors in the Alias structs should be
		// of the auth backend and not of the token store. Therefore, avoiding
		// the overwriting of accessors by having a check for path prefix
		// having "renew". This gets applied for "renew" and "renew-self"
		// requests.
		if resp != nil &&
			resp.Auth != nil &&
			!strings.HasPrefix(req.Path, "renew") {
			if resp.Auth.Alias != nil {
				resp.Auth.Alias.MountAccessor = re.mountEntry.Accessor
			}
			for _, alias := range resp.Auth.GroupAliases {
				alias.MountAccessor = re.mountEntry.Accessor
			}
		}
		return resp, false, false, err
	}
}

// RootPath checks if the given path requires root privileges
func (r *Router) RootPath(path string) bool {
	r.l.RLock()
	mount, raw, ok := r.root.LongestPrefix(path)
	r.l.RUnlock()
	if !ok {
		return false
	}
	re := raw.(*routeEntry)

	// Trim to get remaining path
	remain := strings.TrimPrefix(path, mount)

	// Check the rootPaths of this backend
	match, raw, ok := re.rootPaths.LongestPrefix(remain)
	if !ok {
		return false
	}
	prefixMatch := raw.(bool)

	// Handle the prefix match case
	if prefixMatch {
		return strings.HasPrefix(remain, match)
	}

	// Handle the exact match case
	return match == remain
}

// LoginPath checks if the given path is used for logins
func (r *Router) LoginPath(path string) bool {
	r.l.RLock()
	mount, raw, ok := r.root.LongestPrefix(path)
	r.l.RUnlock()
	if !ok {
		return false
	}
	re := raw.(*routeEntry)

	// Trim to get remaining path
	remain := strings.TrimPrefix(path, mount)

	// Check the loginPaths of this backend
	match, raw, ok := re.loginPaths.LongestPrefix(remain)
	if !ok {
		return false
	}
	prefixMatch := raw.(bool)

	// Handle the prefix match case
	if prefixMatch {
		return strings.HasPrefix(remain, match)
	}

	// Handle the exact match case
	return match == remain
}

// pathsToRadix converts a the mapping of special paths to a mapping
// of special paths to radix trees.
func pathsToRadix(paths []string) *radix.Tree {
	tree := radix.New()
	for _, path := range paths {
		// Check if this is a prefix or exact match
		prefixMatch := len(path) >= 1 && path[len(path)-1] == '*'
		if prefixMatch {
			path = path[:len(path)-1]
		}

		tree.Insert(path, prefixMatch)
	}

	return tree
}
