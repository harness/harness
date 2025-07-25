// Copyright 2023 Harness, Inc.
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

package utils

import (
	"context"
	"regexp"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
)

func IsImagePresentLocally(ctx context.Context, imageName string, dockerClient *client.Client) (bool, error) {
	filterArgs := filters.NewArgs()
	filterArgs.Add("reference", imageName)

	images, err := dockerClient.ImageList(ctx, image.ListOptions{Filters: filterArgs})
	if err != nil {
		return false, err
	}

	return len(images) > 0, nil
}

// Image Expression Classification types.
const (
	ExpressionTypeInvalid            = "invalid"
	ExpressionTypeRepository         = "repository"
	ExpressionTypeImageTag           = "image (tag)"
	ExpressionTypeImageDigest        = "image (digest)"
	ExpressionTypeWildcardTag        = "wildcard image (tag)"
	ExpressionTypeWildcardRepo       = "wildcard (repo)"
	ExpressionTypeWildcardRepoPrefix = "wildcard (repo-prefix)"
	ExpressionTypeWildcardTagPrefix  = "wildcard image (tag-prefix)"
)

func CheckContainerImageExpression(input string) string {
	// Reject wildcard not at the end
	if idx := strings.IndexByte(input, '*'); idx != -1 && idx != len(input)-1 {
		return ExpressionTypeInvalid
	}

	switch {
	case strings.HasSuffix(input, "/*"):
		return classifyWildcardRepo(input)

	case strings.HasSuffix(input, ":*"):
		return classifyWildcardTag(input)

	case strings.HasSuffix(input, "*"):
		if strings.Contains(input, ":") {
			return classifyWildcardTagPrefix(input)
		}
		return classifyWildcardRepoPrefix(input)

	default:
		return classifyStandardImage(input)
	}
}

func classifyWildcardRepo(input string) string {
	base := strings.TrimSuffix(input, "/*")
	if _, err := reference.ParseAnyReference(base); err == nil {
		return ExpressionTypeWildcardRepo
	}
	return ExpressionTypeInvalid
}

func classifyWildcardTag(input string) string {
	base := strings.TrimSuffix(input, ":*")
	if _, err := reference.ParseAnyReference(base); err == nil {
		return ExpressionTypeWildcardTag
	}
	return ExpressionTypeInvalid
}

func classifyWildcardTagPrefix(input string) string {
	colonIndex := strings.LastIndex(input, ":")
	if colonIndex == -1 {
		return ExpressionTypeInvalid
	}

	repo := input[:colonIndex]
	tag := input[colonIndex+1:]

	if !strings.HasSuffix(tag, "*") {
		return ExpressionTypeInvalid
	}

	tagPrefix := strings.TrimSuffix(tag, "*")
	if !isValidTag(tagPrefix) {
		log.Warn().Str("tag", tagPrefix).Msg("Invalid tag prefix")
		return ExpressionTypeInvalid
	}

	if _, err := reference.ParseAnyReference(repo); err == nil {
		return ExpressionTypeWildcardTagPrefix
	}

	log.Error().Msgf("invalid repository in tag prefix: %s", repo)
	return ExpressionTypeInvalid
}

func classifyWildcardRepoPrefix(input string) string {
	base := strings.TrimSuffix(input, "*")
	if _, err := reference.ParseAnyReference(base); err == nil {
		return ExpressionTypeWildcardRepoPrefix
	}
	return ExpressionTypeInvalid
}

func classifyStandardImage(input string) string {
	ref, err := reference.ParseAnyReference(input)
	if err != nil {
		return ExpressionTypeInvalid
	}
	switch ref.(type) {
	case reference.Canonical:
		return ExpressionTypeImageDigest
	case reference.Tagged:
		return ExpressionTypeImageTag
	case reference.Named:
		return ExpressionTypeRepository
	default:
		return ExpressionTypeInvalid
	}
}

var tagRegexp = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}$`)

func isValidTag(tag string) bool {
	return tagRegexp.MatchString(tag)
}

func MatchesContainerImageExpression(expr, image string) bool {
	switch CheckContainerImageExpression(expr) {
	case ExpressionTypeImageDigest, ExpressionTypeImageTag:
		return expr == image

	case ExpressionTypeRepository:
		return getRepo(image) == expr

	case ExpressionTypeWildcardRepo:
		return strings.HasPrefix(getRepo(image), strings.TrimSuffix(expr, "/*")+"/")

	case ExpressionTypeWildcardTag:
		repoExpr := strings.TrimSuffix(expr, ":*")
		repo, tag := splitImage(image)
		return repo == repoExpr && tag != ""

	case ExpressionTypeWildcardRepoPrefix:
		return strings.HasPrefix(getRepo(image), strings.TrimSuffix(expr, "*"))

	case ExpressionTypeWildcardTagPrefix:
		colon := strings.LastIndex(expr, ":")
		if colon == -1 {
			return false
		}
		repoExpr := expr[:colon]
		tagPrefix := strings.TrimSuffix(expr[colon+1:], "*")

		repo, tag := splitImage(image)
		return repo == repoExpr && strings.HasPrefix(tag, tagPrefix)

	default:
		return false
	}
}

func getRepo(image string) string {
	if i := strings.IndexAny(image, "@:"); i != -1 {
		return image[:i]
	}
	return image
}

func splitImage(image string) (repo, tag string) {
	if i := strings.LastIndex(image, ":"); i != -1 && !strings.Contains(image[i:], "/") {
		return image[:i], image[i+1:]
	}
	return image, ""
}

func IsValidContainerImage(image string) bool {
	_, err := reference.ParseAnyReference(image)
	return err == nil
}
