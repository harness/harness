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

package request

import (
	"context"

	"github.com/harness/gitness/registry/app/pkg"

	"github.com/rs/zerolog/log"
)

type contextKey string

const OriginalURLKey contextKey = "originalURL"
const ArtifactInfoKey contextKey = "artifactInfo"

func OriginalURLFrom(ctx context.Context) string {
	originalURL, ok := ctx.Value(OriginalURLKey).(string)
	if !ok {
		log.Ctx(ctx).Warn().Msg("Original URL not found in context")
	}
	return originalURL
}

func WithOriginalURL(parent context.Context, originalURL string) context.Context {
	return context.WithValue(parent, OriginalURLKey, originalURL)
}

func ArtifactInfoFrom(ctx context.Context) pkg.PackageArtifactInfo {
	baseInfo, ok := ctx.Value(ArtifactInfoKey).(pkg.PackageArtifactInfo)
	if !ok {
		log.Ctx(ctx).Warn().Msg("Failed to create artifact info")
	}
	return baseInfo
}

func WithArtifactInfo(parent context.Context, artifactInfo pkg.PackageArtifactInfo) context.Context {
	return context.WithValue(parent, ArtifactInfoKey, artifactInfo)
}
