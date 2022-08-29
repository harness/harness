// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package mocks provides mock interfaces.
package mocks

//go:generate mockgen -package=mocks -destination=mock_store.go github.com/harness/gitness/internal/store ExecutionStore,PipelineStore,SystemStore,UserStore
//go:generate mockgen -package=mocks -destination=mock_client.go github.com/harness/gitness/client Client
