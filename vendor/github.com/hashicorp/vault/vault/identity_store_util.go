package vault

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/golang/protobuf/ptypes"
	memdb "github.com/hashicorp/go-memdb"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/vault/helper/consts"
	"github.com/hashicorp/vault/helper/identity"
	"github.com/hashicorp/vault/helper/locksutil"
	"github.com/hashicorp/vault/helper/storagepacker"
	"github.com/hashicorp/vault/helper/strutil"
	"github.com/hashicorp/vault/logical"
)

func (c *Core) loadIdentityStoreArtifacts(ctx context.Context) error {
	var err error
	if c.identityStore == nil {
		return fmt.Errorf("identity store is not setup")
	}

	err = c.identityStore.loadEntities(ctx)
	if err != nil {
		return err
	}

	err = c.identityStore.loadGroups(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (i *IdentityStore) loadGroups(ctx context.Context) error {
	i.logger.Debug("identity loading groups")
	existing, err := i.groupPacker.View().List(ctx, groupBucketsPrefix)
	if err != nil {
		return fmt.Errorf("failed to scan for groups: %v", err)
	}
	i.logger.Debug("identity: groups collected", "num_existing", len(existing))

	i.groupLock.Lock()
	defer i.groupLock.Unlock()

	for _, key := range existing {
		bucket, err := i.groupPacker.GetBucket(i.groupPacker.BucketPath(key))
		if err != nil {
			return err
		}

		if bucket == nil {
			continue
		}

		for _, item := range bucket.Items {
			group, err := i.parseGroupFromBucketItem(item)
			if err != nil {
				return err
			}
			if group == nil {
				continue
			}

			if i.logger.IsTrace() {
				i.logger.Trace("loading group", "name", group.Name, "id", group.ID)
			}

			txn := i.db.Txn(true)

			err = i.upsertGroupInTxn(txn, group, false)
			if err != nil {
				txn.Abort()
				return fmt.Errorf("failed to update group in memdb: %v", err)
			}

			txn.Commit()
		}
	}

	if i.logger.IsInfo() {
		i.logger.Info("identity: groups restored")
	}

	return nil
}

func (i *IdentityStore) loadEntities(ctx context.Context) error {
	// Accumulate existing entities
	i.logger.Debug("identity: loading entities")
	existing, err := i.entityPacker.View().List(ctx, storagepacker.StoragePackerBucketsPrefix)
	if err != nil {
		return fmt.Errorf("failed to scan for entities: %v", err)
	}
	i.logger.Debug("identity: entities collected", "num_existing", len(existing))

	// Make the channels used for the worker pool
	broker := make(chan string)
	quit := make(chan bool)

	// Buffer these channels to prevent deadlocks
	errs := make(chan error, len(existing))
	result := make(chan *storagepacker.Bucket, len(existing))

	// Use a wait group
	wg := &sync.WaitGroup{}

	// Create 64 workers to distribute work to
	for j := 0; j < consts.ExpirationRestoreWorkerCount; j++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case bucketKey, ok := <-broker:
					// broker has been closed, we are done
					if !ok {
						return
					}

					bucket, err := i.entityPacker.GetBucket(i.entityPacker.BucketPath(bucketKey))
					if err != nil {
						errs <- err
						continue
					}

					// Write results out to the result channel
					result <- bucket

				// quit early
				case <-quit:
					return
				}
			}
		}()
	}

	// Distribute the collected keys to the workers in a go routine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j, bucketKey := range existing {
			if j%500 == 0 {
				i.logger.Trace("identity: enities loading", "progress", j)
			}

			select {
			case <-quit:
				return

			default:
				broker <- bucketKey
			}
		}

		// Close the broker, causing worker routines to exit
		close(broker)
	}()

	// Restore each key by pulling from the result chan
	for j := 0; j < len(existing); j++ {
		select {
		case err := <-errs:
			// Close all go routines
			close(quit)

			return err

		case bucket := <-result:
			// If there is no entry, nothing to restore
			if bucket == nil {
				continue
			}

			for _, item := range bucket.Items {
				entity, err := i.parseEntityFromBucketItem(item)
				if err != nil {
					return err
				}

				if entity == nil {
					continue
				}

				// Only update MemDB and don't hit the storage again
				err = i.upsertEntity(entity, nil, false)
				if err != nil {
					return fmt.Errorf("failed to update entity in MemDB: %v", err)
				}
			}
		}
	}

	// Let all go routines finish
	wg.Wait()

	if i.logger.IsInfo() {
		i.logger.Info("identity: entities restored")
	}

	return nil
}

// LockForEntityID returns the lock used to modify the entity.
func (i *IdentityStore) LockForEntityID(entityID string) *locksutil.LockEntry {
	return locksutil.LockForKey(i.entityLocks, entityID)
}

// upsertEntityInTxn either creates or updates an existing entity. The
// operations will be updated in both MemDB and storage. If 'persist' is set to
// false, then storage will not be updated. When an alias is transferred from
// one entity to another, both the source and destination entities should get
// updated, in which case, callers should send in both entity and
// previousEntity.
func (i *IdentityStore) upsertEntityInTxn(txn *memdb.Txn, entity *identity.Entity, previousEntity *identity.Entity, persist, lockHeld bool) error {
	var err error

	if txn == nil {
		return fmt.Errorf("txn is nil")
	}

	if entity == nil {
		return fmt.Errorf("entity is nil")
	}

	// Acquire the lock to modify the entity storage entry
	if !lockHeld {
		lock := locksutil.LockForKey(i.entityLocks, entity.ID)
		lock.Lock()
		defer lock.Unlock()
	}

	for _, alias := range entity.Aliases {
		// Verify that alias is not associated to a different one already
		aliasByFactors, err := i.MemDBAliasByFactors(alias.MountAccessor, alias.Name, false, false)
		if err != nil {
			return err
		}

		if aliasByFactors != nil && aliasByFactors.CanonicalID != entity.ID {
			return fmt.Errorf("alias %q in already tied to a different entity %q", alias.ID, aliasByFactors.CanonicalID)
		}

		// Insert or update alias in MemDB using the transaction created above
		err = i.MemDBUpsertAliasInTxn(txn, alias, false)
		if err != nil {
			return err
		}
	}

	// If previous entity is set, update it in MemDB and persist it
	if previousEntity != nil && persist {
		err = i.MemDBUpsertEntityInTxn(txn, previousEntity)
		if err != nil {
			return err
		}

		// Persist the previous entity object
		marshaledPreviousEntity, err := ptypes.MarshalAny(previousEntity)
		if err != nil {
			return err
		}
		err = i.entityPacker.PutItem(&storagepacker.Item{
			ID:      previousEntity.ID,
			Message: marshaledPreviousEntity,
		})
		if err != nil {
			return err
		}
	}

	// Insert or update entity in MemDB using the transaction created above
	err = i.MemDBUpsertEntityInTxn(txn, entity)
	if err != nil {
		return err
	}

	if persist {
		entityAsAny, err := ptypes.MarshalAny(entity)
		if err != nil {
			return err
		}
		item := &storagepacker.Item{
			ID:      entity.ID,
			Message: entityAsAny,
		}

		// Persist the entity object
		err = i.entityPacker.PutItem(item)
		if err != nil {
			return err
		}
	}

	return nil
}

// upsertEntity either creates or updates an existing entity. The operations
// will be updated in both MemDB and storage. If 'persist' is set to false,
// then storage will not be updated. When an alias is transferred from one
// entity to another, both the source and destination entities should get
// updated, in which case, callers should send in both entity and
// previousEntity.
func (i *IdentityStore) upsertEntity(entity *identity.Entity, previousEntity *identity.Entity, persist bool) error {

	// Create a MemDB transaction to update both alias and entity
	txn := i.db.Txn(true)
	defer txn.Abort()

	err := i.upsertEntityInTxn(txn, entity, previousEntity, persist, false)
	if err != nil {
		return err
	}

	txn.Commit()

	return nil
}

