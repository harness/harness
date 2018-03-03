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

const (
	groupBucketsPrefix = "packer/group/buckets/"
)

func (c *Core) IdentityStore() *IdentityStore {
	return c.identityStore
}

// NewIdentityStore creates a new identity store
func NewIdentityStore(ctx context.Context, core *Core, config *logical.BackendConfig) (*IdentityStore, error) {
	var err error

	// Create a new in-memory database for the identity store
	db, err := memdb.NewMemDB(identityStoreSchema())
	if err != nil {
		return nil, fmt.Errorf("failed to create memdb for identity store: %v", err)
	}

	iStore := &IdentityStore{
		view:        config.StorageView,
		db:          db,
		entityLocks: locksutil.CreateLocks(),
		logger:      core.logger,
		validateMountAccessorFunc: core.router.validateMountByAccessor,
	}

	iStore.entityPacker, err = storagepacker.NewStoragePacker(iStore.view, iStore.logger, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create entity packer: %v", err)
	}

	iStore.groupPacker, err = storagepacker.NewStoragePacker(iStore.view, iStore.logger, groupBucketsPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to create group packer: %v", err)
	}

	iStore.Backend = &framework.Backend{
		BackendType: logical.TypeLogical,
		Paths: framework.PathAppend(
			entityPaths(iStore),
			aliasPaths(iStore),
			groupAliasPaths(iStore),
			groupPaths(iStore),
			lookupPaths(iStore),
			upgradePaths(iStore),
		),
		Invalidate: iStore.Invalidate,
	}

	err = iStore.Setup(ctx, config)
	if err != nil {
		return nil, err
	}

	return iStore, nil
}

// Invalidate is a callback wherein the backend is informed that the value at
// the given key is updated. In identity store's case, it would be the entity
// storage entries that get updated. The value needs to be read and MemDB needs
// to be updated accordingly.
func (i *IdentityStore) Invalidate(ctx context.Context, key string) {
	i.logger.Debug("identity: invalidate notification received", "key", key)

	switch {
	// Check if the key is a storage entry key for an entity bucket
	case strings.HasPrefix(key, storagepacker.StoragePackerBucketsPrefix):
		// Get the hash value of the storage bucket entry key
		bucketKeyHash := i.entityPacker.BucketKeyHashByKey(key)
		if len(bucketKeyHash) == 0 {
			i.logger.Error("failed to get the bucket entry key hash")
			return
		}

		// Create a MemDB transaction
		txn := i.db.Txn(true)
		defer txn.Abort()

		// Each entity object in MemDB holds the MD5 hash of the storage
		// entry key of the entity bucket. Fetch all the entities that
		// belong to this bucket using the hash value. Remove these entities
		// from MemDB along with all the aliases of each entity.
		entitiesFetched, err := i.MemDBEntitiesByBucketEntryKeyHashInTxn(txn, string(bucketKeyHash))
		if err != nil {
			i.logger.Error("failed to fetch entities using the bucket entry key hash", "bucket_entry_key_hash", bucketKeyHash)
			return
		}

		for _, entity := range entitiesFetched {
			// Delete all the aliases in the entity. This function will also remove
			// the corresponding alias indexes too.
			err = i.deleteAliasesInEntityInTxn(txn, entity, entity.Aliases)
			if err != nil {
				i.logger.Error("failed to delete aliases in entity", "entity_id", entity.ID, "error", err)
				return
			}

			// Delete the entity using the same transaction
			err = i.MemDBDeleteEntityByIDInTxn(txn, entity.ID)
			if err != nil {
				i.logger.Error("failed to delete entity from MemDB", "entity_id", entity.ID, "error", err)
				return
			}
		}

		// Get the storage bucket entry
		bucket, err := i.entityPacker.GetBucket(key)
		if err != nil {
			i.logger.Error("failed to refresh entities", "key", key, "error", err)
			return
		}

		// If the underlying entry is nil, it means that this invalidation
		// notification is for the deletion of the underlying storage entry. At
		// this point, since all the entities belonging to this bucket are
		// already removed, there is nothing else to be done. But, if the
		// storage entry is non-nil, its an indication of an update. In this
		// case, entities in the updated bucket needs to be reinserted into
		// MemDB.
		if bucket != nil {
			for _, item := range bucket.Items {
				entity, err := i.parseEntityFromBucketItem(item)
				if err != nil {
					i.logger.Error("failed to parse entity from bucket entry item", "error", err)
					return
				}

				// Only update MemDB and don't touch the storage
				err = i.upsertEntityInTxn(txn, entity, nil, false, false)
				if err != nil {
					i.logger.Error("failed to update entity in MemDB", "error", err)
					return
				}
			}
		}

		txn.Commit()
		return

	// Check if the key is a storage entry key for an group bucket
	case strings.HasPrefix(key, groupBucketsPrefix):
		// Get the hash value of the storage bucket entry key
		bucketKeyHash := i.groupPacker.BucketKeyHashByKey(key)
		if len(bucketKeyHash) == 0 {
			i.logger.Error("failed to get the bucket entry key hash")
			return
		}

		// Create a MemDB transaction
		txn := i.db.Txn(true)
		defer txn.Abort()

		groupsFetched, err := i.MemDBGroupsByBucketEntryKeyHashInTxn(txn, string(bucketKeyHash))
		if err != nil {
			i.logger.Error("failed to fetch groups using the bucket entry key hash", "bucket_entry_key_hash", bucketKeyHash)
			return
		}

		for _, group := range groupsFetched {
			// Delete the group using the same transaction
			err = i.MemDBDeleteGroupByIDInTxn(txn, group.ID)
			if err != nil {
				i.logger.Error("failed to delete group from MemDB", "group_id", group.ID, "error", err)
				return
			}
		}

		// Get the storage bucket entry
		bucket, err := i.groupPacker.GetBucket(key)
		if err != nil {
			i.logger.Error("failed to refresh group", "key", key, "error", err)
			return
		}

		if bucket != nil {
			for _, item := range bucket.Items {
				group, err := i.parseGroupFromBucketItem(item)
				if err != nil {
					i.logger.Error("failed to parse group from bucket entry item", "error", err)
					return
				}

				// Only update MemDB and don't touch the storage
				err = i.upsertGroupInTxn(txn, group, false)
				if err != nil {
					i.logger.Error("failed to update group in MemDB", "error", err)
					return
				}
			}
		}

		txn.Commit()
		return
	}
}

