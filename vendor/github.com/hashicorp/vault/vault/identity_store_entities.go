package vault

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/protobuf/ptypes"
	memdb "github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/helper/identity"
	"github.com/hashicorp/vault/helper/locksutil"
	"github.com/hashicorp/vault/helper/storagepacker"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

// entityPaths returns the API endpoints supported to operate on entities.
// Following are the paths supported:
// entity - To register a new entity
// entity/id - To lookup, modify, delete and list entities based on ID
// entity/merge - To merge entities based on ID
func entityPaths(i *IdentityStore) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "entity$",
			Fields: map[string]*framework.FieldSchema{
				"id": {
					Type:        framework.TypeString,
					Description: "ID of the entity. If set, updates the corresponding existing entity.",
				},
				"name": {
					Type:        framework.TypeString,
					Description: "Name of the entity",
				},
				"metadata": {
					Type: framework.TypeKVPairs,
					Description: `Metadata to be associated with the entity.
In CLI, this parameter can be repeated multiple times, and it all gets merged together.
For example:
vault <command> <path> metadata=key1=value1 metadata=key2=value2
					`,
				},
				"policies": {
					Type:        framework.TypeCommaStringSlice,
					Description: "Policies to be tied to the entity.",
				},
			},
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.UpdateOperation: i.pathEntityRegister(),
			},

			HelpSynopsis:    strings.TrimSpace(entityHelp["entity"][0]),
			HelpDescription: strings.TrimSpace(entityHelp["entity"][1]),
		},
		{
			Pattern: "entity/id/" + framework.GenericNameRegex("id"),
			Fields: map[string]*framework.FieldSchema{
				"id": {
					Type:        framework.TypeString,
					Description: "ID of the entity.",
				},
				"name": {
					Type:        framework.TypeString,
					Description: "Name of the entity.",
				},
				"metadata": {
					Type: framework.TypeKVPairs,
					Description: `Metadata to be associated with the entity.
In CLI, this parameter can be repeated multiple times, and it all gets merged together.
For example:
vault <command> <path> metadata=key1=value1 metadata=key2=value2
					`,
				},
				"policies": {
					Type:        framework.TypeCommaStringSlice,
					Description: "Policies to be tied to the entity.",
				},
			},
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.UpdateOperation: i.pathEntityIDUpdate(),
				logical.ReadOperation:   i.pathEntityIDRead(),
				logical.DeleteOperation: i.pathEntityIDDelete(),
			},

			HelpSynopsis:    strings.TrimSpace(entityHelp["entity-id"][0]),
			HelpDescription: strings.TrimSpace(entityHelp["entity-id"][1]),
		},
		{
			Pattern: "entity/id/?$",
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ListOperation: i.pathEntityIDList(),
			},

			HelpSynopsis:    strings.TrimSpace(entityHelp["entity-id-list"][0]),
			HelpDescription: strings.TrimSpace(entityHelp["entity-id-list"][1]),
		},
		{
			Pattern: "entity/merge/?$",
			Fields: map[string]*framework.FieldSchema{
				"from_entity_ids": {
					Type:        framework.TypeCommaStringSlice,
					Description: "Entity IDs which needs to get merged",
				},
				"to_entity_id": {
					Type:        framework.TypeString,
					Description: "Entity ID into which all the other entities need to get merged",
				},
				"force": {
					Type:        framework.TypeBool,
					Description: "Setting this will follow the 'mine' strategy for merging MFA secrets. If there are secrets of the same type both in entities that are merged from and in entity into which all others are getting merged, secrets in the destination will be unaltered. If not set, this API will throw an error containing all the conflicts.",
				},
			},
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.UpdateOperation: i.pathEntityMergeID(),
			},

			HelpSynopsis:    strings.TrimSpace(entityHelp["entity-merge-id"][0]),
			HelpDescription: strings.TrimSpace(entityHelp["entity-merge-id"][1]),
		},
	}
}

