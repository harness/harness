// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"github.com/go-swagger/go-swagger/fixtures/goparsing/classification/transitive/mods"
	"github.com/go-swagger/go-swagger/strfmt"
)

// NoModel is a struct without an annotation.
// NoModel exists in a package
// but is not annotated with the swagger model annotations
// so it should now show up in a test.
//
type NoModel struct {
	// ID of this no model instance.
	// ids in this application start at 11 and are smaller than 1000
	//
	// required: true
	// minimum: > 10
	// maximum: < 1000
	ID int64 `json:"id"`

	Ignored      string `json:"-"`
	IgnoredOther string `json:"-,omitempty"`

	// The Score of this model
	//
	// required: true
	// minimum: 3
	// maximum: 45
	// multiple of: 3
	Score int32 `json:"score"`

	// Name of this no model instance
	//
	// min length: 4
	// max length: 50
	// pattern: [A-Za-z0-9-.]*
	// required: true
	Name string `json:"name"`

	// Created holds the time when this entry was created
	//
	// required: false
	// read only: true
	Created strfmt.DateTime `json:"created"`

	// a FooSlice has foos which are strings
	//
	// min items: 3
	// max items: 10
	// unique: true
	// items.minLength: 3
	// items.maxLength: 10
	// items.pattern: \w+
	FooSlice []string `json:"foo_slice"`

	// a BarSlice has bars which are strings
	//
	// min items: 3
	// max items: 10
	// unique: true
	// items.minItems: 4
	// items.maxItems: 9
	// items.items.minItems: 5
	// items.items.maxItems: 8
	// items.items.items.minLength: 3
	// items.items.items.maxLength: 10
	// items.items.items.pattern: \w+
	BarSlice [][][]string `json:"bar_slice"`

	// the items for this order
	Items []struct {
		// ID of this no model instance.
		// ids in this application start at 11 and are smaller than 1000
		//
		// required: true
		// minimum: > 10
		// maximum: < 1000
		ID int32 `json:"id"`

		// The Pet to add to this NoModel items bucket.
		// Pets can appear more than once in the bucket
		//
		// required: true
		Pet *mods.Pet `json:"pet"`

		// The amount of pets to add to this bucket.
		//
		// required: true
		// minimum: 1
		// maximum: 10
		Quantity int16 `json:"quantity"`

		// Notes to add to this item.
		// This can be used to add special instructions.
		//
		// required: false
		Notes string `json:"notes"`

		AlsoIgnored string `json:"-"`
	} `json:"items"`
}

// A OtherTypes struct contains type aliases
type OtherTypes struct {
	Named       SomeStringType `json:"named"`
	Numbered    SomeIntType    `json:"numbered"`
	Dated       SomeTimeType   `json:"dated"`
	Timed       SomeTimedType  `json:"timed"`
	Petted      SomePettedType `json:"petted"`
	Somethinged SomethingType  `json:"somethinged"`

	ManyNamed       SomeStringsType `json:"manyNamed"`
	ManyNumbered    SomeIntsType    `json:"manyNumbered"`
	ManyDated       SomeTimesType   `json:"manyDated"`
	ManyTimed       SomeTimedsType  `json:"manyTimed"`
	ManyPetted      SomePettedsType `json:"manyPetted"`
	ManySomethinged SomethingsType  `json:"manySomethinged"`

	Nameds       []SomeStringType `json:"nameds"`
	Numbereds    []SomeIntType    `json:"numbereds"`
	Dateds       []SomeTimeType   `json:"dateds"`
	Timeds       []SomeTimedType  `json:"timeds"`
	Petteds      []SomePettedType `json:"petteds"`
	Somethingeds []SomethingType  `json:"somethingeds"`

	ModsNamed    mods.SomeStringType `json:"modsNamed"`
	ModsNumbered mods.SomeIntType    `json:"modsNumbered"`
	ModsDated    mods.SomeTimeType   `json:"modsDated"`
	ModsTimed    mods.SomeTimedType  `json:"modsTimed"`
	ModsPetted   mods.SomePettedType `json:"modsPetted"`

	ModsNameds    []mods.SomeStringType `json:"modsNameds"`
	ModsNumbereds []mods.SomeIntType    `json:"modsNumbereds"`
	ModsDateds    []mods.SomeTimeType   `json:"modsDateds"`
	ModsTimeds    []mods.SomeTimedType  `json:"modsTimeds"`
	ModsPetteds   []mods.SomePettedType `json:"modsPetteds"`

	ManyModsNamed    mods.SomeStringsType `json:"manyModsNamed"`
	ManyModsNumbered mods.SomeIntsType    `json:"manyModsNumbered"`
	ManyModsDated    mods.SomeTimesType   `json:"manyModsDated"`
	ManyModsTimed    mods.SomeTimedsType  `json:"manyModsTimed"`
	ManyModsPetted   mods.SomePettedsType `json:"manyModsPetted"`
}

// A SimpleOne is a model with a few simple fields
type SimpleOne struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Age  int32  `json:"age"`
}

