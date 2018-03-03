package vault

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/ast"
	"github.com/hashicorp/vault/helper/parseutil"
	"github.com/mitchellh/copystructure"
)

const (
	DenyCapability   = "deny"
	CreateCapability = "create"
	ReadCapability   = "read"
	UpdateCapability = "update"
	DeleteCapability = "delete"
	ListCapability   = "list"
	SudoCapability   = "sudo"
	RootCapability   = "root"

	// Backwards compatibility
	OldDenyPathPolicy  = "deny"
	OldReadPathPolicy  = "read"
	OldWritePathPolicy = "write"
	OldSudoPathPolicy  = "sudo"
)

const (
	DenyCapabilityInt uint32 = 1 << iota
	CreateCapabilityInt
	ReadCapabilityInt
	UpdateCapabilityInt
	DeleteCapabilityInt
	ListCapabilityInt
	SudoCapabilityInt
)

type PolicyType uint32

const (
	PolicyTypeACL PolicyType = iota
	PolicyTypeRGP
	PolicyTypeEGP

	// Triggers a lookup in the map to figure out if ACL or RGP
	PolicyTypeToken
)

func (p PolicyType) String() string {
	switch p {
	case PolicyTypeACL:
		return "acl"
	case PolicyTypeRGP:
		return "rgp"
	case PolicyTypeEGP:
		return "egp"
	}

	return ""
}

var (
	cap2Int = map[string]uint32{
		DenyCapability:   DenyCapabilityInt,
		CreateCapability: CreateCapabilityInt,
		ReadCapability:   ReadCapabilityInt,
		UpdateCapability: UpdateCapabilityInt,
		DeleteCapability: DeleteCapabilityInt,
		ListCapability:   ListCapabilityInt,
		SudoCapability:   SudoCapabilityInt,
	}
)

// Policy is used to represent the policy specified by
// an ACL configuration.
type Policy struct {
	Name  string       `hcl:"name"`
	Paths []*PathRules `hcl:"-"`
	Raw   string
	Type  PolicyType
}

// PathRules represents a policy for a path in the namespace.
type PathRules struct {
	Prefix       string
	Policy       string
	Permissions  *ACLPermissions
	Glob         bool
	Capabilities []string

	// These keys are used at the top level to make the HCL nicer; we store in
	// the ACLPermissions object though
	MinWrappingTTLHCL     interface{}              `hcl:"min_wrapping_ttl"`
	MaxWrappingTTLHCL     interface{}              `hcl:"max_wrapping_ttl"`
	AllowedParametersHCL  map[string][]interface{} `hcl:"allowed_parameters"`
	DeniedParametersHCL   map[string][]interface{} `hcl:"denied_parameters"`
	RequiredParametersHCL []string                 `hcl:"required_parameters"`
}

type ACLPermissions struct {
	CapabilitiesBitmap uint32
	MinWrappingTTL     time.Duration
	MaxWrappingTTL     time.Duration
	AllowedParameters  map[string][]interface{}
	DeniedParameters   map[string][]interface{}
	RequiredParameters []string
}

func (p *ACLPermissions) Clone() (*ACLPermissions, error) {
	ret := &ACLPermissions{
		CapabilitiesBitmap: p.CapabilitiesBitmap,
		MinWrappingTTL:     p.MinWrappingTTL,
		MaxWrappingTTL:     p.MaxWrappingTTL,
		RequiredParameters: p.RequiredParameters[:],
	}

	switch {
	case p.AllowedParameters == nil:
	case len(p.AllowedParameters) == 0:
		ret.AllowedParameters = make(map[string][]interface{})
	default:
		clonedAllowed, err := copystructure.Copy(p.AllowedParameters)
		if err != nil {
			return nil, err
		}
		ret.AllowedParameters = clonedAllowed.(map[string][]interface{})
	}

	switch {
	case p.DeniedParameters == nil:
	case len(p.DeniedParameters) == 0:
		ret.DeniedParameters = make(map[string][]interface{})
	default:
		clonedDenied, err := copystructure.Copy(p.DeniedParameters)
		if err != nil {
			return nil, err
		}
		ret.DeniedParameters = clonedDenied.(map[string][]interface{})
	}

	return ret, nil
}

// Parse is used to parse the specified ACL rules into an
// intermediary set of policies, before being compiled into
// the ACL
func ParseACLPolicy(rules string) (*Policy, error) {
	// Parse the rules
	root, err := hcl.Parse(rules)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse policy: %s", err)
	}

	// Top-level item should be the object list
	list, ok := root.Node.(*ast.ObjectList)
	if !ok {
		return nil, fmt.Errorf("Failed to parse policy: does not contain a root object")
	}

	// Check for invalid top-level keys
	valid := []string{
		"name",
		"path",
	}
	if err := checkHCLKeys(list, valid); err != nil {
		return nil, fmt.Errorf("Failed to parse policy: %s", err)
	}

	// Create the initial policy and store the raw text of the rules
	var p Policy
	p.Raw = rules
	p.Type = PolicyTypeACL
	if err := hcl.DecodeObject(&p, list); err != nil {
		return nil, fmt.Errorf("Failed to parse policy: %s", err)
	}

	if o := list.Filter("path"); len(o.Items) > 0 {
		if err := parsePaths(&p, o); err != nil {
			return nil, fmt.Errorf("Failed to parse policy: %s", err)
		}
	}

	return &p, nil
}

