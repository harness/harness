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

package migrate

import (
	"context"
	"testing"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestSvaFindClosestColor(t *testing.T) {
	t.Parallel()

	_, defaultColor := enum.GetAllLabelColors()

	tests := []struct {
		name  string
		input string
		want  enum.LabelColor
	}{
		{name: "exact red", input: "#C7292F", want: enum.LabelColorRed},
		{name: "exact green without hash", input: "16794c", want: enum.LabelColorGreen},
		{name: "exact yellow", input: "#92582D", want: enum.LabelColorYellow},
		{name: "exact blue", input: "#236E93", want: enum.LabelColorBlue},
		{name: "exact pink", input: "#C41B87", want: enum.LabelColorPink},
		{name: "exact purple", input: "#9C2AAD", want: enum.LabelColorPurple},
		{name: "exact violet", input: "#5645AF", want: enum.LabelColorViolet},
		{name: "exact indigo", input: "#3250B2", want: enum.LabelColorIndigo},
		{name: "exact cyan", input: "#0B7792", want: enum.LabelColorCyan},
		{name: "exact orange", input: "#995137", want: enum.LabelColorOrange},
		{name: "exact brown", input: "#805C43", want: enum.LabelColorBrown},
		{name: "exact mint", input: "#247469", want: enum.LabelColorMint},
		{name: "exact lime", input: "#586729", want: enum.LabelColorLime},
		{name: "close to red", input: "#FF0000", want: enum.LabelColorRed},
		{name: "invalid hex falls back to default", input: "not-a-color", want: defaultColor},
		{name: "empty falls back to default", input: "", want: defaultColor},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, test.want, findClosestColor(context.Background(), test.input))
		})
	}
}

func TestSvaConvertToColorfulDefault(t *testing.T) {
	t.Parallel()

	blue := convertToColorful(enum.LabelColorBlue)
	require.Equal(t, blue, convertToColorful(enum.LabelColor("unknown")))
	require.NotEqual(t, blue, convertToColorful(enum.LabelColorRed))
}

func TestSvaConvertLabelWithSanitization(t *testing.T) {
	t.Parallel()

	migrator := types.Principal{ID: 7}

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		label, err := convertLabelWithSanitization(
			context.Background(), migrator, 3, 2,
			ExternalLabel{Name: "bug", Description: "a bug", Color: "#C7292F"},
		)
		require.NoError(t, err)
		require.Equal(t, "bug", label.Key)
		require.Equal(t, "a bug", label.Description)
		require.Equal(t, enum.LabelColorRed, label.Color)
		require.Equal(t, enum.LabelTypeStatic, label.Type)
		require.Equal(t, int64(3), *label.SpaceID)
		require.Nil(t, label.RepoID)
		require.Equal(t, int64(2), label.Scope)
		require.Equal(t, int64(7), label.CreatedBy)
		require.Equal(t, int64(7), label.UpdatedBy)
		require.NotZero(t, label.Created)
	})

	t.Run("empty key is rejected", func(t *testing.T) {
		t.Parallel()

		_, err := convertLabelWithSanitization(context.Background(), migrator, 3, 2, ExternalLabel{})
		require.Error(t, err)
	})
}