func (i *IdentityStore) parseEntityFromBucketItem(item *storagepacker.Item) (*identity.Entity, error) {
	if item == nil {
		return nil, fmt.Errorf("nil item")
	}

	var entity identity.Entity
	err := ptypes.UnmarshalAny(item.Message, &entity)
	if err != nil {
		return nil, fmt.Errorf("failed to decode entity from storage bucket item: %v", err)
	}

	return &entity, nil
}

func (i *IdentityStore) parseGroupFromBucketItem(item *storagepacker.Item) (*identity.Group, error) {
	if item == nil {
		return nil, fmt.Errorf("nil item")
	}

	var group identity.Group
	err := ptypes.UnmarshalAny(item.Message, &group)
	if err != nil {
		return nil, fmt.Errorf("failed to decode group from storage bucket item: %v", err)
	}

	return &group, nil
}

// entityByAliasFactors fetches the entity based on factors of alias, i.e mount
// accessor and the alias name.
func (i *IdentityStore) entityByAliasFactors(mountAccessor, aliasName string, clone bool) (*identity.Entity, error) {
	if mountAccessor == "" {
		return nil, fmt.Errorf("missing mount accessor")
	}

	if aliasName == "" {
		return nil, fmt.Errorf("missing alias name")
	}

	txn := i.db.Txn(false)

	return i.entityByAliasFactorsInTxn(txn, mountAccessor, aliasName, clone)
}

// entityByAlaisFactorsInTxn fetches the entity based on factors of alias, i.e
// mount accessor and the alias name.
func (i *IdentityStore) entityByAliasFactorsInTxn(txn *memdb.Txn, mountAccessor, aliasName string, clone bool) (*identity.Entity, error) {
	if txn == nil {
		return nil, fmt.Errorf("nil txn")
	}

	if mountAccessor == "" {
		return nil, fmt.Errorf("missing mount accessor")
	}

	if aliasName == "" {
		return nil, fmt.Errorf("missing alias name")
	}

	alias, err := i.MemDBAliasByFactorsInTxn(txn, mountAccessor, aliasName, false, false)
	if err != nil {
		return nil, err
	}

	if alias == nil {
		return nil, nil
	}

	return i.MemDBEntityByAliasIDInTxn(txn, alias.ID, clone)
}

// CreateOrFetchEntity creates a new entity. This is used by core to
// associate each login attempt by an alias to a unified entity in Vault.
func (i *IdentityStore) CreateOrFetchEntity(alias *logical.Alias) (*identity.Entity, error) {
	var entity *identity.Entity
	var err error

	if alias == nil {
		return nil, fmt.Errorf("alias is nil")
	}

	if alias.Name == "" {
		return nil, fmt.Errorf("empty alias name")
	}

	mountValidationResp := i.validateMountAccessorFunc(alias.MountAccessor)
	if mountValidationResp == nil {
		return nil, fmt.Errorf("invalid mount accessor %q", alias.MountAccessor)
	}

	if mountValidationResp.MountType != alias.MountType {
		return nil, fmt.Errorf("mount accessor %q is not a mount of type %q", alias.MountAccessor, alias.MountType)
	}

	// Check if an entity already exists for the given alais
	entity, err = i.entityByAliasFactors(alias.MountAccessor, alias.Name, false)
	if err != nil {
		return nil, err
	}
	if entity != nil {
		return entity, nil
	}

	// Create a MemDB transaction to update both alias and entity
	txn := i.db.Txn(true)
	defer txn.Abort()

	// Check if an entity was created before acquiring the lock
	entity, err = i.entityByAliasFactorsInTxn(txn, alias.MountAccessor, alias.Name, false)
	if err != nil {
		return nil, err
	}
	if entity != nil {
		return entity, nil
	}

	i.logger.Debug("identity: creating a new entity", "alias", alias)

	entity = &identity.Entity{}

	err = i.sanitizeEntity(entity)
	if err != nil {
		return nil, err
	}

	// Create a new alias
	newAlias := &identity.Alias{
		CanonicalID:   entity.ID,
		Name:          alias.Name,
		MountAccessor: alias.MountAccessor,
		MountPath:     mountValidationResp.MountPath,
		MountType:     mountValidationResp.MountType,
	}

	err = i.sanitizeAlias(newAlias)
	if err != nil {
		return nil, err
	}

	// Append the new alias to the new entity
	entity.Aliases = []*identity.Alias{
		newAlias,
	}

	// Update MemDB and persist entity object
	err = i.upsertEntityInTxn(txn, entity, nil, true, false)
	if err != nil {
		return nil, err
	}

	txn.Commit()

	return entity, nil
}
