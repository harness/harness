package vault

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/armon/go-metrics"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/golang-lru"
	"github.com/hashicorp/vault/helper/consts"
	"github.com/hashicorp/vault/helper/strutil"
	"github.com/hashicorp/vault/logical"
	log "github.com/mgutz/logxi/v1"
)

const (
	// policySubPath is the sub-path used for the policy store
	// view. This is nested under the system view.
	policyACLSubPath = "policy/"

	// policyCacheSize is the number of policies that are kept cached
	policyCacheSize = 1024

	// defaultPolicyName is the name of the default policy
	defaultPolicyName = "default"

	// responseWrappingPolicyName is the name of the fixed policy
	responseWrappingPolicyName = "response-wrapping"

	// controlGroupPolicyName is the name of the fixed policy for control group
	// tokens
	controlGroupPolicyName = "control-group"

	// responseWrappingPolicy is the policy that ensures cubbyhole response
	// wrapping can always succeed.
	responseWrappingPolicy = `
path "cubbyhole/response" {
    capabilities = ["create", "read"]
}

path "sys/wrapping/unwrap" {
    capabilities = ["update"]
}
`

	// defaultPolicy is the "default" policy
	defaultPolicy = `
# Allow tokens to look up their own properties
path "auth/token/lookup-self" {
    capabilities = ["read"]
}

# Allow tokens to renew themselves
path "auth/token/renew-self" {
    capabilities = ["update"]
}

# Allow tokens to revoke themselves
path "auth/token/revoke-self" {
    capabilities = ["update"]
}

# Allow a token to look up its own capabilities on a path
path "sys/capabilities-self" {
    capabilities = ["update"]
}

# Allow a token to renew a lease via lease_id in the request body; old path for
# old clients, new path for newer
path "sys/renew" {
    capabilities = ["update"]
}
path "sys/leases/renew" {
    capabilities = ["update"]
}

# Allow looking up lease properties. This requires knowing the lease ID ahead
# of time and does not divulge any sensitive information.
path "sys/leases/lookup" {
    capabilities = ["update"]
}

# Allow a token to manage its own cubbyhole
path "cubbyhole/*" {
    capabilities = ["create", "read", "update", "delete", "list"]
}

# Allow a token to wrap arbitrary values in a response-wrapping token
path "sys/wrapping/wrap" {
    capabilities = ["update"]
}

# Allow a token to look up the creation time and TTL of a given
# response-wrapping token
path "sys/wrapping/lookup" {
    capabilities = ["update"]
}

# Allow a token to unwrap a response-wrapping token. This is a convenience to
# avoid client token swapping since this is also part of the response wrapping
# policy.
path "sys/wrapping/unwrap" {
    capabilities = ["update"]
}

# Allow general purpose tools
path "sys/tools/hash" {
	capabilities = ["update"]
}
path "sys/tools/hash/*" {
	capabilities = ["update"]
}
path "sys/tools/random" {
	capabilities = ["update"]
}
path "sys/tools/random/*" {
	capabilities = ["update"]
}
`
)

var (
	immutablePolicies = []string{
		"root",
		responseWrappingPolicyName,
		controlGroupPolicyName,
	}
	nonAssignablePolicies = []string{
		responseWrappingPolicyName,
		controlGroupPolicyName,
	}
)

// PolicyStore is used to provide durable storage of policy, and to
// manage ACLs associated with them.
type PolicyStore struct {
	core             *Core
	aclView          *BarrierView
	tokenPoliciesLRU *lru.TwoQueueCache
	// This is used to ensure that writes to the store (acl/rgp) or to the egp
	// path tree don't happen concurrently. We are okay reading stale data so
	// long as there aren't concurrent writes.
	modifyLock *sync.RWMutex
	// Stores whether a token policy is ACL or RGP
	policyTypeMap sync.Map
	// logger is the server logger copied over from core
	logger log.Logger
}

// PolicyEntry is used to store a policy by name
type PolicyEntry struct {
	Version int
	Raw     string
	Type    PolicyType
}

// NewPolicyStore creates a new PolicyStore that is backed
// using a given view. It used used to durable store and manage named policy.
func NewPolicyStore(ctx context.Context, core *Core, baseView *BarrierView, system logical.SystemView, logger log.Logger) *PolicyStore {
	ps := &PolicyStore{
		aclView:    baseView.SubView(policyACLSubPath),
		modifyLock: new(sync.RWMutex),
		logger:     logger,
		core:       core,
	}
	if !system.CachingDisabled() {
		cache, _ := lru.New2Q(policyCacheSize)
		ps.tokenPoliciesLRU = cache
	}

	keys, err := logical.CollectKeys(ctx, ps.aclView)
	if err != nil {
		ps.logger.Error("policy: error collecting acl policy keys", "error", err)
		return nil
	}
	for _, key := range keys {
		ps.policyTypeMap.Store(ps.sanitizeName(key), PolicyTypeACL)
	}
	// Special-case root; doesn't exist on disk but does need to be found
	ps.policyTypeMap.Store("root", PolicyTypeACL)
	return ps
}

