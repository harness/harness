package toml

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/naoina/toml/ast"
)

const (
	tableSeparator = '.'
)

var (
	escapeReplacer = strings.NewReplacer(
		"\b", "\\n",
		"\f", "\\f",
		"\n", "\\n",
		"\r", "\\r",
		"\t", "\\t",
	)
	underscoreReplacer = strings.NewReplacer(
		"_", "",
	)
)

// Unmarshal parses the TOML data and stores the result in the value pointed to by v.
//
// Unmarshal will mapped to v that according to following rules:
//
//	TOML strings to string
//	TOML integers to any int type
//	TOML floats to float32 or float64
//	TOML booleans to bool
//	TOML datetimes to time.Time
//	TOML arrays to any type of slice or []interface{}
//	TOML tables to struct
//	TOML array of tables to slice of struct
func Unmarshal(data []byte, v interface{}) error {
	table, err := Parse(data)
	if err != nil {
		return err
	}
	if err := UnmarshalTable(table, v); err != nil {
		return fmt.Errorf("toml: unmarshal: %v", err)
	}
	return nil
}

// Unmarshaler is the interface implemented by objects that can unmarshal a
// TOML description of themselves.
// The input can be assumed to be a valid encoding of a TOML value.
// UnmarshalJSON must copy the TOML data if it wishes to retain the data after
// returning.
type Unmarshaler interface {
	UnmarshalTOML([]byte) error
}

// UnmarshalTable applies the contents of an ast.Table to the value pointed at by v.
//
// UnmarshalTable will mapped to v that according to following rules:
//
//	TOML strings to string
//	TOML integers to any int type
//	TOML floats to float32 or float64
//	TOML booleans to bool
//	TOML datetimes to time.Time
//	TOML arrays to any type of slice or []interface{}
//	TOML tables to struct
//	TOML array of tables to slice of struct
func UnmarshalTable(t *ast.Table, v interface{}) (err error) {
	if v == nil {
		return fmt.Errorf("v must not be nil")
	}
	rv := reflect.ValueOf(v)
	if kind := rv.Kind(); kind != reflect.Ptr && kind != reflect.Map {
		return fmt.Errorf("v must be a pointer or map")
	}
	for rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	for key, val := range t.Fields {
		switch av := val.(type) {
		case *ast.KeyValue:
			fv, fieldName, found := findField(rv, key)
			if !found {
				return fmt.Errorf("line %d: field corresponding to `%s' is not defined in `%T'", av.Line, key, v)
			}
			switch fv.Kind() {
			case reflect.Map:
				mv := reflect.New(fv.Type().Elem()).Elem()
				if err := UnmarshalTable(t, mv.Addr().Interface()); err != nil {
					return err
				}
				fv.SetMapIndex(reflect.ValueOf(fieldName), mv)
			default:
				if err := setValue(fv, av.Value); err != nil {
					return fmt.Errorf("line %d: %v.%s: %v", av.Line, rv.Type(), fieldName, err)
				}
				if rv.Kind() == reflect.Map {
					rv.SetMapIndex(reflect.ValueOf(fieldName), fv)
				}
			}
		case *ast.Table:
			fv, fieldName, found := findField(rv, key)
			if !found {
				return fmt.Errorf("line %d: field corresponding to `%s' is not defined in `%T'", av.Line, key, v)
			}
			if err, ok := setUnmarshaler(fv, string(av.Data)); ok {
				if err != nil {
					return err
				}
				continue
			}
			for fv.Kind() == reflect.Ptr {
				fv.Set(reflect.New(fv.Type().Elem()))
				fv = fv.Elem()
			}
			switch fv.Kind() {
			case reflect.Struct:
				vv := reflect.New(fv.Type()).Elem()
				if err := UnmarshalTable(av, vv.Addr().Interface()); err != nil {
					return err
				}
				fv.Set(vv)
				if rv.Kind() == reflect.Map {
					rv.SetMapIndex(reflect.ValueOf(fieldName), fv)
				}
			case reflect.Map:
				mv := reflect.MakeMap(fv.Type())
				if err := UnmarshalTable(av, mv.Interface()); err != nil {
					return err
				}
				fv.Set(mv)
			default:
				return fmt.Errorf("line %d: `%v.%s' must be struct or map, but %v given", av.Line, rv.Type(), fieldName, fv.Kind())
			}
		case []*ast.Table:
			fv, fieldName, found := findField(rv, key)
			if !found {
				return fmt.Errorf("line %d: field corresponding to `%s' is not defined in `%T'", av[0].Line, key, v)
			}
			data := make([]string, 0, len(av))
			for _, tbl := range av {
				data = append(data, string(tbl.Data))
			}
			if err, ok := setUnmarshaler(fv, strings.Join(data, "\n")); ok {
				if err != nil {
					return err
				}
				continue
			}
			t := fv.Type().Elem()
			pc := 0
			for ; t.Kind() == reflect.Ptr; pc++ {
				t = t.Elem()
			}
			if fv.Kind() != reflect.Slice {
				return fmt.Errorf("line %d: `%v.%s' must be slice type, but %v given", av[0].Line, rv.Type(), fieldName, fv.Kind())
			}
			for _, tbl := range av {
				var vv reflect.Value
				switch t.Kind() {
				case reflect.Map:
					vv = reflect.MakeMap(t)
					if err := UnmarshalTable(tbl, vv.Interface()); err != nil {
						return err
					}
				default:
					vv = reflect.New(t).Elem()
					if err := UnmarshalTable(tbl, vv.Addr().Interface()); err != nil {
						return err
					}
				}
				for i := 0; i < pc; i++ {
					vv = vv.Addr()
					pv := reflect.New(vv.Type()).Elem()
					pv.Set(vv)
					vv = pv
				}
				fv.Set(reflect.Append(fv, vv))
			}
			if rv.Kind() == reflect.Map {
				rv.SetMapIndex(reflect.ValueOf(fieldName), fv)
			}
		default:
			return fmt.Errorf("BUG: unknown type `%T'", t)
		}
	}
	return nil
}

