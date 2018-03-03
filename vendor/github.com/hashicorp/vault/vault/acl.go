package vault

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/armon/go-radix"
	"github.com/hashicorp/errwrap"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/vault/helper/identity"
	"github.com/hashicorp/vault/helper/strutil"
	"github.com/hashicorp/vault/logical"
)

// ACL is used to wrap a set of policies to provide
// an efficient interface for access control.
type ACL struct {
	// exactRules contains the path policies that are exact
	exactRules *radix.Tree

	// globRules contains the path policies that glob
	globRules *radix.Tree

	// root is enabled if the "root" named policy is present.
	root bool
}

type PolicyCheckOpts struct {
	RootPrivsRequired bool
	Unauth            bool
}

type AuthResults struct {
	ACLResults *ACLResults
	Allowed    bool
	RootPrivs  bool
	Error      *multierror.Error
}

type ACLResults struct {
	Allowed    bool
	RootPrivs  bool
	IsRoot     bool
	MFAMethods []string
}

// New is used to construct a policy based ACL from a set of policies.
func NewACL(policies []*Policy) (*ACL, error) {
	// Initialize
	a := &ACL{
		exactRules: radix.New(),
		globRules:  radix.New(),
		root:       false,
	}

	// Inject each policy
	for _, policy := range policies {
		// Ignore a nil policy object
		if policy == nil {
			continue
		}

		switch policy.Type {
		case PolicyTypeACL:
		default:
			return nil, fmt.Errorf("unable to parse policy (wrong type)")
		}

		// Check if this is root
		if policy.Name == "root" {
			a.root = true
		}
		for _, pc := range policy.Paths {
			// Check which tree to use
			tree := a.exactRules
			if pc.Glob {
				tree = a.globRules
			}

			// Check for an existing policy
			raw, ok := tree.Get(pc.Prefix)
			if !ok {
				clonedPerms, err := pc.Permissions.Clone()
				if err != nil {
					return nil, errwrap.Wrapf("error cloning ACL permissions: {{err}}", err)
				}
				tree.Insert(pc.Prefix, clonedPerms)
				continue
			}

			// these are the ones already in the tree
			existingPerms := raw.(*ACLPermissions)

			switch {
			case existingPerms.CapabilitiesBitmap&DenyCapabilityInt > 0:
				// If we are explicitly denied in the existing capability set,
				// don't save anything else
				continue

			case pc.Permissions.CapabilitiesBitmap&DenyCapabilityInt > 0:
				// If this new policy explicitly denies, only save the deny value
				existingPerms.CapabilitiesBitmap = DenyCapabilityInt
				existingPerms.AllowedParameters = nil
				existingPerms.DeniedParameters = nil
				goto INSERT

			default:
				// Insert the capabilities in this new policy into the existing
				// value
				existingPerms.CapabilitiesBitmap = existingPerms.CapabilitiesBitmap | pc.Permissions.CapabilitiesBitmap
			}

			// Note: In these stanzas, we're preferring minimum lifetimes. So
			// we take the lesser of two specified max values, or we take the
			// lesser of two specified min values, the idea being, allowing
			// token lifetime to be minimum possible.
			//
			// If we have an existing max, and we either don't have a current
			// max, or the current is greater than the previous, use the
			// existing.
			if pc.Permissions.MaxWrappingTTL > 0 &&
				(existingPerms.MaxWrappingTTL == 0 ||
					pc.Permissions.MaxWrappingTTL < existingPerms.MaxWrappingTTL) {
				existingPerms.MaxWrappingTTL = pc.Permissions.MaxWrappingTTL
			}
			// If we have an existing min, and we either don't have a current
			// min, or the current is greater than the previous, use the
			// existing
			if pc.Permissions.MinWrappingTTL > 0 &&
				(existingPerms.MinWrappingTTL == 0 ||
					pc.Permissions.MinWrappingTTL < existingPerms.MinWrappingTTL) {
				existingPerms.MinWrappingTTL = pc.Permissions.MinWrappingTTL
			}

			if len(pc.Permissions.AllowedParameters) > 0 {
				if existingPerms.AllowedParameters == nil {
					existingPerms.AllowedParameters = pc.Permissions.AllowedParameters
				} else {
					for key, value := range pc.Permissions.AllowedParameters {
						pcValue, ok := existingPerms.AllowedParameters[key]
						// If an empty array exist it should overwrite any other
						// value.
						if len(value) == 0 || (ok && len(pcValue) == 0) {
							existingPerms.AllowedParameters[key] = []interface{}{}
						} else {
							// Merge the two maps, appending values on key conflict.
							existingPerms.AllowedParameters[key] = append(value, existingPerms.AllowedParameters[key]...)
						}
					}
				}
			}

			if len(pc.Permissions.DeniedParameters) > 0 {
				if existingPerms.DeniedParameters == nil {
					existingPerms.DeniedParameters = pc.Permissions.DeniedParameters
				} else {
					for key, value := range pc.Permissions.DeniedParameters {
						pcValue, ok := existingPerms.DeniedParameters[key]
						// If an empty array exist it should overwrite any other
						// value.
						if len(value) == 0 || (ok && len(pcValue) == 0) {
							existingPerms.DeniedParameters[key] = []interface{}{}
						} else {
							// Merge the two maps, appending values on key conflict.
							existingPerms.DeniedParameters[key] = append(value, existingPerms.DeniedParameters[key]...)
						}
					}
				}
			}

			if len(pc.Permissions.RequiredParameters) > 0 {
				if len(existingPerms.RequiredParameters) == 0 {
					existingPerms.RequiredParameters = pc.Permissions.RequiredParameters
				} else {
					for _, v := range pc.Permissions.RequiredParameters {
						if !strutil.StrListContains(existingPerms.RequiredParameters, v) {
							existingPerms.RequiredParameters = append(existingPerms.RequiredParameters, v)
						}
					}
				}
			}

		INSERT:
			tree.Insert(pc.Prefix, existingPerms)
		}
	}
	return a, nil
}

