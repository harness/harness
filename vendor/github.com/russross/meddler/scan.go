package meddler

import (
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// the name of our struct tag
const tagName = "meddler"

// Database contains database-specific options.
// MySQL, PostgreSQL, and SQLite are provided for convenience.
// Setting Default to any of these lets you use the package-level convenience functions.
type Database struct {
	Quote               string // the quote character for table and column names
	Placeholder         string // the placeholder style to use in generated queries
	UseReturningToGetID bool   // use PostgreSQL-style RETURNING "ID" instead of calling sql.Result.LastInsertID
}

var MySQL = &Database{
	Quote:               "`",
	Placeholder:         "?",
	UseReturningToGetID: false,
}

var PostgreSQL = &Database{
	Quote:               `"`,
	Placeholder:         "$1",
	UseReturningToGetID: true,
}

var SQLite = &Database{
	Quote:               `"`,
	Placeholder:         "?",
	UseReturningToGetID: false,
}

var Default = MySQL

func (d *Database) quoted(s string) string {
	return d.Quote + s + d.Quote
}

func (d *Database) placeholder(n int) string {
	return strings.Replace(d.Placeholder, "1", strconv.FormatInt(int64(n), 10), 1)
}

// Debug enables debug mode, where unused columns and struct fields will be logged
var Debug = true

type structField struct {
	column     string
	index      int
	primaryKey bool
	meddler    Meddler
}

type structData struct {
	columns []string
	fields  map[string]*structField
	pk      string
}

// cache reflection data
var fieldsCache = make(map[reflect.Type]*structData)
var fieldsCacheMutex sync.Mutex

// getFields gathers the list of columns from a struct using reflection.
func getFields(dstType reflect.Type) (*structData, error) {
	fieldsCacheMutex.Lock()
	defer fieldsCacheMutex.Unlock()

	if result, present := fieldsCache[dstType]; present {
		return result, nil
	}

	// make sure dst is a non-nil pointer to a struct
	if dstType.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("meddler called with non-pointer destination %v", dstType)
	}
	structType := dstType.Elem()
	if structType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("meddler called with pointer to non-struct %v", dstType)
	}

	// gather the list of fields in the struct
	data := new(structData)
	data.fields = make(map[string]*structField)

	for i := 0; i < structType.NumField(); i++ {
		f := structType.Field(i)

		// skip non-exported fields
		if f.PkgPath != "" {
			continue
		}

		// examine the tag for metadata
		tag := strings.Split(f.Tag.Get(tagName), ",")

		// was this field marked for skipping?
		if len(tag) > 0 && tag[0] == "-" {
			continue
		}

		// default to the field name
		name := f.Name

		// the tag can override the field name
		if len(tag) > 0 && tag[0] != "" {
			name = tag[0]
		}

		// check for a meddler
		var meddler Meddler = registry["identity"]
		for j := 1; j < len(tag); j++ {
			if tag[j] == "pk" {
				if f.Type.Kind() == reflect.Ptr {
					return nil, fmt.Errorf("meddler found field %s which is marked as the primary key but is a pointer", f.Name)
				}

				// make sure it is an int of some kind
				switch f.Type.Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				default:
					return nil, fmt.Errorf("meddler found field %s which is marked as the primary key, but is not an integer type", f.Name)
				}

				if data.pk != "" {
					return nil, fmt.Errorf("meddler found field %s which is marked as the primary key, but a primary key field was already found", f.Name)
				}
				data.pk = name
			} else if m, present := registry[tag[j]]; present {
				meddler = m
			} else {
				return nil, fmt.Errorf("meddler found field %s with meddler %s, but that meddler is not registered", f.Name, tag[j])
			}
		}

		if _, present := data.fields[name]; present {
			return nil, fmt.Errorf("meddler found multiple fields for column %s", name)
		}
		data.fields[name] = &structField{
			column:     name,
			primaryKey: name == data.pk,
			index:      i,
			meddler:    meddler,
		}
		data.columns = append(data.columns, name)
	}

	fieldsCache[dstType] = data
	return data, nil
}