// upsertEntityNonLocked creates or updates an entity. The lock to modify the
// entity should be held before calling this function.
func (i *IdentityStore) upsertEntityNonLocked(entity *identity.Entity, previousEntity *identity.Entity, persist bool) error {
	// Create a MemDB transaction to update both alias and entity
	txn := i.db.Txn(true)
	defer txn.Abort()

	err := i.upsertEntityInTxn(txn, entity, previousEntity, persist, true)
	if err != nil {
		return err
	}

	txn.Commit()

	return nil
}

func (i *IdentityStore) deleteEntity(entityID string) error {
	var err error
	var entity *identity.Entity

	if entityID == "" {
		return fmt.Errorf("missing entity id")
	}

	// Since an entity ID is required to acquire the lock to modify the
	// storage, fetch the entity without acquiring the lock

	lockEntity, err := i.MemDBEntityByID(entityID, false)
	if err != nil {
		return err
	}

	if lockEntity == nil {
		return nil
	}

	// Acquire the lock to modify the entity storage entry
	lock := locksutil.LockForKey(i.entityLocks, lockEntity.ID)
	lock.Lock()
	defer lock.Unlock()

	// Create a MemDB transaction to delete entity
	txn := i.db.Txn(true)
	defer txn.Abort()

	// Fetch the entity using its ID
	entity, err = i.MemDBEntityByIDInTxn(txn, entityID, true)
	if err != nil {
		return err
	}

	// If there is no entity for the ID, do nothing
	if entity == nil {
		return nil
	}

	// Delete all the aliases in the entity. This function will also remove
	// the corresponding alias indexes too.
	err = i.deleteAliasesInEntityInTxn(txn, entity, entity.Aliases)
	if err != nil {
		return err
	}

	// Delete the entity using the same transaction
	err = i.MemDBDeleteEntityByIDInTxn(txn, entity.ID)
	if err != nil {
		return err
	}

	// Delete the entity from storage
	err = i.entityPacker.DeleteItem(entity.ID)
	if err != nil {
		return err
	}

	// Committing the transaction *after* successfully deleting entity
	txn.Commit()

	return nil
}

func (i *IdentityStore) deleteAlias(aliasID string) error {
	var err error
	var alias *identity.Alias
	var entity *identity.Entity

	if aliasID == "" {
		return fmt.Errorf("missing alias ID")
	}

	// Since an entity ID is required to acquire the lock to modify the
	// storage, fetch the entity without acquiring the lock

	// Fetch the alias using its ID

	alias, err = i.MemDBAliasByID(aliasID, false, false)
	if err != nil {
		return err
	}

	// If there is no alias for the ID, do nothing
	if alias == nil {
		return nil
	}

	// Find the entity to which the alias is tied to
	lockEntity, err := i.MemDBEntityByAliasID(alias.ID, false)
	if err != nil {
		return err
	}

	// If there is no entity tied to a valid alias, something is wrong
	if lockEntity == nil {
		return fmt.Errorf("alias not associated to an entity")
	}

	// Acquire the lock to modify the entity storage entry
	lock := locksutil.LockForKey(i.entityLocks, lockEntity.ID)
	lock.Lock()
	defer lock.Unlock()

	// Create a MemDB transaction to delete entity
	txn := i.db.Txn(true)
	defer txn.Abort()

	// Fetch the alias again after acquiring the lock using the transaction
	// created above
	alias, err = i.MemDBAliasByIDInTxn(txn, aliasID, false, false)
	if err != nil {
		return err
	}

	// If there is no alias for the ID, do nothing
	if alias == nil {
		return nil
	}

	// Fetch the entity again after acquiring the lock using the transaction
	// created above
	entity, err = i.MemDBEntityByAliasIDInTxn(txn, alias.ID, true)
	if err != nil {
		return err
	}

	// If there is no entity tied to a valid alias, something is wrong
	if entity == nil {
		return fmt.Errorf("alias not associated to an entity")
	}

	// Lock switching should not end up in this code pointing to different
	// entities
	if entity.ID != entity.ID {
		return fmt.Errorf("operating on an entity to which the lock doesn't belong to")
	}

	aliases := []*identity.Alias{
		alias,
	}

	// Delete alias from the entity object
	err = i.deleteAliasesInEntityInTxn(txn, entity, aliases)
	if err != nil {
		return err
	}

	// Update the entity index in the entities table
	err = i.MemDBUpsertEntityInTxn(txn, entity)
	if err != nil {
		return err
	}

	// Persist the entity object
	entityAsAny, err := ptypes.MarshalAny(entity)
	if err != nil {
		return err
	}
	item := &storagepacker.Item{
		ID:      entity.ID,
		Message: entityAsAny,
	}

	err = i.entityPacker.PutItem(item)
	if err != nil {
		return err
	}

	// Committing the transaction *after* successfully updating entity in
	// storage
	txn.Commit()

	return nil
}

func (i *IdentityStore) MemDBUpsertAliasInTxn(txn *memdb.Txn, alias *identity.Alias, groupAlias bool) error {
	if txn == nil {
		return fmt.Errorf("nil txn")
	}

	if alias == nil {
		return fmt.Errorf("alias is nil")
	}

	tableName := entityAliasesTable
	if groupAlias {
		tableName = groupAliasesTable
	}

	aliasRaw, err := txn.First(tableName, "id", alias.ID)
	if err != nil {
		return fmt.Errorf("failed to lookup alias from memdb using alias ID: %v", err)
	}

	if aliasRaw != nil {
		err = txn.Delete(tableName, aliasRaw)
		if err != nil {
			return fmt.Errorf("failed to delete alias from memdb: %v", err)
		}
	}

	if err := txn.Insert(tableName, alias); err != nil {
		return fmt.Errorf("failed to update alias into memdb: %v", err)
	}

	return nil
}

func (i *IdentityStore) MemDBUpsertAlias(alias *identity.Alias, groupAlias bool) error {
	if alias == nil {
		return fmt.Errorf("alias is nil")
	}

	txn := i.db.Txn(true)
	defer txn.Abort()

	err := i.MemDBUpsertAliasInTxn(txn, alias, groupAlias)
	if err != nil {
		return err
	}

	txn.Commit()

	return nil
}

func (i *IdentityStore) MemDBAliasByCanonicalIDInTxn(txn *memdb.Txn, canonicalID string, clone bool, groupAlias bool) (*identity.Alias, error) {
	if canonicalID == "" {
		return nil, fmt.Errorf("missing canonical ID")
	}

	if txn == nil {
		return nil, fmt.Errorf("txn is nil")
	}

	tableName := entityAliasesTable
	if groupAlias {
		tableName = groupAliasesTable
	}

	aliasRaw, err := txn.First(tableName, "canonical_id", canonicalID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch alias from memdb using canonical ID: %v", err)
	}

	if aliasRaw == nil {
		return nil, nil
	}

	alias, ok := aliasRaw.(*identity.Alias)
	if !ok {
		return nil, fmt.Errorf("failed to declare the type of fetched alias")
	}

	if clone {
		return alias.Clone()
	}

	return alias, nil
}

func (i *IdentityStore) MemDBAliasByCanonicalID(canonicalID string, clone bool, groupAlias bool) (*identity.Alias, error) {
	if canonicalID == "" {
		return nil, fmt.Errorf("missing canonical ID")
	}

	txn := i.db.Txn(false)

	return i.MemDBAliasByCanonicalIDInTxn(txn, canonicalID, clone, groupAlias)
}

func (i *IdentityStore) MemDBAliasByIDInTxn(txn *memdb.Txn, aliasID string, clone bool, groupAlias bool) (*identity.Alias, error) {
	if aliasID == "" {
		return nil, fmt.Errorf("missing alias ID")
	}

	if txn == nil {
		return nil, fmt.Errorf("txn is nil")
	}

	tableName := entityAliasesTable
	if groupAlias {
		tableName = groupAliasesTable
	}

	aliasRaw, err := txn.First(tableName, "id", aliasID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch alias from memdb using alias ID: %v", err)
	}

	if aliasRaw == nil {
		return nil, nil
	}

	alias, ok := aliasRaw.(*identity.Alias)
	if !ok {
		return nil, fmt.Errorf("failed to declare the type of fetched alias")
	}

	if clone {
		return alias.Clone()
	}

	return alias, nil
}

