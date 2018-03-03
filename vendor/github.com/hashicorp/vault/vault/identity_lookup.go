package vault

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/helper/identity"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

func lookupPaths(i *IdentityStore) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "lookup/entity$",
			Fields: map[string]*framework.FieldSchema{
				"name": {
					Type:        framework.TypeString,
					Description: "Name of the entity.",
				},
				"id": {
					Type:        framework.TypeString,
					Description: "ID of the entity.",
				},
				"alias_id": {
					Type:        framework.TypeString,
					Description: "ID of the alias.",
				},
				"alias_name": {
					Type:        framework.TypeString,
					Description: "Name of the alias. This should be supplied in conjuction with 'alias_mount_accessor'.",
				},
				"alias_mount_accessor": {
					Type:        framework.TypeString,
					Description: "Accessor of the mount to which the alias belongs to. This should be supplied in conjunction with 'alias_name'.",
				},
			},
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.UpdateOperation: i.pathLookupEntityUpdate(),
			},

			HelpSynopsis:    strings.TrimSpace(lookupHelp["lookup-entity"][0]),
			HelpDescription: strings.TrimSpace(lookupHelp["lookup-entity"][1]),
		},
		{
			Pattern: "lookup/group$",
			Fields: map[string]*framework.FieldSchema{
				"name": {
					Type:        framework.TypeString,
					Description: "Name of the group.",
				},
				"id": {
					Type:        framework.TypeString,
					Description: "ID of the group.",
				},
				"alias_id": {
					Type:        framework.TypeString,
					Description: "ID of the alias.",
				},
				"alias_name": {
					Type:        framework.TypeString,
					Description: "Name of the alias. This should be supplied in conjuction with 'alias_mount_accessor'.",
				},
				"alias_mount_accessor": {
					Type:        framework.TypeString,
					Description: "Accessor of the mount to which the alias belongs to. This should be supplied in conjunction with 'alias_name'.",
				},
			},
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.UpdateOperation: i.pathLookupGroupUpdate(),
			},

			HelpSynopsis:    strings.TrimSpace(lookupHelp["lookup-group"][0]),
			HelpDescription: strings.TrimSpace(lookupHelp["lookup-group"][1]),
		},
	}
}

func (i *IdentityStore) pathLookupEntityUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
		var entity *identity.Entity
		var err error

		inputCount := 0

		id := ""
		idRaw, ok := d.GetOk("id")
		if ok {
			inputCount++
			id = idRaw.(string)
		}

		name := ""
		nameRaw, ok := d.GetOk("name")
		if ok {
			inputCount++
			name = nameRaw.(string)
		}

		aliasID := ""
		aliasIDRaw, ok := d.GetOk("alias_id")
		if ok {
			inputCount++
			aliasID = aliasIDRaw.(string)
		}

		aliasName := ""
		aliasNameRaw, ok := d.GetOk("alias_name")
		if ok {
			inputCount++
			aliasName = aliasNameRaw.(string)
		}

		aliasMountAccessor := ""
		aliasMountAccessorRaw, ok := d.GetOk("alias_mount_accessor")
		if ok {
			inputCount++
			aliasMountAccessor = aliasMountAccessorRaw.(string)
		}

		switch {
		case inputCount == 0:
			return logical.ErrorResponse(fmt.Sprintf("query parameter not supplied")), nil

		case inputCount != 1:
			switch {
			case inputCount == 2 && aliasName != "" && aliasMountAccessor != "":
			default:
				return logical.ErrorResponse(fmt.Sprintf("query parameter conflict; please supply distinct set of query parameters")), nil
			}

		case inputCount == 1:
			switch {
			case aliasName != "" || aliasMountAccessor != "":
				return logical.ErrorResponse(fmt.Sprintf("both 'alias_name' and 'alias_mount_accessor' needs to be set")), nil
			}
		}

		switch {
		case id != "":
			entity, err = i.MemDBEntityByID(id, false)
			if err != nil {
				return nil, err
			}

		case name != "":
			entity, err = i.MemDBEntityByName(name, false)
			if err != nil {
				return nil, err
			}

		case aliasID != "":
			alias, err := i.MemDBAliasByID(aliasID, false, false)
			if err != nil {
				return nil, err
			}

			if alias == nil {
				break
			}

			entity, err = i.MemDBEntityByAliasID(alias.ID, false)
			if err != nil {
				return nil, err
			}

		case aliasName != "" && aliasMountAccessor != "":
			alias, err := i.MemDBAliasByFactors(aliasMountAccessor, aliasName, false, false)
			if err != nil {
				return nil, err
			}

			if alias == nil {
				break
			}

			entity, err = i.MemDBEntityByAliasID(alias.ID, false)
			if err != nil {
				return nil, err
			}
		}

		if entity == nil {
			return nil, nil
		}

		return i.handleEntityReadCommon(entity)
	}
}

