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
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/errors"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/rs/zerolog/log"
)

const defaultLabelValueColor = enum.LabelColorGreen

// Label is label migrate.
type Label struct {
	labelStore      store.LabelStore
	labelValueStore store.LabelValueStore
	spaceStore      store.SpaceStore
	tx              dbtx.Transactor
}

func NewLabel(
	labelStore store.LabelStore,
	labelValueStore store.LabelValueStore,
	spaceStore store.SpaceStore,
	tx dbtx.Transactor,
) *Label {
	return &Label{
		labelStore:      labelStore,
		labelValueStore: labelValueStore,
		spaceStore:      spaceStore,
		tx:              tx,
	}
}

//nolint:gocognit
func (migrate Label) Import(
	ctx context.Context,
	migrator types.Principal,
	space *types.SpaceCore,
	extLabels []*ExternalLabel,
) ([]*types.Label, error) {
	labels := make([]*types.Label, len(extLabels))
	labelValues := make(map[string][]string)

	spaceIDs, err := migrate.spaceStore.GetAncestorIDs(ctx, space.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get space ids hierarchy: %w", err)
	}
	scope := int64(len(spaceIDs))
	for i, extLabel := range extLabels {
		label, err := convertLabelWithSanitization(ctx, migrator, space.ID, scope, *extLabel)
		if err != nil {
			return nil, fmt.Errorf("failed to sanitize and convert external label input: %w", err)
		}
		labels[i] = label

		if extLabel.Value != "" {
			valueIn := &types.DefineValueInput{
				Value: extLabel.Value,
				Color: defaultLabelValueColor,
			}
			if err := valueIn.Sanitize(); err != nil {
				return nil, fmt.Errorf("failed to sanitize external label value input: %w", err)
			}
			labelValues[label.Key] = append(labelValues[label.Key], valueIn.Value)
		}
	}

	err = migrate.tx.WithTx(ctx, func(ctx context.Context) error {
		for _, label := range labels {
			err := migrate.defineLabelsAndValues(ctx, migrator.ID, space.ID, label, labelValues[label.Key])
			if err != nil {
				return fmt.Errorf("failed to define labels and/or values: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to define external labels: %w", err)
	}

	return labels, nil
}

func (migrate Label) defineLabelsAndValues(
	ctx context.Context,
	migratorID int64,
	spaceID int64,
	labelIn *types.Label,
	extValues []string) error {
	var label *types.Label
	var err error
	// try to find the label first as it might have been defined already.
	label, err = migrate.labelStore.Find(ctx, &spaceID, nil, labelIn.Key)
	if errors.Is(err, gitness_store.ErrResourceNotFound) {
		err := migrate.labelStore.Define(ctx, labelIn)
		if err != nil {
			return fmt.Errorf("failed to define label: %w", err)
		}
		label = labelIn
	} else if err != nil {
		return fmt.Errorf("failed to find the label: %w", err)
	}

	values, err := migrate.labelValueStore.List(
		ctx,
		label.ID,
		types.ListQueryFilter{
			Pagination: types.Pagination{
				Size: database.MaxLabelValueSize,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to list label values: %w", err)
	}

	now := time.Now().UnixMilli()
	existingValues := make(map[string]bool)
	for _, val := range values {
		existingValues[val.Value] = true
	}

	var newValuesCount int
	for _, val := range extValues {
		if existingValues[val] {
			continue
		}
		// define new label values
		if err := migrate.labelValueStore.Define(ctx, &types.LabelValue{
			LabelID:   label.ID,
			Value:     val,
			Color:     defaultLabelValueColor,
			Created:   now,
			Updated:   now,
			CreatedBy: migratorID,
			UpdatedBy: migratorID,
		}); err != nil {
			return fmt.Errorf("failed to create label value: %w", err)
		}
		newValuesCount++
	}

	_, err = migrate.labelStore.IncrementValueCount(ctx, label.ID, newValuesCount)
	if err != nil {
		return fmt.Errorf("failed to update label value count: %w", err)
	}

	return nil
}

func convertLabelWithSanitization(
	ctx context.Context,
	migrator types.Principal,
	spaceID int64,
	scope int64,
	extLabel ExternalLabel,
) (*types.Label, error) {
	in := &types.DefineLabelInput{
		Key:         extLabel.Name,
		Type:        enum.LabelTypeStatic,
		Description: extLabel.Description,
		Color:       findClosestColor(ctx, extLabel.Color),
	}

	if err := in.Sanitize(); err != nil {
		return nil, fmt.Errorf("failed to sanitize external labels input: %w", err)
	}

	now := time.Now().UnixMilli()
	label := &types.Label{
		SpaceID:     &spaceID,
		RepoID:      nil,
		Scope:       scope,
		Key:         in.Key,
		Color:       in.Color,
		Description: in.Description,
		Type:        in.Type,
		Created:     now,
		Updated:     now,
		CreatedBy:   migrator.ID,
		UpdatedBy:   migrator.ID,
	}

	return label, nil
}

// findClosestColor finds the visually closest color to a provided value using go-colorful library.
func findClosestColor(ctx context.Context, extColor string) enum.LabelColor {
	supportedColors, defColor := enum.GetAllLabelColors()
	if len(extColor) > 1 && string(extColor[0]) != "#" {
		extColor = "#" + extColor
	}
	targetColor, err := colorful.Hex(strings.ToUpper(extColor))
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to convert the color to hex. choosing default color instead.")
		return defColor
	}
	closestColor := supportedColors[0]
	minDistance := targetColor.DistanceLab(convertToColorful(supportedColors[0]))

	for _, labelColor := range supportedColors[1:] {
		distance := targetColor.DistanceLab(convertToColorful(labelColor))
		if distance < minDistance {
			closestColor = labelColor
			minDistance = distance
		}
	}

	return closestColor
}

// convertToColorful converts Gitness supported label colors to the hex (text value) using web/src/utils:ColorDetails.
func convertToColorful(color enum.LabelColor) colorful.Color {
	var hexColor colorful.Color
	switch color {
	case enum.LabelColorRed:
		hexColor, _ = colorful.Hex("#C7292F")
	case enum.LabelColorGreen:
		hexColor, _ = colorful.Hex("#16794C")
	case enum.LabelColorYellow:
		hexColor, _ = colorful.Hex("#92582D")
	case enum.LabelColorBlue:
		hexColor, _ = colorful.Hex("#236E93")
	case enum.LabelColorPink:
		hexColor, _ = colorful.Hex("#C41B87")
	case enum.LabelColorPurple:
		hexColor, _ = colorful.Hex("#9C2AAD")
	case enum.LabelColorViolet:
		hexColor, _ = colorful.Hex("#5645AF")
	case enum.LabelColorIndigo:
		hexColor, _ = colorful.Hex("#3250B2")
	case enum.LabelColorCyan:
		hexColor, _ = colorful.Hex("#0B7792")
	case enum.LabelColorOrange:
		hexColor, _ = colorful.Hex("#995137")
	case enum.LabelColorBrown:
		hexColor, _ = colorful.Hex("#805C43")
	case enum.LabelColorMint:
		hexColor, _ = colorful.Hex("#247469")
	case enum.LabelColorLime:
		hexColor, _ = colorful.Hex("#586729")
	default:
		// blue is the default color on Gitness
		hexColor, _ = colorful.Hex("#236E93")
	}

	return hexColor
}