func setUnmarshaler(lhs reflect.Value, data string) (error, bool) {
	for lhs.Kind() == reflect.Ptr {
		lhs.Set(reflect.New(lhs.Type().Elem()))
		lhs = lhs.Elem()
	}
	if lhs.CanAddr() {
		if u, ok := lhs.Addr().Interface().(Unmarshaler); ok {
			return u.UnmarshalTOML([]byte(data)), true
		}
	}
	return nil, false
}

func setValue(lhs reflect.Value, val ast.Value) error {
	for lhs.Kind() == reflect.Ptr {
		lhs.Set(reflect.New(lhs.Type().Elem()))
		lhs = lhs.Elem()
	}
	if err, ok := setUnmarshaler(lhs, val.Source()); ok {
		return err
	}
	switch v := val.(type) {
	case *ast.Integer:
		if err := setInt(lhs, v); err != nil {
			return err
		}
	case *ast.Float:
		if err := setFloat(lhs, v); err != nil {
			return err
		}
	case *ast.String:
		if err := setString(lhs, v); err != nil {
			return err
		}
	case *ast.Boolean:
		if err := setBoolean(lhs, v); err != nil {
			return err
		}
	case *ast.Datetime:
		if err := setDatetime(lhs, v); err != nil {
			return err
		}
	case *ast.Array:
		if err := setArray(lhs, v); err != nil {
			return err
		}
	}
	return nil
}

func setInt(fv reflect.Value, v *ast.Integer) error {
	i, err := v.Int()
	if err != nil {
		return err
	}
	switch fv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if fv.OverflowInt(i) {
			return &errorOutOfRange{fv.Kind(), i}
		}
		fv.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fv.SetUint(uint64(i))
	case reflect.Interface:
		fv.Set(reflect.ValueOf(i))
	default:
		return fmt.Errorf("`%v' is not any types of int", fv.Type())
	}
	return nil
}

