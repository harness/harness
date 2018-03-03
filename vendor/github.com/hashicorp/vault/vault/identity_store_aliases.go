package vault

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/protobuf/ptypes"
	memdb "github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/helper/identity"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

// aliasPaths returns the API endpoints to operate on aliases.
// Following are the paths supported:
// entity-alias - To register/modify an alias
// entity-alias/id - To read, modify, delete and list aliases based on their ID
func aliasPaths(i *IdentityStore) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "entity-alias$",
			Fields: map[string]*framework.FieldSchema{
				"id": {
					Type:        framework.TypeString,
					Description: "ID of the entity alias. If set, updates the corresponding entity alias.",
				},
				// entity_id is deprecated in favor of canonical_id
				"entity_id": {
					Type:        framework.TypeString,
					Description: "Entity ID to which this alias belongs to",
				},
				"canonical_id": {
					Type:        framework.TypeString,
					Description: "Entity ID to which this alias belongs to",
				},
				"mount_accessor": {
					Type:        framework.TypeString,
					Description: "Mount accessor to which this alias belongs to",
				},
				"name": {
					Type:        framework.TypeString,
					Description: "Name of the alias",
				},
				"metadata": {
					Type: framework.TypeKVPairs,
					Description: `Metadata to be associated with the alias.
In CLI, this parameter can be repeated multiple times, and it all gets merged together.
For example:
vault <command> <path> metadata=key1=value1 metadata=key2=value2
					`,
				},
			},
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.UpdateOperation: i.pathAliasRegister(),
			},

			HelpSynopsis:    strings.TrimSpace(aliasHelp["alias"][0]),
			HelpDescription: strings.TrimSpace(aliasHelp["alias"][1]),
		},
		// BC path for identity/entity-alias
		{
			Pattern: "alias$",
			Fields: map[string]*framework.FieldSchema{
				"id": {
					Type:        framework.TypeString,
					Description: "ID of the alias",
				},
				// entity_id is deprecated
				"entity_id": {
					Type:        framework.TypeString,
					Description: "Entity ID to which this alias belongs to",
				},
				"canonical_id": {
					Type:        framework.TypeString,
					Description: "Entity ID to which this alias belongs to",
				},
				"mount_accessor": {
					Type:        framework.TypeString,
					Description: "Mount accessor to which this alias belongs to",
				},
				"name": {
					Type:        framework.TypeString,
					Description: "Name of the alias",
				},
				"metadata": {
					Type: framework.TypeKVPairs,
					Description: `Metadata to be associated with the alias.
In CLI, this parameter can be repeated multiple times, and it all gets merged together.
For example:
vault <command> <path> metadata=key1=value1 metadata=key2=value2
					`,
				},
			},
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.UpdateOperation: i.pathAliasRegister(),
			},

			HelpSynopsis:    strings.TrimSpace(aliasHelp["alias"][0]),
			HelpDescription: strings.TrimSpace(aliasHelp["alias"][1]),
		},
		{
			Pattern: "entity-alias/id/" + framework.GenericNameRegex("id"),
			Fields: map[string]*framework.FieldSchema{
				"id": {
					Type:        framework.TypeString,
					Description: "ID of the alias",
				},
				// entity_id is deprecated
				"entity_id": {
					Type:        framework.TypeString,
					Description: "Entity ID to which this alias belongs to",
				},
				"canonical_id": {
					Type:        framework.TypeString,
					Description: "Entity ID to which this alias should be tied to",
				},
				"mount_accessor": {
					Type:        framework.TypeString,
					Description: "Mount accessor to which this alias belongs to",
				},
				"name": {
					Type:        framework.TypeString,
					Description: "Name of the alias",
				},
				"metadata": {
					Type: framework.TypeKVPairs,
					Description: `Metadata to be associated with the alias.
In CLI, this parameter can be repeated multiple times, and it all gets merged together.
For example:
vault <command> <path> metadata=key1=value1 metadata=key2=value2
					`,
				},
			},
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.UpdateOperation: i.pathAliasIDUpdate(),
				logical.ReadOperation:   i.pathAliasIDRead(),
				logical.DeleteOperation: i.pathAliasIDDelete(),
			},

			HelpSynopsis:    strings.TrimSpace(aliasHelp["alias-id"][0]),
			HelpDescription: strings.TrimSpace(aliasHelp["alias-id"][1]),
		},
		{
			Pattern: "entity-alias/id/?$",
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ListOperation: i.pathAliasIDList(),
			},

			HelpSynopsis:    strings.TrimSpace(aliasHelp["alias-id-list"][0]),
			HelpDescription: strings.TrimSpace(aliasHelp["alias-id-list"][1]),
		},
	}
}

