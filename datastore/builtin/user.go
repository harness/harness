package builtin

import (
	"time"

	"github.com/boltdb/bolt"
	"github.com/drone/drone/common"
)

// User returns a user by user login.
func (db *DB) User(login string) (*common.User, error) {
	user := &common.User{}
	key := []byte(login)

	err := db.View(func(t *bolt.Tx) error {
		return get(t, bucketUser, key, user)
	})

	return user, err
}

// UserCount returns a count of all registered users.
func (db *DB) UserCount() (int, error) {
	var out int
	var err = db.View(func(t *bolt.Tx) error {
		out = t.Bucket(bucketUser).Stats().KeyN
		return nil
	})
	return out, err
}

// UserList returns a list of all registered users.
func (db *DB) UserList() ([]*common.User, error) {
	users := []*common.User{}
	err := db.View(func(t *bolt.Tx) error {
		return t.Bucket(bucketUser).ForEach(func(key, raw []byte) error {
			user := &common.User{}
			err := decode(raw, user)
			if err != nil {
				return err
			}
			users = append(users, user)
			return nil
		})
	})
	return users, err
}

// SetUser inserts or updates a user.
func (db *DB) SetUser(user *common.User) error {
	key := []byte(user.Login)
	user.Updated = time.Now().UTC().Unix()

	return db.Update(func(t *bolt.Tx) error {
		return update(t, bucketUser, key, user)
	})
}

// SetUserNotExists inserts a new user into the datastore.
// If the user login already exists ErrConflict is returned.
func (db *DB) SetUserNotExists(user *common.User) error {
	key := []byte(user.Login)
	user.Created = time.Now().UTC().Unix()
	user.Updated = time.Now().UTC().Unix()

	return db.Update(func(t *bolt.Tx) error {
		return insert(t, bucketUser, key, user)
	})
}

// DelUser deletes the user.
func (db *DB) DelUser(user *common.User) error {
	key := []byte(user.Login)
	return db.Update(func(t *bolt.Tx) error {
		err := delete(t, bucketUserTokens, key)
		if err != nil {
			return err
		}
		err = delete(t, bucketUserRepos, key)
		if err != nil {
			return err
		}
		// IDEA: deleteKeys(t, bucketTokens, keys)
		deleteWithPrefix(t, bucketTokens, append(key, '/'))
		return delete(t, bucketUser, key)
	})
}
