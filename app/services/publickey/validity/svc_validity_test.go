// Copyright 2026 Harness, Inc.
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

package validity

import (
	"testing"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var svcBase = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

func svcDays(n int) time.Duration { return time.Duration(n) * 24 * time.Hour }

func svcLifetime(secs uint32) *uint32 { return &secs }

func TestSvcFromSignature(t *testing.T) {
	tests := []struct {
		name         string
		lifetime     *uint32
		wantDuration time.Duration
	}{
		{"no lifetime", nil, 0},
		{"zero lifetime means forever", svcLifetime(0), 0},
		{"one hour", svcLifetime(3600), time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := FromSignature(&packet.Signature{CreationTime: svcBase, SigLifetimeSecs: tt.lifetime})
			assert.Equal(t, svcBase, p.CreatedAt)
			assert.Equal(t, tt.wantDuration, p.Duration)
			assert.False(t, p.Invalid)
		})
	}
}

func TestSvcFromPublicKey(t *testing.T) {
	key := &packet.PublicKey{CreationTime: svcBase}

	p := FromPublicKey(key, &packet.Signature{KeyLifetimeSecs: svcLifetime(86400)})
	assert.Equal(t, svcBase, p.CreatedAt)
	assert.Equal(t, svcDays(1), p.Duration)

	p = FromPublicKey(key, &packet.Signature{})
	assert.Zero(t, p.Duration)

	p = FromPublicKey(key, &packet.Signature{KeyLifetimeSecs: svcLifetime(0)})
	assert.Zero(t, p.Duration)
}

func TestSvcInvalidate(t *testing.T) {
	p := Period{CreatedAt: svcBase, Duration: time.Hour}
	p.Invalidate()
	assert.True(t, p.Invalid)
	assert.Zero(t, p.Duration)
}

func TestSvcIntersect(t *testing.T) {
	tests := []struct {
		name          string
		a, b          Period
		wantInvalid   bool
		wantCreatedAt time.Time
		wantDuration  time.Duration
	}{
		{
			name: "both unlimited keeps the later start",
			a:    Period{CreatedAt: svcBase}, b: Period{CreatedAt: svcBase.Add(svcDays(2))},
			wantCreatedAt: svcBase.Add(svcDays(2)),
		},
		{
			name: "first limited",
			a:    Period{CreatedAt: svcBase, Duration: svcDays(10)}, b: Period{CreatedAt: svcBase.Add(svcDays(4))},
			wantCreatedAt: svcBase.Add(svcDays(4)), wantDuration: svcDays(6),
		},
		{
			name: "second limited",
			a:    Period{CreatedAt: svcBase.Add(svcDays(1))}, b: Period{CreatedAt: svcBase, Duration: svcDays(5)},
			wantCreatedAt: svcBase.Add(svcDays(1)), wantDuration: svcDays(4),
		},
		{
			name: "both limited uses the earlier end (first ends later)",
			a:    Period{CreatedAt: svcBase, Duration: svcDays(10)}, b: Period{CreatedAt: svcBase, Duration: svcDays(3)},
			wantCreatedAt: svcBase, wantDuration: svcDays(3),
		},
		{
			name: "both limited uses the earlier end (second ends later)",
			a:    Period{CreatedAt: svcBase, Duration: svcDays(2)}, b: Period{CreatedAt: svcBase, Duration: svcDays(9)},
			wantCreatedAt: svcBase, wantDuration: svcDays(2),
		},
		{
			name:        "disjoint periods are invalid",
			a:           Period{CreatedAt: svcBase, Duration: svcDays(1)},
			b:           Period{CreatedAt: svcBase.Add(svcDays(5)), Duration: svcDays(1)},
			wantInvalid: true, wantCreatedAt: svcBase,
		},
		{
			name: "invalid receiver stays invalid",
			a:    Period{CreatedAt: svcBase, Invalid: true}, b: Period{CreatedAt: svcBase},
			wantInvalid: true, wantCreatedAt: svcBase,
		},
		{
			name: "invalid argument invalidates the receiver",
			a:    Period{CreatedAt: svcBase, Duration: svcDays(3)}, b: Period{CreatedAt: svcBase, Invalid: true},
			wantInvalid: true, wantCreatedAt: svcBase,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.a.Intersect(tt.b)
			assert.Equal(t, tt.wantInvalid, tt.a.Invalid)
			assert.Equal(t, tt.wantCreatedAt, tt.a.CreatedAt)
			assert.Equal(t, tt.wantDuration, tt.a.Duration)
		})
	}
}