// pathAliasRegister is used to register new alias
func (i *IdentityStore) pathAliasRegister() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
		_, ok := d.GetOk("id")
		if ok {
			return i.pathAliasIDUpdate()(ctx, req, d)
		}

		return i.handleAliasUpdateCommon(req, d, nil)
	}
}

// pathAliasIDUpdate is used to update an alias based on the given
// alias ID
func (i *IdentityStore) pathAliasIDUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
		// Get alias id
		aliasID := d.Get("id").(string)

		if aliasID == "" {
			return logical.ErrorResponse("empty alias ID"), nil
		}

		alias, err := i.MemDBAliasByID(aliasID, true, false)
		if err != nil {
			return nil, err
		}
		if alias == nil {
			return logical.ErrorResponse("invalid alias id"), nil
		}

		return i.handleAliasUpdateCommon(req, d, alias)
	}
}

// handleAliasUpdateCommon is used to update an alias
func (i *IdentityStore) handleAliasUpdateCommon(req *logical.Request, d *framework.FieldData, alias *identity.Alias) (*logical.Response, error) {
	var err error
	var newAlias bool
	var entity *identity.Entity
	var previousEntity *identity.Entity

	// Alias will be nil when a new alias is being registered; create a
	// new struct in that case.
	if alias == nil {
		alias = &identity.Alias{}
		newAlias = true
	}

	// Get entity id
	canonicalID := d.Get("entity_id").(string)
	if canonicalID == "" {
		canonicalID = d.Get("canonical_id").(string)
	}

	if canonicalID != "" {
		entity, err = i.MemDBEntityByID(canonicalID, true)
		if err != nil {
			return nil, err
		}
		if entity == nil {
			return logical.ErrorResponse("invalid entity ID"), nil
		}
	}

	// Get alias name
	aliasName := d.Get("name").(string)
	if aliasName == "" {
		return logical.ErrorResponse("missing alias name"), nil
	}

	mountAccessor := d.Get("mount_accessor").(string)
	if mountAccessor == "" {
		return logical.ErrorResponse("missing mount_accessor"), nil
	}

	mountValidationResp := i.validateMountAccessorFunc(mountAccessor)
	if mountValidationResp == nil {
		return logical.ErrorResponse(fmt.Sprintf("invalid mount accessor %q", mountAccessor)), nil
	}

	// Get alias metadata
	metadata, ok, err := d.GetOkErr("metadata")
	if err != nil {
		return logical.ErrorResponse(fmt.Sprintf("failed to parse metadata: %v", err)), nil
	}
	var aliasMetadata map[string]string
	if ok {
		aliasMetadata = metadata.(map[string]string)
	}

	aliasByFactors, err := i.MemDBAliasByFactors(mountValidationResp.MountAccessor, aliasName, false, false)
	if err != nil {
		return nil, err
	}

	resp := &logical.Response{}

	if newAlias {
		if aliasByFactors != nil {
			return logical.ErrorResponse("combination of mount and alias name is already in use"), nil
		}

		// If this is an alias being tied to a non-existent entity, create
		// a new entity for it.
		if entity == nil {
			entity = &identity.Entity{
				Aliases: []*identity.Alias{
					alias,
				},
			}
		} else {
			entity.Aliases = append(entity.Aliases, alias)
		}
	} else {
		// Verify that the combination of alias name and mount is not
		// already tied to a different alias
		if aliasByFactors != nil && aliasByFactors.ID != alias.ID {
			return logical.ErrorResponse("combination of mount and alias name is already in use"), nil
		}

		// Fetch the entity to which the alias is tied to
		existingEntity, err := i.MemDBEntityByAliasID(alias.ID, true)
		if err != nil {
			return nil, err
		}

		if existingEntity == nil {
			return nil, fmt.Errorf("alias is not associated with an entity")
		}

		if entity != nil && entity.ID != existingEntity.ID {
			// Alias should be transferred from 'existingEntity' to 'entity'
			err = i.deleteAliasFromEntity(existingEntity, alias)
			if err != nil {
				return nil, err
			}
			previousEntity = existingEntity
			entity.Aliases = append(entity.Aliases, alias)
			resp.AddWarning(fmt.Sprintf("alias is being transferred from entity %q to %q", existingEntity.ID, entity.ID))
		} else {
			// Update entity with modified alias
			err = i.updateAliasInEntity(existingEntity, alias)
			if err != nil {
				return nil, err
			}
			entity = existingEntity
		}
	}

	// ID creation and other validations; This is more useful for new entities
	// and may not perform anything for the existing entities. Placing the
	// check here to make the flow common for both new and existing entities.
	err = i.sanitizeEntity(entity)
	if err != nil {
		return nil, err
	}

	// Update the fields
	alias.Name = aliasName
	alias.Metadata = aliasMetadata
	alias.MountType = mountValidationResp.MountType
	alias.MountAccessor = mountValidationResp.MountAccessor
	alias.MountPath = mountValidationResp.MountPath

	// Set the canonical ID in the alias index. This should be done after
	// sanitizing entity.
	alias.CanonicalID = entity.ID

	// ID creation and other validations
	err = i.sanitizeAlias(alias)
	if err != nil {
		return nil, err
	}

	// Index entity and its aliases in MemDB and persist entity along with
	// aliases in storage. If the alias is being transferred over from
	// one entity to another, previous entity needs to get refreshed in MemDB
	// and persisted in storage as well.
	err = i.upsertEntity(entity, previousEntity, true)
	if err != nil {
		return nil, err
	}

	// Return ID of both alias and entity
	resp.Data = map[string]interface{}{
		"id":           alias.ID,
		"canonical_id": entity.ID,
	}

	return resp, nil
}