func setFloat(fv reflect.Value, v *ast.Float) error {
	f, err := v.Float()
	if err != nil {
		return err
	}
	switch fv.Kind() {
	case reflect.Float32, reflect.Float64:
		if fv.OverflowFloat(f) {
			return &errorOutOfRange{fv.Kind(), f}
		}
		fv.SetFloat(f)
	case reflect.Interface:
		fv.Set(reflect.ValueOf(f))
	default:
		return fmt.Errorf("`%v' is not float32 or float64", fv.Type())
	}
	return nil
}

func setString(fv reflect.Value, v *ast.String) error {
	return set(fv, v.Value)
}

func setBoolean(fv reflect.Value, v *ast.Boolean) error {
	b, err := v.Boolean()
	if err != nil {
		return err
	}
	return set(fv, b)
}

func setDatetime(fv reflect.Value, v *ast.Datetime) error {
	tm, err := v.Time()
	if err != nil {
		return err
	}
	return set(fv, tm)
}

func setArray(fv reflect.Value, v *ast.Array) error {
	if len(v.Value) == 0 {
		return nil
	}
	typ := reflect.TypeOf(v.Value[0])
	for _, vv := range v.Value[1:] {
		if typ != reflect.TypeOf(vv) {
			return fmt.Errorf("array cannot contain multiple types")
		}
	}
	sliceType := fv.Type()
	if fv.Kind() == reflect.Interface {
		sliceType = reflect.SliceOf(sliceType)
	}
	slice := reflect.MakeSlice(sliceType, 0, len(v.Value))
	t := sliceType.Elem()
	for _, vv := range v.Value {
		tmp := reflect.New(t).Elem()
		if err := setValue(tmp, vv); err != nil {
			return err
		}
		slice = reflect.Append(slice, tmp)
	}
	fv.Set(slice)
	return nil
}

func set(fv reflect.Value, v interface{}) error {
	rhs := reflect.ValueOf(v)
	if !rhs.Type().AssignableTo(fv.Type()) {
		return fmt.Errorf("`%v' type is not assignable to `%v' type", rhs.Type(), fv.Type())
	}
	fv.Set(rhs)
	return nil
}

type stack struct {
	key   string
	table *ast.Table
}

type toml struct {
	table        *ast.Table
	line         int
	currentTable *ast.Table
	s            string
	key          string
	val          ast.Value
	arr          *array
	tableMap     map[string]*ast.Table
	stack        []*stack
	skip         bool
}

func (p *toml) init() {
	p.line = 1
	p.table = &ast.Table{
		Line: p.line,
		Type: ast.TableTypeNormal,
	}
	p.tableMap = map[string]*ast.Table{
		"": p.table,
	}
	p.currentTable = p.table
}

func (p *toml) Error(err error) {
	panic(convertError{fmt.Errorf("toml: line %d: %v", p.line, err)})
}

func (p *tomlParser) SetTime(begin, end int) {
	p.val = &ast.Datetime{
		Position: ast.Position{Begin: begin, End: end},
		Data:     p.buffer[begin:end],
		Value:    string(p.buffer[begin:end]),
	}
}

func (p *tomlParser) SetFloat64(begin, end int) {
	p.val = &ast.Float{
		Position: ast.Position{Begin: begin, End: end},
		Data:     p.buffer[begin:end],
		Value:    underscoreReplacer.Replace(string(p.buffer[begin:end])),
	}
}

func (p *tomlParser) SetInt64(begin, end int) {
	p.val = &ast.Integer{
		Position: ast.Position{Begin: begin, End: end},
		Data:     p.buffer[begin:end],
		Value:    underscoreReplacer.Replace(string(p.buffer[begin:end])),
	}
}