// Columns returns a list of column names for its input struct.
func (d *Database) Columns(src interface{}, includePk bool) ([]string, error) {
	data, err := getFields(reflect.TypeOf(src))
	if err != nil {
		return nil, err
	}

	var names []string
	for _, elt := range data.columns {
		if !includePk && elt == data.pk {
			continue
		}
		names = append(names, elt)
	}

	return names, nil
}

// Columns using the Default Database type
func Columns(src interface{}, includePk bool) ([]string, error) {
	return Default.Columns(src, includePk)
}

// ColumnsQuoted is similar to Columns, but it return the list of columns in the form:
//   `column1`,`column2`,...
// using Quote as the quote character.
func (d *Database) ColumnsQuoted(src interface{}, includePk bool) (string, error) {
	unquoted, err := Columns(src, includePk)
	if err != nil {
		return "", err
	}

	var parts []string
	for _, elt := range unquoted {
		parts = append(parts, d.quoted(elt))
	}

	return strings.Join(parts, ","), nil
}

// ColumnsQuoted using the Default Database type
func ColumnsQuoted(src interface{}, includePk bool) (string, error) {
	return Default.ColumnsQuoted(src, includePk)
}

// PrimaryKey returns the name and value of the primary key field. The name
// is the empty string if there is not primary key field marked.
func (d *Database) PrimaryKey(src interface{}) (name string, pk int64, err error) {
	data, err := getFields(reflect.TypeOf(src))
	if err != nil {
		return "", 0, err
	}

	if data.pk == "" {
		return "", 0, nil
	}

	name = data.pk
	field := reflect.ValueOf(src).Elem().Field(data.fields[name].index)
	switch field.Type().Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		pk = field.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		pk = int64(field.Uint())
	default:
		return "", 0, fmt.Errorf("meddler found field %s which is marked as the primary key, but is not an integer type", name)
	}

	return name, pk, nil
}

// PrimaryKey using the Default Database type
func PrimaryKey(src interface{}) (name string, pk int64, err error) {
	return Default.PrimaryKey(src)
}

// SetPrimaryKey sets the primary key field to the given int value.
func (d *Database) SetPrimaryKey(src interface{}, pk int64) error {
	data, err := getFields(reflect.TypeOf(src))
	if err != nil {
		return err
	}

	if data.pk == "" {
		return fmt.Errorf("meddler.SetPrimaryKey: no primary key field found")
	}

	field := reflect.ValueOf(src).Elem().Field(data.fields[data.pk].index)
	switch field.Type().Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		field.SetInt(pk)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		field.SetUint(uint64(pk))
	default:
		return fmt.Errorf("meddler found field %s which is marked as the primary key, but is not an integer type", data.pk)
	}

	return nil
}

// SetPrimaryKey using the Default Database type
func SetPrimaryKey(src interface{}, pk int64) error {
	return Default.SetPrimaryKey(src, pk)
}

// Values returns a list of PreWrite processed values suitable for
// use in an INSERT or UPDATE query. If includePk is false, the primary
// key field is omitted. The columns used are the same ones (in the same
// order) as returned by Columns.
func (d *Database) Values(src interface{}, includePk bool) ([]interface{}, error) {
	columns, err := d.Columns(src, includePk)
	if err != nil {
		return nil, err
	}
	return d.SomeValues(src, columns)
}

// Values using the Default Database type
func Values(src interface{}, includePk bool) ([]interface{}, error) {
	return Default.Values(src, includePk)
}