// pathEntityMergeID merges two or more entities into a single entity
func (i *IdentityStore) pathEntityMergeID() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
		toEntityID := d.Get("to_entity_id").(string)
		if toEntityID == "" {
			return logical.ErrorResponse("missing entity id to merge to"), nil
		}

		fromEntityIDs := d.Get("from_entity_ids").([]string)
		if len(fromEntityIDs) == 0 {
			return logical.ErrorResponse("missing entity ids to merge from"), nil
		}

		force := d.Get("force").(bool)

		toEntityForLocking, err := i.MemDBEntityByID(toEntityID, false)
		if err != nil {
			return nil, err
		}

		if toEntityForLocking == nil {
			return logical.ErrorResponse("entity id to merge to is invalid"), nil
		}

		// Acquire the lock to modify the entity storage entry to merge to
		toEntityLock := locksutil.LockForKey(i.entityLocks, toEntityForLocking.ID)
		toEntityLock.Lock()
		defer toEntityLock.Unlock()

		// Create a MemDB transaction to merge entities
		txn := i.db.Txn(true)
		defer txn.Abort()

		// Re-read post lock acquisition
		toEntity, err := i.MemDBEntityByID(toEntityID, true)
		if err != nil {
			return nil, err
		}

		if toEntity == nil {
			return logical.ErrorResponse("entity id to merge to is invalid"), nil
		}

		if toEntity.ID != toEntityForLocking.ID {
			return logical.ErrorResponse("acquired lock for an undesired entity"), nil
		}

		var conflictErrors error
		for _, fromEntityID := range fromEntityIDs {
			if fromEntityID == toEntityID {
				return logical.ErrorResponse("to_entity_id should not be present in from_entity_ids"), nil
			}

			lockFromEntity, err := i.MemDBEntityByID(fromEntityID, false)
			if err != nil {
				return nil, err
			}

			if lockFromEntity == nil {
				return logical.ErrorResponse("entity id to merge from is invalid"), nil
			}

			// Acquire the lock to modify the entity storage entry to merge from
			fromEntityLock := locksutil.LockForKey(i.entityLocks, lockFromEntity.ID)

			fromLockHeld := false

			// There are only 256 lock buckets and the chances of entity ID collision
			// is fairly high. When we are merging entities belonging to the same
			// bucket, multiple attempts to acquire the same lock should be avoided.
			if fromEntityLock != toEntityLock {
				fromEntityLock.Lock()
				fromLockHeld = true
			}

			// Re-read the entities post lock acquisition
			fromEntity, err := i.MemDBEntityByID(fromEntityID, false)
			if err != nil {
				if fromLockHeld {
					fromEntityLock.Unlock()
				}
				return nil, err
			}

			if fromEntity == nil {
				if fromLockHeld {
					fromEntityLock.Unlock()
				}
				return logical.ErrorResponse("entity id to merge from is invalid"), nil
			}

			if fromEntity.ID != lockFromEntity.ID {
				if fromLockHeld {
					fromEntityLock.Unlock()
				}
				return logical.ErrorResponse("acquired lock for an undesired entity"), nil
			}

			for _, alias := range fromEntity.Aliases {
				// Set the desired canonical ID
				alias.CanonicalID = toEntity.ID

				alias.MergedFromCanonicalIDs = append(alias.MergedFromCanonicalIDs, fromEntity.ID)

				err = i.MemDBUpsertAliasInTxn(txn, alias, false)
				if err != nil {
					if fromLockHeld {
						fromEntityLock.Unlock()
					}
					return nil, fmt.Errorf("failed to update alias during merge: %v", err)
				}

				// Add the alias to the desired entity
				toEntity.Aliases = append(toEntity.Aliases, alias)
			}

			// If the entity from which we are merging from was already a merged
			// entity, transfer over the Merged set to the entity we are
			// merging into.
			toEntity.MergedEntityIDs = append(toEntity.MergedEntityIDs, fromEntity.MergedEntityIDs...)

			// Add the entity from which we are merging from to the list of entities
			// the entity we are merging into is composed of.
			toEntity.MergedEntityIDs = append(toEntity.MergedEntityIDs, fromEntity.ID)

			// Delete the entity which we are merging from in MemDB using the same transaction
			err = i.MemDBDeleteEntityByIDInTxn(txn, fromEntity.ID)
			if err != nil {
				if fromLockHeld {
					fromEntityLock.Unlock()
				}
				return nil, err
			}

			// Delete the entity which we are merging from in storage
			err = i.entityPacker.DeleteItem(fromEntity.ID)
			if err != nil {
				if fromLockHeld {
					fromEntityLock.Unlock()
				}
				return nil, err
			}

			if fromLockHeld {
				fromEntityLock.Unlock()
			}
		}

		if conflictErrors != nil && !force {
			return logical.ErrorResponse(conflictErrors.Error()), nil
		}

		// Update MemDB with changes to the entity we are merging to
		err = i.MemDBUpsertEntityInTxn(txn, toEntity)
		if err != nil {
			return nil, err
		}

		// Persist the entity which we are merging to
		toEntityAsAny, err := ptypes.MarshalAny(toEntity)
		if err != nil {
			return nil, err
		}
		item := &storagepacker.Item{
			ID:      toEntity.ID,
			Message: toEntityAsAny,
		}

		err = i.entityPacker.PutItem(item)
		if err != nil {
			return nil, err
		}

		// Committing the transaction *after* successfully performing storage
		// persistence
		txn.Commit()

		return nil, nil
	}
}

