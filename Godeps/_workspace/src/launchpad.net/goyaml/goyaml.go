// Package goyaml implements YAML support for the Go language.
package goyaml

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
)

func handleErr(err *error) {
	if r := recover(); r != nil {
		if _, ok := r.(runtime.Error); ok {
			panic(r)
		} else if _, ok := r.(*reflect.ValueError); ok {
			panic(r)
		} else if _, ok := r.(externalPanic); ok {
			panic(r)
		} else if s, ok := r.(string); ok {
			*err = errors.New("YAML error: " + s)
		} else if e, ok := r.(error); ok {
			*err = e
		} else {
			panic(r)
		}
	}
}

// Objects implementing the goyaml.Setter interface will receive the YAML
// tag and value via the SetYAML method during unmarshaling, rather than
// being implicitly assigned by the goyaml machinery.  If setting the value
// works, the method should return true.  If it returns false, the given
// value will be omitted from maps and slices.
type Setter interface {
	SetYAML(tag string, value interface{}) bool
}

// Objects implementing the goyaml.Getter interface will get the GetYAML()
// method called when goyaml is requested to marshal the given value, and
// the result of this method will be marshaled in place of the actual object.
type Getter interface {
	GetYAML() (tag string, value interface{})
}

// Unmarshal decodes the first document found within the in byte slice
// and assigns decoded values into the object pointed by out.
//
// Maps, pointers to structs and ints, etc, may all be used as out values.
// If an internal pointer within a struct is not initialized, goyaml
// will initialize it if necessary for unmarshalling the provided data,
// but the struct provided as out must not be a nil pointer.
//
// The type of the decoded values and the type of out will be considered,
// and Unmarshal() will do the best possible job to unmarshal values
// appropriately.  It is NOT considered an error, though, to skip values
// because they are not available in the decoded YAML, or if they are not
// compatible with the out value. To ensure something was properly
// unmarshaled use a map or compare against the previous value for the
// field (usually the zero value).
//
// Struct fields are only unmarshalled if they are exported (have an
// upper case first letter), and will be unmarshalled using the field
// name lowercased by default. When custom field names are desired, the
// tag value may be used to tweak the name. Everything before the first
// comma in the field tag will be used as the name. The values following
// the comma are used to tweak the marshalling process (see Marshal).
// Conflicting names result in a runtime error.
//
// For example:
//
//     type T struct {
//         F int `yaml:"a,omitempty"`
//         B int
//     }
//     var T t
//     goyaml.Unmarshal([]byte("a: 1\nb: 2"), &t)
//
// See the documentation of Marshal for the format of tags and a list of
// supported tag options.
//
func Unmarshal(in []byte, out interface{}) (err error) {
	defer handleErr(&err)
	d := newDecoder()
	p := newParser(in)
	defer p.destroy()
	node := p.parse()
	if node != nil {
		d.unmarshal(node, reflect.ValueOf(out))
	}
	return nil
}

// Marshal serializes the value provided into a YAML document. The structure
// of the generated document will reflect the structure of the value itself.
// Maps, pointers to structs and ints, etc, may all be used as the in value.
//
// In the case of struct values, only exported fields will be serialized.
// The lowercased field name is used as the key for each exported field,
// but this behavior may be changed using the respective field tag.
// The tag may also contain flags to tweak the marshalling behavior for
// the field. Conflicting names result in a runtime error. The tag format
// accepted is:
//
//     `(...) yaml:"[<key>][,<flag1>[,<flag2>]]" (...)`
//
// The following flags are currently supported:
//
//     omitempty    Only include the field if it's not set to the zero
//                  value for the type or to empty slices or maps.
//                  Does not apply to zero valued structs.
//
//     flow         Marshal using a flow style (useful for structs,
//                  sequences and maps.
//
//     inline       Inline the struct it's applied to, so its fields
//                  are processed as if they were part of the outer
//                  struct.
//
// In addition, if the key is "-", the field is ignored.
//
// For example:
//
//     type T struct {
//         F int "a,omitempty"
//         B int
//     }
//     goyaml.Marshal(&T{B: 2}) // Returns "b: 2\n"
//     goyaml.Marshal(&T{F: 1}} // Returns "a: 1\nb: 0\n"
//
func Marshal(in interface{}) (out []byte, err error) {
	defer handleErr(&err)
	e := newEncoder()
	defer e.destroy()
	e.marshal("", reflect.ValueOf(in))
	e.finish()
	out = e.out
	return
}

