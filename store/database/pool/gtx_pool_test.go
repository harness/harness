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

package pool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type gtxDB struct {
	maxOpen     int
	maxIdle     int
	maxLifetime time.Duration
	calls       int
}

func (d *gtxDB) SetMaxOpenConns(n int)              { d.maxOpen = n; d.calls++ }
func (d *gtxDB) SetMaxIdleConns(n int)              { d.maxIdle = n; d.calls++ }
func (d *gtxDB) SetConnMaxLifetime(l time.Duration) { d.maxLifetime = l; d.calls++ }

func TestGtxApply(t *testing.T) {
	valid := Config{MaxOpenConns: 20, MaxIdleConns: 5, ConnMaxLifetime: time.Minute}

	tests := []struct {
		name    string
		mutate  func(c *Config)
		wantErr string
	}{
		{name: "valid", mutate: func(*Config) {}},
		{
			name: "unset open connections", mutate: func(c *Config) { c.MaxOpenConns = 0 },
			wantErr: "MaxOpenConns must be > 0, got 0",
		},
		{name: "negative open connections", mutate: func(c *Config) { c.MaxOpenConns = -1 }, wantErr: "MaxOpenConns"},
		{
			name: "unset idle connections", mutate: func(c *Config) { c.MaxIdleConns = 0 },
			wantErr: "MaxIdleConns must be > 0, got 0",
		},
		{
			name: "unset lifetime", mutate: func(c *Config) { c.ConnMaxLifetime = 0 },
			wantErr: "ConnMaxLifetime must be > 0, got 0s",
		},
		{name: "negative lifetime", mutate: func(c *Config) { c.ConnMaxLifetime = -time.Second }, wantErr: "ConnMaxLifetime"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := valid
			test.mutate(&cfg)

			db := &gtxDB{}
			err := Apply(db, cfg)
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				assert.Zero(t, db.calls, "nothing is applied if the config is invalid")
				return
			}

			require.NoError(t, err)
			assert.Equal(t, 20, db.maxOpen)
			assert.Equal(t, 5, db.maxIdle)
			assert.Equal(t, time.Minute, db.maxLifetime)
		})
	}
}
