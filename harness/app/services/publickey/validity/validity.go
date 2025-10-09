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

package validity

import (
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/gotidy/ptr"
)

type Period struct {
	Invalid   bool
	CreatedAt time.Time
	Duration  time.Duration
}

func FromSignature(sig *packet.Signature) Period {
	v := Period{CreatedAt: sig.CreationTime}

	if sig.SigLifetimeSecs != nil && *sig.SigLifetimeSecs != 0 {
		v.Duration = time.Duration(*sig.SigLifetimeSecs) * time.Second
	}

	return v
}

func FromPublicKey(key *packet.PublicKey, sig *packet.Signature) Period {
	v := Period{CreatedAt: key.CreationTime}

	if sig.KeyLifetimeSecs != nil && *sig.KeyLifetimeSecs != 0 {
		v.Duration = time.Duration(*sig.KeyLifetimeSecs) * time.Second
	}

	return v
}

func (v *Period) Invalidate() {
	v.Invalid = true
	v.Duration = 0
}

func (v *Period) Intersect(v2 Period) {
	if v.Invalid {
		return
	}

	if v2.Invalid {
		v.Invalidate()
		return
	}

	createdAt := v.CreatedAt
	if createdAt.Before(v2.CreatedAt) {
		createdAt = v2.CreatedAt
	}

	if v.Duration == 0 && v2.Duration == 0 {
		v.CreatedAt = createdAt
		return
	}

	var duration time.Duration

	switch {
	case v.Duration == 0:
		duration = v2.CreatedAt.Add(v2.Duration).Sub(createdAt)
	case v2.Duration == 0:
		duration = v.CreatedAt.Add(v.Duration).Sub(createdAt)
	default:
		end1 := v.CreatedAt.Add(v.Duration)
		end2 := v2.CreatedAt.Add(v2.Duration)
		if end1.After(end2) {
			duration = end2.Sub(createdAt)
		} else {
			duration = end1.Sub(createdAt)
		}
	}

	if duration < 0 {
		v.Invalidate()
		return
	}

	v.CreatedAt = createdAt
	v.Duration = duration
}

func (v *Period) Revoke(revocations []*packet.Signature) {
	if v.Invalid {
		return // The period is already invalid - nothing to do.
	}

	for _, rev := range revocations {
		if rev.RevocationReason != nil && *rev.RevocationReason == packet.KeyCompromised {
			// If the key is compromised, the key is considered revoked even before the revocation date.
			v.Invalidate()
			return
		}

		revokedFrom := rev.CreationTime // Note: Lifetime (rev.SigLifetimeSecs) isn't used in revocations.
		duration := revokedFrom.Sub(v.CreatedAt)

		if duration <= 0 {
			v.Invalidate()
			return
		}

		if v.Duration == 0 {
			v.Duration = duration
		} else if v.Duration > duration {
			v.Duration = duration
		}
	}
}

func (v *Period) String() string {
	if v.Invalid {
		return "not-valid"
	}

	var sb strings.Builder

	sb.WriteString("from=")
	sb.WriteString(v.CreatedAt.Format(time.RFC3339))

	if v.Duration > 0 {
		sb.WriteString(" to=")
		sb.WriteString(v.CreatedAt.Add(v.Duration).Format(time.RFC3339))
	}

	return sb.String()
}

func (v *Period) Milliseconds() (int64, *int64) {
	var (
		validFrom int64
		validTo   *int64
	)

	validFrom = v.CreatedAt.UnixMilli()
	if v.Invalid {
		return validFrom, &validFrom // zero duration
	}

	if v.Duration > 0 {
		validTo = ptr.Int64(v.CreatedAt.Add(v.Duration).UnixMilli())
	}

	return validFrom, validTo
}
