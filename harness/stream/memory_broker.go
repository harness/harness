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

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/go-multierror"
	gonanoid "github.com/matoous/go-nanoid"
)

const (
	idPrefixUIDAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	idPrefixUIDLength   = 8
)

// MemoryBroker is a very basic in memory broker implementation that supports multiple streams and consumer groups.
type MemoryBroker struct {
	// idPrefix is a random prefix the memory broker is seeded with to avoid overlaps with previous runs!
	idPrefix string
	// latestID is used to generate unique, sequentially increasing message IDs
	latestID uint64
	// queueSize is the max size of the queues before enqueing is blocking
	queueSize int64
	mx        sync.RWMutex
	// messageQueues contains all streams requested for a specific group
	// messageQueues[streamID][groupName]: queue of messages for a specific stream and group
	messageQueues map[string]map[string]chan message
}

func NewMemoryBroker(queueSize int64) (*MemoryBroker, error) {
	idPrefix, err := gonanoid.Generate(idPrefixUIDAlphabet, idPrefixUIDLength)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random prefix for in-memory event IDs: %w", err)
	}
	return &MemoryBroker{
		idPrefix:      idPrefix,
		latestID:      0,
		queueSize:     queueSize,
		mx:            sync.RWMutex{},
		messageQueues: make(map[string]map[string]chan message),
	}, err
}

// enqueue enqueues a message to a stream.
// NOTE: messages are only send to existing groups - groups created after completion won't get the message.
// NOTE: if any of the groups' message queues is full, an aggregated error is returned.
// However, the error is only returned after attempting to send the message to all groups.
func (b *MemoryBroker) enqueue(streamID string, m message) (string, error) {
	// similar to redis, only populate the ID if's empty or requested explicitly
	if m.id == "" || m.id == "*" {
		id := atomic.AddUint64(&b.latestID, 1)
		m.id = fmt.Sprintf("%s-%d", b.idPrefix, id)
	}

	// the lock is for reading from the messageQueues map
	// NOTE: this method isn't blocking anywhere so we should be safe from deadlocking
	// NOTE: might be possible to optimize (potentially not even required), but okay for initial local solution.
	b.mx.RLock()
	defer b.mx.RUnlock()

	// get all group queues for the stream - no lock needed as it's read-only
	queues, ok := b.messageQueues[streamID]
	if !ok {
		// if there are no groups - discard message
		return "", nil
	}

	// we have to queue up the message to ALL EXISTING groups of a stream
	var errs error
	for groupName, queue := range queues {
		select {
		case queue <- m:
			continue
		default:
			errs = multierror.Append(errs, fmt.Errorf("queue for group '%s' is full", groupName))
		}
	}

	return m.id, errs
}

// messages returns a read-only stream that can be used to receive messages of a stream under a specific group.
// If no such stream exists yet, a new one is created, otherwise, the existing one is returned.
func (b *MemoryBroker) messages(streamID string, groupName string) <-chan message {
	// the lock is for writing in the messageQueues map
	b.mx.Lock()
	defer b.mx.Unlock()

	groups, ok := b.messageQueues[streamID]
	if !ok {
		groups = make(map[string]chan message)
		b.messageQueues[streamID] = groups
	}

	groupQueue, ok := groups[groupName]
	if !ok {
		groupQueue = make(chan message, b.queueSize)
		groups[groupName] = groupQueue
	}

	return groupQueue
}
