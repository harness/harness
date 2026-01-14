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

package pullreq

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

func (c *Controller) processMentions(
	ctx context.Context,
	text string,
) (map[int64]*types.PrincipalInfo, error) {
	mentions := parseMentions(ctx, text)
	if len(mentions) == 0 {
		return map[int64]*types.PrincipalInfo{}, nil
	}

	infos, err := c.principalInfoCache.Map(ctx, mentions)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch info from principalInfoCache: %w", err)
	}

	return infos, nil
}

var mentionRegex = regexp.MustCompile(`@\[(\d+)\]`)

func parseMentions(ctx context.Context, text string) []int64 {
	matches := mentionRegex.FindAllStringSubmatch(text, -1)

	var mentions []int64
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		if mention, err := strconv.ParseInt(match[1], 10, 64); err == nil {
			mentions = append(mentions, mention)
		} else {
			log.Ctx(ctx).Warn().Err(err).Msgf("failed to parse mention %q", match[1])
		}
	}

	return mentions
}
