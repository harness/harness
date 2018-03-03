package vault

import (
	"context"
	"errors"
	"time"

	"github.com/hashicorp/vault/logical"
)

var (
	// ErrBarrierSealed is returned if an operation is performed on
	// a sealed barrier. No operation is expected to succeed before unsealing
	ErrBarrierSealed = errors.New("Vault is sealed")

	// ErrBarrierAlreadyInit is returned if the barrier is already
	// initialized. This prevents a re-initialization.
	ErrBarrierAlreadyInit = errors.New("Vault is already initialized")

	// ErrBarrierNotInit is returned if a non-initialized barrier
	// is attempted to be unsealed.
	ErrBarrierNotInit = errors.New("Vault is not initialized")

	// ErrBarrierInvalidKey is returned if the Unseal key is invalid
	ErrBarrierInvalidKey = errors.New("Unseal failed, invalid key")
)

const (
	// barrierInitPath is the path used to store our init sentinel file
	barrierInitPath = "barrier/init"

	// keyringPath is the location of the keyring data. This is encrypted
	// by the master key.
	keyringPath   = "core/keyring"
	keyringPrefix = "core/"

	// keyringUpgradePrefix is the path used to store keyring update entries.
	// When running in HA mode, the active instance will install the new key
	// and re-write the keyring. For standby instances, they need an upgrade
	// path from key N to N+1. They cannot just use the master key because
	// in the event of a rekey, that master key can no longer decrypt the keyring.
	// When key N+1 is installed, we create an entry at "prefix/N" which uses
	// encryption key N to provide the N+1 key. The standby instances scan
	// for this periodically and refresh their keyring. The upgrade keys
	// are deleted after a few minutes, but this provides enough time for the
	// standby instances to upgrade without causing any disruption.
	keyringUpgradePrefix = "core/upgrade/"

	// masterKeyPath is the location of the master key. This is encrypted
	// by the latest key in the keyring. This is only used by standby instances
	// to handle the case of a rekey. If the active instance does a rekey,
	// the standby instances can no longer reload the keyring since they
	// have the old master key. This key can be decrypted if you have the
	// keyring to discover the new master key. The new master key is then
	// used to reload the keyring itself.
	masterKeyPath = "core/master"
)

// SecurityBarrier is a critical component of Vault. It is used to wrap
// an untrusted physical backend and provide a single point of encryption,
// decryption and checksum verification. The goal is to ensure that any
// data written to the barrier is confidential and that integrity is preserved.
// As a real-world analogy, this is the steel and concrete wrapper around
// a Vault. The barrier should only be Unlockable given its key.
type SecurityBarrier interface {
	// Initialized checks if the barrier has been initialized
	// and has a master key set.
	Initialized(ctx context.Context) (bool, error)

	// Initialize works only if the barrier has not been initialized
	// and makes use of the given master key.
	Initialize(context.Context, []byte) error

	// GenerateKey is used to generate a new key
	GenerateKey() ([]byte, error)

	// KeyLength is used to sanity check a key
	KeyLength() (int, int)

	// Sealed checks if the barrier has been unlocked yet. The Barrier
	// is not expected to be able to perform any CRUD until it is unsealed.
	Sealed() (bool, error)

	// Unseal is used to provide the master key which permits the barrier
	// to be unsealed. If the key is not correct, the barrier remains sealed.
	Unseal(ctx context.Context, key []byte) error

	// VerifyMaster is used to check if the given key matches the master key
	VerifyMaster(key []byte) error

	// SetMasterKey is used to directly set a new master key. This is used in
	// repliated scenarios due to the chicken and egg problem of reloading the
	// keyring from disk before we have the master key to decrypt it.
	SetMasterKey(key []byte) error

	// ReloadKeyring is used to re-read the underlying keyring.
	// This is used for HA deployments to ensure the latest keyring
	// is present in the leader.
	ReloadKeyring(ctx context.Context) error

	// ReloadMasterKey is used to re-read the underlying masterkey.
	// This is used for HA deployments to ensure the latest master key
	// is available for keyring reloading.
	ReloadMasterKey(ctx context.Context) error

	// Seal is used to re-seal the barrier. This requires the barrier to
	// be unsealed again to perform any further operations.
	Seal() error

	// Rotate is used to create a new encryption key. All future writes
	// should use the new key, while old values should still be decryptable.
	Rotate(ctx context.Context) (uint32, error)

	// CreateUpgrade creates an upgrade path key to the given term from the previous term
	CreateUpgrade(ctx context.Context, term uint32) error

	// DestroyUpgrade destroys the upgrade path key to the given term
	DestroyUpgrade(ctx context.Context, term uint32) error

	// CheckUpgrade looks for an upgrade to the current term and installs it
	CheckUpgrade(ctx context.Context) (bool, uint32, error)

	// ActiveKeyInfo is used to inform details about the active key
	ActiveKeyInfo() (*KeyInfo, error)

	// Rekey is used to change the master key used to protect the keyring
	Rekey(context.Context, []byte) error

	// For replication we must send over the keyring, so this must be available
	Keyring() (*Keyring, error)

	// SecurityBarrier must provide the storage APIs
	BarrierStorage

	// SecurityBarrier must provide the encryption APIs
	BarrierEncryptor
}

// BarrierStorage is the storage only interface required for a Barrier.
type BarrierStorage interface {
	// Put is used to insert or update an entry
	Put(ctx context.Context, entry *Entry) error

	// Get is used to fetch an entry
	Get(ctx context.Context, key string) (*Entry, error)

	// Delete is used to permanently delete an entry
	Delete(ctx context.Context, key string) error

	// List is used ot list all the keys under a given
	// prefix, up to the next prefix.
	List(ctx context.Context, prefix string) ([]string, error)
}

// BarrierEncryptor is the in memory only interface that does not actually
// use the underlying barrier. It is used for lower level modules like the
// Write-Ahead-Log and Merkle index to allow them to use the barrier.
type BarrierEncryptor interface {
	Encrypt(ctx context.Context, key string, plaintext []byte) ([]byte, error)
	Decrypt(ctx context.Context, key string, ciphertext []byte) ([]byte, error)
}

// Entry is used to represent data stored by the security barrier
type Entry struct {
	Key      string
	Value    []byte
	SealWrap bool
}

// Logical turns the Entry into a logical storage entry.
func (e *Entry) Logical() *logical.StorageEntry {
	return &logical.StorageEntry{
		Key:      e.Key,
		Value:    e.Value,
		SealWrap: e.SealWrap,
	}
}

// KeyInfo is used to convey information about the encryption key
type KeyInfo struct {
	Term        int
	InstallTime time.Time
}