// setupPolicyStore is used to initialize the policy store
// when the vault is being unsealed.
func (c *Core) setupPolicyStore(ctx context.Context) error {
	// Create the policy store
	sysView := &dynamicSystemView{core: c}
	c.policyStore = NewPolicyStore(ctx, c, c.systemBarrierView, sysView, c.logger)

	if c.ReplicationState().HasState(consts.ReplicationPerformanceSecondary) {
		// Policies will sync from the primary
		return nil
	}

	// Ensure that the default policy exists, and if not, create it
	if err := c.policyStore.loadACLPolicy(ctx, defaultPolicyName, defaultPolicy); err != nil {
		return err
	}
	// Ensure that the response wrapping policy exists
	if err := c.policyStore.loadACLPolicy(ctx, responseWrappingPolicyName, responseWrappingPolicy); err != nil {
		return err
	}

	return nil
}

// teardownPolicyStore is used to reverse setupPolicyStore
// when the vault is being sealed.
func (c *Core) teardownPolicyStore() error {
	c.policyStore = nil
	return nil
}

func (ps *PolicyStore) invalidate(ctx context.Context, name string, policyType PolicyType) {
	// This may come with a prefixed "/" due to joining the file path
	saneName := strings.TrimPrefix(name, "/")

	// We don't lock before removing from the LRU here because the worst that
	// can happen is we load again if something since added it
	switch policyType {
	case PolicyTypeACL:
		if ps.tokenPoliciesLRU != nil {
			ps.tokenPoliciesLRU.Remove(saneName)
		}

	default:
		// Can't do anything
		return
	}

	// Force a reload
	_, err := ps.GetPolicy(ctx, name, policyType)
	if err != nil {
		ps.logger.Error("policy: error fetching policy after invalidation", "name", saneName)
	}
}

// SetPolicy is used to create or update the given policy
func (ps *PolicyStore) SetPolicy(ctx context.Context, p *Policy) error {
	defer metrics.MeasureSince([]string{"policy", "set_policy"}, time.Now())
	if p == nil {
		return fmt.Errorf("nil policy passed in for storage")
	}
	if p.Name == "" {
		return fmt.Errorf("policy name missing")
	}
	// Policies are normalized to lower-case
	p.Name = ps.sanitizeName(p.Name)
	if strutil.StrListContains(immutablePolicies, p.Name) {
		return fmt.Errorf("cannot update %s policy", p.Name)
	}

	return ps.setPolicyInternal(ctx, p)
}

func (ps *PolicyStore) setPolicyInternal(ctx context.Context, p *Policy) error {
	ps.modifyLock.Lock()
	defer ps.modifyLock.Unlock()
	// Create the entry
	entry, err := logical.StorageEntryJSON(p.Name, &PolicyEntry{
		Version: 2,
		Raw:     p.Raw,
		Type:    p.Type,
	})
	if err != nil {
		return fmt.Errorf("failed to create entry: %v", err)
	}
	switch p.Type {
	case PolicyTypeACL:
		if err := ps.aclView.Put(ctx, entry); err != nil {
			return errwrap.Wrapf("failed to persist policy: {{err}}", err)
		}
		ps.policyTypeMap.Store(p.Name, PolicyTypeACL)

		if ps.tokenPoliciesLRU != nil {
			// Update the LRU cache
			ps.tokenPoliciesLRU.Add(p.Name, p)
		}

	default:
		return fmt.Errorf("unknown policy type, cannot set")
	}

	return nil
}

// GetPolicy is used to fetch the named policy
func (ps *PolicyStore) GetPolicy(ctx context.Context, name string, policyType PolicyType) (*Policy, error) {
	defer metrics.MeasureSince([]string{"policy", "get_policy"}, time.Now())

	// Policies are normalized to lower-case
	name = ps.sanitizeName(name)

	var cache *lru.TwoQueueCache
	var view *BarrierView
	switch policyType {
	case PolicyTypeACL:
		cache = ps.tokenPoliciesLRU
		view = ps.aclView
	case PolicyTypeToken:
		cache = ps.tokenPoliciesLRU
		val, ok := ps.policyTypeMap.Load(name)
		if !ok {
			// Doesn't exist
			return nil, nil
		}
		policyType = val.(PolicyType)
		switch policyType {
		case PolicyTypeACL:
			view = ps.aclView
		default:
			return nil, fmt.Errorf("invalid type of policy in type map: %s", policyType)
		}
	}

	if cache != nil {
		// Check for cached policy
		if raw, ok := cache.Get(name); ok {
			return raw.(*Policy), nil
		}
	}

	// Special case the root policy
	if policyType == PolicyTypeACL && name == "root" {
		p := &Policy{Name: "root"}
		if cache != nil {
			cache.Add(p.Name, p)
		}
		return p, nil
	}

	ps.modifyLock.Lock()
	defer ps.modifyLock.Unlock()

	// See if anything has added it since we got the lock
	if cache != nil {
		if raw, ok := cache.Get(name); ok {
			return raw.(*Policy), nil
		}
	}

	out, err := view.Get(ctx, name)
	if err != nil {
		return nil, errwrap.Wrapf("failed to read policy: {{err}}", err)
	}

	if out == nil {
		return nil, nil
	}

	policyEntry := new(PolicyEntry)
	policy := new(Policy)
	err = out.DecodeJSON(policyEntry)
	if err != nil {
		return nil, errwrap.Wrapf("failed to parse policy: {{err}}", err)
	}

	// Set these up here so that they're available for loading into
	// Sentinel
	policy.Name = name
	policy.Raw = policyEntry.Raw
	policy.Type = policyEntry.Type
	switch policyEntry.Type {
	case PolicyTypeACL:
		// Parse normally
		p, err := ParseACLPolicy(policyEntry.Raw)
		if err != nil {
			return nil, errwrap.Wrapf("failed to parse policy: {{err}}", err)
		}
		policy.Paths = p.Paths
		// Reset this in case they set the name in the policy itself
		policy.Name = name

		ps.policyTypeMap.Store(name, PolicyTypeACL)

	default:
		return nil, fmt.Errorf("unknown policy type %q", policyEntry.Type.String())
	}

	if cache != nil {
		// Update the LRU cache
		cache.Add(name, policy)
	}

	return policy, nil
}

