package http

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/vault"
)

func handleSysInit(core *vault.Core) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			handleSysInitGet(core, w, r)
		case "PUT", "POST":
			handleSysInitPut(core, w, r)
		default:
			respondError(w, http.StatusMethodNotAllowed, nil)
		}
	})
}

func handleSysInitGet(core *vault.Core, w http.ResponseWriter, r *http.Request) {
	init, err := core.Initialized(context.Background())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	respondOk(w, &InitStatusResponse{
		Initialized: init,
	})
}

func handleSysInitPut(core *vault.Core, w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Parse the request
	var req InitRequest
	if err := parseRequest(r, w, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	// Initialize
	barrierConfig := &vault.SealConfig{
		SecretShares:    req.SecretShares,
		SecretThreshold: req.SecretThreshold,
		StoredShares:    req.StoredShares,
		PGPKeys:         req.PGPKeys,
	}

	recoveryConfig := &vault.SealConfig{
		SecretShares:    req.RecoveryShares,
		SecretThreshold: req.RecoveryThreshold,
		PGPKeys:         req.RecoveryPGPKeys,
	}

	// N.B. Although the core is capable of handling situations where some keys
	// are stored and some aren't, in practice, replication + HSMs makes this
	// extremely hard to reason about, to the point that it will probably never
	// be supported. The reason is that each HSM needs to encode the master key
	// separately, which means the shares must be generated independently,
	// which means both that the shares will be different *AND* there would
	// need to be a way to actually allow fetching of the generated keys by
	// operators.
	if core.SealAccess().StoredKeysSupported() {
		if len(barrierConfig.PGPKeys) > 0 {
			respondError(w, http.StatusBadRequest, fmt.Errorf("PGP keys not supported when storing shares"))
			return
		}
		barrierConfig.SecretShares = 1
		barrierConfig.SecretThreshold = 1
		barrierConfig.StoredShares = 1
		core.Logger().Warn("init: stored keys supported, forcing shares/threshold to 1")
	} else {
		if barrierConfig.StoredShares > 0 {
			respondError(w, http.StatusBadRequest, fmt.Errorf("stored keys are not supported by the current seal type"))
			return
		}
	}

	if len(barrierConfig.PGPKeys) > 0 && len(barrierConfig.PGPKeys) != barrierConfig.SecretShares {
		respondError(w, http.StatusBadRequest, fmt.Errorf("incorrect number of PGP keys"))
		return
	}

	if core.SealAccess().RecoveryKeySupported() {
		if len(recoveryConfig.PGPKeys) > 0 && len(recoveryConfig.PGPKeys) != recoveryConfig.SecretShares {
			respondError(w, http.StatusBadRequest, fmt.Errorf("incorrect number of PGP keys for recovery"))
			return
		}
	}

	initParams := &vault.InitParams{
		BarrierConfig:   barrierConfig,
		RecoveryConfig:  recoveryConfig,
		RootTokenPGPKey: req.RootTokenPGPKey,
	}

	result, initErr := core.Initialize(ctx, initParams)
	if initErr != nil {
		if !errwrap.ContainsType(initErr, new(vault.NonFatalError)) {
			respondError(w, http.StatusBadRequest, initErr)
			return
		} else {
			// Add a warnings field? The error will be logged in the vault log
			// already.
		}
	}

	// Encode the keys
	keys := make([]string, 0, len(result.SecretShares))
	keysB64 := make([]string, 0, len(result.SecretShares))
	for _, k := range result.SecretShares {
		keys = append(keys, hex.EncodeToString(k))
		keysB64 = append(keysB64, base64.StdEncoding.EncodeToString(k))
	}

	resp := &InitResponse{
		Keys:      keys,
		KeysB64:   keysB64,
		RootToken: result.RootToken,
	}

	if len(result.RecoveryShares) > 0 {
		resp.RecoveryKeys = make([]string, 0, len(result.RecoveryShares))
		resp.RecoveryKeysB64 = make([]string, 0, len(result.RecoveryShares))
		for _, k := range result.RecoveryShares {
			resp.RecoveryKeys = append(resp.RecoveryKeys, hex.EncodeToString(k))
			resp.RecoveryKeysB64 = append(resp.RecoveryKeysB64, base64.StdEncoding.EncodeToString(k))
		}
	}

	core.UnsealWithStoredKeys(ctx)

	respondOk(w, resp)
}

type InitRequest struct {
	SecretShares      int      `json:"secret_shares"`
	SecretThreshold   int      `json:"secret_threshold"`
	StoredShares      int      `json:"stored_shares"`
	PGPKeys           []string `json:"pgp_keys"`
	RecoveryShares    int      `json:"recovery_shares"`
	RecoveryThreshold int      `json:"recovery_threshold"`
	RecoveryPGPKeys   []string `json:"recovery_pgp_keys"`
	RootTokenPGPKey   string   `json:"root_token_pgp_key"`
}

type InitResponse struct {
	Keys            []string `json:"keys"`
	KeysB64         []string `json:"keys_base64"`
	RecoveryKeys    []string `json:"recovery_keys,omitempty"`
	RecoveryKeysB64 []string `json:"recovery_keys_base64,omitempty"`
	RootToken       string   `json:"root_token"`
}

type InitStatusResponse struct {
	Initialized bool `json:"initialized"`
}
