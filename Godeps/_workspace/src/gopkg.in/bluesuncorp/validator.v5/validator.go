/**
 * Package validator
 *
 * MISC:
 * - anonymous structs - they don't have names so expect the Struct name within StructErrors to be blank
 *
 */

package validator

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	utf8HexComma    = "0x2C"
	tagSeparator    = ","
	orSeparator     = "|"
	noValidationTag = "-"
	tagKeySeparator = "="
	structOnlyTag   = "structonly"
	omitempty       = "omitempty"
	fieldErrMsg     = "Field validation for \"%s\" failed on the \"%s\" tag"
	structErrMsg    = "Struct:%s\n"
)

var structPool *pool

// Pool holds a channelStructErrors.
type pool struct {
	pool chan *StructErrors
}

// NewPool creates a new pool of Clients.
func newPool(max int) *pool {
	return &pool{
		pool: make(chan *StructErrors, max),
	}
}

// Borrow a StructErrors from the pool.
func (p *pool) Borrow() *StructErrors {
	var c *StructErrors

	select {
	case c = <-p.pool:
	default:
		c = &StructErrors{
			Errors:       map[string]*FieldError{},
			StructErrors: map[string]*StructErrors{},
		}
	}

	return c
}

// Return returns a StructErrors to the pool.
func (p *pool) Return(c *StructErrors) {

	// c.Struct = ""

	select {
	case p.pool <- c:
	default:
		// let it go, let it go...
	}
}

type cachedTags struct {
	keyVals [][]string
	isOrVal bool
}

type cachedField struct {
	index  int
	name   string
	tags   []*cachedTags
	tag    string
	kind   reflect.Kind
	typ    reflect.Type
	isTime bool
}

type cachedStruct struct {
	children int
	name     string
	kind     reflect.Kind
	fields   []*cachedField
}

type structsCacheMap struct {
	lock sync.RWMutex
	m    map[reflect.Type]*cachedStruct
}

func (s *structsCacheMap) Get(key reflect.Type) (*cachedStruct, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	value, ok := s.m[key]
	return value, ok
}

func (s *structsCacheMap) Set(key reflect.Type, value *cachedStruct) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.m[key] = value
}

var structCache = &structsCacheMap{m: map[reflect.Type]*cachedStruct{}}

type fieldsCacheMap struct {
	lock sync.RWMutex
	m    map[string][]*cachedTags
}

func (s *fieldsCacheMap) Get(key string) ([]*cachedTags, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	value, ok := s.m[key]
	return value, ok
}

func (s *fieldsCacheMap) Set(key string, value []*cachedTags) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.m[key] = value
}

var fieldsCache = &fieldsCacheMap{m: map[string][]*cachedTags{}}

// FieldError contains a single field's validation error along
// with other properties that may be needed for error message creation
type FieldError struct {
	Field string
	Tag   string
	Kind  reflect.Kind
	Type  reflect.Type
	Param string
	Value interface{}
}

// This is intended for use in development + debugging and not intended to be a production error message.
// it also allows FieldError to be used as an Error interface
func (e *FieldError) Error() string {
	return fmt.Sprintf(fieldErrMsg, e.Field, e.Tag)
}

// StructErrors is hierarchical list of field and struct validation errors
// for a non hierarchical representation please see the Flatten method for StructErrors
type StructErrors struct {
	// Name of the Struct
	Struct string
	// Struct Field Errors
	Errors map[string]*FieldError
	// Struct Fields of type struct and their errors
	// key = Field Name of current struct, but internally Struct will be the actual struct name unless anonymous struct, it will be blank
	StructErrors map[string]*StructErrors
}

// This is intended for use in development + debugging and not intended to be a production error message.
// it also allows StructErrors to be used as an Error interface
func (e *StructErrors) Error() string {
	buff := bytes.NewBufferString(fmt.Sprintf(structErrMsg, e.Struct))

	for _, err := range e.Errors {
		buff.WriteString(err.Error())
		buff.WriteString("\n")
	}

	for _, err := range e.StructErrors {
		buff.WriteString(err.Error())
	}

	return buff.String()
}

// Flatten flattens the StructErrors hierarchical structure into a flat namespace style field name
// for those that want/need it
func (e *StructErrors) Flatten() map[string]*FieldError {

	if e == nil {
		return nil
	}

	errs := map[string]*FieldError{}

	for _, f := range e.Errors {

		errs[f.Field] = f
	}

	for key, val := range e.StructErrors {

		otherErrs := val.Flatten()

		for _, f2 := range otherErrs {

			f2.Field = fmt.Sprintf("%s.%s", key, f2.Field)
			errs[f2.Field] = f2
		}
	}

	return errs
}

// Func accepts all values needed for file and cross field validation
// top     = top level struct when validating by struct otherwise nil
// current = current level struct when validating by struct otherwise optional comparison value
// f       = field value for validation
// param   = parameter used in validation i.e. gt=0 param would be 0
type Func func(top interface{}, current interface{}, f interface{}, param string) bool

