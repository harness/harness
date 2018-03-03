package vault

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/hashicorp/vault/helper/jsonutil"
	"github.com/hashicorp/vault/physical"

	"github.com/keybase/go-crypto/openpgp"
	"github.com/keybase/go-crypto/openpgp/packet"
)

const (
	// barrierSealConfigPath is the path used to store our seal configuration.
	// This value is stored in plaintext, since we must be able to read it even
	// with the Vault sealed. This is required so that we know how many secret
	// parts must be used to reconstruct the master key.
	barrierSealConfigPath = "core/seal-config"

	// recoverySealConfigPath is the path to the recovery key seal
	// configuration. It lives inside the barrier.
	// DEPRECATED: Use recoverySealConfigPlaintextPath instead.
	recoverySealConfigPath = "core/recovery-seal-config"

	// recoverySealConfigPlaintextPath is the path to the recovery key seal
	// configuration. This is stored in plaintext so that we can perform
	// auto-unseal.
	recoverySealConfigPlaintextPath = "core/recovery-config"

	// recoveryKeyPath is the path to the recovery key
	recoveryKeyPath = "core/recovery-key"

	// storedBarrierKeysPath is the path used for storing HSM-encrypted unseal keys
	storedBarrierKeysPath = "core/hsm/barrier-unseal-keys"

	// hsmStoredIVPath is the path to the initialization vector for stored keys
	hsmStoredIVPath = "core/hsm/iv"
)

const (
	SealTypeShamir = "shamir"
	SealTypePKCS11 = "pkcs11"
	SealTypeAWSKMS = "awskms"
	SealTypeTest   = "test-auto"

	RecoveryTypeUnsupported = "unsupported"
	RecoveryTypeShamir      = "shamir"
)

type KeyNotFoundError struct {
	Err error
}

func (e *KeyNotFoundError) WrappedErrors() []error {
	return []error{e.Err}
}

func (e *KeyNotFoundError) Error() string {
	return e.Err.Error()
}

type Seal interface {
	SetCore(*Core)
	Init(context.Context) error
	Finalize(context.Context) error

	StoredKeysSupported() bool
	SetStoredKeys(context.Context, [][]byte) error
	GetStoredKeys(context.Context) ([][]byte, error)

	BarrierType() string
	BarrierConfig(context.Context) (*SealConfig, error)
	SetBarrierConfig(context.Context, *SealConfig) error

	RecoveryKeySupported() bool
	RecoveryType() string
	RecoveryConfig(context.Context) (*SealConfig, error)
	SetRecoveryConfig(context.Context, *SealConfig) error
	SetRecoveryKey(context.Context, []byte) error
	VerifyRecoveryKey(context.Context, []byte) error
}

type defaultSeal struct {
	config atomic.Value
	core   *Core
}

func NewDefaultSeal() Seal {
	ret := &defaultSeal{}
	ret.config.Store((*SealConfig)(nil))
	return ret
}

func (d *defaultSeal) checkCore() error {
	if d.core == nil {
		return fmt.Errorf("seal does not have a core set")
	}
	return nil
}

func (d *defaultSeal) SetCore(core *Core) {
	d.core = core
}

func (d *defaultSeal) Init(ctx context.Context) error {
	return nil
}

func (d *defaultSeal) Finalize(ctx context.Context) error {
	return nil
}

func (d *defaultSeal) BarrierType() string {
	return SealTypeShamir
}

func (d *defaultSeal) StoredKeysSupported() bool {
	return false
}

func (d *defaultSeal) RecoveryKeySupported() bool {
	return false
}

func (d *defaultSeal) SetStoredKeys(ctx context.Context, keys [][]byte) error {
	return fmt.Errorf("core: stored keys are not supported")
}

func (d *defaultSeal) GetStoredKeys(ctx context.Context) ([][]byte, error) {
	return nil, fmt.Errorf("core: stored keys are not supported")
}

func (d *defaultSeal) BarrierConfig(ctx context.Context) (*SealConfig, error) {
	if d.config.Load().(*SealConfig) != nil {
		return d.config.Load().(*SealConfig).Clone(), nil
	}

	if err := d.checkCore(); err != nil {
		return nil, err
	}

	// Fetch the core configuration
	pe, err := d.core.physical.Get(ctx, barrierSealConfigPath)
	if err != nil {
		d.core.logger.Error("core: failed to read seal configuration", "error", err)
		return nil, fmt.Errorf("failed to check seal configuration: %v", err)
	}

	// If the seal configuration is missing, we are not initialized
	if pe == nil {
		d.core.logger.Info("core: seal configuration missing, not initialized")
		return nil, nil
	}

	var conf SealConfig

	// Decode the barrier entry
	if err := jsonutil.DecodeJSON(pe.Value, &conf); err != nil {
		d.core.logger.Error("core: failed to decode seal configuration", "error", err)
		return nil, fmt.Errorf("failed to decode seal configuration: %v", err)
	}

	switch conf.Type {
	// This case should not be valid for other types as only this is the default
	case "":
		conf.Type = d.BarrierType()
	case d.BarrierType():
	default:
		d.core.logger.Error("core: barrier seal type does not match loaded type", "barrier_seal_type", conf.Type, "loaded_seal_type", d.BarrierType())
		return nil, fmt.Errorf("barrier seal type of %s does not match loaded type of %s", conf.Type, d.BarrierType())
	}

	// Check for a valid seal configuration
	if err := conf.Validate(); err != nil {
		d.core.logger.Error("core: invalid seal configuration", "error", err)
		return nil, fmt.Errorf("seal validation failed: %v", err)
	}

	d.config.Store(&conf)
	return conf.Clone(), nil
}