// A ComplexerOne is composed of a SimpleOne and some extra fields
type ComplexerOne struct {
	SimpleOne
	mods.NotSelected
	mods.Notable
	CreatedAt strfmt.DateTime `json:"createdAt"`
}

// An OverridingOne is composed of a SimpleOne and overrides a field
type OverridingOne struct {
	SimpleOne
	Age int64
}

// An AllOfModel is composed out of embedded structs but it should build
// an allOf property
type AllOfModel struct {
	// swagger:allOf
	SimpleOne
	// swagger:allOf
	mods.Notable

	Something // not annotated with anything, so should be included

	CreatedAt strfmt.DateTime `json:"createdAt"`
}

// A PrimateModel is a struct with nothing but primitives.
//
// It only has values 1 level deep and each of those is of a very simple
// builtin type.
type PrimateModel struct {
	A bool `json:"a"`

	B rune   `json:"b"`
	C string `json:"c"`

	D int   `json:"d"`
	E int8  `json:"e"`
	F int16 `json:"f"`
	G int32 `json:"g"`
	H int64 `json:"h"`

	I uint   `json:"i"`
	J uint8  `json:"j"`
	K uint16 `json:"k"`
	L uint32 `json:"l"`
	M uint64 `json:"m"`

	N float32 `json:"n"`
	O float64 `json:"o"`
}

// A FormattedModel is a struct with only strfmt types
//
// It only has values 1 level deep and is used for testing the conversion
type FormattedModel struct {
	A strfmt.Base64     `json:"a"`
	B strfmt.CreditCard `json:"b"`
	C strfmt.Date       `json:"c"`
	D strfmt.DateTime   `json:"d"`
	E strfmt.Duration   `json:"e"`
	F strfmt.Email      `json:"f"`
	G strfmt.HexColor   `json:"g"`
	H strfmt.Hostname   `json:"h"`
	I strfmt.IPv4       `json:"i"`
	J strfmt.IPv6       `json:"j"`
	K strfmt.ISBN       `json:"k"`
	L strfmt.ISBN10     `json:"l"`
	M strfmt.ISBN13     `json:"m"`
	N strfmt.RGBColor   `json:"n"`
	O strfmt.SSN        `json:"o"`
	P strfmt.URI        `json:"p"`
	Q strfmt.UUID       `json:"q"`
	R strfmt.UUID3      `json:"r"`
	S strfmt.UUID4      `json:"s"`
	T strfmt.UUID5      `json:"t"`
}

// A SimpleComplexModel is a struct with only other struct types
//
// It doesn't have slices or arrays etc but only complex types
// so also no primitives or string formatters
type SimpleComplexModel struct {
	Top Something `json:"top"`

	NotSel mods.NotSelected `json:"notSel"`

	Emb struct {
		CID int64  `json:"cid"`
		Baz string `json:"baz"`
	} `json:"emb"`
}

// Pointdexter is a struct with only pointers
type Pointdexter struct {
	ID   *int64        `json:"id"`
	Name *string       `json:"name"`
	T    *strfmt.UUID5 `json:"t"`
	Top  *Something    `json:"top"`

	NotSel *mods.NotSelected `json:"notSel"`

	Emb *struct {
		CID *int64  `json:"cid"`
		Baz *string `json:"baz"`
	} `json:"emb"`
}

// A SliceAndDice struct contains only slices
//
// the elements of the slices are structs, primitives or string formats
// there is also a pointer version of each property
type SliceAndDice struct {
	IDs     []int64            `json:"ids"`
	Names   []string           `json:"names"`
	UUIDs   []strfmt.UUID      `json:"uuids"`
	Tops    []Something        `json:"tops"`
	NotSels []mods.NotSelected `json:"notSels"`
	Embs    []struct {
		CID []int64  `json:"cid"`
		Baz []string `json:"baz"`
	} `json:"embs"`

	PtrIDs     []*int64            `json:"ptrIds"`
	PtrNames   []*string           `json:"ptrNames"`
	PtrUUIDs   []*strfmt.UUID      `json:"ptrUuids"`
	PtrTops    []*Something        `json:"ptrTops"`
	PtrNotSels []*mods.NotSelected `json:"ptrNotSels"`
	PtrEmbs    []*struct {
		PtrCID []*int64  `json:"ptrCid"`
		PtrBaz []*string `json:"ptrBaz"`
	} `json:"ptrEmbs"`
}

// A MapTastic struct contains only maps
//
// the values of the maps are structs, primitives or string formats
// there is also a pointer version of each property
type MapTastic struct {
	IDs     map[string]int64            `json:"ids"`
	Names   map[string]string           `json:"names"`
	UUIDs   map[string]strfmt.UUID      `json:"uuids"`
	Tops    map[string]Something        `json:"tops"`
	NotSels map[string]mods.NotSelected `json:"notSels"`
	Embs    map[string]struct {
		CID map[string]int64  `json:"cid"`
		Baz map[string]string `json:"baz"`
	} `json:"embs"`

	PtrIDs     map[string]*int64            `json:"ptrIds"`
	PtrNames   map[string]*string           `json:"ptrNames"`
	PtrUUIDs   map[string]*strfmt.UUID      `json:"ptrUuids"`
	PtrTops    map[string]*Something        `json:"ptrTops"`
	PtrNotSels map[string]*mods.NotSelected `json:"ptrNotSels"`
	PtrEmbs    map[string]*struct {
		PtrCID map[string]*int64  `json:"ptrCid"`
		PtrBaz map[string]*string `json:"ptrBaz"`
	} `json:"ptrEmbs"`
}

