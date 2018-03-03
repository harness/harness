package vault

import (
	"context"
	"sort"

	"github.com/hashicorp/vault/logical"
)

// Capabilities is used to fetch the capabilities of the given token on the given path
func (c *Core) Capabilities(ctx context.Context, token, path string) ([]string, error) {
	if path == "" {
		return nil, &logical.StatusBadRequest{Err: "missing path"}
	}

	if token == "" {
		return nil, &logical.StatusBadRequest{Err: "missing token"}
	}

	te, err := c.tokenStore.Lookup(ctx, token)
	if err != nil {
		return nil, err
	}
	if te == nil {
		return nil, &logical.StatusBadRequest{Err: "invalid token"}
	}

	if te.Policies == nil {
		return []string{DenyCapability}, nil
	}

	var policies []*Policy
	for _, tePolicy := range te.Policies {
		policy, err := c.policyStore.GetPolicy(ctx, tePolicy, PolicyTypeToken)
		if err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}

	_, derivedPolicies, err := c.fetchEntityAndDerivedPolicies(te.EntityID)
	if err != nil {
		return nil, err
	}

	for _, item := range derivedPolicies {
		policy, err := c.policyStore.GetPolicy(ctx, item, PolicyTypeToken)
		if err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}

	if len(policies) == 0 {
		return []string{DenyCapability}, nil
	}

	acl, err := NewACL(policies)
	if err != nil {
		return nil, err
	}

	capabilities := acl.Capabilities(path)
	sort.Strings(capabilities)
	return capabilities, nil
}
