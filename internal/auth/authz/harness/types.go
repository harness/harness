// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package harness

/*
 * Classes required for harness ACL.
 * For now keep it here, as it shouldn't even be part of the code base in the first place
 * (should be in its own harness wide client library).
 */
type aclRequest struct {
	Principal   aclPrincipal    `json:"principal"`
	Permissions []aclPermission `json:"permissions"`
}

type aclResponse struct {
	Status        string          `json:"status"`
	CorrelationID string          `json:"correlationId"`
	Data          aclResponseData `json:"data"`
}

type aclResponseData struct {
	Principal         aclPrincipal        `json:"principal"`
	AccessControlList []aclControlElement `json:"accessControlList"`
}

type aclControlElement struct {
	Permission         string           `json:"permission"`
	ResourceScope      aclResourceScope `json:"resourceScope,omitempty"`
	ResourceType       string           `json:"resourceType"`
	ResourceIdentifier string           `json:"resourceIdentifier"`
	Permitted          bool             `json:"permitted"`
}

type aclResourceScope struct {
	AccountIdentifier string `json:"accountIdentifier"`
	OrgIdentifier     string `json:"orgIdentifier,omitempty"`
	ProjectIdentifier string `json:"projectIdentifier,omitempty"`
}

type aclPermission struct {
	ResourceScope      aclResourceScope `json:"resourceScope,omitempty"`
	ResourceType       string           `json:"resourceType"`
	ResourceIdentifier string           `json:"resourceIdentifier"`
	Permission         string           `json:"permission"`
}

type aclPrincipal struct {
	PrincipalIdentifier string `json:"principalIdentifier"`
	PrincipalType       string `json:"principalType"`
}
