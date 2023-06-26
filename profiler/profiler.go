// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package profiler

import (
	"fmt"
	"strings"
)

type Profiler interface {
	StartProfiling(serviceName, serviceVersion string)
}

type Type string

const (
	TypeGCP Type = "gcp"
)

func ParseType(profilerType string) (Type, bool) {
	switch strings.TrimSpace(strings.ToLower(profilerType)) {
	case string(TypeGCP):
		return TypeGCP, true
	default:
		return "", false
	}
}

func New(profiler Type) (Profiler, error) {
	switch profiler {
	case TypeGCP:
		return &GCPProfiler{}, nil
	default:
		return &NoopProfiler{}, fmt.Errorf("profiler '%s' not supported", profiler)
	}
}