func (i *IdentityStore) MemDBAliasByID(aliasID string, clone bool, groupAlias bool) (*identity.Alias, error) {
	if aliasID == "" {
		return nil, fmt.Errorf("missing alias ID")
	}

	txn := i.db.Txn(false)

	return i.MemDBAliasByIDInTxn(txn, aliasID, clone, groupAlias)
}

func (i *IdentityStore) MemDBAliasByFactors(mountAccessor, aliasName string, clone bool, groupAlias bool) (*identity.Alias, error) {
	if aliasName == "" {
		return nil, fmt.Errorf("missing alias name")
	}

	if mountAccessor == "" {
		return nil, fmt.Errorf("missing mount accessor")
	}

	txn := i.db.Txn(false)

	return i.MemDBAliasByFactorsInTxn(txn, mountAccessor, aliasName, clone, groupAlias)
}

func (i *IdentityStore) MemDBAliasByFactorsInTxn(txn *memdb.Txn, mountAccessor, aliasName string, clone bool, groupAlias bool) (*identity.Alias, error) {
	if txn == nil {
		return nil, fmt.Errorf("nil txn")
	}

	if aliasName == "" {
		return nil, fmt.Errorf("missing alias name")
	}

	if mountAccessor == "" {
		return nil, fmt.Errorf("missing mount accessor")
	}

	tableName := entityAliasesTable
	if groupAlias {
		tableName = groupAliasesTable
	}

	aliasRaw, err := txn.First(tableName, "factors", mountAccessor, aliasName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch alias from memdb using factors: %v", err)
	}

	if aliasRaw == nil {
		return nil, nil
	}

	alias, ok := aliasRaw.(*identity.Alias)
	if !ok {
		return nil, fmt.Errorf("failed to declare the type of fetched alias")
	}

	if clone {
		return alias.Clone()
	}

	return alias, nil
}

func (i *IdentityStore) MemDBAliasesByMetadata(filters map[string]string, clone bool, groupAlias bool) ([]*identity.Alias, error) {
	if filters == nil {
		return nil, fmt.Errorf("map filter is nil")
	}

	txn := i.db.Txn(false)
	defer txn.Abort()

	var args []interface{}
	for key, value := range filters {
		args = append(args, key, value)
		break
	}

	tableName := entityAliasesTable
	if groupAlias {
		tableName = groupAliasesTable
	}

	aliasesIter, err := txn.Get(tableName, "metadata", args...)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup aliases using metadata: %v", err)
	}

	var aliases []*identity.Alias
	for alias := aliasesIter.Next(); alias != nil; alias = aliasesIter.Next() {
		entry := alias.(*identity.Alias)
		if len(filters) <= 1 || satisfiesMetadataFilters(entry.Metadata, filters) {
			if clone {
				entry, err = entry.Clone()
				if err != nil {
					return nil, err
				}
			}
			aliases = append(aliases, entry)
		}
	}
	return aliases, nil
}

func (i *IdentityStore) MemDBDeleteAliasByID(aliasID string, groupAlias bool) error {
	if aliasID == "" {
		return nil
	}

	txn := i.db.Txn(true)
	defer txn.Abort()

	err := i.MemDBDeleteAliasByIDInTxn(txn, aliasID, groupAlias)
	if err != nil {
		return err
	}

	txn.Commit()

	return nil
}

func (i *IdentityStore) MemDBDeleteAliasByIDInTxn(txn *memdb.Txn, aliasID string, groupAlias bool) error {
	if aliasID == "" {
		return nil
	}

	if txn == nil {
		return fmt.Errorf("txn is nil")
	}

	alias, err := i.MemDBAliasByIDInTxn(txn, aliasID, false, groupAlias)
	if err != nil {
		return err
	}

	if alias == nil {
		return nil
	}

	tableName := entityAliasesTable
	if groupAlias {
		tableName = groupAliasesTable
	}

	err = txn.Delete(tableName, alias)
	if err != nil {
		return fmt.Errorf("failed to delete alias from memdb: %v", err)
	}

	return nil
}

func (i *IdentityStore) MemDBAliases(ws memdb.WatchSet, groupAlias bool) (memdb.ResultIterator, error) {
	txn := i.db.Txn(false)

	tableName := entityAliasesTable
	if groupAlias {
		tableName = groupAliasesTable
	}

	iter, err := txn.Get(tableName, "id")
	if err != nil {
		return nil, err
	}

	ws.Add(iter.WatchCh())

	return iter, nil
}

func (i *IdentityStore) MemDBUpsertEntityInTxn(txn *memdb.Txn, entity *identity.Entity) error {
	if txn == nil {
		return fmt.Errorf("nil txn")
	}

	if entity == nil {
		return fmt.Errorf("entity is nil")
	}

	entityRaw, err := txn.First(entitiesTable, "id", entity.ID)
	if err != nil {
		return fmt.Errorf("failed to lookup entity from memdb using entity id: %v", err)
	}

	if entityRaw != nil {
		err = txn.Delete(entitiesTable, entityRaw)
		if err != nil {
			return fmt.Errorf("failed to delete entity from memdb: %v", err)
		}
	}

	if err := txn.Insert(entitiesTable, entity); err != nil {
		return fmt.Errorf("failed to update entity into memdb: %v", err)
	}

	return nil
}

func (i *IdentityStore) MemDBUpsertEntity(entity *identity.Entity) error {
	if entity == nil {
		return fmt.Errorf("entity to upsert is nil")
	}

	txn := i.db.Txn(true)
	defer txn.Abort()

	err := i.MemDBUpsertEntityInTxn(txn, entity)
	if err != nil {
		return err
	}

	txn.Commit()

	return nil
}

func (i *IdentityStore) MemDBEntityByIDInTxn(txn *memdb.Txn, entityID string, clone bool) (*identity.Entity, error) {
	if entityID == "" {
		return nil, fmt.Errorf("missing entity id")
	}

	if txn == nil {
		return nil, fmt.Errorf("txn is nil")
	}

	entityRaw, err := txn.First(entitiesTable, "id", entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch entity from memdb using entity id: %v", err)
	}

	if entityRaw == nil {
		return nil, nil
	}

	entity, ok := entityRaw.(*identity.Entity)
	if !ok {
		return nil, fmt.Errorf("failed to declare the type of fetched entity")
	}

	if clone {
		return entity.Clone()
	}

	return entity, nil
}

func (i *IdentityStore) MemDBEntityByID(entityID string, clone bool) (*identity.Entity, error) {
	if entityID == "" {
		return nil, fmt.Errorf("missing entity id")
	}

	txn := i.db.Txn(false)

	return i.MemDBEntityByIDInTxn(txn, entityID, clone)
}

func (i *IdentityStore) MemDBEntityByNameInTxn(txn *memdb.Txn, entityName string, clone bool) (*identity.Entity, error) {
	if entityName == "" {
		return nil, fmt.Errorf("missing entity name")
	}

	if txn == nil {
		return nil, fmt.Errorf("txn is nil")
	}

	entityRaw, err := txn.First(entitiesTable, "name", entityName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch entity from memdb using entity name: %v", err)
	}

	if entityRaw == nil {
		return nil, nil
	}

	entity, ok := entityRaw.(*identity.Entity)
	if !ok {
		return nil, fmt.Errorf("failed to declare the type of fetched entity")
	}

	if clone {
		return entity.Clone()
	}

	return entity, nil
}

