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
	utf8HexComma        = "0x2C"
	tagSeparator        = ","
	orSeparator         = "|"
	noValidationTag     = "-"
	tagKeySeparator     = "="
	structOnlyTag       = "structonly"
	omitempty           = "omitempty"
	required            = "required"
	fieldErrMsg         = "Field validation for \"%s\" failed on the \"%s\" tag"
	sliceErrMsg         = "Field validation for \"%s\" failed at index \"%d\" with error(s): %s"
	mapErrMsg           = "Field validation for \"%s\" failed on key \"%v\" with error(s): %s"
	structErrMsg        = "Struct:%s\n"
	diveTag             = "dive"
	existsTag           = "exists"
	arrayIndexFieldName = "%s[%d]"
	mapIndexFieldName   = "%s[%v]"
)

var structPool *sync.Pool

// returns new *StructErrors to the pool
func newStructErrors() interface{} {
	return &StructErrors{
		Errors:       map[string]*FieldError{},
		StructErrors: map[string]*StructErrors{},
	}
}

type cachedTags struct {
	keyVals [][]string
	isOrVal bool
}

type cachedField struct {
	index          int
	name           string
	tags           []*cachedTags
	tag            string
	kind           reflect.Kind
	typ            reflect.Type
	isTime         bool
	isSliceOrArray bool
	isMap          bool
	isTimeSubtype  bool
	sliceSubtype   reflect.Type
	mapSubtype     reflect.Type
	sliceSubKind   reflect.Kind
	mapSubKind     reflect.Kind
	dive           bool
	diveTag        string
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
	Field            string
	Tag              string
	Kind             reflect.Kind
	Type             reflect.Type
	Param            string
	Value            interface{}
	IsPlaceholderErr bool
	IsSliceOrArray   bool
	IsMap            bool
	SliceOrArrayErrs map[int]error         // counld be FieldError, StructErrors
	MapErrs          map[interface{}]error // counld be FieldError, StructErrors
}

// This is intended for use in development + debugging and not intended to be a production error message.
// it also allows FieldError to be used as an Error interface
func (e *FieldError) Error() string {

	if e.IsPlaceholderErr {

		buff := bytes.NewBufferString("")

		if e.IsSliceOrArray {

			for j, err := range e.SliceOrArrayErrs {
				buff.WriteString("\n")
				buff.WriteString(fmt.Sprintf(sliceErrMsg, e.Field, j, "\n"+err.Error()))
			}

		} else if e.IsMap {

			for key, err := range e.MapErrs {
				buff.WriteString(fmt.Sprintf(mapErrMsg, e.Field, key, "\n"+err.Error()))
			}
		}

		return strings.TrimSpace(buff.String())
	}

	return fmt.Sprintf(fieldErrMsg, e.Field, e.Tag)
}

// Flatten flattens the FieldError hierarchical structure into a flat namespace style field name
// for those that want/need it.
// This is now needed because of the new dive functionality
func (e *FieldError) Flatten() map[string]*FieldError {

	errs := map[string]*FieldError{}

	if e.IsPlaceholderErr {

		if e.IsSliceOrArray {
			for key, err := range e.SliceOrArrayErrs {

				fe, ok := err.(*FieldError)

				if ok {

					if flat := fe.Flatten(); flat != nil && len(flat) > 0 {
						for k, v := range flat {
							if fe.IsPlaceholderErr {
								errs[fmt.Sprintf("[%#v]%s", key, k)] = v
							} else {
								errs[fmt.Sprintf("[%#v]", key)] = v
							}

						}
					}
				} else {

					se := err.(*StructErrors)

					if flat := se.Flatten(); flat != nil && len(flat) > 0 {
						for k, v := range flat {
							errs[fmt.Sprintf("[%#v].%s.%s", key, se.Struct, k)] = v
						}
					}
				}
			}
		}

		if e.IsMap {
			for key, err := range e.MapErrs {

				fe, ok := err.(*FieldError)

				if ok {

					if flat := fe.Flatten(); flat != nil && len(flat) > 0 {
						for k, v := range flat {
							if fe.IsPlaceholderErr {
								errs[fmt.Sprintf("[%#v]%s", key, k)] = v
							} else {
								errs[fmt.Sprintf("[%#v]", key)] = v
							}
						}
					}
				} else {

					se := err.(*StructErrors)

					if flat := se.Flatten(); flat != nil && len(flat) > 0 {
						for k, v := range flat {
							errs[fmt.Sprintf("[%#v].%s.%s", key, se.Struct, k)] = v
						}
					}
				}
			}
		}

		return errs
	}

	errs[e.Field] = e

	return errs
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

	return strings.TrimSpace(buff.String())
}