func (a *ACL) Capabilities(path string) (pathCapabilities []string) {
	// Fast-path root
	if a.root {
		return []string{RootCapability}
	}

	// Find an exact matching rule, look for glob if no match
	var capabilities uint32
	raw, ok := a.exactRules.Get(path)

	if ok {
		perm := raw.(*ACLPermissions)
		capabilities = perm.CapabilitiesBitmap
		goto CHECK
	}

	// Find a glob rule, default deny if no match
	_, raw, ok = a.globRules.LongestPrefix(path)
	if !ok {
		return []string{DenyCapability}
	} else {
		perm := raw.(*ACLPermissions)
		capabilities = perm.CapabilitiesBitmap
	}

CHECK:
	if capabilities&SudoCapabilityInt > 0 {
		pathCapabilities = append(pathCapabilities, SudoCapability)
	}
	if capabilities&ReadCapabilityInt > 0 {
		pathCapabilities = append(pathCapabilities, ReadCapability)
	}
	if capabilities&ListCapabilityInt > 0 {
		pathCapabilities = append(pathCapabilities, ListCapability)
	}
	if capabilities&UpdateCapabilityInt > 0 {
		pathCapabilities = append(pathCapabilities, UpdateCapability)
	}
	if capabilities&DeleteCapabilityInt > 0 {
		pathCapabilities = append(pathCapabilities, DeleteCapability)
	}
	if capabilities&CreateCapabilityInt > 0 {
		pathCapabilities = append(pathCapabilities, CreateCapability)
	}

	// If "deny" is explicitly set or if the path has no capabilities at all,
	// set the path capabilities to "deny"
	if capabilities&DenyCapabilityInt > 0 || len(pathCapabilities) == 0 {
		pathCapabilities = []string{DenyCapability}
	}
	return
}

