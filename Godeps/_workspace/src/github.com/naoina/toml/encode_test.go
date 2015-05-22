package toml_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/naoina/toml"
)

func TestMarshal(t *testing.T) {
	for _, v := range []struct {
		v      interface{}
		expect string
	}{
		{struct{ Name string }{"alice"}, "name=\"alice\"\n"},
		{struct{ Age int }{7}, "age=7\n"},
		{struct {
			Name string
			Age  int
		}{"alice", 7}, "name=\"alice\"\nage=7\n"},
		{struct {
			Name string `toml:"-"`
			Age  int
		}{"alice", 7}, "age=7\n"},
		{struct {
			Name string `toml:"my_name"`
		}{"bob"}, "my_name=\"bob\"\n"},
		{struct {
			Name string `toml:"my_name,omitempty"`
		}{"bob"}, "my_name=\"bob\"\n"},
		{struct {
			Name string `toml:",omitempty"`
		}{"bob"}, "name=\"bob\"\n"},
		{struct {
			Name string `toml:",omitempty"`
		}{""}, ""},
	} {
		b, err := toml.Marshal(v.v)
		var actual interface{} = err
		var expect interface{} = nil
		if !reflect.DeepEqual(actual, expect) {
			t.Errorf(`Marshal(%#v) => %#v; want %#v`, v.v, actual, expect)
		}

		actual = string(b)
		expect = v.expect
		if !reflect.DeepEqual(actual, expect) {
			t.Errorf(`Marshal(%#v); v => %#v; want %#v`, v, actual, expect)
		}
	}
}

