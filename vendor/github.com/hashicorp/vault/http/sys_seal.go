package http

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/helper/consts"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/vault"
	"github.com/hashicorp/vault/version"
)

func handleSysSeal(core *vault.Core) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, statusCode, err := buildLogicalRequest(core, w, r)
		if err != nil || statusCode != 0 {
			respondError(w, statusCode, err)
			return
		}

		switch req.Operation {
		case logical.UpdateOperation:
		default:
			respondError(w, http.StatusMethodNotAllowed, nil)
			return
		}

		// Seal with the token above
		// We use context.Background since there won't be a request context if the node isn't active
		if err := core.SealWithRequest(req); err != nil {
			if errwrap.Contains(err, logical.ErrPermissionDenied.Error()) {
				respondError(w, http.StatusForbidden, err)
				return
			} else {
				respondError(w, http.StatusInternalServerError, err)
				return
			}
		}

		respondOk(w, nil)
	})
}

func handleSysStepDown(core *vault.Core) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, statusCode, err := buildLogicalRequest(core, w, r)
		if err != nil || statusCode != 0 {
			respondError(w, statusCode, err)
			return
		}

		switch req.Operation {
		case logical.UpdateOperation:
		default:
			respondError(w, http.StatusMethodNotAllowed, nil)
			return
		}

		// Seal with the token above
		if err := core.StepDown(req); err != nil {
			respondError(w, http.StatusInternalServerError, err)
			return
		}

		respondOk(w, nil)
	})
}

func handleSysUnseal(core *vault.Core) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "PUT":
		case "POST":
		default:
			respondError(w, http.StatusMethodNotAllowed, nil)
			return
		}

		// Parse the request
		var req UnsealRequest
		if err := parseRequest(r, w, &req); err != nil {
			respondError(w, http.StatusBadRequest, err)
			return
		}
		if !req.Reset && req.Key == "" {
			respondError(
				w, http.StatusBadRequest,
				errors.New("'key' must be specified in request body as JSON, or 'reset' set to true"))
			return
		}

		if req.Reset {
			sealed, err := core.Sealed()
			if err != nil {
				respondError(w, http.StatusInternalServerError, err)
				return
			}
			if !sealed {
				respondError(w, http.StatusBadRequest, errors.New("vault is unsealed"))
				return
			}
			core.ResetUnsealProcess()
		} else {
			// Decode the key, which is base64 or hex encoded
			min, max := core.BarrierKeyLength()
			key, err := hex.DecodeString(req.Key)
			// We check min and max here to ensure that a string that is base64
			// encoded but also valid hex will not be valid and we instead base64
			// decode it
			if err != nil || len(key) < min || len(key) > max {
				key, err = base64.StdEncoding.DecodeString(req.Key)
				if err != nil {
					respondError(
						w, http.StatusBadRequest,
						errors.New("'key' must be a valid hex or base64 string"))
					return
				}
			}

			// Attempt the unseal
			ctx := context.Background()
			if core.SealAccess().RecoveryKeySupported() {
				_, err = core.UnsealWithRecoveryKeys(ctx, key)
			} else {
				_, err = core.Unseal(key)
			}
			if err != nil {
				switch {
				case errwrap.ContainsType(err, new(vault.ErrInvalidKey)):
				case errwrap.Contains(err, vault.ErrBarrierInvalidKey.Error()):
				case errwrap.Contains(err, vault.ErrBarrierNotInit.Error()):
				case errwrap.Contains(err, vault.ErrBarrierSealed.Error()):
				case errwrap.Contains(err, consts.ErrStandby.Error()):
				default:
					respondError(w, http.StatusInternalServerError, err)
					return
				}
				respondError(w, http.StatusBadRequest, err)
				return
			}
		}

		// Return the seal status
		handleSysSealStatusRaw(core, w, r)
	})
}

func handleSysSealStatus(core *vault.Core) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			respondError(w, http.StatusMethodNotAllowed, nil)
			return
		}

		handleSysSealStatusRaw(core, w, r)
	})
}

func handleSysSealStatusRaw(core *vault.Core, w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	sealed, err := core.Sealed()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	var sealConfig *vault.SealConfig
	if core.SealAccess().RecoveryKeySupported() {
		sealConfig, err = core.SealAccess().RecoveryConfig(ctx)
	} else {
		sealConfig, err = core.SealAccess().BarrierConfig(ctx)
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	if sealConfig == nil {
		respondError(w, http.StatusBadRequest, fmt.Errorf(
			"server is not yet initialized"))
		return
	}

	// Fetch the local cluster name and identifier
	var clusterName, clusterID string
	if !sealed {
		cluster, err := core.Cluster(ctx)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err)
			return
		}
		if cluster == nil {
			respondError(w, http.StatusInternalServerError, fmt.Errorf("failed to fetch cluster details"))
			return
		}
		clusterName = cluster.Name
		clusterID = cluster.ID
	}

	progress, nonce := core.SecretProgress()

	respondOk(w, &SealStatusResponse{
		Type:        sealConfig.Type,
		Sealed:      sealed,
		T:           sealConfig.SecretThreshold,
		N:           sealConfig.SecretShares,
		Progress:    progress,
		Nonce:       nonce,
		Version:     version.GetVersion().VersionNumber(),
		ClusterName: clusterName,
		ClusterID:   clusterID,
	})
}

type SealStatusResponse struct {
	Type        string `json:"type"`
	Sealed      bool   `json:"sealed"`
	T           int    `json:"t"`
	N           int    `json:"n"`
	Progress    int    `json:"progress"`
	Nonce       string `json:"nonce"`
	Version     string `json:"version"`
	ClusterName string `json:"cluster_name,omitempty"`
	ClusterID   string `json:"cluster_id,omitempty"`
}

type UnsealRequest struct {
	Key   string
	Reset bool
}