// AllowOperation is used to check if the given operation is permitted.
func (a *ACL) AllowOperation(req *logical.Request) (ret *ACLResults) {
	ret = new(ACLResults)

	// Fast-path root
	if a.root {
		ret.Allowed = true
		ret.RootPrivs = true
		ret.IsRoot = true
		return
	}
	op := req.Operation
	path := req.Path

	// Help is always allowed
	if op == logical.HelpOperation {
		ret.Allowed = true
		return
	}

	var permissions *ACLPermissions

	// Find an exact matching rule, look for glob if no match
	var capabilities uint32
	raw, ok := a.exactRules.Get(path)
	if ok {
		permissions = raw.(*ACLPermissions)
		capabilities = permissions.CapabilitiesBitmap
		goto CHECK
	}

	// Find a glob rule, default deny if no match
	_, raw, ok = a.globRules.LongestPrefix(path)
	if !ok {
		return
	} else {
		permissions = raw.(*ACLPermissions)
		capabilities = permissions.CapabilitiesBitmap
	}

CHECK:
	// Check if the minimum permissions are met
	// If "deny" has been explicitly set, only deny will be in the map, so we
	// only need to check for the existence of other values
	ret.RootPrivs = capabilities&SudoCapabilityInt > 0

	operationAllowed := false
	switch op {
	case logical.ReadOperation:
		operationAllowed = capabilities&ReadCapabilityInt > 0
	case logical.ListOperation:
		operationAllowed = capabilities&ListCapabilityInt > 0
	case logical.UpdateOperation:
		operationAllowed = capabilities&UpdateCapabilityInt > 0
	case logical.DeleteOperation:
		operationAllowed = capabilities&DeleteCapabilityInt > 0
	case logical.CreateOperation:
		operationAllowed = capabilities&CreateCapabilityInt > 0

	// These three re-use UpdateCapabilityInt since that's the most appropriate
	// capability/operation mapping
	case logical.RevokeOperation, logical.RenewOperation, logical.RollbackOperation:
		operationAllowed = capabilities&UpdateCapabilityInt > 0

	default:
		return
	}

	if !operationAllowed {
		return
	}

	if permissions.MaxWrappingTTL > 0 {
		if req.WrapInfo == nil || req.WrapInfo.TTL > permissions.MaxWrappingTTL {
			return
		}
	}
	if permissions.MinWrappingTTL > 0 {
		if req.WrapInfo == nil || req.WrapInfo.TTL < permissions.MinWrappingTTL {
			return
		}
	}
	// This situation can happen because of merging, even though in a single
	// path statement we check on ingress
	if permissions.MinWrappingTTL != 0 &&
		permissions.MaxWrappingTTL != 0 &&
		permissions.MaxWrappingTTL < permissions.MinWrappingTTL {
		return
	}

	// Only check parameter permissions for operations that can modify
	// parameters.
	if op == logical.UpdateOperation || op == logical.CreateOperation {
		for _, parameter := range permissions.RequiredParameters {
			if _, ok := req.Data[strings.ToLower(parameter)]; !ok {
				return
			}
		}

		// If there are no data fields, allow
		if len(req.Data) == 0 {
			ret.Allowed = true
			return
		}

		if len(permissions.DeniedParameters) == 0 {
			goto ALLOWED_PARAMETERS
		}

		// Check if all parameters have been denied
		if _, ok := permissions.DeniedParameters["*"]; ok {
			return
		}

		for parameter, value := range req.Data {
			// Check if parameter has been explictly denied
			if valueSlice, ok := permissions.DeniedParameters[strings.ToLower(parameter)]; ok {
				// If the value exists in denied values slice, deny
				if valueInParameterList(value, valueSlice) {
					return
				}
			}
		}

	ALLOWED_PARAMETERS:
		// If we don't have any allowed parameters set, allow
		if len(permissions.AllowedParameters) == 0 {
			ret.Allowed = true
			return
		}

		_, allowedAll := permissions.AllowedParameters["*"]
		if len(permissions.AllowedParameters) == 1 && allowedAll {
			ret.Allowed = true
			return
		}

		for parameter, value := range req.Data {
			valueSlice, ok := permissions.AllowedParameters[strings.ToLower(parameter)]
			// Requested parameter is not in allowed list
			if !ok && !allowedAll {
				return
			}

			// If the value doesn't exists in the allowed values slice,
			// deny
			if ok && !valueInParameterList(value, valueSlice) {
				return
			}
		}
	}

	ret.Allowed = true
	return
}
func (c *Core) performPolicyChecks(ctx context.Context, acl *ACL, te *TokenEntry, req *logical.Request, inEntity *identity.Entity, opts *PolicyCheckOpts) (ret *AuthResults) {
	ret = new(AuthResults)

	// First, perform normal ACL checks if requested. The only time no ACL
	// should be applied is if we are only processing EGPs against a login
	// path in which case opts.Unauth will be set.
	if acl != nil && !opts.Unauth {
		ret.ACLResults = acl.AllowOperation(req)
		ret.RootPrivs = ret.ACLResults.RootPrivs
		// Root is always allowed; skip Sentinel/MFA checks
		if ret.ACLResults.IsRoot {
			//c.logger.Warn("policy: token is root, skipping checks")
			ret.Allowed = true
			return
		}
		if !ret.ACLResults.Allowed {
			return
		}
		if !ret.RootPrivs && opts.RootPrivsRequired {
			return
		}
	}

	ret.Allowed = true
	return
}

func valueInParameterList(v interface{}, list []interface{}) bool {
	// Empty list is equivalent to the item always existing in the list
	if len(list) == 0 {
		return true
	}

	return valueInSlice(v, list)
}

func valueInSlice(v interface{}, list []interface{}) bool {
	for _, el := range list {
		if reflect.TypeOf(el).String() == "string" && reflect.TypeOf(v).String() == "string" {
			item := el.(string)
			val := v.(string)

			if strutil.GlobbedStringsMatch(item, val) {
				return true
			}
		} else if reflect.DeepEqual(el, v) {
			return true
		}
	}

	return false
}