func (p *tomlParser) SetString(begin, end int) {
	p.val = &ast.String{
		Position: ast.Position{Begin: begin, End: end},
		Data:     p.buffer[begin:end],
		Value:    p.s,
	}
	p.s = ""
}

func (p *tomlParser) SetBool(begin, end int) {
	p.val = &ast.Boolean{
		Position: ast.Position{Begin: begin, End: end},
		Data:     p.buffer[begin:end],
		Value:    string(p.buffer[begin:end]),
	}
}

func (p *tomlParser) StartArray() {
	if p.arr == nil {
		p.arr = &array{line: p.line, current: &ast.Array{}}
		return
	}
	p.arr.child = &array{parent: p.arr, line: p.line, current: &ast.Array{}}
	p.arr = p.arr.child
}

func (p *tomlParser) AddArrayVal() {
	if p.arr.current == nil {
		p.arr.current = &ast.Array{}
	}
	p.arr.current.Value = append(p.arr.current.Value, p.val)
}

func (p *tomlParser) SetArray(begin, end int) {
	p.arr.current.Position = ast.Position{Begin: begin, End: end}
	p.arr.current.Data = p.buffer[begin:end]
	p.val = p.arr.current
	p.arr = p.arr.parent
}

func (p *toml) SetTable(buf []rune, begin, end int) {
	p.setTable(p.table, buf, begin, end)
}

func (p *toml) setTable(t *ast.Table, buf []rune, begin, end int) {
	name := string(buf[begin:end])
	names := splitTableKey(name)
	if t, exists := p.tableMap[name]; exists {
		if lt := p.tableMap[names[len(names)-1]]; t.Type == ast.TableTypeArray || lt != nil && lt.Type == ast.TableTypeNormal {
			p.Error(fmt.Errorf("table `%s' is in conflict with %v table in line %d", name, t.Type, t.Line))
		}
	}
	t, err := p.lookupTable(t, names)
	if err != nil {
		p.Error(err)
	}
	p.currentTable = t
	p.tableMap[name] = p.currentTable
}

func (p *tomlParser) SetTableString(begin, end int) {
	p.currentTable.Data = p.buffer[begin:end]

	p.currentTable.Position.Begin = begin
	p.currentTable.Position.End = end
}

func (p *toml) SetArrayTable(buf []rune, begin, end int) {
	p.setArrayTable(p.table, buf, begin, end)
}

func (p *toml) setArrayTable(t *ast.Table, buf []rune, begin, end int) {
	name := string(buf[begin:end])
	if t, exists := p.tableMap[name]; exists && t.Type == ast.TableTypeNormal {
		p.Error(fmt.Errorf("table `%s' is in conflict with %v table in line %d", name, t.Type, t.Line))
	}
	names := splitTableKey(name)
	t, err := p.lookupTable(t, names[:len(names)-1])
	if err != nil {
		p.Error(err)
	}
	last := names[len(names)-1]
	tbl := &ast.Table{
		Position: ast.Position{begin, end},
		Line:     p.line,
		Name:     last,
		Type:     ast.TableTypeArray,
	}
	switch v := t.Fields[last].(type) {
	case nil:
		if t.Fields == nil {
			t.Fields = make(map[string]interface{})
		}
		t.Fields[last] = []*ast.Table{tbl}
	case []*ast.Table:
		t.Fields[last] = append(v, tbl)
	case *ast.KeyValue:
		p.Error(fmt.Errorf("key `%s' is in conflict with line %d", last, v.Line))
	default:
		p.Error(fmt.Errorf("BUG: key `%s' is in conflict but it's unknown type `%T'", last, v))
	}
	p.currentTable = tbl
	p.tableMap[name] = p.currentTable
}

func (p *toml) StartInlineTable() {
	p.skip = false
	p.stack = append(p.stack, &stack{p.key, p.currentTable})
	buf := []rune(p.key)
	if p.arr == nil {
		p.setTable(p.currentTable, buf, 0, len(buf))
	} else {
		p.setArrayTable(p.currentTable, buf, 0, len(buf))
	}
}