// pathEntityRegister is used to register a new entity
func (i *IdentityStore) pathEntityRegister() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
		_, ok := d.GetOk("id")
		if ok {
			return i.pathEntityIDUpdate()(ctx, req, d)
		}

		return i.handleEntityUpdateCommon(req, d, nil)
	}
}

// pathEntityIDUpdate is used to update an entity based on the given entity ID
func (i *IdentityStore) pathEntityIDUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
		// Get entity id
		entityID := d.Get("id").(string)

		if entityID == "" {
			return logical.ErrorResponse("missing entity id"), nil
		}

		entity, err := i.MemDBEntityByID(entityID, true)
		if err != nil {
			return nil, err
		}
		if entity == nil {
			return nil, fmt.Errorf("invalid entity id")
		}

		return i.handleEntityUpdateCommon(req, d, entity)
	}
}

// handleEntityUpdateCommon is used to update an entity
func (i *IdentityStore) handleEntityUpdateCommon(req *logical.Request, d *framework.FieldData, entity *identity.Entity) (*logical.Response, error) {
	var err error
	var newEntity bool

	// Entity will be nil when a new entity is being registered; create a new
	// struct in that case.
	if entity == nil {
		entity = &identity.Entity{}
		newEntity = true
	}

	// Update the policies if supplied
	entityPoliciesRaw, ok := d.GetOk("policies")
	if ok {
		entity.Policies = entityPoliciesRaw.([]string)
	}

	// Get the name
	entityName := d.Get("name").(string)
	if entityName != "" {
		entityByName, err := i.MemDBEntityByName(entityName, false)
		if err != nil {
			return nil, err
		}
		switch {
		case (newEntity && entityByName != nil), (entityByName != nil && entity.ID != "" && entityByName.ID != entity.ID):
			return logical.ErrorResponse("entity name is already in use"), nil
		}
		entity.Name = entityName
	}

	// Get entity metadata
	metadata, ok, err := d.GetOkErr("metadata")
	if err != nil {
		return logical.ErrorResponse(fmt.Sprintf("failed to parse metadata: %v", err)), nil
	}
	if ok {
		entity.Metadata = metadata.(map[string]string)
	}
	// ID creation and some validations
	err = i.sanitizeEntity(entity)
	if err != nil {
		return nil, err
	}

	// Prepare the response
	respData := map[string]interface{}{
		"id": entity.ID,
	}

	var aliasIDs []string
	for _, alias := range entity.Aliases {
		aliasIDs = append(aliasIDs, alias.ID)
	}

	respData["aliases"] = aliasIDs

	// Update MemDB and persist entity object
	err = i.upsertEntity(entity, nil, true)
	if err != nil {
		return nil, err
	}

	// Return ID of the entity that was either created or updated along with
	// its aliases
	return &logical.Response{
		Data: respData,
	}, nil
}