// SomeValues returns a list of PreWrite processed values suitable for
// use in an INSERT or UPDATE query. The columns used are the same ones (in
// the same order) as specified in the columns argument.
func (d *Database) SomeValues(src interface{}, columns []string) ([]interface{}, error) {
	data, err := getFields(reflect.TypeOf(src))
	if err != nil {
		return nil, err
	}
	structVal := reflect.ValueOf(src).Elem()

	var values []interface{}
	for _, name := range columns {
		field, present := data.fields[name]
		if !present {
			// write null to the database
			values = append(values, nil)

			if Debug {
				log.Printf("meddler.SomeValues: column [%s] not found in struct", name)
			}
			continue
		}

		saveVal, err := field.meddler.PreWrite(structVal.Field(field.index).Interface())
		if err != nil {
			return nil, fmt.Errorf("meddler.SomeValues: PreWrite error on column [%s]: %v", name, err)
		}
		values = append(values, saveVal)
	}

	return values, nil
}

// SomeValues using the Default Database type
func SomeValues(src interface{}, columns []string) ([]interface{}, error) {
	return Default.SomeValues(src, columns)
}

// Placeholders returns a list of placeholders suitable for an INSERT or UPDATE query.
// If includePk is false, the primary key field is omitted.
func (d *Database) Placeholders(src interface{}, includePk bool) ([]string, error) {
	data, err := getFields(reflect.TypeOf(src))
	if err != nil {
		return nil, err
	}

	var placeholders []string
	for _, name := range data.columns {
		if !includePk && name == data.pk {
			continue
		}
		ph := d.placeholder(len(placeholders) + 1)
		placeholders = append(placeholders, ph)
	}

	return placeholders, nil
}

// Placeholders using the Default Database type
func Placeholders(src interface{}, includePk bool) ([]string, error) {
	return Default.Placeholders(src, includePk)
}

// PlaceholdersString returns a list of placeholders suitable for an INSERT
// or UPDATE query in string form, e.g.:
//   ?,?,?,?
// if includePk is false, the primary key field is omitted.
func (d *Database) PlaceholdersString(src interface{}, includePk bool) (string, error) {
	lst, err := d.Placeholders(src, includePk)
	if err != nil {
		return "", err
	}
	return strings.Join(lst, ","), nil
}

// PlaceholdersString using the Default Database type
func PlaceholdersString(src interface{}, includePk bool) (string, error) {
	return Default.PlaceholdersString(src, includePk)
}

// scan a single row of data into a struct.
func (d *Database) scanRow(data *structData, rows *sql.Rows, dst interface{}, columns []string) error {
	// check if there is data waiting
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}

	// get a list of targets
	targets, err := d.Targets(dst, columns)
	if err != nil {
		return err
	}

	// perform the scan
	if err := rows.Scan(targets...); err != nil {
		return err
	}

	// post-process and copy the target values into the struct
	if err := d.WriteTargets(dst, columns, targets); err != nil {
		return err
	}

	return rows.Err()
}

// Targets returns a list of values suitable for handing to a
// Scan function in the sql package, complete with meddling. After
// the Scan is performed, the same values should be handed to
// WriteTargets to finalize the values and record them in the struct.
func (d *Database) Targets(dst interface{}, columns []string) ([]interface{}, error) {
	data, err := getFields(reflect.TypeOf(dst))
	if err != nil {
		return nil, err
	}

	structVal := reflect.ValueOf(dst).Elem()

	var targets []interface{}
	for _, name := range columns {
		if field, present := data.fields[name]; present {
			fieldAddr := structVal.Field(field.index).Addr().Interface()
			scanTarget, err := field.meddler.PreRead(fieldAddr)
			if err != nil {
				return nil, fmt.Errorf("meddler.Targets: PreRead error on column %s: %v", name, err)
			}
			targets = append(targets, scanTarget)
		} else {
			// no destination, so throw this away
			targets = append(targets, new(interface{}))

			if Debug {
				log.Printf("meddler.Targets: column [%s] not found in struct", name)
			}
		}
	}

	return targets, nil
}

// Targets using the Default Database type
func Targets(dst interface{}, columns []string) ([]interface{}, error) {
	return Default.Targets(dst, columns)
}