func (p *toml) EndInlineTable() {
	st := p.stack[len(p.stack)-1]
	p.key, p.currentTable = st.key, st.table
	p.stack[len(p.stack)-1] = nil
	p.stack = p.stack[:len(p.stack)-1]
	p.skip = true
}

func (p *toml) AddLineCount(i int) {
	p.line += i
}

func (p *toml) SetKey(buf []rune, begin, end int) {
	p.key = string(buf[begin:end])
}

func (p *toml) AddKeyValue() {
	if p.skip {
		p.skip = false
		return
	}
	if val, exists := p.currentTable.Fields[p.key]; exists {
		switch v := val.(type) {
		case *ast.Table:
			p.Error(fmt.Errorf("key `%s' is in conflict with %v table in line %d", p.key, v.Type, v.Line))
		case *ast.KeyValue:
			p.Error(fmt.Errorf("key `%s' is in conflict with line %d", p.key, v.Line))
		default:
			p.Error(fmt.Errorf("BUG: key `%s' is in conflict but it's unknown type `%T'", p.key, v))
		}
	}
	if p.currentTable.Fields == nil {
		p.currentTable.Fields = make(map[string]interface{})
	}
	p.currentTable.Fields[p.key] = &ast.KeyValue{
		Key:   p.key,
		Value: p.val,
		Line:  p.line,
	}
}

func (p *toml) SetBasicString(buf []rune, begin, end int) {
	p.s = p.unquote(string(buf[begin:end]))
}

func (p *toml) SetMultilineString() {
	p.s = p.unquote(`"` + escapeReplacer.Replace(strings.TrimLeft(p.s, "\r\n")) + `"`)
}

func (p *toml) AddMultilineBasicBody(buf []rune, begin, end int) {
	p.s += string(buf[begin:end])
}

func (p *toml) SetLiteralString(buf []rune, begin, end int) {
	p.s = string(buf[begin:end])
}

func (p *toml) SetMultilineLiteralString(buf []rune, begin, end int) {
	p.s = strings.TrimLeft(string(buf[begin:end]), "\r\n")
}

func (p *toml) unquote(s string) string {
	s, err := strconv.Unquote(s)
	if err != nil {
		p.Error(err)
	}
	return s
}

func (p *toml) lookupTable(t *ast.Table, keys []string) (*ast.Table, error) {
	for _, s := range keys {
		val, exists := t.Fields[s]
		if !exists {
			tbl := &ast.Table{
				Line: p.line,
				Name: s,
				Type: ast.TableTypeNormal,
			}
			if t.Fields == nil {
				t.Fields = make(map[string]interface{})
			}
			t.Fields[s] = tbl
			t = tbl
			continue
		}
		switch v := val.(type) {
		case *ast.Table:
			t = v
		case []*ast.Table:
			t = v[len(v)-1]
		case *ast.KeyValue:
			return nil, fmt.Errorf("key `%s' is in conflict with line %d", s, v.Line)
		default:
			return nil, fmt.Errorf("BUG: key `%s' is in conflict but it's unknown type `%T'", s, v)
		}
	}
	return t, nil
}

func splitTableKey(tk string) []string {
	key := make([]byte, 0, 1)
	keys := make([]string, 0, 1)
	inQuote := false
	for i := 0; i < len(tk); i++ {
		k := tk[i]
		switch {
		case k == tableSeparator && !inQuote:
			keys = append(keys, string(key))
			key = key[:0] // reuse buffer.
		case k == '"':
			inQuote = !inQuote
		case (k == ' ' || k == '\t') && !inQuote:
			// skip.
		default:
			key = append(key, k)
		}
	}
	keys = append(keys, string(key))
	return keys
}

type convertError struct {
	err error
}

func (e convertError) Error() string {
	return e.err.Error()
}

type array struct {
	parent  *array
	child   *array
	current *ast.Array
	line    int
}