// --------------------------------------------------------------------------
// Maintain a mapping of keys to structure field indexes

// The code in this section was copied from gobson.

// structInfo holds details for the serialization of fields of
// a given struct.
type structInfo struct {
	FieldsMap  map[string]fieldInfo
	FieldsList []fieldInfo

	// InlineMap is the number of the field in the struct that
	// contains an ,inline map, or -1 if there's none.
	InlineMap int
}

type fieldInfo struct {
	Key       string
	Num       int
	OmitEmpty bool
	Flow      bool

	// Inline holds the field index if the field is part of an inlined struct.
	Inline []int
}

var structMap = make(map[reflect.Type]*structInfo)
var fieldMapMutex sync.RWMutex

type externalPanic string

func (e externalPanic) String() string {
	return string(e)
}

func getStructInfo(st reflect.Type) (*structInfo, error) {
	fieldMapMutex.RLock()
	sinfo, found := structMap[st]
	fieldMapMutex.RUnlock()
	if found {
		return sinfo, nil
	}

	n := st.NumField()
	fieldsMap := make(map[string]fieldInfo)
	fieldsList := make([]fieldInfo, 0, n)
	inlineMap := -1
	for i := 0; i != n; i++ {
		field := st.Field(i)
		if field.PkgPath != "" {
			continue // Private field
		}

		info := fieldInfo{Num: i}

		tag := field.Tag.Get("yaml")
		if tag == "" && strings.Index(string(field.Tag), ":") < 0 {
			tag = string(field.Tag)
		}
		if tag == "-" {
			continue
		}

		inline := false
		fields := strings.Split(tag, ",")
		if len(fields) > 1 {
			for _, flag := range fields[1:] {
				switch flag {
				case "omitempty":
					info.OmitEmpty = true
				case "flow":
					info.Flow = true
				case "inline":
					inline = true
				default:
					msg := fmt.Sprintf("Unsupported flag %q in tag %q of type %s", flag, tag, st)
					panic(externalPanic(msg))
				}
			}
			tag = fields[0]
		}

		if inline {
			switch field.Type.Kind() {
			//case reflect.Map:
			//	if inlineMap >= 0 {
			//		return nil, errors.New("Multiple ,inline maps in struct " + st.String())
			//	}
			//	if field.Type.Key() != reflect.TypeOf("") {
			//		return nil, errors.New("Option ,inline needs a map with string keys in struct " + st.String())
			//	}
			//	inlineMap = info.Num
			case reflect.Struct:
				sinfo, err := getStructInfo(field.Type)
				if err != nil {
					return nil, err
				}
				for _, finfo := range sinfo.FieldsList {
					if _, found := fieldsMap[finfo.Key]; found {
						msg := "Duplicated key '" + finfo.Key + "' in struct " + st.String()
						return nil, errors.New(msg)
					}
					if finfo.Inline == nil {
						finfo.Inline = []int{i, finfo.Num}
					} else {
						finfo.Inline = append([]int{i}, finfo.Inline...)
					}
					fieldsMap[finfo.Key] = finfo
					fieldsList = append(fieldsList, finfo)
				}
			default:
				//panic("Option ,inline needs a struct value or map field")
				panic("Option ,inline needs a struct value field")
			}
			continue
		}

		if tag != "" {
			info.Key = tag
		} else {
			info.Key = strings.ToLower(field.Name)
		}

		if _, found = fieldsMap[info.Key]; found {
			msg := "Duplicated key '" + info.Key + "' in struct " + st.String()
			return nil, errors.New(msg)
		}

		fieldsList = append(fieldsList, info)
		fieldsMap[info.Key] = info
	}

	sinfo = &structInfo{fieldsMap, fieldsList, inlineMap}

	fieldMapMutex.Lock()
	structMap[st] = sinfo
	fieldMapMutex.Unlock()
	return sinfo, nil
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return len(v.String()) == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	case reflect.Slice:
		return v.Len() == 0
	case reflect.Map:
		return v.Len() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Bool:
		return !v.Bool()
	}
	return false
}
