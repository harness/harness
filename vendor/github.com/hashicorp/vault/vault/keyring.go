package vault

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/vault/helper/jsonutil"
)

// Keyring is used to manage multiple encryption keys used by
// the barrier. New keys can be installed and each has a sequential term.
// The term used to encrypt a key is prefixed to the key written out.
// All data is encrypted with the latest key, but storing the old keys
// allows for decryption of keys written previously. Along with the encryption
// keys, the keyring also tracks the master key. This is necessary so that
// when a new key is added to the keyring, we can encrypt with the master key
// and write out the new keyring.
type Keyring struct {
	masterKey  []byte
	keys       map[uint32]*Key
	activeTerm uint32
}

// EncodedKeyring is used for serialization of the keyring
type EncodedKeyring struct {
	MasterKey []byte
	Keys      []*Key
}

// Key represents a single term, along with the key used.
type Key struct {
	Term        uint32
	Version     int
	Value       []byte
	InstallTime time.Time
}

// Serialize is used to create a byte encoded key
func (k *Key) Serialize() ([]byte, error) {
	return json.Marshal(k)
}

// DeserializeKey is used to deserialize and return a new key
func DeserializeKey(buf []byte) (*Key, error) {
	k := new(Key)
	if err := jsonutil.DecodeJSON(buf, k); err != nil {
		return nil, fmt.Errorf("deserialization failed: %v", err)
	}
	return k, nil
}

// NewKeyring creates a new keyring
func NewKeyring() *Keyring {
	k := &Keyring{
		keys:       make(map[uint32]*Key),
		activeTerm: 0,
	}
	return k
}

// Clone returns a new copy of the keyring
func (k *Keyring) Clone() *Keyring {
	clone := &Keyring{
		masterKey:  k.masterKey,
		keys:       make(map[uint32]*Key, len(k.keys)),
		activeTerm: k.activeTerm,
	}
	for idx, key := range k.keys {
		clone.keys[idx] = key
	}
	return clone
}

// AddKey adds a new key to the keyring
func (k *Keyring) AddKey(key *Key) (*Keyring, error) {
	// Ensure there is no conflict
	if exist, ok := k.keys[key.Term]; ok {
		if !bytes.Equal(key.Value, exist.Value) {
			return nil, fmt.Errorf("Conflicting key for term %d already installed", key.Term)
		}
		return k, nil
	}

	// Add a time if none
	if key.InstallTime.IsZero() {
		key.InstallTime = time.Now()
	}

	// Make a new keyring
	clone := k.Clone()

	// Install the new key
	clone.keys[key.Term] = key

	// Update the active term if newer
	if key.Term > clone.activeTerm {
		clone.activeTerm = key.Term
	}
	return clone, nil
}

// RemoveKey removes a key from the keyring
func (k *Keyring) RemoveKey(term uint32) (*Keyring, error) {
	// Ensure this is not the active key
	if term == k.activeTerm {
		return nil, fmt.Errorf("Cannot remove active key")
	}

	// Check if this term does not exist
	if _, ok := k.keys[term]; !ok {
		return k, nil
	}

	// Delete the key
	clone := k.Clone()
	delete(clone.keys, term)
	return clone, nil
}

// ActiveTerm returns the currently active term
func (k *Keyring) ActiveTerm() uint32 {
	return k.activeTerm
}

// ActiveKey returns the active encryption key, or nil
func (k *Keyring) ActiveKey() *Key {
	return k.keys[k.activeTerm]
}

// TermKey returns the key for the given term, or nil
func (k *Keyring) TermKey(term uint32) *Key {
	return k.keys[term]
}

// SetMasterKey is used to update the master key
func (k *Keyring) SetMasterKey(val []byte) *Keyring {
	valCopy := make([]byte, len(val))
	copy(valCopy, val)
	clone := k.Clone()
	clone.masterKey = valCopy
	return clone
}

// MasterKey returns the master key
func (k *Keyring) MasterKey() []byte {
	return k.masterKey
}

// Serialize is used to create a byte encoded keyring
func (k *Keyring) Serialize() ([]byte, error) {
	// Create the encoded entry
	enc := EncodedKeyring{
		MasterKey: k.masterKey,
	}
	for _, key := range k.keys {
		enc.Keys = append(enc.Keys, key)
	}

	// JSON encode the keyring
	buf, err := json.Marshal(enc)
	return buf, err
}

// DeserializeKeyring is used to deserialize and return a new keyring
func DeserializeKeyring(buf []byte) (*Keyring, error) {
	// Deserialize the keyring
	var enc EncodedKeyring
	if err := jsonutil.DecodeJSON(buf, &enc); err != nil {
		return nil, fmt.Errorf("deserialization failed: %v", err)
	}

	// Create a new keyring
	k := NewKeyring()
	k.masterKey = enc.MasterKey
	for _, key := range enc.Keys {
		k.keys[key.Term] = key
		if key.Term > k.activeTerm {
			k.activeTerm = key.Term
		}
	}
	return k, nil
}

// N.B.:
// Since Go 1.5 these are not reliable; see the documentation around the memzero
// function. These are best-effort.
func (k *Keyring) Zeroize(keysToo bool) {
	if k == nil {
		return
	}
	if k.masterKey != nil {
		memzero(k.masterKey)
	}
	if !keysToo || k.keys == nil {
		return
	}
	for _, key := range k.keys {
		memzero(key.Value)
	}
}