// ListPolicies is used to list the available policies
func (ps *PolicyStore) ListPolicies(ctx context.Context, policyType PolicyType) ([]string, error) {
	defer metrics.MeasureSince([]string{"policy", "list_policies"}, time.Now())
	// Scan the view, since the policy names are the same as the
	// key names.
	var keys []string
	var err error
	switch policyType {
	case PolicyTypeACL:
		keys, err = logical.CollectKeys(ctx, ps.aclView)
	default:
		return nil, fmt.Errorf("unknown policy type %s", policyType)
	}

	// We only have non-assignable ACL policies at the moment
	for _, nonAssignable := range nonAssignablePolicies {
		deleteIndex := -1
		//Find indices of non-assignable policies in keys
		for index, key := range keys {
			if key == nonAssignable {
				// Delete collection outside the loop
				deleteIndex = index
				break
			}
		}
		// Remove non-assignable policies when found
		if deleteIndex != -1 {
			keys = append(keys[:deleteIndex], keys[deleteIndex+1:]...)
		}
	}

	return keys, err
}

// DeletePolicy is used to delete the named policy
func (ps *PolicyStore) DeletePolicy(ctx context.Context, name string, policyType PolicyType) error {
	defer metrics.MeasureSince([]string{"policy", "delete_policy"}, time.Now())

	ps.modifyLock.Lock()
	defer ps.modifyLock.Unlock()

	// Policies are normalized to lower-case
	name = ps.sanitizeName(name)

	switch policyType {
	case PolicyTypeACL:
		if strutil.StrListContains(immutablePolicies, name) {
			return fmt.Errorf("cannot delete %s policy", name)
		}
		if name == "default" {
			return fmt.Errorf("cannot delete default policy")
		}

		err := ps.aclView.Delete(ctx, name)
		if err != nil {
			return errwrap.Wrapf("failed to delete policy: {{err}}", err)
		}

		if ps.tokenPoliciesLRU != nil {
			// Clear the cache
			ps.tokenPoliciesLRU.Remove(name)
		}

		ps.policyTypeMap.Delete(name)

	}
	return nil
}

// ACL is used to return an ACL which is built using the
// named policies.
func (ps *PolicyStore) ACL(ctx context.Context, names ...string) (*ACL, error) {
	// Fetch the policies
	var policies []*Policy
	for _, name := range names {
		p, err := ps.GetPolicy(ctx, name, PolicyTypeToken)
		if err != nil {
			return nil, errwrap.Wrapf("failed to get policy: {{err}}", err)
		}
		policies = append(policies, p)
	}

	// Construct the ACL
	acl, err := NewACL(policies)
	if err != nil {
		return nil, errwrap.Wrapf("failed to construct ACL: {{err}}", err)
	}
	return acl, nil
}

func (ps *PolicyStore) loadACLPolicy(ctx context.Context, policyName, policyText string) error {
	// Check if the policy already exists
	policy, err := ps.GetPolicy(ctx, policyName, PolicyTypeACL)

	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("error fetching %s policy from store: {{err}}", policyName), err)
	}

	if policy != nil {
		if !strutil.StrListContains(immutablePolicies, policyName) || policyText == policy.Raw {
			return nil
		}
	}

	policy, err = ParseACLPolicy(policyText)
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("error parsing %s policy: {{err}}", policyName), err)
	}

	if policy == nil {
		return fmt.Errorf("parsing %s policy resulted in nil policy", policyName)
	}

	policy.Name = policyName
	policy.Type = PolicyTypeACL
	return ps.setPolicyInternal(ctx, policy)
}

func (ps *PolicyStore) sanitizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
