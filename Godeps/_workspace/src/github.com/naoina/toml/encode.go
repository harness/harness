package toml

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"go/ast"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/naoina/go-stringutil"
)

const (
	tagOmitempty = "omitempty"
	tagSkip      = "-"
)

// Marshal returns the TOML encoding of v.
//
// Struct values encode as TOML. Each exported struct field becomes a field of
// the TOML structure unless
//   - the field's tag is "-", or
//   - the field is empty and its tag specifies the "omitempty" option.
// The "toml" key in the struct field's tag value is the key name, followed by
// an optional comma and options. Examples:
//
//   // Field is ignored by this package.
//   Field int `toml:"-"`
//
//   // Field appears in TOML as key "myName".
//   Field int `toml:"myName"`
//
//   // Field appears in TOML as key "myName" and the field is omitted from the
//   // result of encoding if its value is empty.
//   Field int `toml:"myName,omitempty"`
//
//   // Field appears in TOML as key "field", but the field is skipped if
//   // empty.
//   // Note the leading comma.
//   Field int `toml:",omitempty"`
func Marshal(v interface{}) ([]byte, error) {
	return marshal(nil, "", reflect.ValueOf(v), false, false)
}

// Marshaler is the interface implemented by objects that can marshal themshelves into valid TOML.
type Marshaler interface {
	MarshalTOML() ([]byte, error)
}

func marshal(buf []byte, prefix string, rv reflect.Value, inArray, arrayTable bool) ([]byte, error) {
	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		ft := rt.Field(i)
		if !ast.IsExported(ft.Name) {
			continue
		}
		colName, rest := extractTag(rt.Field(i).Tag.Get(fieldTagName))
		if colName == tagSkip {
			continue
		}
		if colName == "" {
			colName = stringutil.ToSnakeCase(ft.Name)
		}
		fv := rv.Field(i)
		switch rest {
		case tagOmitempty:
			if fv.Interface() == reflect.Zero(ft.Type).Interface() {
				continue
			}
		}
		var err error
		if buf, err = encodeValue(buf, prefix, colName, fv, inArray, arrayTable); err != nil {
			return nil, err
		}
	}
	return buf, nil
}

func encodeValue(buf []byte, prefix, name string, fv reflect.Value, inArray, arrayTable bool) ([]byte, error) {
	switch t := fv.Interface().(type) {
	case Marshaler:
		b, err := t.MarshalTOML()
		if err != nil {
			return nil, err
		}
		return appendNewline(append(appendKey(buf, name, inArray, arrayTable), b...), inArray, arrayTable), nil
	case time.Time:
		return appendNewline(encodeTime(appendKey(buf, name, inArray, arrayTable), t), inArray, arrayTable), nil
	}
	switch fv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return appendNewline(encodeInt(appendKey(buf, name, inArray, arrayTable), fv.Int()), inArray, arrayTable), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return appendNewline(encodeUint(appendKey(buf, name, inArray, arrayTable), fv.Uint()), inArray, arrayTable), nil
	case reflect.Float32, reflect.Float64:
		return appendNewline(encodeFloat(appendKey(buf, name, inArray, arrayTable), fv.Float()), inArray, arrayTable), nil
	case reflect.Bool:
		return appendNewline(encodeBool(appendKey(buf, name, inArray, arrayTable), fv.Bool()), inArray, arrayTable), nil
	case reflect.String:
		return appendNewline(encodeString(appendKey(buf, name, inArray, arrayTable), fv.String()), inArray, arrayTable), nil
	case reflect.Slice, reflect.Array:
		ft := fv.Type().Elem()
		for ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Struct {
			name := tableName(prefix, name)
			var err error
			for i := 0; i < fv.Len(); i++ {
				if buf, err = marshal(append(append(append(buf, '[', '['), name...), ']', ']', '\n'), name, fv.Index(i), false, true); err != nil {
					return nil, err
				}
			}
			return buf, nil
		}
		buf = append(appendKey(buf, name, inArray, arrayTable), '[')
		var err error
		for i := 0; i < fv.Len(); i++ {
			if i != 0 {
				buf = append(buf, ',')
			}
			if buf, err = encodeValue(buf, prefix, name, fv.Index(i), true, false); err != nil {
				return nil, err
			}
		}
		return appendNewline(append(buf, ']'), inArray, arrayTable), nil
	case reflect.Struct:
		name := tableName(prefix, name)
		return marshal(append(append(append(buf, '['), name...), ']', '\n'), name, fv, inArray, arrayTable)
	case reflect.Interface:
		var err error
		if buf, err = encodeInterface(appendKey(buf, name, inArray, arrayTable), fv.Interface()); err != nil {
			return nil, err
		}
		return appendNewline(buf, inArray, arrayTable), nil
	}
	return nil, fmt.Errorf("toml: marshal: unsupported type %v", fv.Kind())
}

func appendKey(buf []byte, key string, inArray, arrayTable bool) []byte {
	if !inArray {
		return append(append(buf, key...), '=')
	}
	return buf
}

func appendNewline(buf []byte, inArray, arrayTable bool) []byte {
	if !inArray {
		return append(buf, '\n')
	}
	return buf
}

func encodeInterface(buf []byte, v interface{}) ([]byte, error) {
	switch v := v.(type) {
	case int:
		return encodeInt(buf, int64(v)), nil
	case int8:
		return encodeInt(buf, int64(v)), nil
	case int16:
		return encodeInt(buf, int64(v)), nil
	case int32:
		return encodeInt(buf, int64(v)), nil
	case int64:
		return encodeInt(buf, v), nil
	case uint:
		return encodeUint(buf, uint64(v)), nil
	case uint8:
		return encodeUint(buf, uint64(v)), nil
	case uint16:
		return encodeUint(buf, uint64(v)), nil
	case uint32:
		return encodeUint(buf, uint64(v)), nil
	case uint64:
		return encodeUint(buf, v), nil
	case float32:
		return encodeFloat(buf, float64(v)), nil
	case float64:
		return encodeFloat(buf, v), nil
	case bool:
		return encodeBool(buf, v), nil
	case string:
		return encodeString(buf, v), nil
	}
	return nil, fmt.Errorf("toml: marshal: unable to detect a type of value `%v'", v)
}

func encodeInt(buf []byte, i int64) []byte {
	return strconv.AppendInt(buf, i, 10)
}

func encodeUint(buf []byte, u uint64) []byte {
	return strconv.AppendUint(buf, u, 10)
}

func encodeFloat(buf []byte, f float64) []byte {
	return strconv.AppendFloat(buf, f, 'e', -1, 64)
}

func encodeBool(buf []byte, b bool) []byte {
	return strconv.AppendBool(buf, b)
}

func encodeString(buf []byte, s string) []byte {
	return strconv.AppendQuote(buf, s)
}

func encodeTime(buf []byte, t time.Time) []byte {
	return append(buf, t.Format(time.RFC3339Nano)...)
}