func (i *IdentityStore) MemDBEntityByName(entityName string, clone bool) (*identity.Entity, error) {
	if entityName == "" {
		return nil, fmt.Errorf("missing entity name")
	}

	txn := i.db.Txn(false)

	return i.MemDBEntityByNameInTxn(txn, entityName, clone)
}

func (i *IdentityStore) MemDBEntitiesByMetadata(filters map[string]string, clone bool) ([]*identity.Entity, error) {
	if filters == nil {
		return nil, fmt.Errorf("map filter is nil")
	}

	txn := i.db.Txn(false)
	defer txn.Abort()

	var args []interface{}
	for key, value := range filters {
		args = append(args, key, value)
		break
	}

	entitiesIter, err := txn.Get(entitiesTable, "metadata", args...)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup entities using metadata: %v", err)
	}

	var entities []*identity.Entity
	for entity := entitiesIter.Next(); entity != nil; entity = entitiesIter.Next() {
		entry := entity.(*identity.Entity)
		if clone {
			entry, err = entry.Clone()
			if err != nil {
				return nil, err
			}
		}
		if len(filters) <= 1 || satisfiesMetadataFilters(entry.Metadata, filters) {
			entities = append(entities, entry)
		}
	}
	return entities, nil
}

func (i *IdentityStore) MemDBEntitiesByBucketEntryKeyHash(hashValue string) ([]*identity.Entity, error) {
	if hashValue == "" {
		return nil, fmt.Errorf("empty hash value")
	}

	txn := i.db.Txn(false)
	defer txn.Abort()

	return i.MemDBEntitiesByBucketEntryKeyHashInTxn(txn, hashValue)
}

func (i *IdentityStore) MemDBEntitiesByBucketEntryKeyHashInTxn(txn *memdb.Txn, hashValue string) ([]*identity.Entity, error) {
	if txn == nil {
		return nil, fmt.Errorf("nil txn")
	}

	if hashValue == "" {
		return nil, fmt.Errorf("empty hash value")
	}

	entitiesIter, err := txn.Get(entitiesTable, "bucket_key_hash", hashValue)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup entities using bucket entry key hash: %v", err)
	}

	var entities []*identity.Entity
	for entity := entitiesIter.Next(); entity != nil; entity = entitiesIter.Next() {
		entities = append(entities, entity.(*identity.Entity))
	}

	return entities, nil
}

func (i *IdentityStore) MemDBEntityByMergedEntityIDInTxn(txn *memdb.Txn, mergedEntityID string, clone bool) (*identity.Entity, error) {
	if mergedEntityID == "" {
		return nil, fmt.Errorf("missing merged entity id")
	}

	if txn == nil {
		return nil, fmt.Errorf("txn is nil")
	}

	entityRaw, err := txn.First(entitiesTable, "merged_entity_ids", mergedEntityID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch entity from memdb using merged entity id: %v", err)
	}

	if entityRaw == nil {
		return nil, nil
	}

	entity, ok := entityRaw.(*identity.Entity)
	if !ok {
		return nil, fmt.Errorf("failed to declare the type of fetched entity")
	}

	if clone {
		return entity.Clone()
	}

	return entity, nil
}

func (i *IdentityStore) MemDBEntityByMergedEntityID(mergedEntityID string, clone bool) (*identity.Entity, error) {
	if mergedEntityID == "" {
		return nil, fmt.Errorf("missing merged entity id")
	}

	txn := i.db.Txn(false)

	return i.MemDBEntityByMergedEntityIDInTxn(txn, mergedEntityID, clone)
}

func (i *IdentityStore) MemDBEntityByAliasIDInTxn(txn *memdb.Txn, aliasID string, clone bool) (*identity.Entity, error) {
	if aliasID == "" {
		return nil, fmt.Errorf("missing alias ID")
	}

	if txn == nil {
		return nil, fmt.Errorf("txn is nil")
	}

	alias, err := i.MemDBAliasByIDInTxn(txn, aliasID, false, false)
	if err != nil {
		return nil, err
	}

	if alias == nil {
		return nil, nil
	}

	return i.MemDBEntityByIDInTxn(txn, alias.CanonicalID, clone)
}

func (i *IdentityStore) MemDBEntityByAliasID(aliasID string, clone bool) (*identity.Entity, error) {
	if aliasID == "" {
		return nil, fmt.Errorf("missing alias ID")
	}

	txn := i.db.Txn(false)

	return i.MemDBEntityByAliasIDInTxn(txn, aliasID, clone)
}

func (i *IdentityStore) MemDBDeleteEntityByID(entityID string) error {
	if entityID == "" {
		return nil
	}

	txn := i.db.Txn(true)
	defer txn.Abort()

	err := i.MemDBDeleteEntityByIDInTxn(txn, entityID)
	if err != nil {
		return err
	}

	txn.Commit()

	return nil
}

func (i *IdentityStore) MemDBDeleteEntityByIDInTxn(txn *memdb.Txn, entityID string) error {
	if entityID == "" {
		return nil
	}

	if txn == nil {
		return fmt.Errorf("txn is nil")
	}

	entity, err := i.MemDBEntityByIDInTxn(txn, entityID, false)
	if err != nil {
		return err
	}

	if entity == nil {
		return nil
	}

	err = txn.Delete(entitiesTable, entity)
	if err != nil {
		return fmt.Errorf("failed to delete entity from memdb: %v", err)
	}

	return nil
}

func (i *IdentityStore) MemDBEntities(ws memdb.WatchSet) (memdb.ResultIterator, error) {
	txn := i.db.Txn(false)

	iter, err := txn.Get(entitiesTable, "id")
	if err != nil {
		return nil, err
	}

	ws.Add(iter.WatchCh())

	return iter, nil
}

func (i *IdentityStore) sanitizeAlias(alias *identity.Alias) error {
	var err error

	if alias == nil {
		return fmt.Errorf("alias is nil")
	}

	// Alias must always be tied to a canonical object
	if alias.CanonicalID == "" {
		return fmt.Errorf("missing canonical ID")
	}

	// Alias must have a name
	if alias.Name == "" {
		return fmt.Errorf("missing alias name %q", alias.Name)
	}

	// Alias metadata should always be map[string]string
	err = validateMetadata(alias.Metadata)
	if err != nil {
		return fmt.Errorf("invalid alias metadata: %v", err)
	}

	// Create an ID if there isn't one already
	if alias.ID == "" {
		alias.ID, err = uuid.GenerateUUID()
		if err != nil {
			return fmt.Errorf("failed to generate alias ID")
		}
	}

	// Set the creation and last update times
	if alias.CreationTime == nil {
		alias.CreationTime = ptypes.TimestampNow()
		alias.LastUpdateTime = alias.CreationTime
	} else {
		alias.LastUpdateTime = ptypes.TimestampNow()
	}

	return nil
}

func (i *IdentityStore) sanitizeEntity(entity *identity.Entity) error {
	var err error

	if entity == nil {
		return fmt.Errorf("entity is nil")
	}

	// Create an ID if there isn't one already
	if entity.ID == "" {
		entity.ID, err = uuid.GenerateUUID()
		if err != nil {
			return fmt.Errorf("failed to generate entity id")
		}

		// Set the hash value of the storage bucket key in entity
		entity.BucketKeyHash = i.entityPacker.BucketKeyHashByItemID(entity.ID)
	}

	// Create a name if there isn't one already
	if entity.Name == "" {
		entity.Name, err = i.generateName("entity")
		if err != nil {
			return fmt.Errorf("failed to generate entity name")
		}
	}

	// Entity metadata should always be map[string]string
	err = validateMetadata(entity.Metadata)
	if err != nil {
		return fmt.Errorf("invalid entity metadata: %v", err)
	}

	// Set the creation and last update times
	if entity.CreationTime == nil {
		entity.CreationTime = ptypes.TimestampNow()
		entity.LastUpdateTime = entity.CreationTime
	} else {
		entity.LastUpdateTime = ptypes.TimestampNow()
	}

	return nil
}