// Validate implements the Validate Struct
// NOTE: Fields within are not thread safe and that is on purpose
// Functions and Tags should all be predifined before use, so subscribe to the philosiphy
// or make it thread safe on your end
type Validate struct {
	// tagName being used.
	tagName string
	// validateFuncs is a map of validation functions and the tag keys
	validationFuncs map[string]Func
}

// New creates a new Validate instance for use.
func New(tagName string, funcs map[string]Func) *Validate {

	structPool = newPool(10)

	return &Validate{
		tagName:         tagName,
		validationFuncs: funcs,
	}
}

// SetTag sets tagName of the Validator to one of your choosing after creation
// perhaps to dodge a tag name conflict in a specific section of code
// NOTE: this method is not thread-safe
func (v *Validate) SetTag(tagName string) {
	v.tagName = tagName
}

// SetStructPoolMax sets the  struct pools max size. this may be usefull for fine grained
// performance tuning towards your application, however, the default should be fine for
// nearly all cases. only increase if you have a deeply nested struct structure.
// NOTE: this method is not thread-safe
// NOTE: this is only here to keep compatibility with v5, in v6 the method will be removed
// and the max pool size will be passed into the New function
func (v *Validate) SetMaxStructPoolSize(max int) {
	structPool = newPool(max)
}

// AddFunction adds a validation Func to a Validate's map of validators denoted by the key
// NOTE: if the key already exists, it will get replaced.
// NOTE: this method is not thread-safe
func (v *Validate) AddFunction(key string, f Func) error {

	if len(key) == 0 {
		return errors.New("Function Key cannot be empty")
	}

	if f == nil {
		return errors.New("Function cannot be empty")
	}

	v.validationFuncs[key] = f

	return nil
}

// Struct validates a struct, even it's nested structs, and returns a struct containing the errors
// NOTE: Nested Arrays, or Maps of structs do not get validated only the Array or Map itself; the reason is that there is no good
// way to represent or report which struct within the array has the error, besides can validate the struct prior to adding it to
// the Array or Map.
func (v *Validate) Struct(s interface{}) *StructErrors {

	return v.structRecursive(s, s, s)
}

// structRecursive validates a struct recursivly and passes the top level and current struct around for use in validator functions and returns a struct containing the errors
func (v *Validate) structRecursive(top interface{}, current interface{}, s interface{}) *StructErrors {

	structValue := reflect.ValueOf(s)

	if structValue.Kind() == reflect.Ptr && !structValue.IsNil() {
		return v.structRecursive(top, current, structValue.Elem().Interface())
	}

	if structValue.Kind() != reflect.Struct && structValue.Kind() != reflect.Interface {
		panic("interface passed for validation is not a struct")
	}

	structType := reflect.TypeOf(s)

	var structName string
	var numFields int
	var cs *cachedStruct
	var isCached bool

	cs, isCached = structCache.Get(structType)

	if isCached {
		structName = cs.name
		numFields = cs.children
	} else {
		structName = structType.Name()
		numFields = structValue.NumField()
		cs = &cachedStruct{name: structName, children: numFields}
		structCache.Set(structType, cs)
	}

	validationErrors := structPool.Borrow()
	validationErrors.Struct = structName

	for i := 0; i < numFields; i++ {

		var valueField reflect.Value
		var cField *cachedField
		var typeField reflect.StructField

		if isCached {
			cField = cs.fields[i]
			valueField = structValue.Field(cField.index)

			if valueField.Kind() == reflect.Ptr && !valueField.IsNil() {
				valueField = valueField.Elem()
			}
		} else {
			valueField = structValue.Field(i)

			if valueField.Kind() == reflect.Ptr && !valueField.IsNil() {
				valueField = valueField.Elem()
			}

			typeField = structType.Field(i)

			cField = &cachedField{index: i, tag: typeField.Tag.Get(v.tagName)}

			if cField.tag == noValidationTag {
				cs.children--
				continue
			}

			// if no validation and not a struct (which may containt fields for validation)
			if cField.tag == "" && ((valueField.Kind() != reflect.Struct && valueField.Kind() != reflect.Interface) || valueField.Type() == reflect.TypeOf(time.Time{})) {
				cs.children--
				continue
			}

			cField.name = typeField.Name
			cField.kind = valueField.Kind()
			cField.typ = valueField.Type()
		}

		// this can happen if the first cache value was nil
		// but the second actually has a value
		if cField.kind == reflect.Ptr {
			cField.kind = valueField.Kind()
		}

		switch cField.kind {

		case reflect.Struct, reflect.Interface:

			if !unicode.IsUpper(rune(cField.name[0])) {
				cs.children--
				continue
			}

			if cField.isTime || valueField.Type() == reflect.TypeOf(time.Time{}) {

				cField.isTime = true

				if fieldError := v.fieldWithNameAndValue(top, current, valueField.Interface(), cField.tag, cField.name, false, cField); fieldError != nil {
					validationErrors.Errors[fieldError.Field] = fieldError
					// free up memory reference
					fieldError = nil
				}

			} else {

				if strings.Contains(cField.tag, structOnlyTag) {
					cs.children--
					continue
				}

				if structErrors := v.structRecursive(top, valueField.Interface(), valueField.Interface()); structErrors != nil {
					validationErrors.StructErrors[cField.name] = structErrors
					// free up memory map no longer needed
					structErrors = nil
				}
			}

		default:

			if fieldError := v.fieldWithNameAndValue(top, current, valueField.Interface(), cField.tag, cField.name, false, cField); fieldError != nil {
				validationErrors.Errors[fieldError.Field] = fieldError
				// free up memory reference
				fieldError = nil
			}
		}

		if !isCached {
			cs.fields = append(cs.fields, cField)
		}
	}

	if len(validationErrors.Errors) == 0 && len(validationErrors.StructErrors) == 0 {
		structPool.Return(validationErrors)
		return nil
	}

	return validationErrors
}