func TestSvaLabelImport(t *testing.T) {
	t.Parallel()

	migrator := types.Principal{ID: 7}
	space := &types.SpaceCore{ID: 3}

	t.Run("defines labels and values", func(t *testing.T) {
		t.Parallel()

		labels := svaNewLabelStore()
		values := svaNewLabelValueStore()
		spaces := &svaSpaceStore{ancestors: []int64{1, 2, 3}}

		got, err := NewLabel(labels, values, spaces, svaTx{}).Import(
			context.Background(), migrator, space, []*ExternalLabel{
				{Name: "bug", Color: "#C7292F", Value: "high"},
				{Name: "bug", Color: "#C7292F", Value: "low"},
				{Name: "feature", Color: "#16794C"},
			})
		require.NoError(t, err)
		require.Len(t, got, 3)
		require.Equal(t, int64(3), got[0].Scope)

		bug := labels.labels["bug"]
		require.NotNil(t, bug)
		require.Len(t, values.values[bug.ID], 2)
		require.Equal(t, 2, labels.increments[bug.ID])
		require.Equal(t, enum.LabelColorGreen, values.values[bug.ID][0].Color)
		require.Equal(t, int64(7), values.values[bug.ID][0].CreatedBy)
		require.Contains(t, labels.labels, "feature")
		require.Empty(t, values.values[labels.labels["feature"].ID])
	})

	t.Run("reuses existing labels and skips existing values", func(t *testing.T) {
		t.Parallel()

		labels := svaNewLabelStore()
		labels.labels["bug"] = &types.Label{ID: 42, Key: "bug"}
		values := svaNewLabelValueStore()
		values.values[42] = []*types.LabelValue{{LabelID: 42, Value: "high"}}

		_, err := NewLabel(labels, values, &svaSpaceStore{}, svaTx{}).Import(
			context.Background(), migrator, space, []*ExternalLabel{
				{Name: "bug", Value: "high"},
				{Name: "bug", Value: "low"},
			})
		require.NoError(t, err)
		require.Empty(t, labels.defined)
		require.Len(t, values.values[42], 2)
		require.Equal(t, 1, labels.increments[42])
	})

	t.Run("no labels", func(t *testing.T) {
		t.Parallel()

		got, err := NewLabel(svaNewLabelStore(), svaNewLabelValueStore(), &svaSpaceStore{}, svaTx{}).Import(
			context.Background(), migrator, space, nil)
		require.NoError(t, err)
		require.Empty(t, got)
	})

	errTests := []struct {
		name   string
		labels []*ExternalLabel
		setup  func(l *svaLabelStore, v *svaLabelValueStore, s *svaSpaceStore) svaTx
	}{
		{
			name:   "ancestors lookup",
			labels: []*ExternalLabel{{Name: "bug"}},
			setup: func(_ *svaLabelStore, _ *svaLabelValueStore, s *svaSpaceStore) svaTx {
				s.err = errSvaBoom
				return svaTx{}
			},
		},
		{
			name:   "invalid label",
			labels: []*ExternalLabel{{Name: ""}},
			setup:  func(*svaLabelStore, *svaLabelValueStore, *svaSpaceStore) svaTx { return svaTx{} },
		},
		{
			name:   "transaction",
			labels: []*ExternalLabel{{Name: "bug"}},
			setup: func(*svaLabelStore, *svaLabelValueStore, *svaSpaceStore) svaTx {
				return svaTx{err: errSvaBoom}
			},
		},
		{
			name:   "label lookup",
			labels: []*ExternalLabel{{Name: "bug"}},
			setup: func(l *svaLabelStore, _ *svaLabelValueStore, _ *svaSpaceStore) svaTx {
				l.findErr = errSvaBoom
				return svaTx{}
			},
		},
		{
			name:   "label define",
			labels: []*ExternalLabel{{Name: "bug"}},
			setup: func(l *svaLabelStore, _ *svaLabelValueStore, _ *svaSpaceStore) svaTx {
				l.defineErr = errSvaBoom
				return svaTx{}
			},
		},
		{
			name:   "value list",
			labels: []*ExternalLabel{{Name: "bug"}},
			setup: func(_ *svaLabelStore, v *svaLabelValueStore, _ *svaSpaceStore) svaTx {
				v.listErr = errSvaBoom
				return svaTx{}
			},
		},
		{
			name:   "value define",
			labels: []*ExternalLabel{{Name: "bug", Value: "high"}},
			setup: func(_ *svaLabelStore, v *svaLabelValueStore, _ *svaSpaceStore) svaTx {
				v.defineErr = errSvaBoom
				return svaTx{}
			},
		},
		{
			name:   "value count",
			labels: []*ExternalLabel{{Name: "bug"}},
			setup: func(l *svaLabelStore, _ *svaLabelValueStore, _ *svaSpaceStore) svaTx {
				l.incrementErr = errSvaBoom
				return svaTx{}
			},
		},
	}

	for _, test := range errTests {
		t.Run("error "+test.name, func(t *testing.T) {
			t.Parallel()

			labels, values, spaces := svaNewLabelStore(), svaNewLabelValueStore(), &svaSpaceStore{}
			tx := test.setup(labels, values, spaces)

			_, err := NewLabel(labels, values, spaces, tx).Import(context.Background(), migrator, space, test.labels)
			require.Error(t, err)
		})
	}
}