func (i *IdentityStore) sanitizeAndUpsertGroup(group *identity.Group, memberGroupIDs []string) error {
	var err error

	if group == nil {
		return fmt.Errorf("group is nil")
	}

	// Create an ID if there isn't one already
	if group.ID == "" {
		group.ID, err = uuid.GenerateUUID()
		if err != nil {
			return fmt.Errorf("failed to generate group id")
		}

		// Set the hash value of the storage bucket key in group
		group.BucketKeyHash = i.groupPacker.BucketKeyHashByItemID(group.ID)
	}

	// Create a name if there isn't one already
	if group.Name == "" {
		group.Name, err = i.generateName("group")
		if err != nil {
			return fmt.Errorf("failed to generate group name")
		}
	}

	// Entity metadata should always be map[string]string
	err = validateMetadata(group.Metadata)
	if err != nil {
		return fmt.Errorf("invalid group metadata: %v", err)
	}

	// Set the creation and last update times
	if group.CreationTime == nil {
		group.CreationTime = ptypes.TimestampNow()
		group.LastUpdateTime = group.CreationTime
	} else {
		group.LastUpdateTime = ptypes.TimestampNow()
	}

	// Remove duplicate entity IDs and check if all IDs are valid
	group.MemberEntityIDs = strutil.RemoveDuplicates(group.MemberEntityIDs, false)
	for _, entityID := range group.MemberEntityIDs {
		err = i.validateEntityID(entityID)
		if err != nil {
			return err
		}
	}

	txn := i.db.Txn(true)
	defer txn.Abort()

	memberGroupIDs = strutil.RemoveDuplicates(memberGroupIDs, false)
	// After the group lock is held, make membership updates to all the
	// relevant groups
	for _, memberGroupID := range memberGroupIDs {
		memberGroup, err := i.MemDBGroupByID(memberGroupID, true)
		if err != nil {
			return err
		}
		if memberGroup == nil {
			return fmt.Errorf("invalid member group ID %q", memberGroupID)
		}

		// Skip if memberGroupID is already a member of group.ID
		if strutil.StrListContains(memberGroup.ParentGroupIDs, group.ID) {
			continue
		}

		// Ensure that adding memberGroupID does not lead to cyclic
		// relationships
		err = i.validateMemberGroupID(group.ID, memberGroupID)
		if err != nil {
			return err
		}

		memberGroup.ParentGroupIDs = append(memberGroup.ParentGroupIDs, group.ID)

		// This technically is not upsert. It is only update, only the method name is upsert here.
		err = i.upsertGroupInTxn(txn, memberGroup, true)
		if err != nil {
			// Ideally we would want to revert the whole operation in case of
			// errors while persisting in member groups. But there is no
			// storage transaction support yet. When we do have it, this will need
			// an update.
			return err
		}
	}

	// Sanitize the group alias
	if group.Alias != nil {
		group.Alias.CanonicalID = group.ID

		err = i.sanitizeAlias(group.Alias)
		if err != nil {
			return err
		}

		err = i.MemDBUpsertAliasInTxn(txn, group.Alias, true)
		if err != nil {
			return err
		}
	}

	err = i.upsertGroupInTxn(txn, group, true)
	if err != nil {
		return err
	}

	txn.Commit()

	return nil
}

func (i *IdentityStore) validateMemberGroupID(groupID string, memberGroupID string) error {
	// Detect self loop
	if groupID == memberGroupID {
		return fmt.Errorf("member group ID %q is same as the ID of the group", groupID)
	}

	group, err := i.MemDBGroupByID(groupID, true)
	if err != nil {
		return err
	}

	// If group is nil, that means that a group doesn't already exist and its
	// okay to add any group as its member group.
	if group == nil {
		return nil
	}

	// If adding the memberGroupID to groupID creates a cycle, then groupID must
	// be a hop in that loop. Start a DFS traversal from memberGroupID and see if
	// it reaches back to groupID. If it does, then it's a loop.

	// Created a visited set
	visited := make(map[string]bool)
	cycleDetected, err := i.detectCycleDFS(visited, groupID, memberGroupID)
	if err != nil {
		return fmt.Errorf("failed to perform cyclic relationship detection for member group ID %q", memberGroupID)
	}
	if cycleDetected {
		return fmt.Errorf("cyclic relationship detected for member group ID %q", memberGroupID)
	}

	return nil
}

func (i *IdentityStore) validateEntityID(entityID string) error {
	entity, err := i.MemDBEntityByID(entityID, false)
	if err != nil {
		return fmt.Errorf("failed to validate entity ID %q: %v", entityID, err)
	}
	if entity == nil {
		return fmt.Errorf("invalid entity ID %q", entityID)
	}
	return nil
}

func (i *IdentityStore) validateGroupID(groupID string) error {
	group, err := i.MemDBGroupByID(groupID, false)
	if err != nil {
		return fmt.Errorf("failed to validate group ID %q: %v", groupID, err)
	}
	if group == nil {
		return fmt.Errorf("invalid group ID %q", groupID)
	}
	return nil
}

func (i *IdentityStore) deleteAliasesInEntityInTxn(txn *memdb.Txn, entity *identity.Entity, aliases []*identity.Alias) error {
	if entity == nil {
		return fmt.Errorf("entity is nil")
	}

	if txn == nil {
		return fmt.Errorf("txn is nil")
	}

	var remainList []*identity.Alias
	var removeList []*identity.Alias

	for _, item := range aliases {
		for _, alias := range entity.Aliases {
			if alias.ID == item.ID {
				removeList = append(removeList, alias)
			} else {
				remainList = append(remainList, alias)
			}
		}
	}

	// Remove identity indices from aliases table for those that needs to
	// be removed
	for _, alias := range removeList {
		aliasToBeRemoved, err := i.MemDBAliasByIDInTxn(txn, alias.ID, false, false)
		if err != nil {
			return err
		}
		if aliasToBeRemoved == nil {
			return fmt.Errorf("alias was not indexed")
		}
		err = i.MemDBDeleteAliasByIDInTxn(txn, aliasToBeRemoved.ID, false)
		if err != nil {
			return err
		}
	}

	// Update the entity with remaining items
	entity.Aliases = remainList

	return nil
}

func (i *IdentityStore) deleteAliasFromEntity(entity *identity.Entity, alias *identity.Alias) error {
	if entity == nil {
		return fmt.Errorf("entity is nil")
	}

	if alias == nil {
		return fmt.Errorf("alias is nil")
	}

	for aliasIndex, item := range entity.Aliases {
		if item.ID == alias.ID {
			entity.Aliases = append(entity.Aliases[:aliasIndex], entity.Aliases[aliasIndex+1:]...)
			break
		}
	}

	return nil
}

func (i *IdentityStore) updateAliasInEntity(entity *identity.Entity, alias *identity.Alias) error {
	if entity == nil {
		return fmt.Errorf("entity is nil")
	}

	if alias == nil {
		return fmt.Errorf("alias is nil")
	}

	aliasFound := false
	for aliasIndex, item := range entity.Aliases {
		if item.ID == alias.ID {
			aliasFound = true
			entity.Aliases[aliasIndex] = alias
		}
	}

	if !aliasFound {
		return fmt.Errorf("alias does not exist in entity")
	}

	return nil
}

// validateMeta validates a set of key/value pairs from the agent config
func validateMetadata(meta map[string]string) error {
	if len(meta) > metaMaxKeyPairs {
		return fmt.Errorf("metadata cannot contain more than %d key/value pairs", metaMaxKeyPairs)
	}

	for key, value := range meta {
		if err := validateMetaPair(key, value); err != nil {
			return fmt.Errorf("failed to load metadata pair (%q, %q): %v", key, value, err)
		}
	}

	return nil
}