func (i *IdentityStore) pathLookupGroupUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
		var group *identity.Group
		var err error

		inputCount := 0

		id := ""
		idRaw, ok := d.GetOk("id")
		if ok {
			inputCount++
			id = idRaw.(string)
		}

		name := ""
		nameRaw, ok := d.GetOk("name")
		if ok {
			inputCount++
			name = nameRaw.(string)
		}

		aliasID := ""
		aliasIDRaw, ok := d.GetOk("alias_id")
		if ok {
			inputCount++
			aliasID = aliasIDRaw.(string)
		}

		aliasName := ""
		aliasNameRaw, ok := d.GetOk("alias_name")
		if ok {
			inputCount++
			aliasName = aliasNameRaw.(string)
		}

		aliasMountAccessor := ""
		aliasMountAccessorRaw, ok := d.GetOk("alias_mount_accessor")
		if ok {
			inputCount++
			aliasMountAccessor = aliasMountAccessorRaw.(string)
		}

		switch {
		case inputCount == 0:
			return logical.ErrorResponse(fmt.Sprintf("query parameter not supplied")), nil

		case inputCount != 1:
			switch {
			case inputCount == 2 && aliasName != "" && aliasMountAccessor != "":
			default:
				return logical.ErrorResponse(fmt.Sprintf("query parameter conflict; please supply distinct set of query parameters")), nil
			}

		case inputCount == 1:
			switch {
			case aliasName != "" || aliasMountAccessor != "":
				return logical.ErrorResponse(fmt.Sprintf("both 'alias_name' and 'alias_mount_accessor' needs to be set")), nil
			}
		}

		switch {
		case id != "":
			group, err = i.MemDBGroupByID(id, false)
			if err != nil {
				return nil, err
			}
		case name != "":
			group, err = i.MemDBGroupByName(name, false)
			if err != nil {
				return nil, err
			}
		case aliasID != "":
			alias, err := i.MemDBAliasByID(aliasID, false, true)
			if err != nil {
				return nil, err
			}

			if alias == nil {
				break
			}

			group, err = i.MemDBGroupByAliasID(alias.ID, false)
			if err != nil {
				return nil, err
			}

		case aliasName != "" && aliasMountAccessor != "":
			alias, err := i.MemDBAliasByFactors(aliasMountAccessor, aliasName, false, true)
			if err != nil {
				return nil, err
			}

			if alias == nil {
				break
			}

			group, err = i.MemDBGroupByAliasID(alias.ID, false)
			if err != nil {
				return nil, err
			}
		}

		if group == nil {
			return nil, nil
		}

		return i.handleGroupReadCommon(group)
	}
}

var lookupHelp = map[string][2]string{
	"lookup-entity": {
		"Query entities based on various properties.",
		`Distinct query parameters to be set:
		- 'id'
		To query the entity by its ID.
		- 'name'
		To query the entity by its name.
		- 'alias_id'
		To query the entity by the ID of any of its aliases.
		- 'alias_name' and 'alias_mount_accessor'
		To query the entity by the unique factors that represent an alias; the name and the mount accessor.
		`,
	},
	"lookup-group": {
		"Query groups based on various properties.",
		`Distinct query parameters to be set:
		- 'id'
		To query the group by its ID.
		- 'name'
		To query the group by its name.
		- 'alias_id'
		To query the group by the ID of any of its aliases.
		- 'alias_name' and 'alias_mount_accessor'
		To query the group by the unique factors that represent an alias; the name and the mount accessor.
		`,
	},
}