func (d *defaultSeal) SetBarrierConfig(ctx context.Context, config *SealConfig) error {
	if err := d.checkCore(); err != nil {
		return err
	}

	// Provide a way to wipe out the cached value (also prevents actually
	// saving a nil config)
	if config == nil {
		d.config.Store((*SealConfig)(nil))
		return nil
	}

	config.Type = d.BarrierType()

	// Encode the seal configuration
	buf, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to encode seal configuration: %v", err)
	}

	// Store the seal configuration
	pe := &physical.Entry{
		Key:   barrierSealConfigPath,
		Value: buf,
	}

	if err := d.core.physical.Put(ctx, pe); err != nil {
		d.core.logger.Error("core: failed to write seal configuration", "error", err)
		return fmt.Errorf("failed to write seal configuration: %v", err)
	}

	d.config.Store(config.Clone())

	return nil
}

func (d *defaultSeal) RecoveryType() string {
	return RecoveryTypeUnsupported
}

func (d *defaultSeal) RecoveryConfig(ctx context.Context) (*SealConfig, error) {
	return nil, fmt.Errorf("recovery not supported")
}

func (d *defaultSeal) SetRecoveryConfig(ctx context.Context, config *SealConfig) error {
	return fmt.Errorf("recovery not supported")
}

func (d *defaultSeal) VerifyRecoveryKey(context.Context, []byte) error {
	return fmt.Errorf("recovery not supported")
}

func (d *defaultSeal) SetRecoveryKey(ctx context.Context, key []byte) error {
	return fmt.Errorf("recovery not supported")
}

// SealConfig is used to describe the seal configuration
type SealConfig struct {
	// The type, for sanity checking
	Type string `json:"type"`

	// SecretShares is the number of shares the secret is split into. This is
	// the N value of Shamir.
	SecretShares int `json:"secret_shares"`

	// SecretThreshold is the number of parts required to open the vault. This
	// is the T value of Shamir.
	SecretThreshold int `json:"secret_threshold"`

	// PGPKeys is the array of public PGP keys used, if requested, to encrypt
	// the output unseal tokens. If provided, it sets the value of
	// SecretShares. Ordering is important.
	PGPKeys []string `json:"pgp_keys"`

	// Nonce is a nonce generated by Vault used to ensure that when unseal keys
	// are submitted for a rekey operation, the rekey operation itself is the
	// one intended. This prevents hijacking of the rekey operation, since it
	// is unauthenticated.
	Nonce string `json:"nonce"`

	// Backup indicates whether or not a backup of PGP-encrypted unseal keys
	// should be stored at coreUnsealKeysBackupPath after successful rekeying.
	Backup bool `json:"backup"`

	// How many keys to store, for seals that support storage.
	StoredShares int `json:"stored_shares"`
}

// Validate is used to sanity check the seal configuration
func (s *SealConfig) Validate() error {
	if s.SecretShares < 1 {
		return fmt.Errorf("shares must be at least one")
	}
	if s.SecretThreshold < 1 {
		return fmt.Errorf("threshold must be at least one")
	}
	if s.SecretShares > 1 && s.SecretThreshold == 1 {
		return fmt.Errorf("threshold must be greater than one for multiple shares")
	}
	if s.SecretShares > 255 {
		return fmt.Errorf("shares must be less than 256")
	}
	if s.SecretThreshold > 255 {
		return fmt.Errorf("threshold must be less than 256")
	}
	if s.SecretThreshold > s.SecretShares {
		return fmt.Errorf("threshold cannot be larger than shares")
	}
	if s.StoredShares > s.SecretShares {
		return fmt.Errorf("stored keys cannot be larger than shares")
	}
	if len(s.PGPKeys) > 0 && len(s.PGPKeys) != s.SecretShares-s.StoredShares {
		return fmt.Errorf("count mismatch between number of provided PGP keys and number of shares")
	}
	if len(s.PGPKeys) > 0 {
		for _, keystring := range s.PGPKeys {
			data, err := base64.StdEncoding.DecodeString(keystring)
			if err != nil {
				return fmt.Errorf("Error decoding given PGP key: %s", err)
			}
			_, err = openpgp.ReadEntity(packet.NewReader(bytes.NewBuffer(data)))
			if err != nil {
				return fmt.Errorf("Error parsing given PGP key: %s", err)
			}
		}
	}
	return nil
}

func (s *SealConfig) Clone() *SealConfig {
	ret := &SealConfig{
		Type:            s.Type,
		SecretShares:    s.SecretShares,
		SecretThreshold: s.SecretThreshold,
		Nonce:           s.Nonce,
		Backup:          s.Backup,
		StoredShares:    s.StoredShares,
	}
	if len(s.PGPKeys) > 0 {
		ret.PGPKeys = make([]string, len(s.PGPKeys))
		copy(ret.PGPKeys, s.PGPKeys)
	}
	return ret
}
