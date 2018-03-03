package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/hashicorp/vault/helper/consts"
	"github.com/hashicorp/vault/vault"
	"github.com/hashicorp/vault/version"
)

func handleSysHealth(core *vault.Core) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			handleSysHealthGet(core, w, r)
		case "HEAD":
			handleSysHealthHead(core, w, r)
		default:
			respondError(w, http.StatusMethodNotAllowed, nil)
		}
	})
}

func fetchStatusCode(r *http.Request, field string) (int, bool, bool) {
	var err error
	statusCode := http.StatusOK
	if statusCodeStr, statusCodeOk := r.URL.Query()[field]; statusCodeOk {
		statusCode, err = strconv.Atoi(statusCodeStr[0])
		if err != nil || len(statusCodeStr) < 1 {
			return http.StatusBadRequest, false, false
		}
		return statusCode, true, true
	}
	return statusCode, false, true
}

func handleSysHealthGet(core *vault.Core, w http.ResponseWriter, r *http.Request) {
	code, body, err := getSysHealth(core, r)
	if err != nil {
		core.Logger().Error("error checking health", "error", err)
		respondError(w, http.StatusInternalServerError, nil)
		return
	}

	if body == nil {
		respondError(w, code, nil)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	// Generate the response
	enc := json.NewEncoder(w)
	enc.Encode(body)
}

func handleSysHealthHead(core *vault.Core, w http.ResponseWriter, r *http.Request) {
	code, body, err := getSysHealth(core, r)
	if err != nil {
		code = http.StatusInternalServerError
	}

	if body != nil {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(code)
}

func getSysHealth(core *vault.Core, r *http.Request) (int, *HealthResponse, error) {
	// Check if being a standby is allowed for the purpose of a 200 OK
	_, standbyOK := r.URL.Query()["standbyok"]

	uninitCode := http.StatusNotImplemented
	if code, found, ok := fetchStatusCode(r, "uninitcode"); !ok {
		return http.StatusBadRequest, nil, nil
	} else if found {
		uninitCode = code
	}

	sealedCode := http.StatusServiceUnavailable
	if code, found, ok := fetchStatusCode(r, "sealedcode"); !ok {
		return http.StatusBadRequest, nil, nil
	} else if found {
		sealedCode = code
	}

	standbyCode := http.StatusTooManyRequests // Consul warning code
	if code, found, ok := fetchStatusCode(r, "standbycode"); !ok {
		return http.StatusBadRequest, nil, nil
	} else if found {
		standbyCode = code
	}

	activeCode := http.StatusOK
	if code, found, ok := fetchStatusCode(r, "activecode"); !ok {
		return http.StatusBadRequest, nil, nil
	} else if found {
		activeCode = code
	}

	drSecondaryCode := 472 // unofficial 4xx status code
	if code, found, ok := fetchStatusCode(r, "drsecondarycode"); !ok {
		return http.StatusBadRequest, nil, nil
	} else if found {
		drSecondaryCode = code
	}

	ctx := context.Background()

	// Check system status
	sealed, _ := core.Sealed()
	standby, _ := core.Standby()
	var replicationState consts.ReplicationState
	if standby {
		replicationState = core.ActiveNodeReplicationState()
	} else {
		replicationState = core.ReplicationState()
	}

	init, err := core.Initialized(ctx)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	// Determine the status code
	code := activeCode
	switch {
	case !init:
		code = uninitCode
	case sealed:
		code = sealedCode
	case replicationState.HasState(consts.ReplicationDRSecondary):
		code = drSecondaryCode
	case !standbyOK && standby:
		code = standbyCode
	}

	// Fetch the local cluster name and identifier
	var clusterName, clusterID string
	if !sealed {
		cluster, err := core.Cluster(ctx)
		if err != nil {
			return http.StatusInternalServerError, nil, err
		}
		if cluster == nil {
			return http.StatusInternalServerError, nil, fmt.Errorf("failed to fetch cluster details")
		}
		clusterName = cluster.Name
		clusterID = cluster.ID
	}

	// Format the body
	body := &HealthResponse{
		Initialized:                init,
		Sealed:                     sealed,
		Standby:                    standby,
		ReplicationPerformanceMode: replicationState.GetPerformanceString(),
		ReplicationDRMode:          replicationState.GetDRString(),
		ServerTimeUTC:              time.Now().UTC().Unix(),
		Version:                    version.GetVersion().VersionNumber(),
		ClusterName:                clusterName,
		ClusterID:                  clusterID,
	}
	return code, body, nil
}

type HealthResponse struct {
	Initialized                bool   `json:"initialized"`
	Sealed                     bool   `json:"sealed"`
	Standby                    bool   `json:"standby"`
	ReplicationPerformanceMode string `json:"replication_performance_mode"`
	ReplicationDRMode          string `json:"replication_dr_mode"`
	ServerTimeUTC              int64  `json:"server_time_utc"`
	Version                    string `json:"version"`
	ClusterName                string `json:"cluster_name,omitempty"`
	ClusterID                  string `json:"cluster_id,omitempty"`
}
