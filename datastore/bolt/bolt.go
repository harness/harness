package bolt

import (
	"errors"

	"github.com/boltdb/bolt"

	"github.com/drone/drone/datastore"
)

var (
	ErrKeyNotFound = errors.New("Key not found")
	ErrKeyExists   = errors.New("Key exists")
)

var (
	bucketUser        = []byte("user")
	bucketUserRepos   = []byte("user_repos")
	bucketUserTokens  = []byte("user_tokens")
	bucketTokens      = []byte("token")
	bucketRepo        = []byte("repo")
	bucketRepoKeys    = []byte("repo_keys")
	bucketRepoParams  = []byte("repo_params")
	bucketRepoUsers   = []byte("repo_users")
	bucketBuild       = []byte("build")
	bucketBuildAgent  = []byte("build_agents")
	bucketBuildStatus = []byte("build_status")
	bucketBuildLogs   = []byte("build_logs")
	bucketBuildSeq    = []byte("build_seq")
)

type DB struct {
	*bolt.DB
}

func New(path string) (*DB, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	// Initialize all the required buckets.
	db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists(bucketUser)
		tx.CreateBucketIfNotExists(bucketUserRepos)
		tx.CreateBucketIfNotExists(bucketUserTokens)
		tx.CreateBucketIfNotExists(bucketTokens)
		tx.CreateBucketIfNotExists(bucketRepo)
		tx.CreateBucketIfNotExists(bucketRepoKeys)
		tx.CreateBucketIfNotExists(bucketRepoParams)
		tx.CreateBucketIfNotExists(bucketRepoUsers)
		tx.CreateBucketIfNotExists(bucketBuild)
		tx.CreateBucketIfNotExists(bucketBuildAgent)
		tx.CreateBucketIfNotExists(bucketBuildStatus)
		tx.CreateBucketIfNotExists(bucketBuildLogs)
		tx.CreateBucketIfNotExists(bucketBuildSeq)
		return nil
	})

	// REMOVE BELOW
	var ds datastore.Datastore
	if ds == nil {
		ds = &DB{db}
	}
	// REMOVE ABOVE

	return &DB{db}, nil
}

func Must(path string) *DB {
	db, err := New(path)
	if err != nil {
		panic(err)
	}
	return db
}
