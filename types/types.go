// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package types defines common data structures.
package types

import (
	"time"

	"github.com/harness/gitness/types/enum"
)

type (
	// Scope defines the data scope.
	Scope struct {
		Account      string
		Organization string
		Project      string
		Redirect     string
	}

	// Params stores query parameters.
	Params struct {
		Page  int        `json:"page"`
		Size  int        `json:"size"`
		Sort  string     `json:"sort"`
		Order enum.Order `json:"direction"`
	}

	// Execution stores execution details.
	Execution struct {
		ID       int64  `db:"execution_id"          json:"id"`
		Pipeline int64  `db:"execution_pipeline_id" json:"pipeline,omitempty"`
		Slug     string `db:"execution_slug"        json:"slug"`
		Name     string `db:"execution_name"        json:"name"`
		Desc     string `db:"execution_desc"        json:"desc"`
		Created  int64  `db:"execution_created"     json:"created"`
		Updated  int64  `db:"execution_updated"     json:"updated"`
	}

	// ExecutionParams stores execution parameters.
	ExecutionParams struct {
		Pipeline int64
		Slug     string

		Scope Scope
	}

	// ExecutionListParams stores execution list
	// parameters.
	ExecutionListParams struct {
		Pipeline int64

		Query Params
		Scope Scope
	}

	// ExecutionInput store details used to create or
	// update a execution.
	ExecutionInput struct {
		Slug *string `json:"slug"`
		Name *string `json:"name"`
		Desc *string `json:"desc"`
	}

	// Pipeline stores pipeline details.
	Pipeline struct {
		ID      int64  `db:"pipeline_id"      json:"id"`
		Name    string `db:"pipeline_name"    json:"name"`
		Slug    string `db:"pipeline_slug"    json:"slug"`
		Desc    string `db:"pipeline_desc"    json:"desc"`
		Token   string `db:"pipeline_token"   json:"-"`
		Active  bool   `db:"pipeline_active"  json:"active"`
		Created int64  `db:"pipeline_created" json:"created"`
		Updated int64  `db:"pipeline_updated" json:"updated"`
	}

	// PipelineParams stores pipeline parameters.
	PipelineParams struct {
		Slug string

		Scope Scope
	}

	// PipelineListParams stores pipeline list
	// parameters.
	PipelineListParams struct {
		Query Params
		Scope Scope
	}

	// PipelineInput store user pipeline details used to
	// create or update a pipeline.
	PipelineInput struct {
		Slug *string `json:"slug"`
		Name *string `json:"name"`
		Desc *string `json:"desc"`
	}

	// User stores user account details.
	User struct {
		ID       int64  `db:"user_id"        json:"id"`
		Email    string `db:"user_email"     json:"email"`
		Password string `db:"user_password"  json:"-"`
		Salt     string `db:"user_salt"      json:"-"`
		Name     string `db:"user_name"      json:"name"`
		Company  string `db:"user_company"   json:"company"`
		Admin    bool   `db:"user_admin"     json:"admin"`
		Blocked  bool   `db:"user_blocked"   json:"-"`
		Created  int64  `db:"user_created"   json:"created"`
		Updated  int64  `db:"user_updated"   json:"updated"`
		Authed   int64  `db:"user_authed"    json:"authed"`
	}

	// UserInput store user account details used to
	// create or update a user.
	UserInput struct {
		Username *string `json:"email"`
		Password *string `json:"password"`
		Name     *string `json:"name"`
		Company  *string `json:"company"`
		Admin    *bool   `json:"admin"`
	}

	// UserFilter stores user query parameters.
	UserFilter struct {
		Page  int           `json:"page"`
		Size  int           `json:"size"`
		Sort  enum.UserAttr `json:"sort"`
		Order enum.Order    `json:"direction"`
	}

	// Token stores token  details.
	Token struct {
		Value   string    `json:"access_token"`
		Address string    `json:"uri,omitempty"`
		Expires time.Time `json:"expires_at,omitempty"`
	}

	// UserToken stores user account and token details.
	UserToken struct {
		User  *User  `json:"user"`
		Token *Token `json:"token"`
	}

	// Project stores project details.
	Project struct {
		Identifier string            `json:"identifier"`
		Color      string            `json:"color"`
		Desc       string            `json:"description"`
		Name       string            `json:"name"`
		Modules    []string          `json:"modules"`
		Org        string            `json:"orgIdentifier"`
		Tags       map[string]string `json:"tags"`
	}

	// ProjectList stores the project list and project
	// result set metdata.
	ProjectList struct {
		Data []*Project `json:"data"`

		Empty         bool `json:"empty"`
		PageIndex     int  `json:"pageIndex,omitempty"`
		PageItemCount int  `json:"pageItemCount,omitempty"`
		PageSize      int  `json:"pageSize,omitempty"`
		TotalItems    int  `json:"totalItems,omitempty"`
		TotalPages    int  `json:"totalPages,omitempty"`
	}
)