func TestMarshalWhole(t *testing.T) {
	for _, v := range []struct {
		v      interface{}
		expect string
	}{
		{
			testStruct{
				Table: Table{
					Key: "value",
					Subtable: Subtable{
						Key: "another value",
					},
					Inline: Inline{
						Name: Name{
							First: "Tom",
							Last:  "Preston-Werner",
						},
						Point: Point{
							X: 1,
							Y: 2,
						},
					},
				},
				X: X{},
				String: String{
					Basic: Basic{
						Basic: "I'm a string. \"You can quote me\". Name\tJos\u00E9\nLocation\tSF.",
					},
					Multiline: Multiline{
						Key1: "One\nTwo",
						Continued: Continued{
							Key1: "The quick brown fox jumps over the lazy dog.",
						},
					},
					Literal: Literal{
						Winpath:  `C:\Users\nodejs\templates`,
						Winpath2: `\\ServerX\admin$\system32\`,
						Quoted:   `Tom "Dubs" Preston-Werner`,
						Regex:    `<\i\c*\s*>`,
						Multiline: LiteralMultiline{
							Regex2: `I [dw]on't need \d{2} apples`,
							Lines:  "The first newline is\ntrimmed in raw strings.\n   All other whitespace\n   is preserved.\n",
						},
					},
				},
				Integer: Integer{
					Key1: 99,
					Key2: 42,
					Key3: 0,
					Key4: -17,
					Underscores: IntegerUnderscores{
						Key1: 1000,
						Key2: 5349221,
						Key3: 12345,
					},
				},
				Float: Float{
					Fractional: Fractional{
						Key1: 1.0,
						Key2: 3.1415,
						Key3: -0.01,
					},
					Exponent: Exponent{
						Key1: 5e22,
						Key2: 1e6,
						Key3: -2e-2,
					},
					Both: Both{
						Key: 6.626e-34,
					},
					Underscores: FloatUnderscores{
						Key1: 9224617.445991228313,
						Key2: 1e100,
					},
				},
				Boolean: Boolean{
					True:  true,
					False: false,
				},
				Datetime: Datetime{
					Key1: mustTime(time.Parse(time.RFC3339Nano, "1979-05-27T07:32:00Z")),
					Key2: mustTime(time.Parse(time.RFC3339Nano, "1979-05-27T00:32:00-07:00")),
					Key3: mustTime(time.Parse(time.RFC3339Nano, "1979-05-27T00:32:00.999999-07:00")),
				},
				Array: Array{
					Key1: []int{1, 2, 3},
					Key2: []string{"red", "yellow", "green"},
					Key3: [][]int{{1, 2}, {3, 4, 5}},
					Key4: [][]interface{}{{int64(1), int64(2)}, {"a", "b", "c"}},
					Key5: []int{1, 2, 3},
					Key6: []int{1, 2},
				},
				Products: []Product{
					{Name: "Hammer", Sku: 738594937},
					{},
					{Name: "Nail", Sku: 284758393, Color: "gray"},
				},
				Fruit: []Fruit{
					{
						Name: "apple",
						Physical: Physical{
							Color: "red",
							Shape: "round",
						},
						Variety: []Variety{
							{Name: "red delicious"},
							{Name: "granny smith"},
						},
					},
					{
						Name: "banana",
						Variety: []Variety{
							{Name: "plantain"},
						},
					},
				},
			},
			`[table]
key="value"
[table.subtable]
key="another value"
[table.inline]
[table.inline.name]
first="Tom"
last="Preston-Werner"
[table.inline.point]
x=1
y=2
[x]
[x.y]
[x.y.z]
[x.y.z.w]
[string]
[string.basic]
basic="I'm a string. \"You can quote me\". Name\tJosé\nLocation\tSF."
[string.multiline]
key1="One\nTwo"
key2=""
key3=""
[string.multiline.continued]
key1="The quick brown fox jumps over the lazy dog."
key2=""
key3=""
[string.literal]
winpath="C:\\Users\\nodejs\\templates"
winpath2="\\\\ServerX\\admin$\\system32\\"
quoted="Tom \"Dubs\" Preston-Werner"
regex="<\\i\\c*\\s*>"
[string.literal.multiline]
regex2="I [dw]on't need \\d{2} apples"
lines="The first newline is\ntrimmed in raw strings.\n   All other whitespace\n   is preserved.\n"
[integer]
key1=99
key2=42
key3=0
key4=-17
[integer.underscores]
key1=1000
key2=5349221
key3=12345
[float]
[float.fractional]
key1=1e+00
key2=3.1415e+00
key3=-1e-02
[float.exponent]
key1=5e+22
key2=1e+06
key3=-2e-02
[float.both]
key=6.626e-34
[float.underscores]
key1=9.224617445991227e+06
key2=1e+100
[boolean]
true=true
false=false
[datetime]
key1=1979-05-27T07:32:00Z
key2=1979-05-27T00:32:00-07:00
key3=1979-05-27T00:32:00.999999-07:00
[array]
key1=[1,2,3]
key2=["red","yellow","green"]
key3=[[1,2],[3,4,5]]
key4=[[1,2],["a","b","c"]]
key5=[1,2,3]
key6=[1,2]
[[products]]
name="Hammer"
sku=738594937
color=""
[[products]]
name=""
sku=0
color=""
[[products]]
name="Nail"
sku=284758393
color="gray"
[[fruit]]
name="apple"
[fruit.physical]
color="red"
shape="round"
[[fruit.variety]]
name="red delicious"
[[fruit.variety]]
name="granny smith"
[[fruit]]
name="banana"
[fruit.physical]
color=""
shape=""
[[fruit.variety]]
name="plantain"
`,
		},
	} {
		b, err := toml.Marshal(v.v)
		var actual interface{} = err
		var expect interface{} = nil
		if !reflect.DeepEqual(actual, expect) {
			t.Errorf(`Marshal(%#v) => %#v; want %#v`, v.v, actual, expect)
		}
		actual = string(b)
		expect = v.expect
		if !reflect.DeepEqual(actual, expect) {
			t.Errorf(`Marshal(%#v); v => %#v; want %#v`, v.v, actual, expect)
		}

		// test for reversible.
		dest := testStruct{}
		actual = toml.Unmarshal(b, &dest)
		expect = nil
		if !reflect.DeepEqual(actual, expect) {
			t.Errorf(`Unmarshal after Marshal => %#v; want %#v`, actual, expect)
		}
		actual = dest
		expect = v.v
		if !reflect.DeepEqual(actual, expect) {
			t.Errorf(`Unmarshal after Marshal => %#v; want %#v`, v, actual, expect)
		}
	}
}
