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

package stream

import "fmt"

// DiscardedMessageError is returned (via pushError) when a message is force-acknowledged
// after exceeding the maximum retry count, regardless of whether the XAck call itself succeeded.
type DiscardedMessageError struct {
	MessageID  string
	StreamID   string
	RetryCount int64
	AckErr     error // non-nil if the XAck call also failed
}

func (e *DiscardedMessageError) Error() string {
	if e.AckErr != nil {
		return fmt.Sprintf(
			"failed to force acknowledge (discard) message '%s' (Retries: %d) in stream '%s': %s",
			e.MessageID, e.RetryCount, e.StreamID, e.AckErr)
	}
	return fmt.Sprintf(
		"force acknowledged (discarded) message '%s' (Retries: %d) in stream '%s'",
		e.MessageID, e.RetryCount, e.StreamID)
}

func (e *DiscardedMessageError) Unwrap() error { return e.AckErr }
