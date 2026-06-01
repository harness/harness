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

package enum

// PullReqCommentReactionEmoji represents a validated emoji name for a PR comment reaction.
type PullReqCommentReactionEmoji string

func (PullReqCommentReactionEmoji) Enum() []any {
	return toInterfaceSlice(pullReqCommentReactionEmojis)
}

func (e PullReqCommentReactionEmoji) Sanitize() (PullReqCommentReactionEmoji, bool) {
	return Sanitize(e, GetAllPullReqCommentReactionEmojis)
}

func GetAllPullReqCommentReactionEmojis() ([]PullReqCommentReactionEmoji, PullReqCommentReactionEmoji) {
	return pullReqCommentReactionEmojis, ""
}

const (
	PullReqCommentReactionEmojiPlusOne  PullReqCommentReactionEmoji = "plusone"
	PullReqCommentReactionEmojiMinusOne PullReqCommentReactionEmoji = "minusone"
	PullReqCommentReactionEmojiSmile    PullReqCommentReactionEmoji = "smile"
	PullReqCommentReactionEmojiTada     PullReqCommentReactionEmoji = "tada"
	PullReqCommentReactionEmojiConfused PullReqCommentReactionEmoji = "confused"
	PullReqCommentReactionEmojiHeart    PullReqCommentReactionEmoji = "heart"
	PullReqCommentReactionEmojiRocket   PullReqCommentReactionEmoji = "rocket"
	PullReqCommentReactionEmojiEyes     PullReqCommentReactionEmoji = "eyes"
)

var pullReqCommentReactionEmojis = sortEnum([]PullReqCommentReactionEmoji{
	PullReqCommentReactionEmojiPlusOne,
	PullReqCommentReactionEmojiMinusOne,
	PullReqCommentReactionEmojiSmile,
	PullReqCommentReactionEmojiTada,
	PullReqCommentReactionEmojiConfused,
	PullReqCommentReactionEmojiHeart,
	PullReqCommentReactionEmojiRocket,
	PullReqCommentReactionEmojiEyes,
})
