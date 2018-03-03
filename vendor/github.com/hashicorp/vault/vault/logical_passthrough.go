package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/helper/jsonutil"
	"github.com/hashicorp/vault/helper/parseutil"
	"github.com/hashicorp/vault/helper/wrapping"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

// PassthroughBackendFactory returns a PassthroughBackend
// with leases switched off
func PassthroughBackendFactory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	return LeaseSwitchedPassthroughBackend(ctx, conf, false)
}

// LeasedPassthroughBackendFactory returns a PassthroughBackend
// with leases switched on
func LeasedPassthroughBackendFactory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	return LeaseSwitchedPassthroughBackend(ctx, conf, true)
}

// LeaseSwitchedPassthroughBackend returns a PassthroughBackend
// with leases switched on or off
func LeaseSwitchedPassthroughBackend(ctx context.Context, conf *logical.BackendConfig, leases bool) (logical.Backend, error) {
	var b PassthroughBackend
	b.generateLeases = leases
	b.Backend = &framework.Backend{
		Help: strings.TrimSpace(passthroughHelp),

		PathsSpecial: &logical.Paths{
			SealWrapStorage: []string{
				"/",
			},
		},

		Paths: []*framework.Path{
			&framework.Path{
				Pattern: ".*",

				Callbacks: map[logical.Operation]framework.OperationFunc{
					logical.ReadOperation:   b.handleRead,
					logical.CreateOperation: b.handleWrite,
					logical.UpdateOperation: b.handleWrite,
					logical.DeleteOperation: b.handleDelete,
					logical.ListOperation:   b.handleList,
				},

				ExistenceCheck: b.handleExistenceCheck,

				HelpSynopsis:    strings.TrimSpace(passthroughHelpSynopsis),
				HelpDescription: strings.TrimSpace(passthroughHelpDescription),
			},
		},
	}

	b.Backend.Secrets = []*framework.Secret{
		&framework.Secret{
			Type: "kv",

			Renew:  b.handleRead,
			Revoke: b.handleRevoke,
		},
	}

	if conf == nil {
		return nil, fmt.Errorf("Configuation passed into backend is nil")
	}
	b.Backend.Setup(ctx, conf)

	return &b, nil
}

// PassthroughBackend is used storing secrets directly into the physical
// backend. The secrets are encrypted in the durable storage and custom TTL
// information can be specified, but otherwise this backend doesn't do anything
// fancy.
type PassthroughBackend struct {
	*framework.Backend
	generateLeases bool
}

func (b *PassthroughBackend) handleRevoke(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	// This is a no-op
	return nil, nil
}

func (b *PassthroughBackend) handleExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	out, err := req.Storage.Get(ctx, req.Path)
	if err != nil {
		return false, fmt.Errorf("existence check failed: %v", err)
	}

	return out != nil, nil
}

func (b *PassthroughBackend) handleRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	// Read the path
	out, err := req.Storage.Get(ctx, req.Path)
	if err != nil {
		return nil, fmt.Errorf("read failed: %v", err)
	}

	// Fast-path the no data case
	if out == nil {
		return nil, nil
	}

	// Decode the data
	var rawData map[string]interface{}

	if err := jsonutil.DecodeJSON(out.Value, &rawData); err != nil {
		return nil, fmt.Errorf("json decoding failed: %v", err)
	}

	var resp *logical.Response
	if b.generateLeases {
		// Generate the response
		resp = b.Secret("kv").Response(rawData, nil)
		resp.Secret.Renewable = false
	} else {
		resp = &logical.Response{
			Secret: &logical.Secret{},
			Data:   rawData,
		}
	}

	// Ensure seal wrapping is carried through if the response is
	// response-wrapped
	if out.SealWrap {
		if resp.WrapInfo == nil {
			resp.WrapInfo = &wrapping.ResponseWrapInfo{}
		}
		resp.WrapInfo.SealWrap = out.SealWrap
	}

	// Check if there is a ttl key
	ttlDuration := b.System().DefaultLeaseTTL()
	ttlRaw, ok := rawData["ttl"]
	if !ok {
		ttlRaw, ok = rawData["lease"]
	}
	if ok {
		dur, err := parseutil.ParseDurationSecond(ttlRaw)
		if err == nil {
			ttlDuration = dur
		}

		if b.generateLeases {
			resp.Secret.Renewable = true
		}
	}

	resp.Secret.TTL = ttlDuration

	return resp, nil
}

func (b *PassthroughBackend) GeneratesLeases() bool {
	return b.generateLeases
}

func (b *PassthroughBackend) handleWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	// Check that some fields are given
	if len(req.Data) == 0 {
		return logical.ErrorResponse("missing data fields"), nil
	}

	// JSON encode the data
	buf, err := json.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("json encoding failed: %v", err)
	}

	// Write out a new key
	entry := &logical.StorageEntry{
		Key:   req.Path,
		Value: buf,
	}
	if err := req.Storage.Put(ctx, entry); err != nil {
		return nil, fmt.Errorf("failed to write: %v", err)
	}

	return nil, nil
}

func (b *PassthroughBackend) handleDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	// Delete the key at the request path
	if err := req.Storage.Delete(ctx, req.Path); err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *PassthroughBackend) handleList(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	// Right now we only handle directories, so ensure it ends with /; however,
	// some physical backends may not handle the "/" case properly, so only add
	// it if we're not listing the root
	path := req.Path
	if path != "" && !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	// List the keys at the prefix given by the request
	keys, err := req.Storage.List(ctx, path)
	if err != nil {
		return nil, err
	}

	// Generate the response
	return logical.ListResponse(keys), nil
}

const passthroughHelp = `
The kv backend reads and writes arbitrary secrets to the backend.
The secrets are encrypted/decrypted by Vault: they are never stored
unencrypted in the backend and the backend never has an opportunity to
see the unencrypted value.

TTLs can be set on a per-secret basis. These TTLs will be sent down
when that secret is read, and it is assumed that some outside process will
revoke and/or replace the secret at that path.
`

const passthroughHelpSynopsis = `
Pass-through secret storage to the storage backend, allowing you to
read/write arbitrary data into secret storage.
`

const passthroughHelpDescription = `
The pass-through backend reads and writes arbitrary data into secret storage,
encrypting it along the way.

A TTL can be specified when writing with the "ttl" field. If given, the
duration of leases returned by this backend will be set to this value. This
can be used as a hint from the writer of a secret to the consumer of a secret
that the consumer should re-read the value before the TTL has expired.
However, any revocation must be handled by the user of this backend; the lease
duration does not affect the provided data in any way.
`