// validateMetaPair checks that the given key/value pair is in a valid format
func validateMetaPair(key, value string) error {
	if key == "" {
		return fmt.Errorf("key cannot be blank")
	}
	if !metaKeyFormatRegEx(key) {
		return fmt.Errorf("key contains invalid characters")
	}
	if len(key) > metaKeyMaxLength {
		return fmt.Errorf("key is too long (limit: %d characters)", metaKeyMaxLength)
	}
	if strings.HasPrefix(key, metaKeyReservedPrefix) {
		return fmt.Errorf("key prefix %q is reserved for internal use", metaKeyReservedPrefix)
	}
	if len(value) > metaValueMaxLength {
		return fmt.Errorf("value is too long (limit: %d characters)", metaValueMaxLength)
	}
	return nil
}

// satisfiesMetadataFilters returns true if the metadata map contains the given filters
func satisfiesMetadataFilters(meta map[string]string, filters map[string]string) bool {
	for key, value := range filters {
		if v, ok := meta[key]; !ok || v != value {
			return false
		}
	}
	return true
}

func (i *IdentityStore) MemDBGroupByNameInTxn(txn *memdb.Txn, groupName string, clone bool) (*identity.Group, error) {
	if groupName == "" {
		return nil, fmt.Errorf("missing group name")
	}

	if txn == nil {
		return nil, fmt.Errorf("txn is nil")
	}

	groupRaw, err := txn.First(groupsTable, "name", groupName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch group from memdb using group name: %v", err)
	}

	if groupRaw == nil {
		return nil, nil
	}

	group, ok := groupRaw.(*identity.Group)
	if !ok {
		return nil, fmt.Errorf("failed to declare the type of fetched group")
	}

	if clone {
		return group.Clone()
	}

	return group, nil
}

func (i *IdentityStore) MemDBGroupByName(groupName string, clone bool) (*identity.Group, error) {
	if groupName == "" {
		return nil, fmt.Errorf("missing group name")
	}

	txn := i.db.Txn(false)

	return i.MemDBGroupByNameInTxn(txn, groupName, clone)
}

func (i *IdentityStore) UpsertGroup(group *identity.Group, persist bool) error {
	txn := i.db.Txn(true)
	defer txn.Abort()

	err := i.upsertGroupInTxn(txn, group, persist)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (i *IdentityStore) upsertGroupInTxn(txn *memdb.Txn, group *identity.Group, persist bool) error {
	var err error

	if txn == nil {
		return fmt.Errorf("txn is nil")
	}

	if group == nil {
		return fmt.Errorf("group is nil")
	}

	// Increment the modify index of the group
	group.ModifyIndex++

	// Insert or update group in MemDB using the transaction created above
	err = i.MemDBUpsertGroupInTxn(txn, group)
	if err != nil {
		return err
	}

	if persist {
		groupAsAny, err := ptypes.MarshalAny(group)
		if err != nil {
			return err
		}

		item := &storagepacker.Item{
			ID:      group.ID,
			Message: groupAsAny,
		}

		err = i.groupPacker.PutItem(item)
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *IdentityStore) MemDBUpsertGroup(group *identity.Group) error {
	txn := i.db.Txn(true)
	defer txn.Abort()

	err := i.MemDBUpsertGroupInTxn(txn, group)
	if err != nil {
		return err
	}

	txn.Commit()

	return nil
}

func (i *IdentityStore) MemDBUpsertGroupInTxn(txn *memdb.Txn, group *identity.Group) error {
	if txn == nil {
		return fmt.Errorf("nil txn")
	}

	if group == nil {
		return fmt.Errorf("group is nil")
	}

	groupRaw, err := txn.First(groupsTable, "id", group.ID)
	if err != nil {
		return fmt.Errorf("failed to lookup group from memdb using group id: %v", err)
	}

	if groupRaw != nil {
		err = txn.Delete(groupsTable, groupRaw)
		if err != nil {
			return fmt.Errorf("failed to delete group from memdb: %v", err)
		}
	}

	if err := txn.Insert(groupsTable, group); err != nil {
		return fmt.Errorf("failed to update group into memdb: %v", err)
	}

	return nil
}

func (i *IdentityStore) deleteGroupByID(groupID string) error {
	var err error
	var group *identity.Group

	if groupID == "" {
		return fmt.Errorf("missing group ID")
	}

	// Acquire the lock to modify the group storage entry
	i.groupLock.Lock()
	defer i.groupLock.Unlock()

	// Create a MemDB transaction to delete group
	txn := i.db.Txn(true)
	defer txn.Abort()

	group, err = i.MemDBGroupByIDInTxn(txn, groupID, false)
	if err != nil {
		return err
	}

	// If there is no group for the ID, do nothing
	if group == nil {
		return nil
	}

	// Delete group alias from memdb
	if group.Type == groupTypeExternal && group.Alias != nil {
		err = i.MemDBDeleteAliasByIDInTxn(txn, group.Alias.ID, true)
		if err != nil {
			return err
		}
	}

	// Delete the group using the same transaction
	err = i.MemDBDeleteGroupByIDInTxn(txn, group.ID)
	if err != nil {
		return err
	}

	// Delete the group from storage
	err = i.groupPacker.DeleteItem(group.ID)
	if err != nil {
		return err
	}

	// Committing the transaction *after* successfully deleting group
	txn.Commit()

	return nil
}

func (i *IdentityStore) MemDBDeleteGroupByIDInTxn(txn *memdb.Txn, groupID string) error {
	if groupID == "" {
		return nil
	}

	if txn == nil {
		return fmt.Errorf("txn is nil")
	}

	group, err := i.MemDBGroupByIDInTxn(txn, groupID, false)
	if err != nil {
		return err
	}

	if group == nil {
		return nil
	}

	err = txn.Delete("groups", group)
	if err != nil {
		return fmt.Errorf("failed to delete group from memdb: %v", err)
	}

	return nil
}

func (i *IdentityStore) deleteGroupByName(groupName string) error {
	var err error
	var group *identity.Group

	if groupName == "" {
		return fmt.Errorf("missing group name")
	}

	// Acquire the lock to modify the group storage entry
	i.groupLock.Lock()
	defer i.groupLock.Unlock()

	// Create a MemDB transaction to delete group
	txn := i.db.Txn(true)
	defer txn.Abort()

	// Fetch the group using its ID
	group, err = i.MemDBGroupByNameInTxn(txn, groupName, false)
	if err != nil {
		return err
	}

	// If there is no entity for the ID, do nothing
	if group == nil {
		return nil
	}

	// Delete the group using the same transaction
	err = i.MemDBDeleteGroupByNameInTxn(txn, group.Name)
	if err != nil {
		return err
	}

	// Delete the entity from storage
	err = i.groupPacker.DeleteItem(group.ID)
	if err != nil {
		return err
	}

	// Committing the transaction *after* successfully deleting group
	txn.Commit()

	return nil
}

func (i *IdentityStore) MemDBDeleteGroupByNameInTxn(txn *memdb.Txn, groupName string) error {
	if groupName == "" {
		return nil
	}

	if txn == nil {
		return fmt.Errorf("txn is nil")
	}

	group, err := i.MemDBGroupByNameInTxn(txn, groupName, false)
	if err != nil {
		return err
	}

	if group == nil {
		return nil
	}

	err = txn.Delete(groupsTable, group)
	if err != nil {
		return fmt.Errorf("failed to delete group from memdb: %v", err)
	}

	return nil
}

func (i *IdentityStore) MemDBGroupByIDInTxn(txn *memdb.Txn, groupID string, clone bool) (*identity.Group, error) {
	if groupID == "" {
		return nil, fmt.Errorf("missing group ID")
	}

	if txn == nil {
		return nil, fmt.Errorf("txn is nil")
	}

	groupRaw, err := txn.First(groupsTable, "id", groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch group from memdb using group ID: %v", err)
	}

	if groupRaw == nil {
		return nil, nil
	}

	group, ok := groupRaw.(*identity.Group)
	if !ok {
		return nil, fmt.Errorf("failed to declare the type of fetched group")
	}

	if clone {
		return group.Clone()
	}

	return group, nil
}

func (i *IdentityStore) MemDBGroupByID(groupID string, clone bool) (*identity.Group, error) {
	if groupID == "" {
		return nil, fmt.Errorf("missing group ID")
	}

	txn := i.db.Txn(false)

	return i.MemDBGroupByIDInTxn(txn, groupID, clone)
}

func (i *IdentityStore) MemDBGroupsByPolicyInTxn(txn *memdb.Txn, policyName string, clone bool) ([]*identity.Group, error) {
	if policyName == "" {
		return nil, fmt.Errorf("missing policy name")
	}

	groupsIter, err := txn.Get(groupsTable, "policies", policyName)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup groups using policy name: %v", err)
	}

	var groups []*identity.Group
	for group := groupsIter.Next(); group != nil; group = groupsIter.Next() {
		entry := group.(*identity.Group)
		if clone {
			entry, err = entry.Clone()
			if err != nil {
				return nil, err
			}
		}
		groups = append(groups, entry)
	}

	return groups, nil
}

func (i *IdentityStore) MemDBGroupsByPolicy(policyName string, clone bool) ([]*identity.Group, error) {
	if policyName == "" {
		return nil, fmt.Errorf("missing policy name")
	}

	txn := i.db.Txn(false)

	return i.MemDBGroupsByPolicyInTxn(txn, policyName, clone)
}

func (i *IdentityStore) MemDBGroupsByParentGroupIDInTxn(txn *memdb.Txn, memberGroupID string, clone bool) ([]*identity.Group, error) {
	if memberGroupID == "" {
		return nil, fmt.Errorf("missing member group ID")
	}

	groupsIter, err := txn.Get(groupsTable, "parent_group_ids", memberGroupID)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup groups using member group ID: %v", err)
	}

	var groups []*identity.Group
	for group := groupsIter.Next(); group != nil; group = groupsIter.Next() {
		entry := group.(*identity.Group)
		if clone {
			entry, err = entry.Clone()
			if err != nil {
				return nil, err
			}
		}
		groups = append(groups, entry)
	}

	return groups, nil
}