// pathEntityIDRead returns the properties of an entity for a given entity ID
func (i *IdentityStore) pathEntityIDRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
		entityID := d.Get("id").(string)
		if entityID == "" {
			return logical.ErrorResponse("missing entity id"), nil
		}

		entity, err := i.MemDBEntityByID(entityID, false)
		if err != nil {
			return nil, err
		}
		if entity == nil {
			return nil, nil
		}

		return i.handleEntityReadCommon(entity)
	}
}

func (i *IdentityStore) handleEntityReadCommon(entity *identity.Entity) (*logical.Response, error) {
	respData := map[string]interface{}{}
	respData["id"] = entity.ID
	respData["name"] = entity.Name
	respData["metadata"] = entity.Metadata
	respData["merged_entity_ids"] = entity.MergedEntityIDs
	respData["policies"] = entity.Policies

	// Convert protobuf timestamp into RFC3339 format
	respData["creation_time"] = ptypes.TimestampString(entity.CreationTime)
	respData["last_update_time"] = ptypes.TimestampString(entity.LastUpdateTime)

	// Convert each alias into a map and replace the time format in each
	aliasesToReturn := make([]interface{}, len(entity.Aliases))
	for aliasIdx, alias := range entity.Aliases {
		aliasMap := map[string]interface{}{}
		aliasMap["id"] = alias.ID
		aliasMap["canonical_id"] = alias.CanonicalID
		aliasMap["mount_type"] = alias.MountType
		aliasMap["mount_accessor"] = alias.MountAccessor
		aliasMap["mount_path"] = alias.MountPath
		aliasMap["metadata"] = alias.Metadata
		aliasMap["name"] = alias.Name
		aliasMap["merged_from_canonical_ids"] = alias.MergedFromCanonicalIDs
		aliasMap["creation_time"] = ptypes.TimestampString(alias.CreationTime)
		aliasMap["last_update_time"] = ptypes.TimestampString(alias.LastUpdateTime)
		aliasesToReturn[aliasIdx] = aliasMap
	}

	// Add the aliases information to the response which has the correct time
	// formats
	respData["aliases"] = aliasesToReturn

	// Fetch the groups this entity belongs to and return their identifiers
	groups, inheritedGroups, err := i.groupsByEntityID(entity.ID)
	if err != nil {
		return nil, err
	}

	groupIDs := make([]string, len(groups))
	for i, group := range groups {
		groupIDs[i] = group.ID
	}
	respData["direct_group_ids"] = groupIDs

	inheritedGroupIDs := make([]string, len(inheritedGroups))
	for i, group := range inheritedGroups {
		inheritedGroupIDs[i] = group.ID
	}
	respData["inherited_group_ids"] = inheritedGroupIDs

	respData["group_ids"] = append(groupIDs, inheritedGroupIDs...)

	return &logical.Response{
		Data: respData,
	}, nil
}

// pathEntityIDDelete deletes the entity for a given entity ID
func (i *IdentityStore) pathEntityIDDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
		entityID := d.Get("id").(string)
		if entityID == "" {
			return logical.ErrorResponse("missing entity id"), nil
		}

		return nil, i.deleteEntity(entityID)
	}
}

// pathEntityIDList lists the IDs of all the valid entities in the identity
// store
func (i *IdentityStore) pathEntityIDList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
		ws := memdb.NewWatchSet()
		iter, err := i.MemDBEntities(ws)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch iterator for entities in memdb: %v", err)
		}

		var entityIDs []string
		for {
			raw := iter.Next()
			if raw == nil {
				break
			}
			entityIDs = append(entityIDs, raw.(*identity.Entity).ID)
		}

		return logical.ListResponse(entityIDs), nil
	}
}

var entityHelp = map[string][2]string{
	"entity": {
		"Create a new entity",
		"",
	},
	"entity-id": {
		"Update, read or delete an entity using entity ID",
		"",
	},
	"entity-id-list": {
		"List all the entity IDs",
		"",
	},
	"entity-merge-id": {
		"Merge two or more entities together",
		"",
	},
}
