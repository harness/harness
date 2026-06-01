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

import (
	"testing"
)

func TestPullReqCommentReactionEmojiSanitize(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantEmoji PullReqCommentReactionEmoji
		wantOK    bool
	}{
		{"plusone valid", "plusone", PullReqCommentReactionEmojiPlusOne, true},
		{"minusone valid", "minusone", PullReqCommentReactionEmojiMinusOne, true},
		{"smile valid", "smile", PullReqCommentReactionEmojiSmile, true},
		{"tada valid", "tada", PullReqCommentReactionEmojiTada, true},
		{"confused valid", "confused", PullReqCommentReactionEmojiConfused, true},
		{"heart valid", "heart", PullReqCommentReactionEmojiHeart, true},
		{"rocket valid", "rocket", PullReqCommentReactionEmojiRocket, true},
		{"eyes valid", "eyes", PullReqCommentReactionEmojiEyes, true},
		{"empty string invalid", "", "", false},
		{"unknown emoji invalid", "unknown", "", false},
		{"old +1 symbol invalid", "+1", "", false},
		{"old -1 symbol invalid", "-1", "", false},
		{"uppercase invalid", "PLUSONE", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := PullReqCommentReactionEmoji(tt.input).Sanitize()
			if ok != tt.wantOK {
				t.Errorf("Sanitize(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if got != tt.wantEmoji {
				t.Errorf("Sanitize(%q) emoji = %q, want %q", tt.input, got, tt.wantEmoji)
			}
		})
	}
}

func TestPullReqCommentReactionEmojiEnum(t *testing.T) {
	vals := PullReqCommentReactionEmojiPlusOne.Enum()
	if len(vals) != 8 {
		t.Errorf("Enum() returned %d values, want 8", len(vals))
	}
}

func TestGetAllPullReqCommentReactionEmojis(t *testing.T) {
	all, def := GetAllPullReqCommentReactionEmojis()
	if len(all) != 8 {
		t.Errorf("GetAllPullReqCommentReactionEmojis() returned %d values, want 8", len(all))
	}
	if def != "" {
		t.Errorf("GetAllPullReqCommentReactionEmojis() default = %q, want empty string", def)
	}
}