// An Interfaced struct contains objects with interface definitions
type Interfaced struct {
	CustomData interface{} `json:"custom_data"`
}

// A BaseStruct is a struct that has subtypes.
//
// it should deserialize into one of the struct types that
// enlist for being implementations of this struct
//
// swagger:model animal
type BaseStruct struct {
	// ID of this no model instance.
	// ids in this application start at 11 and are smaller than 1000
	//
	// required: true
	// minimum: > 10
	// maximum: < 1000
	ID int64 `json:"id"`

	// Name of this no model instance
	//
	// min length: 4
	// max length: 50
	// pattern: [A-Za-z0-9-.]*
	// required: true
	Name string `json:"name"`

	// StructType the type of this polymorphic model
	//
	// discriminator: true
	StructType string `json:"jsonClass"`
}

/* TODO: implement this in the scanner

// A Lion is a struct that "subtypes" the BaseStruct
//
// it does so because it included the fields in the struct body
// The scanner assumes it will follow the rules and describes this type
// as discriminated in the swagger spec based on the discriminator value
// annotation.
//
// swagger:model lion
// swagger:discriminatorValue animal org.horrible.java.fqpn.TheLionDataObjectFactoryInstanceServiceImpl
type Lion struct {
	// ID of this no model instance.
	// ids in this application start at 11 and are smaller than 1000
	//
	// required: true
	// minimum: > 10
	// maximum: < 1000
	ID int64 `json:"id"`

	// Name of this no model instance
	//
	// min length: 4
	// max length: 50
	// pattern: [A-Za-z0-9-.]*
	// required: true
	Name string `json:"name"`

	// StructType the type of this polymorphic model
	StructType string `json:"jsonClass"`

	// Leader is true when this is the leader of its group
	//
	// default value: false
	Leader bool `json:"leader"`
}
*/

// A Giraffe is a struct that embeds BaseStruct
//
// the annotation is not necessary here because of inclusion
// of a discriminated type
// it infers the name of the x-class value from its context
//
// swagger:model giraffe
type Giraffe struct {
	// swagger:allOf
	BaseStruct

	// NeckSize the size of the neck of this giraffe
	NeckSize int64 `json:"neckSize"`
}

// A Gazelle is a struct is discriminated for BaseStruct.
//
// The struct includes the BaseStruct and that embedded value
// is annotated with the discriminator value annotation so it
// where it only requires 1 argument because it knows which
// discriminator type this belongs to
//
// swagger:model gazelle
type Gazelle struct {
	// swagger:allOf a.b.c.d.E
	BaseStruct

	// The size of the horns
	HornSize float32 `json:"hornSize"`
}

// Identifiable is an interface for things that have an ID
type Identifiable interface {
	// ID of this no model instance.
	// ids in this application start at 11 and are smaller than 1000
	//
	// required: true
	// minimum: > 10
	// maximum: < 1000
	// swagger:name id
	ID() int64
}

// WaterType is an interface describing a water type
//
// swagger:model water
type WaterType interface {
	// swagger:name sweetWater
	SweetWater() bool
	// swagger:name saltWater
	SaltWater() bool
}

// Fish represents a base type implemented as interface
// the nullary methods of this interface will be included as
//
// swagger:model fish
type Fish interface {
	Identifiable // interfaces like this are included as if they were defined directly on this type

	// embeds decorated with allOf are included as refs

	// swagger:allOf
	WaterType

	// swagger:allOf
	mods.ExtraInfo

	mods.EmbeddedColor

	Items(id, size int64) []string

	// Name of this no model instance
	//
	// min length: 4
	// max length: 50
	// pattern: [A-Za-z0-9-.]*
	// required: true
	// swagger:name name
	Name() string

	// StructType the type of this polymorphic model
	// Discriminator: true
	// swagger:name jsonClass
	StructType() string
}

// TeslaCar is a tesla car
//
// swagger:model
type TeslaCar interface {
	// The model of tesla car
	//
	// discriminated: true
	// swagger:name model
	Model() string

	// AutoPilot returns true when it supports autopilot
	// swagger:name autoPilot
	AutoPilot() bool
}

// The ModelS version of the tesla car
//
// swagger:model modelS
type ModelS struct {
	// swagger:allOf com.tesla.models.ModelS
	TeslaCar
	// The edition of this Model S
	Edition string `json:"edition"`
}

// The ModelX version of the tesla car
//
// swagger:model modelX
type ModelX struct {
	// swagger:allOf com.tesla.models.ModelX
	TeslaCar
	// The number of doors on this Model X
	Doors int `json:"doors"`
}