// pathAliasIDRead returns the properties of an alias for a given
// alias ID
func (i *IdentityStore) pathAliasIDRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
		aliasID := d.Get("id").(string)
		if aliasID == "" {
			return logical.ErrorResponse("missing alias id"), nil
		}

		alias, err := i.MemDBAliasByID(aliasID, false, false)
		if err != nil {
			return nil, err
		}

		return i.handleAliasReadCommon(alias)
	}
}

func (i *IdentityStore) handleAliasReadCommon(alias *identity.Alias) (*logical.Response, error) {
	if alias == nil {
		return nil, nil
	}

	respData := map[string]interface{}{}
	respData["id"] = alias.ID
	respData["canonical_id"] = alias.CanonicalID
	respData["mount_type"] = alias.MountType
	respData["mount_accessor"] = alias.MountAccessor
	respData["mount_path"] = alias.MountPath
	respData["metadata"] = alias.Metadata
	respData["name"] = alias.Name
	respData["merged_from_canonical_ids"] = alias.MergedFromCanonicalIDs

	// Convert protobuf timestamp into RFC3339 format
	respData["creation_time"] = ptypes.TimestampString(alias.CreationTime)
	respData["last_update_time"] = ptypes.TimestampString(alias.LastUpdateTime)

	return &logical.Response{
		Data: respData,
	}, nil
}

// pathAliasIDDelete deletes the alias for a given alias ID
func (i *IdentityStore) pathAliasIDDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
		aliasID := d.Get("id").(string)
		if aliasID == "" {
			return logical.ErrorResponse("missing alias ID"), nil
		}

		return nil, i.deleteAlias(aliasID)
	}
}

// pathAliasIDList lists the IDs of all the valid aliases in the identity
// store
func (i *IdentityStore) pathAliasIDList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
		ws := memdb.NewWatchSet()
		iter, err := i.MemDBAliases(ws, false)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch iterator for aliases in memdb: %v", err)
		}

		var aliasIDs []string
		for {
			raw := iter.Next()
			if raw == nil {
				break
			}
			aliasIDs = append(aliasIDs, raw.(*identity.Alias).ID)
		}

		return logical.ListResponse(aliasIDs), nil
	}
}

var aliasHelp = map[string][2]string{
	"alias": {
		"Create a new alias.",
		"",
	},
	"alias-id": {
		"Update, read or delete an alias ID.",
		"",
	},
	"alias-id-list": {
		"List all the entity IDs.",
		"",
	},
}