func (i *IdentityStore) MemDBGroupsByParentGroupID(memberGroupID string, clone bool) ([]*identity.Group, error) {
	if memberGroupID == "" {
		return nil, fmt.Errorf("missing member group ID")
	}

	txn := i.db.Txn(false)

	return i.MemDBGroupsByParentGroupIDInTxn(txn, memberGroupID, clone)
}

func (i *IdentityStore) MemDBGroupsByMemberEntityID(entityID string, clone bool, externalOnly bool) ([]*identity.Group, error) {
	txn := i.db.Txn(false)
	defer txn.Abort()

	return i.MemDBGroupsByMemberEntityIDInTxn(txn, entityID, clone, externalOnly)
}

func (i *IdentityStore) MemDBGroupsByMemberEntityIDInTxn(txn *memdb.Txn, entityID string, clone bool, externalOnly bool) ([]*identity.Group, error) {
	if entityID == "" {
		return nil, fmt.Errorf("missing entity ID")
	}

	groupsIter, err := txn.Get(groupsTable, "member_entity_ids", entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup groups using entity ID: %v", err)
	}

	var groups []*identity.Group
	for group := groupsIter.Next(); group != nil; group = groupsIter.Next() {
		entry := group.(*identity.Group)
		if externalOnly && entry.Type == groupTypeInternal {
			continue
		}
		if clone {
			entry, err = entry.Clone()
			if err != nil {
				return nil, err
			}
		}
		groups = append(groups, entry)
	}

	return groups, nil
}

func (i *IdentityStore) groupPoliciesByEntityID(entityID string) ([]string, error) {
	if entityID == "" {
		return nil, fmt.Errorf("empty entity ID")
	}

	groups, err := i.MemDBGroupsByMemberEntityID(entityID, false, false)
	if err != nil {
		return nil, err
	}

	visited := make(map[string]bool)
	var policies []string
	for _, group := range groups {
		groupPolicies, err := i.collectPoliciesReverseDFS(group, visited, nil)
		if err != nil {
			return nil, err
		}
		policies = append(policies, groupPolicies...)
	}

	return strutil.RemoveDuplicates(policies, false), nil
}

func (i *IdentityStore) groupsByEntityID(entityID string) ([]*identity.Group, []*identity.Group, error) {
	if entityID == "" {
		return nil, nil, fmt.Errorf("empty entity ID")
	}

	groups, err := i.MemDBGroupsByMemberEntityID(entityID, true, false)
	if err != nil {
		return nil, nil, err
	}

	visited := make(map[string]bool)
	var tGroups []*identity.Group
	for _, group := range groups {
		gGroups, err := i.collectGroupsReverseDFS(group, visited, nil)
		if err != nil {
			return nil, nil, err
		}
		tGroups = append(tGroups, gGroups...)
	}

	// Remove duplicates
	groupMap := make(map[string]*identity.Group)
	for _, group := range tGroups {
		groupMap[group.ID] = group
	}

	tGroups = make([]*identity.Group, 0, len(groupMap))
	for _, group := range groupMap {
		tGroups = append(tGroups, group)
	}

	diff := diffGroups(groups, tGroups)

	// For sanity
	// There should not be any group that gets deleted
	if len(diff.Deleted) != 0 {
		return nil, nil, fmt.Errorf("failed to diff group memberships")
	}

	return diff.Unmodified, diff.New, nil
}

func (i *IdentityStore) collectGroupsReverseDFS(group *identity.Group, visited map[string]bool, groups []*identity.Group) ([]*identity.Group, error) {
	if group == nil {
		return nil, fmt.Errorf("nil group")
	}

	// If traversal for a groupID is performed before, skip it
	if visited[group.ID] {
		return groups, nil
	}
	visited[group.ID] = true

	groups = append(groups, group)

	// Traverse all the parent groups
	for _, parentGroupID := range group.ParentGroupIDs {
		parentGroup, err := i.MemDBGroupByID(parentGroupID, false)
		if err != nil {
			return nil, err
		}
		pGroups, err := i.collectGroupsReverseDFS(parentGroup, visited, groups)
		if err != nil {
			return nil, fmt.Errorf("failed to collect group at parent group ID %q", parentGroup.ID)
		}
		groups = append(groups, pGroups...)
	}

	return groups, nil
}

func (i *IdentityStore) collectPoliciesReverseDFS(group *identity.Group, visited map[string]bool, policies []string) ([]string, error) {
	if group == nil {
		return nil, fmt.Errorf("nil group")
	}

	// If traversal for a groupID is performed before, skip it
	if visited[group.ID] {
		return policies, nil
	}
	visited[group.ID] = true

	policies = append(policies, group.Policies...)

	// Traverse all the parent groups
	for _, parentGroupID := range group.ParentGroupIDs {
		parentGroup, err := i.MemDBGroupByID(parentGroupID, false)
		if err != nil {
			return nil, err
		}
		parentPolicies, err := i.collectPoliciesReverseDFS(parentGroup, visited, policies)
		if err != nil {
			return nil, fmt.Errorf("failed to collect policies at parent group ID %q", parentGroup.ID)
		}
		policies = append(policies, parentPolicies...)
	}

	return strutil.RemoveDuplicates(policies, false), nil
}

func (i *IdentityStore) detectCycleDFS(visited map[string]bool, startingGroupID, groupID string) (bool, error) {
	// If the traversal reaches the startingGroupID, a loop is detected
	if startingGroupID == groupID {
		return true, nil
	}

	// If traversal for a groupID is performed before, skip it
	if visited[groupID] {
		return false, nil
	}
	visited[groupID] = true

	group, err := i.MemDBGroupByID(groupID, true)
	if err != nil {
		return false, err
	}
	if group == nil {
		return false, nil
	}

	// Fetch all groups in which groupID is present as a ParentGroupID. In
	// other words, find all the subgroups of groupID.
	memberGroups, err := i.MemDBGroupsByParentGroupID(groupID, false)
	if err != nil {
		return false, err
	}

	// DFS traverse the member groups
	for _, memberGroup := range memberGroups {
		cycleDetected, err := i.detectCycleDFS(visited, startingGroupID, memberGroup.ID)
		if err != nil {
			return false, fmt.Errorf("failed to perform cycle detection at member group ID %q", memberGroup.ID)
		}
		if cycleDetected {
			return true, fmt.Errorf("cycle detected at member group ID %q", memberGroup.ID)
		}
	}

	return false, nil
}