// Flatten flattens the StructErrors hierarchical structure into a flat namespace style field name
// for those that want/need it
func (e *StructErrors) Flatten() map[string]*FieldError {

	if e == nil {
		return nil
	}

	errs := map[string]*FieldError{}

	for _, f := range e.Errors {

		if flat := f.Flatten(); flat != nil && len(flat) > 0 {

			for k, fe := range flat {

				if f.IsPlaceholderErr {
					errs[f.Field+k] = fe
				} else {
					errs[k] = fe
				}
			}
		}
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

	structPool = &sync.Pool{New: newStructErrors}

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

// SetMaxStructPoolSize sets the  struct pools max size. this may be usefull for fine grained
// performance tuning towards your application, however, the default should be fine for
// nearly all cases. only increase if you have a deeply nested struct structure.
// NOTE: this method is not thread-safe
// NOTE: this is only here to keep compatibility with v5, in v6 the method will be removed
func (v *Validate) SetMaxStructPoolSize(max int) {
	structPool = &sync.Pool{New: newStructErrors}
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
	}

	validationErrors := structPool.Get().(*StructErrors)
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

			cField = &cachedField{index: i, tag: typeField.Tag.Get(v.tagName), isTime: (valueField.Type() == reflect.TypeOf(time.Time{}) || valueField.Type() == reflect.TypeOf(&time.Time{}))}

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

			if cField.isTime {

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

				if (valueField.Kind() == reflect.Ptr || cField.kind == reflect.Interface) && valueField.IsNil() {

					if strings.Contains(cField.tag, omitempty) {
						goto CACHEFIELD
					}

					tags := strings.Split(cField.tag, tagSeparator)

					if len(tags) > 0 {

						var param string
						vals := strings.SplitN(tags[0], tagKeySeparator, 2)

						if len(vals) > 1 {
							param = vals[1]
						}

						validationErrors.Errors[cField.name] = &FieldError{
							Field: cField.name,
							Tag:   vals[0],
							Param: param,
							Value: valueField.Interface(),
							Kind:  valueField.Kind(),
							Type:  valueField.Type(),
						}

						goto CACHEFIELD
					}
				}

				// if we get here, the field is interface and could be a struct or a field
				// and we need to check the inner type and validate
				if cField.kind == reflect.Interface {

					valueField = valueField.Elem()

					if valueField.Kind() == reflect.Ptr && !valueField.IsNil() {
						valueField = valueField.Elem()
					}

					if valueField.Kind() == reflect.Struct {
						goto VALIDATESTRUCT
					}

					// sending nil for cField as it was type interface and could be anything
					// each time and so must be calculated each time and can't be cached reliably
					if fieldError := v.fieldWithNameAndValue(top, current, valueField.Interface(), cField.tag, cField.name, false, nil); fieldError != nil {
						validationErrors.Errors[fieldError.Field] = fieldError
						// free up memory reference
						fieldError = nil
					}

					goto CACHEFIELD
				}

			VALIDATESTRUCT:
				if structErrors := v.structRecursive(top, valueField.Interface(), valueField.Interface()); structErrors != nil {
					validationErrors.StructErrors[cField.name] = structErrors
					// free up memory map no longer needed
					structErrors = nil
				}
			}

		case reflect.Slice, reflect.Array:
			cField.isSliceOrArray = true
			cField.sliceSubtype = cField.typ.Elem()
			cField.isTimeSubtype = (cField.sliceSubtype == reflect.TypeOf(time.Time{}) || cField.sliceSubtype == reflect.TypeOf(&time.Time{}))
			cField.sliceSubKind = cField.sliceSubtype.Kind()

			if fieldError := v.fieldWithNameAndValue(top, current, valueField.Interface(), cField.tag, cField.name, false, cField); fieldError != nil {
				validationErrors.Errors[fieldError.Field] = fieldError
				// free up memory reference
				fieldError = nil
			}

		case reflect.Map:
			cField.isMap = true
			cField.mapSubtype = cField.typ.Elem()
			cField.isTimeSubtype = (cField.mapSubtype == reflect.TypeOf(time.Time{}) || cField.mapSubtype == reflect.TypeOf(&time.Time{}))
			cField.mapSubKind = cField.mapSubtype.Kind()

			if fieldError := v.fieldWithNameAndValue(top, current, valueField.Interface(), cField.tag, cField.name, false, cField); fieldError != nil {
				validationErrors.Errors[fieldError.Field] = fieldError
				// free up memory reference
				fieldError = nil
			}

		default:
			if fieldError := v.fieldWithNameAndValue(top, current, valueField.Interface(), cField.tag, cField.name, false, cField); fieldError != nil {
				validationErrors.Errors[fieldError.Field] = fieldError
				// free up memory reference
				fieldError = nil
			}
		}

	CACHEFIELD:
		if !isCached {
			cs.fields = append(cs.fields, cField)
		}
	}

	structCache.Set(structType, cs)

	if len(validationErrors.Errors) == 0 && len(validationErrors.StructErrors) == 0 {
		structPool.Put(validationErrors)
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
	var valueField reflect.Value

	// This is a double check if coming from validate.Struct but need to be here in case function is called directly
	if tag == noValidationTag || tag == "" {
		return nil
	}

	if strings.Contains(tag, omitempty) && !hasValue(val, current, f, "") {
		return nil
	}

	valueField = reflect.ValueOf(f)

	if cacheField == nil {

		if valueField.Kind() == reflect.Ptr && !valueField.IsNil() {
			valueField = valueField.Elem()
			f = valueField.Interface()
		}

		cField = &cachedField{name: name, kind: valueField.Kind(), tag: tag}

		if cField.kind != reflect.Invalid {
			cField.typ = valueField.Type()
		}

		switch cField.kind {
		case reflect.Slice, reflect.Array:
			isSingleField = false // cached tags mean nothing because it will be split up while diving
			cField.isSliceOrArray = true
			cField.sliceSubtype = cField.typ.Elem()
			cField.isTimeSubtype = (cField.sliceSubtype == reflect.TypeOf(time.Time{}) || cField.sliceSubtype == reflect.TypeOf(&time.Time{}))
			cField.sliceSubKind = cField.sliceSubtype.Kind()
		case reflect.Map:
			isSingleField = false // cached tags mean nothing because it will be split up while diving
			cField.isMap = true
			cField.mapSubtype = cField.typ.Elem()
			cField.isTimeSubtype = (cField.mapSubtype == reflect.TypeOf(time.Time{}) || cField.mapSubtype == reflect.TypeOf(&time.Time{}))
			cField.mapSubKind = cField.mapSubtype.Kind()
		}
	} else {
		cField = cacheField
	}

	switch cField.kind {
	case reflect.Invalid:
		return &FieldError{
			Field: cField.name,
			Tag:   cField.tag,
			Kind:  cField.kind,
		}

	case reflect.Struct, reflect.Interface:

		if cField.typ != reflect.TypeOf(time.Time{}) {
			panic("Invalid field passed to fieldWithNameAndValue")
		}
	}

	if len(cField.tags) == 0 {

		if isSingleField {
			cField.tags, isCached = fieldsCache.Get(tag)
		}

		if !isCached {

			for _, t := range strings.Split(tag, tagSeparator) {

				if t == diveTag {

					cField.dive = true
					cField.diveTag = strings.TrimLeft(strings.SplitN(tag, diveTag, 2)[1], ",")
					break
				}

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

				// if (idxField.Kind() == reflect.Ptr || idxField.Kind() == reflect.Interface) && idxField.IsNil() {
				// if val[0] == existsTag {
				// 	if (cField.kind == reflect.Ptr || cField.kind == reflect.Interface) && valueField.IsNil() {
				// 		fieldErr = &FieldError{
				// 			Field: name,
				// 			Tag:   val[0],
				// 			Value: f,
				// 			Param: val[1],
				// 		}
				// 		err = errors.New(fieldErr.Tag)
				// 	}

				// } else {

				fieldErr, err = v.fieldWithNameAndSingleTag(val, current, f, val[0], val[1], name)
				// }

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

		if cTag.keyVals[0][0] == existsTag {
			if (cField.kind == reflect.Ptr || cField.kind == reflect.Interface) && valueField.IsNil() {
				return &FieldError{
					Field: name,
					Tag:   cTag.keyVals[0][0],
					Value: f,
					Param: cTag.keyVals[0][1],
				}
			}
			continue
		}

		if fieldErr, err = v.fieldWithNameAndSingleTag(val, current, f, cTag.keyVals[0][0], cTag.keyVals[0][1], name); err != nil {

			fieldErr.Kind = cField.kind
			fieldErr.Type = cField.typ

			return fieldErr
		}
	}

	if cField.dive {

		if cField.isSliceOrArray {

			if errs := v.traverseSliceOrArray(val, current, valueField, cField); errs != nil && len(errs) > 0 {

				return &FieldError{
					Field:            cField.name,
					Kind:             cField.kind,
					Type:             cField.typ,
					Value:            f,
					IsPlaceholderErr: true,
					IsSliceOrArray:   true,
					SliceOrArrayErrs: errs,
				}
			}

		} else if cField.isMap {
			if errs := v.traverseMap(val, current, valueField, cField); errs != nil && len(errs) > 0 {

				return &FieldError{
					Field:            cField.name,
					Kind:             cField.kind,
					Type:             cField.typ,
					Value:            f,
					IsPlaceholderErr: true,
					IsMap:            true,
					MapErrs:          errs,
				}
			}
		} else {
			// throw error, if not a slice or map then should not have gotten here
			panic("dive error! can't dive on a non slice or map")
		}
	}

	return nil
}

func (v *Validate) traverseMap(val interface{}, current interface{}, valueField reflect.Value, cField *cachedField) map[interface{}]error {

	errs := map[interface{}]error{}

	for _, key := range valueField.MapKeys() {

		idxField := valueField.MapIndex(key)

		if cField.mapSubKind == reflect.Ptr && !idxField.IsNil() {
			idxField = idxField.Elem()
			cField.mapSubKind = idxField.Kind()
		}

		switch cField.mapSubKind {
		case reflect.Struct, reflect.Interface:

			if cField.isTimeSubtype {

				if fieldError := v.fieldWithNameAndValue(val, current, idxField.Interface(), cField.diveTag, fmt.Sprintf(mapIndexFieldName, cField.name, key.Interface()), false, nil); fieldError != nil {
					errs[key.Interface()] = fieldError
				}

				continue
			}

			if (idxField.Kind() == reflect.Ptr || idxField.Kind() == reflect.Interface) && idxField.IsNil() {

				if strings.Contains(cField.diveTag, omitempty) {
					continue
				}

				tags := strings.Split(cField.diveTag, tagSeparator)

				if len(tags) > 0 {

					var param string
					vals := strings.SplitN(tags[0], tagKeySeparator, 2)

					if len(vals) > 1 {
						param = vals[1]
					}

					errs[key.Interface()] = &FieldError{
						Field: fmt.Sprintf(mapIndexFieldName, cField.name, key.Interface()),
						Tag:   vals[0],
						Param: param,
						Value: idxField.Interface(),
						Kind:  idxField.Kind(),
						Type:  cField.mapSubtype,
					}
				}

				continue
			}

			// if we get here, the field is interface and could be a struct or a field
			// and we need to check the inner type and validate
			if idxField.Kind() == reflect.Interface {

				idxField = idxField.Elem()

				if idxField.Kind() == reflect.Ptr && !idxField.IsNil() {
					idxField = idxField.Elem()
				}

				if idxField.Kind() == reflect.Struct {
					goto VALIDATESTRUCT
				}

				// sending nil for cField as it was type interface and could be anything
				// each time and so must be calculated each time and can't be cached reliably
				if fieldError := v.fieldWithNameAndValue(val, current, idxField.Interface(), cField.diveTag, fmt.Sprintf(mapIndexFieldName, cField.name, key.Interface()), false, nil); fieldError != nil {
					errs[key.Interface()] = fieldError
				}

				continue
			}

		VALIDATESTRUCT:
			if structErrors := v.structRecursive(val, current, idxField.Interface()); structErrors != nil {
				errs[key.Interface()] = structErrors
			}

		default:
			if fieldError := v.fieldWithNameAndValue(val, current, idxField.Interface(), cField.diveTag, fmt.Sprintf(mapIndexFieldName, cField.name, key.Interface()), false, nil); fieldError != nil {
				errs[key.Interface()] = fieldError
			}
		}
	}

	return errs
}

func (v *Validate) traverseSliceOrArray(val interface{}, current interface{}, valueField reflect.Value, cField *cachedField) map[int]error {

	errs := map[int]error{}

	for i := 0; i < valueField.Len(); i++ {

		idxField := valueField.Index(i)

		if cField.sliceSubKind == reflect.Ptr && !idxField.IsNil() {
			idxField = idxField.Elem()
			cField.sliceSubKind = idxField.Kind()
		}

		switch cField.sliceSubKind {
		case reflect.Struct, reflect.Interface:

			if cField.isTimeSubtype {

				if fieldError := v.fieldWithNameAndValue(val, current, idxField.Interface(), cField.diveTag, fmt.Sprintf(arrayIndexFieldName, cField.name, i), false, nil); fieldError != nil {
					errs[i] = fieldError
				}

				continue
			}

			if (idxField.Kind() == reflect.Ptr || idxField.Kind() == reflect.Interface) && idxField.IsNil() {

				if strings.Contains(cField.diveTag, omitempty) {
					continue
				}

				tags := strings.Split(cField.diveTag, tagSeparator)

				if len(tags) > 0 {

					var param string
					vals := strings.SplitN(tags[0], tagKeySeparator, 2)

					if len(vals) > 1 {
						param = vals[1]
					}

					errs[i] = &FieldError{
						Field: fmt.Sprintf(arrayIndexFieldName, cField.name, i),
						Tag:   vals[0],
						Param: param,
						Value: idxField.Interface(),
						Kind:  idxField.Kind(),
						Type:  cField.sliceSubtype,
					}
				}

				continue
			}

			// if we get here, the field is interface and could be a struct or a field
			// and we need to check the inner type and validate
			if idxField.Kind() == reflect.Interface {

				idxField = idxField.Elem()

				if idxField.Kind() == reflect.Ptr && !idxField.IsNil() {
					idxField = idxField.Elem()
				}

				if idxField.Kind() == reflect.Struct {
					goto VALIDATESTRUCT
				}

				// sending nil for cField as it was type interface and could be anything
				// each time and so must be calculated each time and can't be cached reliably
				if fieldError := v.fieldWithNameAndValue(val, current, idxField.Interface(), cField.diveTag, fmt.Sprintf(arrayIndexFieldName, cField.name, i), false, nil); fieldError != nil {
					errs[i] = fieldError
				}

				continue
			}

		VALIDATESTRUCT:
			if structErrors := v.structRecursive(val, current, idxField.Interface()); structErrors != nil {
				errs[i] = structErrors
			}

		default:
			if fieldError := v.fieldWithNameAndValue(val, current, idxField.Interface(), cField.diveTag, fmt.Sprintf(arrayIndexFieldName, cField.name, i), false, nil); fieldError != nil {
				errs[i] = fieldError
			}
		}
	}

	return errs
}

func (v *Validate) fieldWithNameAndSingleTag(val interface{}, current interface{}, f interface{}, key string, param string, name string) (*FieldError, error) {

	// OK to continue because we checked it's existance before getting into this loop
	if key == omitempty {
		return nil, nil
	}

	// if key == existsTag {
	// 	continue
	// }

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
