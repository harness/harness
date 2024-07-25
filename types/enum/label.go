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

type LabelType string

func (LabelType) Enum() []interface{}            { return toInterfaceSlice(LabelTypes) }
func (t LabelType) Sanitize() (LabelType, bool)  { return Sanitize(t, GetAllLabelTypes) }
func GetAllLabelTypes() ([]LabelType, LabelType) { return LabelTypes, LabelTypeStatic }

const (
	LabelTypeStatic  LabelType = "static"
	LabelTypeDynamic LabelType = "dynamic"
)

var LabelTypes = sortEnum([]LabelType{
	LabelTypeStatic,
	LabelTypeDynamic,
})

type LabelColor string

func (LabelColor) Enum() []interface{}              { return toInterfaceSlice(LabelColors) }
func (t LabelColor) Sanitize() (LabelColor, bool)   { return Sanitize(t, GetAllLabelColors) }
func GetAllLabelColors() ([]LabelColor, LabelColor) { return LabelColors, LabelColorBackground }

const (
	LabelColorBackground LabelColor = "background"
	LabelColorStroke     LabelColor = "stroke"
	LabelColorText       LabelColor = "text"
	LabelColorAccent     LabelColor = "accent"
	LabelColorRed        LabelColor = "red"
	LabelColorGreen      LabelColor = "green"
	LabelColorYellow     LabelColor = "yellow"
	LabelColorBlue       LabelColor = "blue"
	LabelColorPink       LabelColor = "pink"
	LabelColorPurple     LabelColor = "purple"
	LabelColorViolet     LabelColor = "violet"
	LabelColorIndigo     LabelColor = "indigo"
	LabelColorCyan       LabelColor = "cyan"
	LabelColorOrange     LabelColor = "orange"
	LabelColorBrown      LabelColor = "brown"
	LabelColorMint       LabelColor = "mint"
	LabelColorLime       LabelColor = "lime"
)

var LabelColors = sortEnum([]LabelColor{
	LabelColorBackground,
	LabelColorStroke,
	LabelColorText,
	LabelColorAccent,
	LabelColorRed,
	LabelColorGreen,
	LabelColorYellow,
	LabelColorBlue,
	LabelColorPink,
	LabelColorPurple,
	LabelColorViolet,
	LabelColorIndigo,
	LabelColorCyan,
	LabelColorOrange,
	LabelColorBrown,
	LabelColorMint,
	LabelColorLime,
})
