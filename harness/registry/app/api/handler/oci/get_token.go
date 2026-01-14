//  Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package oci

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/jwt"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/token"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type TokenResponseOCI struct {
	Token string `json:"token"`
}

func (h *Handler) GetToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	session, _ := request.AuthSessionFrom(ctx)

	if tokenMetadata, okt := session.Metadata.(*auth.TokenMetadata); okt &&
		tokenMetadata.TokenType != enum.TokenTypePAT {
		returnForbiddenResponse(ctx, w, fmt.Errorf("only personal access token allowed"))
		return
	}

	var user *types.User
	if auth.IsAnonymousSession(session) {
		user = &types.User{
			ID:   session.Principal.ID,
			UID:  session.Principal.UID,
			Salt: h.AnonymousUserSecret,
		}
	} else {
		var err error
		user, err = h.UserCtrl.FindNoAuth(ctx, session.Principal.UID)
		if err != nil {
			returnForbiddenResponse(ctx, w, err)
			return
		}
	}

	requestedOciAccess := GetRequestedResourceActions(getScopes(r.URL))
	var accessPermissionsList = []jwt.AccessPermissions{}
	for _, ra := range requestedOciAccess {
		space, err := h.getSpace(ctx, ra.Name)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			log.Ctx(ctx).Warn().Msgf("failed to find space by ref: %v", err)
			continue
		}

		accessPermissionsList = h.getAccessPermissionList(ctx, space, ra, session, accessPermissionsList)
	}

	subClaimsAccessPermissions := &jwt.SubClaimsAccessPermissions{
		Source:      jwt.OciSource,
		Permissions: accessPermissionsList,
	}

	jwtToken, err := h.getTokenDetails(user, subClaimsAccessPermissions)
	if err != nil {
		returnForbiddenResponse(ctx, w, err)
		return
	}
	if jwtToken != "" {
		w.WriteHeader(http.StatusOK)
		enc := json.NewEncoder(w)
		if err := enc.Encode(
			TokenResponseOCI{
				Token: jwtToken,
			},
		); err != nil {
			log.Ctx(ctx).Error().Msgf("failed to write token response: %v", err)
		}
		return
	}
}

func (h *Handler) getSpace(ctx context.Context, name string) (*types.SpaceCore, error) {
	spaceRef, _, _ := paths.DisectRoot(name)
	space, err := h.SpaceFinder.FindByRef(ctx, spaceRef)
	return space, err
}

func (h *Handler) getAccessPermissionList(
	ctx context.Context, space *types.SpaceCore, ra *ResourceActions, session *auth.Session,
	accessPermissionsList []jwt.AccessPermissions,
) []jwt.AccessPermissions {
	accessPermissions := &jwt.AccessPermissions{SpaceID: space.ID, Permissions: []enum.Permission{}}

	for _, a := range ra.Actions {
		permission, err := getPermissionFromAction(ctx, a)
		if err != nil {
			log.Ctx(ctx).Warn().Msgf("failed to get permission from action: %v", err)
			continue
		}
		scopeErr := apiauth.CheckSpaceScope(
			ctx,
			h.Authorizer,
			session,
			space,
			enum.ResourceTypeRegistry,
			permission,
		)
		if scopeErr != nil {
			log.Ctx(ctx).Warn().Msgf("failed to check space scope: %v", scopeErr)
			continue
		}
		accessPermissions.Permissions = append(accessPermissions.Permissions, permission)
	}
	accessPermissionsList = append(accessPermissionsList, *accessPermissions)
	return accessPermissionsList
}

func getPermissionFromAction(ctx context.Context, action string) (enum.Permission, error) {
	switch action {
	case "pull":
		return enum.PermissionArtifactsDownload, nil
	case "push":
		return enum.PermissionArtifactsUpload, nil
	case "delete":
		return enum.PermissionArtifactsDelete, nil
	default:
		err := fmt.Errorf("unknown action: %s", action)
		log.Ctx(ctx).Err(err).Msgf("Failed to get permission from action: %v", err)
		return "", err
	}
}

func returnForbiddenResponse(ctx context.Context, w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusForbidden)
	_, err2 := w.Write([]byte(fmt.Sprintf("requested access to the resource is denied: %v", err)))
	if err2 != nil {
		log.Ctx(ctx).Error().Msgf("failed to write token response: %v", err2)
	}
}

/*
 * getTokenDetails attempts to get token details.
 */
func (h *Handler) getTokenDetails(
	user *types.User,
	accessPermissions *jwt.SubClaimsAccessPermissions,
) (string, error) {
	return token.CreateUserWithAccessPermissions(user, accessPermissions)
}

// GetRequestedResourceActions ...
func GetRequestedResourceActions(scopes []string) []*ResourceActions {
	var res []*ResourceActions
	for _, s := range scopes {
		if s == "" {
			continue
		}
		items := strings.Split(s, ":")
		length := len(items)

		var resourceType string
		var resourceName string
		actions := make([]string, 0)

		switch length {
		case 1:
			resourceType = items[0]
		case 2:
			resourceType = items[0]
			resourceName = items[1]
		default:
			resourceType = items[0]
			resourceName = strings.Join(items[1:length-1], ":")
			if len(items[length-1]) > 0 {
				actions = strings.Split(items[length-1], ",")
			}
		}

		res = append(
			res, &ResourceActions{
				Type:    resourceType,
				Name:    resourceName,
				Actions: actions,
			},
		)
	}
	return res
}

func getScopes(u *url.URL) []string {
	var sector string
	var result []string
	for _, sector = range u.Query()["scope"] {
		result = append(result, strings.Split(sector, " ")...)
	}
	return result
}

// ResourceActions stores allowed actions on a resource.
type ResourceActions struct {
	Type    string   `json:"type"`
	Name    string   `json:"name"`
	Actions []string `json:"actions"`
}