func svcRevocation(reason packet.ReasonForRevocation, at time.Time) *packet.Signature {
	return &packet.Signature{CreationTime: at, RevocationReason: &reason}
}

func TestSvcRevoke(t *testing.T) {
	tests := []struct {
		name         string
		period       Period
		revocations  []*packet.Signature
		wantInvalid  bool
		wantDuration time.Duration
	}{
		{"no revocations", Period{CreatedAt: svcBase}, nil, false, 0},
		{
			"already invalid", Period{CreatedAt: svcBase, Invalid: true},
			[]*packet.Signature{svcRevocation(packet.KeyRetired, svcBase.Add(svcDays(1)))}, true, 0,
		},
		{
			"compromised invalidates regardless of date", Period{CreatedAt: svcBase},
			[]*packet.Signature{svcRevocation(packet.KeyCompromised, svcBase.Add(svcDays(30)))}, true, 0,
		},
		{
			"retired limits an unlimited period", Period{CreatedAt: svcBase},
			[]*packet.Signature{svcRevocation(packet.KeyRetired, svcBase.Add(svcDays(7)))}, false, svcDays(7),
		},
		{
			"earlier revocation shortens the period", Period{CreatedAt: svcBase, Duration: svcDays(30)},
			[]*packet.Signature{svcRevocation(packet.KeySuperseded, svcBase.Add(svcDays(7)))}, false, svcDays(7),
		},
		{
			"later revocation keeps the period", Period{CreatedAt: svcBase, Duration: svcDays(3)},
			[]*packet.Signature{svcRevocation(packet.KeySuperseded, svcBase.Add(svcDays(7)))}, false, svcDays(3),
		},
		{
			"revocation before creation invalidates", Period{CreatedAt: svcBase},
			[]*packet.Signature{svcRevocation(packet.KeyRetired, svcBase.Add(-svcDays(1)))}, true, 0,
		},
		{
			"revocation without a reason is still honored", Period{CreatedAt: svcBase},
			[]*packet.Signature{{CreationTime: svcBase.Add(svcDays(2))}}, false, svcDays(2),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.period.Revoke(tt.revocations)
			assert.Equal(t, tt.wantInvalid, tt.period.Invalid)
			assert.Equal(t, tt.wantDuration, tt.period.Duration)
		})
	}
}

func TestSvcStringAndMilliseconds(t *testing.T) {
	open := Period{CreatedAt: svcBase}
	assert.Equal(t, "from=2025-01-01T00:00:00Z", open.String())
	from, to := open.Milliseconds()
	assert.Equal(t, svcBase.UnixMilli(), from)
	assert.Nil(t, to)

	limited := Period{CreatedAt: svcBase, Duration: svcDays(1)}
	assert.Equal(t, "from=2025-01-01T00:00:00Z to=2025-01-02T00:00:00Z", limited.String())
	from, to = limited.Milliseconds()
	assert.Equal(t, svcBase.UnixMilli(), from)
	require.NotNil(t, to)
	assert.Equal(t, svcBase.Add(svcDays(1)).UnixMilli(), *to)

	invalid := Period{CreatedAt: svcBase, Invalid: true}
	assert.Equal(t, "not-valid", invalid.String())
	from, to = invalid.Milliseconds()
	require.NotNil(t, to)
	assert.Equal(t, from, *to, "an invalid period ends the moment it starts")
}
