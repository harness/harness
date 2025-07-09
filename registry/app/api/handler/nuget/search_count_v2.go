package nuget

import (
	"encoding/xml"
	"fmt"
	"net/http"

	nugettype "github.com/harness/gitness/registry/app/pkg/types/nuget"
	"github.com/harness/gitness/registry/request"

	"github.com/rs/zerolog/log"
)

func (h *handler) CountPackageV2(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	info, ok := request.ArtifactInfoFrom(ctx).(*nugettype.ArtifactInfo)
	if !ok {
		log.Ctx(ctx).Error().Msg("Failed to get artifact info from context")
		h.HandleErrors(r.Context(), []error{fmt.Errorf("failed to fetch info from context")}, w)
		return
	}
	response := h.controller.CountPackageV2(r.Context(), *info)

	if response.GetError() != nil {
		h.HandleError(r.Context(), w, response.GetError())
		return
	}

	w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	_, err := w.Write([]byte(xml.Header))
	if err != nil {
		h.HandleErrors(r.Context(), []error{err}, w)
		return
	}

	err = xml.NewEncoder(w).Encode(response.Count)
	if err != nil {
		h.HandleErrors(r.Context(), []error{err}, w)
		return
	}
}