func parsePaths(result *Policy, list *ast.ObjectList) error {
	paths := make([]*PathRules, 0, len(list.Items))
	for _, item := range list.Items {
		key := "path"
		if len(item.Keys) > 0 {
			key = item.Keys[0].Token.Value().(string)
		}
		valid := []string{
			"policy",
			"capabilities",
			"allowed_parameters",
			"denied_parameters",
			"required_parameters",
			"min_wrapping_ttl",
			"max_wrapping_ttl",
		}
		if err := checkHCLKeys(item.Val, valid); err != nil {
			return multierror.Prefix(err, fmt.Sprintf("path %q:", key))
		}

		var pc PathRules

		// allocate memory so that DecodeObject can initialize the ACLPermissions struct
		pc.Permissions = new(ACLPermissions)

		pc.Prefix = key
		if err := hcl.DecodeObject(&pc, item.Val); err != nil {
			return multierror.Prefix(err, fmt.Sprintf("path %q:", key))
		}

		// Strip a leading '/' as paths in Vault start after the / in the API path
		if len(pc.Prefix) > 0 && pc.Prefix[0] == '/' {
			pc.Prefix = pc.Prefix[1:]
		}

		// Strip the glob character if found
		if strings.HasSuffix(pc.Prefix, "*") {
			pc.Prefix = strings.TrimSuffix(pc.Prefix, "*")
			pc.Glob = true
		}

		// Map old-style policies into capabilities
		if len(pc.Policy) > 0 {
			switch pc.Policy {
			case OldDenyPathPolicy:
				pc.Capabilities = []string{DenyCapability}
			case OldReadPathPolicy:
				pc.Capabilities = append(pc.Capabilities, []string{ReadCapability, ListCapability}...)
			case OldWritePathPolicy:
				pc.Capabilities = append(pc.Capabilities, []string{CreateCapability, ReadCapability, UpdateCapability, DeleteCapability, ListCapability}...)
			case OldSudoPathPolicy:
				pc.Capabilities = append(pc.Capabilities, []string{CreateCapability, ReadCapability, UpdateCapability, DeleteCapability, ListCapability, SudoCapability}...)
			default:
				return fmt.Errorf("path %q: invalid policy '%s'", key, pc.Policy)
			}
		}

		// Initialize the map
		pc.Permissions.CapabilitiesBitmap = 0
		for _, cap := range pc.Capabilities {
			switch cap {
			// If it's deny, don't include any other capability
			case DenyCapability:
				pc.Capabilities = []string{DenyCapability}
				pc.Permissions.CapabilitiesBitmap = DenyCapabilityInt
				goto PathFinished
			case CreateCapability, ReadCapability, UpdateCapability, DeleteCapability, ListCapability, SudoCapability:
				pc.Permissions.CapabilitiesBitmap |= cap2Int[cap]
			default:
				return fmt.Errorf("path %q: invalid capability '%s'", key, cap)
			}
		}

		if pc.AllowedParametersHCL != nil {
			pc.Permissions.AllowedParameters = make(map[string][]interface{}, len(pc.AllowedParametersHCL))
			for key, val := range pc.AllowedParametersHCL {
				pc.Permissions.AllowedParameters[strings.ToLower(key)] = val
			}
		}
		if pc.DeniedParametersHCL != nil {
			pc.Permissions.DeniedParameters = make(map[string][]interface{}, len(pc.DeniedParametersHCL))

			for key, val := range pc.DeniedParametersHCL {
				pc.Permissions.DeniedParameters[strings.ToLower(key)] = val
			}
		}
		if pc.MinWrappingTTLHCL != nil {
			dur, err := parseutil.ParseDurationSecond(pc.MinWrappingTTLHCL)
			if err != nil {
				return errwrap.Wrapf("error parsing min_wrapping_ttl: {{err}}", err)
			}
			pc.Permissions.MinWrappingTTL = dur
		}
		if pc.MaxWrappingTTLHCL != nil {
			dur, err := parseutil.ParseDurationSecond(pc.MaxWrappingTTLHCL)
			if err != nil {
				return errwrap.Wrapf("error parsing max_wrapping_ttl: {{err}}", err)
			}
			pc.Permissions.MaxWrappingTTL = dur
		}
		if pc.Permissions.MinWrappingTTL != 0 &&
			pc.Permissions.MaxWrappingTTL != 0 &&
			pc.Permissions.MaxWrappingTTL < pc.Permissions.MinWrappingTTL {
			return errors.New("max_wrapping_ttl cannot be less than min_wrapping_ttl")
		}
		if len(pc.RequiredParametersHCL) > 0 {
			pc.Permissions.RequiredParameters = pc.RequiredParametersHCL[:]
		}

	PathFinished:
		paths = append(paths, &pc)
	}

	result.Paths = paths
	return nil
}

func checkHCLKeys(node ast.Node, valid []string) error {
	var list *ast.ObjectList
	switch n := node.(type) {
	case *ast.ObjectList:
		list = n
	case *ast.ObjectType:
		list = n.List
	default:
		return fmt.Errorf("cannot check HCL keys of type %T", n)
	}

	validMap := make(map[string]struct{}, len(valid))
	for _, v := range valid {
		validMap[v] = struct{}{}
	}

	var result error
	for _, item := range list.Items {
		key := item.Keys[0].Token.Value().(string)
		if _, ok := validMap[key]; !ok {
			result = multierror.Append(result, fmt.Errorf(
				"invalid key '%s' on line %d", key, item.Assign.Line))
		}
	}

	return result
}
