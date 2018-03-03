package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/helper/jsonutil"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

// CubbyholeBackendFactory constructs a new cubbyhole backend
func CubbyholeBackendFactory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	var b CubbyholeBackend
	b.Backend = &framework.Backend{
		Help: strings.TrimSpace(cubbyholeHelp),

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

				HelpSynopsis:    strings.TrimSpace(cubbyholeHelpSynopsis),
				HelpDescription: strings.TrimSpace(cubbyholeHelpDescription),
			},
		},
	}

	if conf == nil {
		return nil, fmt.Errorf("Configuation passed into backend is nil")
	}
	b.Backend.Setup(ctx, conf)

	return &b, nil
}

// CubbyholeBackend is used for storing secrets directly into the physical
// backend. The secrets are encrypted in the durable storage.
// This differs from kv in that every token has its own private
// storage view. The view is removed when the token expires.
type CubbyholeBackend struct {
	*framework.Backend

	saltUUID    string
	storageView logical.Storage
}

func (b *CubbyholeBackend) revoke(ctx context.Context, saltedToken string) error {
	if saltedToken == "" {
		return fmt.Errorf("cubbyhole: client token empty during revocation")
	}

	if err := logical.ClearView(ctx, b.storageView.(*BarrierView).SubView(saltedToken+"/")); err != nil {
		return err
	}

	return nil
}

func (b *CubbyholeBackend) handleExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	out, err := req.Storage.Get(ctx, req.ClientToken+"/"+req.Path)
	if err != nil {
		return false, fmt.Errorf("existence check failed: %v", err)
	}

	return out != nil, nil
}

func (b *CubbyholeBackend) handleRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	if req.ClientToken == "" {
		return nil, fmt.Errorf("cubbyhole read: client token empty")
	}

	// Read the path
	out, err := req.Storage.Get(ctx, req.ClientToken+"/"+req.Path)
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

	// Generate the response
	resp := &logical.Response{
		Data: rawData,
	}

	return resp, nil
}

func (b *CubbyholeBackend) handleWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	if req.ClientToken == "" {
		return nil, fmt.Errorf("cubbyhole write: client token empty")
	}
	// Check that some fields are given
	if len(req.Data) == 0 {
		return nil, fmt.Errorf("missing data fields")
	}

	// JSON encode the data
	buf, err := json.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("json encoding failed: %v", err)
	}

	// Write out a new key
	entry := &logical.StorageEntry{
		Key:   req.ClientToken + "/" + req.Path,
		Value: buf,
	}
	if req.WrapInfo != nil && req.WrapInfo.SealWrap {
		entry.SealWrap = true
	}
	if err := req.Storage.Put(ctx, entry); err != nil {
		return nil, fmt.Errorf("failed to write: %v", err)
	}

	return nil, nil
}

func (b *CubbyholeBackend) handleDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	if req.ClientToken == "" {
		return nil, fmt.Errorf("cubbyhole delete: client token empty")
	}
	// Delete the key at the request path
	if err := req.Storage.Delete(ctx, req.ClientToken+"/"+req.Path); err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *CubbyholeBackend) handleList(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	if req.ClientToken == "" {
		return nil, fmt.Errorf("cubbyhole list: client token empty")
	}

	// Right now we only handle directories, so ensure it ends with / We also
	// check if it's empty so we don't end up doing a listing on '<client
	// token>//'
	path := req.Path
	if path != "" && !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	// List the keys at the prefix given by the request
	keys, err := req.Storage.List(ctx, req.ClientToken+"/"+path)
	if err != nil {
		return nil, err
	}

	// Strip the token
	strippedKeys := make([]string, len(keys))
	for i, key := range keys {
		strippedKeys[i] = strings.TrimPrefix(key, req.ClientToken+"/")
	}

	// Generate the response
	return logical.ListResponse(strippedKeys), nil
}

const cubbyholeHelp = `
The cubbyhole backend reads and writes arbitrary secrets to the backend.
The secrets are encrypted/decrypted by Vault: they are never stored
unencrypted in the backend and the backend never has an opportunity to
see the unencrypted value.

This backend differs from the 'kv' backend in that it is namespaced
per-token. Tokens can only read and write their own values, with no
sharing possible (per-token cubbyholes). This can be useful for implementing
certain authentication workflows, as well as "scratch" areas for individual
clients. When the token is revoked, the entire set of stored values for that
token is also removed.
`

const cubbyholeHelpSynopsis = `
Pass-through secret storage to a token-specific cubbyhole in the storage
backend, allowing you to read/write arbitrary data into secret storage.
`

const cubbyholeHelpDescription = `
The cubbyhole backend reads and writes arbitrary data into secret storage,
encrypting it along the way.

The view into the cubbyhole storage space is different for each token; it is
a per-token cubbyhole. When the token is revoked all values are removed.
`