// WriteTargets post-processes values with meddlers after a Scan from the
// sql package has been performed. The list of targets is normally produced
// by Targets.
func (d *Database) WriteTargets(dst interface{}, columns []string, targets []interface{}) error {
	if len(columns) != len(targets) {
		return fmt.Errorf("meddler.WriteTargets: mismatch in number of columns (%d) and targets (%s)",
			len(columns), len(targets))
	}

	data, err := getFields(reflect.TypeOf(dst))
	if err != nil {
		return err
	}
	structVal := reflect.ValueOf(dst).Elem()

	for i, name := range columns {
		if field, present := data.fields[name]; present {
			fieldAddr := structVal.Field(field.index).Addr().Interface()
			err := field.meddler.PostRead(fieldAddr, targets[i])
			if err != nil {
				return fmt.Errorf("meddler.WriteTargets: PostRead error on column [%s]: %v", name, err)
			}
		} else {
			// not destination, so throw this away
			if Debug {
				log.Printf("meddler.WriteTargets: column [%s] not found in struct", name)
			}
		}
	}

	return nil
}

// WriteTargets using the Default Database type
func WriteTargets(dst interface{}, columns []string, targets []interface{}) error {
	return Default.WriteTargets(dst, columns, targets)
}

// Scan scans a single sql result row into a struct.
// It leaves rows ready to be scanned again for the next row.
// Returns sql.ErrNoRows if there is no data to read.
func (d *Database) Scan(rows *sql.Rows, dst interface{}) error {
	// get the list of struct fields
	data, err := getFields(reflect.TypeOf(dst))
	if err != nil {
		return err
	}

	// get the sql columns
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	return d.scanRow(data, rows, dst, columns)
}

// Scan using the Default Database type
func Scan(rows *sql.Rows, dst interface{}) error {
	return Default.Scan(rows, dst)
}

// ScanRow scans a single sql result row into a struct.
// It reads exactly one result row and closes rows when finished.
// Returns sql.ErrNoRows if there is no result row.
func (d *Database) ScanRow(rows *sql.Rows, dst interface{}) error {
	// make sure we always close rows
	defer rows.Close()

	if err := d.Scan(rows, dst); err != nil {
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	return nil
}

// ScanRow using the Default Database type
func ScanRow(rows *sql.Rows, dst interface{}) error {
	return Default.ScanRow(rows, dst)
}

// ScanAll scans all sql result rows into a slice of structs.
// It reads all rows and closes rows when finished.
// dst should be a pointer to a slice of the appropriate type.
// The new results will be appended to any existing data in dst.
func (d *Database) ScanAll(rows *sql.Rows, dst interface{}) error {
	// make sure we always close rows
	defer rows.Close()

	// make sure dst is an appropriate type
	dstVal := reflect.ValueOf(dst)
	if dstVal.Kind() != reflect.Ptr || dstVal.IsNil() {
		return fmt.Errorf("ScanAll called with non-pointer destination: %T", dst)
	}
	sliceVal := dstVal.Elem()
	if sliceVal.Kind() != reflect.Slice {
		return fmt.Errorf("ScanAll called with pointer to non-slice: %T", dst)
	}
	ptrType := sliceVal.Type().Elem()
	if ptrType.Kind() != reflect.Ptr {
		return fmt.Errorf("ScanAll expects element to be pointers, found %T", dst)
	}
	eltType := ptrType.Elem()
	if eltType.Kind() != reflect.Struct {
		return fmt.Errorf("ScanAll expects element to be pointers to structs, found %T", dst)
	}

	// get the list of struct fields
	data, err := getFields(ptrType)
	if err != nil {
		return err
	}

	// get the sql columns
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	// gather the results
	for {
		// create a new element
		eltVal := reflect.New(eltType)
		elt := eltVal.Interface()

		// scan it
		if err := d.scanRow(data, rows, elt, columns); err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}

		// add to the result slice
		sliceVal.Set(reflect.Append(sliceVal, eltVal))
	}
}

// ScanAll using the Default Database type
func ScanAll(rows *sql.Rows, dst interface{}) error {
	return Default.ScanAll(rows, dst)
}