// Field allows validation of a single field, still using tag style validation to check multiple errors
func (v *Validate) Field(f interface{}, tag string) *FieldError {

	return v.FieldWithValue(nil, f, tag)
}

// FieldWithValue allows validation of a single field, possibly even against another fields value, still using tag style validation to check multiple errors
func (v *Validate) FieldWithValue(val interface{}, f interface{}, tag string) *FieldError {

	return v.fieldWithNameAndValue(nil, val, f, tag, "", true, nil)
}

func (v *Validate) fieldWithNameAndValue(val interface{}, current interface{}, f interface{}, tag string, name string, isSingleField bool, cacheField *cachedField) *FieldError {

	var cField *cachedField
	var isCached bool

	// This is a double check if coming from validate.Struct but need to be here in case function is called directly
	if tag == noValidationTag {
		return nil
	}

	if strings.Contains(tag, omitempty) && !hasValue(val, current, f, "") {
		return nil
	}

	if cacheField == nil {
		valueField := reflect.ValueOf(f)

		if valueField.Kind() == reflect.Ptr && !valueField.IsNil() {
			valueField = valueField.Elem()
			f = valueField.Interface()
		}

		cField = &cachedField{name: name, kind: valueField.Kind(), tag: tag, typ: valueField.Type()}
	} else {
		cField = cacheField
	}

	switch cField.kind {

	case reflect.Struct, reflect.Interface, reflect.Invalid:

		if cField.typ != reflect.TypeOf(time.Time{}) {
			panic("Invalid field passed to ValidateFieldWithTag")
		}
	}

	if len(cField.tags) == 0 {

		if isSingleField {
			cField.tags, isCached = fieldsCache.Get(tag)
		}

		if !isCached {

			for _, t := range strings.Split(tag, tagSeparator) {

				orVals := strings.Split(t, orSeparator)
				cTag := &cachedTags{isOrVal: len(orVals) > 1, keyVals: make([][]string, len(orVals))}
				cField.tags = append(cField.tags, cTag)

				for i, val := range orVals {
					vals := strings.SplitN(val, tagKeySeparator, 2)

					key := strings.TrimSpace(vals[0])

					if len(key) == 0 {
						panic(fmt.Sprintf("Invalid validation tag on field %s", name))
					}

					param := ""
					if len(vals) > 1 {
						param = strings.Replace(vals[1], utf8HexComma, ",", -1)
					}

					cTag.keyVals[i] = []string{key, param}
				}
			}

			if isSingleField {
				fieldsCache.Set(cField.tag, cField.tags)
			}
		}
	}

	var fieldErr *FieldError
	var err error

	for _, cTag := range cField.tags {

		if cTag.isOrVal {

			errTag := ""

			for _, val := range cTag.keyVals {

				fieldErr, err = v.fieldWithNameAndSingleTag(val, current, f, val[0], val[1], name)

				if err == nil {
					return nil
				}

				errTag += orSeparator + fieldErr.Tag
			}

			errTag = strings.TrimLeft(errTag, orSeparator)

			fieldErr.Tag = errTag
			fieldErr.Kind = cField.kind
			fieldErr.Type = cField.typ

			return fieldErr
		}

		if fieldErr, err = v.fieldWithNameAndSingleTag(val, current, f, cTag.keyVals[0][0], cTag.keyVals[0][1], name); err != nil {

			fieldErr.Kind = cField.kind
			fieldErr.Type = cField.typ

			return fieldErr
		}
	}

	return nil
}

func (v *Validate) fieldWithNameAndSingleTag(val interface{}, current interface{}, f interface{}, key string, param string, name string) (*FieldError, error) {

	// OK to continue because we checked it's existance before getting into this loop
	if key == omitempty {
		return nil, nil
	}

	valFunc, ok := v.validationFuncs[key]
	if !ok {
		panic(fmt.Sprintf("Undefined validation function on field %s", name))
	}

	if err := valFunc(val, current, f, param); err {
		return nil, nil
	}

	return &FieldError{
		Field: name,
		Tag:   key,
		Value: f,
		Param: param,
	}, errors.New(key)
}