func (i *IdentityStore) memberGroupIDsByID(groupID string) ([]string, error) {
	var memberGroupIDs []string
	memberGroups, err := i.MemDBGroupsByParentGroupID(groupID, false)
	if err != nil {
		return nil, err
	}
	for _, memberGroup := range memberGroups {
		memberGroupIDs = append(memberGroupIDs, memberGroup.ID)
	}
	return memberGroupIDs, nil
}

func (i *IdentityStore) MemDBGroupIterator(ws memdb.WatchSet) (memdb.ResultIterator, error) {
	txn := i.db.Txn(false)

	iter, err := txn.Get(groupsTable, "id")
	if err != nil {
		return nil, err
	}

	ws.Add(iter.WatchCh())

	return iter, nil
}

func (i *IdentityStore) generateName(entryType string) (string, error) {
	var name string
OUTER:
	for {
		randBytes, err := uuid.GenerateRandomBytes(4)
		if err != nil {
			return "", err
		}
		name = fmt.Sprintf("%s_%s", entryType, fmt.Sprintf("%08x", randBytes[0:4]))

		switch entryType {
		case "entity":
			entity, err := i.MemDBEntityByName(name, false)
			if err != nil {
				return "", err
			}
			if entity == nil {
				break OUTER
			}
		case "group":
			group, err := i.MemDBGroupByName(name, false)
			if err != nil {
				return "", err
			}
			if group == nil {
				break OUTER
			}
		default:
			return "", fmt.Errorf("unrecognized type %q", entryType)
		}
	}

	return name, nil
}

func (i *IdentityStore) MemDBGroupsByBucketEntryKeyHash(hashValue string) ([]*identity.Group, error) {
	if hashValue == "" {
		return nil, fmt.Errorf("empty hash value")
	}

	txn := i.db.Txn(false)
	defer txn.Abort()

	return i.MemDBGroupsByBucketEntryKeyHashInTxn(txn, hashValue)
}

func (i *IdentityStore) MemDBGroupsByBucketEntryKeyHashInTxn(txn *memdb.Txn, hashValue string) ([]*identity.Group, error) {
	if txn == nil {
		return nil, fmt.Errorf("nil txn")
	}

	if hashValue == "" {
		return nil, fmt.Errorf("empty hash value")
	}

	groupsIter, err := txn.Get(groupsTable, "bucket_key_hash", hashValue)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup groups using bucket entry key hash: %v", err)
	}

	var groups []*identity.Group
	for group := groupsIter.Next(); group != nil; group = groupsIter.Next() {
		groups = append(groups, group.(*identity.Group))
	}

	return groups, nil
}

func (i *IdentityStore) MemDBGroupByAliasIDInTxn(txn *memdb.Txn, aliasID string, clone bool) (*identity.Group, error) {
	if aliasID == "" {
		return nil, fmt.Errorf("missing alias ID")
	}

	if txn == nil {
		return nil, fmt.Errorf("txn is nil")
	}

	alias, err := i.MemDBAliasByIDInTxn(txn, aliasID, false, true)
	if err != nil {
		return nil, err
	}

	if alias == nil {
		return nil, nil
	}

	return i.MemDBGroupByIDInTxn(txn, alias.CanonicalID, clone)
}

func (i *IdentityStore) MemDBGroupByAliasID(aliasID string, clone bool) (*identity.Group, error) {
	if aliasID == "" {
		return nil, fmt.Errorf("missing alias ID")
	}

	txn := i.db.Txn(false)

	return i.MemDBGroupByAliasIDInTxn(txn, aliasID, clone)
}

func (i *IdentityStore) deleteGroupAlias(aliasID string) error {
	if aliasID == "" {
		return fmt.Errorf("missing alias ID")
	}

	i.groupLock.Lock()
	defer i.groupLock.Unlock()

	txn := i.db.Txn(true)
	defer txn.Abort()

	alias, err := i.MemDBAliasByIDInTxn(txn, aliasID, false, true)
	if err != nil {
		return err
	}

	if alias == nil {
		return nil
	}

	group, err := i.MemDBGroupByAliasIDInTxn(txn, alias.ID, true)
	if err != nil {
		return err
	}

	// If there is no group tied to a valid alias, something is wrong
	if group == nil {
		return fmt.Errorf("alias not associated to a group")
	}

	// Delete group alias in memdb
	err = i.MemDBDeleteAliasByIDInTxn(txn, group.Alias.ID, true)
	if err != nil {
		return err
	}

	// Delete the alias
	group.Alias = nil

	err = i.upsertGroupInTxn(txn, group, true)
	if err != nil {
		return err
	}

	txn.Commit()

	return nil
}

func (i *IdentityStore) refreshExternalGroupMembershipsByEntityID(entityID string, groupAliases []*logical.Alias) error {
	if entityID == "" {
		return fmt.Errorf("empty entity ID")
	}

	i.groupLock.Lock()
	defer i.groupLock.Unlock()

	txn := i.db.Txn(true)
	defer txn.Abort()

	oldGroups, err := i.MemDBGroupsByMemberEntityIDInTxn(txn, entityID, true, true)
	if err != nil {
		return err
	}

	var newGroups []*identity.Group
	for _, alias := range groupAliases {
		aliasByFactors, err := i.MemDBAliasByFactors(alias.MountAccessor, alias.Name, true, true)
		if err != nil {
			return err
		}
		if aliasByFactors == nil {
			continue
		}
		mappingGroup, err := i.MemDBGroupByAliasID(aliasByFactors.ID, true)
		if err != nil {
			return err
		}
		if mappingGroup == nil {
			return fmt.Errorf("group unavailable for a valid alias ID %q", aliasByFactors.ID)
		}
		newGroups = append(newGroups, mappingGroup)
	}

	diff := diffGroups(oldGroups, newGroups)

	// Add the entity ID to all the new groups
	for _, group := range diff.New {
		if group.Type != groupTypeExternal {
			continue
		}

		i.logger.Debug("adding member entity ID to external group", "member_entity_id", entityID, "group_id", group.ID)

		group.MemberEntityIDs = append(group.MemberEntityIDs, entityID)

		err = i.upsertGroupInTxn(txn, group, true)
		if err != nil {
			return err
		}
	}

	// Remove the entity ID from all the deleted groups
	for _, group := range diff.Deleted {
		if group.Type != groupTypeExternal {
			continue
		}

		i.logger.Debug("removing member entity ID from external group", "member_entity_id", entityID, "group_id", group.ID)

		group.MemberEntityIDs = strutil.StrListDelete(group.MemberEntityIDs, entityID)

		err = i.upsertGroupInTxn(txn, group, true)
		if err != nil {
			return err
		}
	}

	txn.Commit()

	return nil
}

// diffGroups is used to diff two sets of groups
func diffGroups(old, new []*identity.Group) *groupDiff {
	diff := &groupDiff{}

	existing := make(map[string]*identity.Group)
	for _, group := range old {
		existing[group.ID] = group
	}

	for _, group := range new {
		// Check if the entry in new is present in the old
		_, ok := existing[group.ID]

		// If its not present, then its a new entry
		if !ok {
			diff.New = append(diff.New, group)
			continue
		}

		// If its present, it means that its unmodified
		diff.Unmodified = append(diff.Unmodified, group)

		// By deleting the unmodified from the old set, we could determine the
		// ones that are stale by looking at the remaining ones.
		delete(existing, group.ID)
	}

	// Any remaining entries must have been deleted
	for _, me := range existing {
		diff.Deleted = append(diff.Deleted, me)
	}

	return diff
}
