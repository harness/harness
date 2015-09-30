package validator

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// BakedInValidators is the default map of ValidationFunc
// you can add, remove or even replace items to suite your needs,
// or even disregard and use your own map if so desired.
var BakedInValidators = map[string]Func{
	"required":     hasValue,
	"len":          hasLengthOf,
	"min":          hasMinOf,
	"max":          hasMaxOf,
	"eq":           isEq,
	"ne":           isNe,
	"lt":           isLt,
	"lte":          isLte,
	"gt":           isGt,
	"gte":          isGte,
	"eqfield":      isEqField,
	"nefield":      isNeField,
	"gtefield":     isGteField,
	"gtfield":      isGtField,
	"ltefield":     isLteField,
	"ltfield":      isLtField,
	"alpha":        isAlpha,
	"alphanum":     isAlphanum,
	"numeric":      isNumeric,
	"number":       isNumber,
	"hexadecimal":  isHexadecimal,
	"hexcolor":     isHexcolor,
	"rgb":          isRgb,
	"rgba":         isRgba,
	"hsl":          isHsl,
	"hsla":         isHsla,
	"email":        isEmail,
	"url":          isURL,
	"uri":          isURI,
	"base64":       isBase64,
	"contains":     contains,
	"containsany":  containsAny,
	"containsrune": containsRune,
	"excludes":     excludes,
	"excludesall":  excludesAll,
	"excludesrune": excludesRune,
	"isbn":         isISBN,
	"isbn10":       isISBN10,
	"isbn13":       isISBN13,
	"uuid":         isUUID,
	"uuid3":        isUUID3,
	"uuid4":        isUUID4,
	"uuid5":        isUUID5,
	"ascii":        isASCII,
	"printascii":   isPrintableASCII,
	"multibyte":    hasMultiByteCharacter,
	"datauri":      isDataURI,
	"latitude":     isLatitude,
	"longitude":    isLongitude,
	"ssn":          isSSN,
}

func isSSN(top interface{}, current interface{}, field interface{}, param string) bool {

	if len(field.(string)) != 11 {
		return false
	}

	return matchesRegex(sSNRegex, field)
}

func isLongitude(top interface{}, current interface{}, field interface{}, param string) bool {
	return matchesRegex(longitudeRegex, field)
}

func isLatitude(top interface{}, current interface{}, field interface{}, param string) bool {
	return matchesRegex(latitudeRegex, field)
}

func isDataURI(top interface{}, current interface{}, field interface{}, param string) bool {

	uri := strings.SplitN(field.(string), ",", 2)

	if len(uri) != 2 {
		return false
	}

	if !matchesRegex(dataURIRegex, uri[0]) {
		return false
	}

	return isBase64(top, current, uri[1], param)
}

func hasMultiByteCharacter(top interface{}, current interface{}, field interface{}, param string) bool {

	if len(field.(string)) == 0 {
		return true
	}

	return matchesRegex(multibyteRegex, field)
}

func isPrintableASCII(top interface{}, current interface{}, field interface{}, param string) bool {
	return matchesRegex(printableASCIIRegex, field)
}

func isASCII(top interface{}, current interface{}, field interface{}, param string) bool {
	return matchesRegex(aSCIIRegex, field)
}

func isUUID5(top interface{}, current interface{}, field interface{}, param string) bool {
	return matchesRegex(uUID5Regex, field)
}

func isUUID4(top interface{}, current interface{}, field interface{}, param string) bool {
	return matchesRegex(uUID4Regex, field)
}

func isUUID3(top interface{}, current interface{}, field interface{}, param string) bool {
	return matchesRegex(uUID3Regex, field)
}

func isUUID(top interface{}, current interface{}, field interface{}, param string) bool {
	return matchesRegex(uUIDRegex, field)
}

func isISBN(top interface{}, current interface{}, field interface{}, param string) bool {
	return isISBN10(top, current, field, param) || isISBN13(top, current, field, param)
}

func isISBN13(top interface{}, current interface{}, field interface{}, param string) bool {

	s := strings.Replace(strings.Replace(field.(string), "-", "", 4), " ", "", 4)

	if !matchesRegex(iSBN13Regex, s) {
		return false
	}

	var checksum int32
	var i int32

	factor := []int32{1, 3}

	for i = 0; i < 12; i++ {
		checksum += factor[i%2] * int32(s[i]-'0')
	}

	if (int32(s[12]-'0'))-((10-(checksum%10))%10) == 0 {
		return true
	}

	return false
}

func isISBN10(top interface{}, current interface{}, field interface{}, param string) bool {

	s := strings.Replace(strings.Replace(field.(string), "-", "", 3), " ", "", 3)

	if !matchesRegex(iSBN10Regex, s) {
		return false
	}

	var checksum int32
	var i int32

	for i = 0; i < 9; i++ {
		checksum += (i + 1) * int32(s[i]-'0')
	}

	if s[9] == 'X' {
		checksum += 10 * 10
	} else {
		checksum += 10 * int32(s[9]-'0')
	}

	if checksum%11 == 0 {
		return true
	}

	return false
}

func excludesRune(top interface{}, current interface{}, field interface{}, param string) bool {
	return !containsRune(top, current, field, param)
}

func excludesAll(top interface{}, current interface{}, field interface{}, param string) bool {
	return !containsAny(top, current, field, param)
}

func excludes(top interface{}, current interface{}, field interface{}, param string) bool {
	return !contains(top, current, field, param)
}

func containsRune(top interface{}, current interface{}, field interface{}, param string) bool {
	r, _ := utf8.DecodeRuneInString(param)

	return strings.ContainsRune(field.(string), r)
}

func containsAny(top interface{}, current interface{}, field interface{}, param string) bool {
	return strings.ContainsAny(field.(string), param)
}

func contains(top interface{}, current interface{}, field interface{}, param string) bool {
	return strings.Contains(field.(string), param)
}

func isNeField(top interface{}, current interface{}, field interface{}, param string) bool {
	return !isEqField(top, current, field, param)
}

func isNe(top interface{}, current interface{}, field interface{}, param string) bool {
	return !isEq(top, current, field, param)
}

func isEqField(top interface{}, current interface{}, field interface{}, param string) bool {

	if current == nil {
		panic("struct not passed for cross validation")
	}

	currentVal := reflect.ValueOf(current)

	if currentVal.Kind() == reflect.Ptr && !currentVal.IsNil() {
		currentVal = reflect.ValueOf(currentVal.Elem().Interface())
	}

	var currentFielVal reflect.Value

	switch currentVal.Kind() {

	case reflect.Struct:

		if currentVal.Type() == reflect.TypeOf(time.Time{}) {
			currentFielVal = currentVal
			break
		}

		f := currentVal.FieldByName(param)

		if f.Kind() == reflect.Invalid {
			panic(fmt.Sprintf("Field \"%s\" not found in struct", param))
		}

		currentFielVal = f

	default:

		currentFielVal = currentVal
	}

	if currentFielVal.Kind() == reflect.Ptr && !currentFielVal.IsNil() {

		currentFielVal = reflect.ValueOf(currentFielVal.Elem().Interface())
	}

	fv := reflect.ValueOf(field)

	switch fv.Kind() {

	case reflect.String:
		return fv.String() == currentFielVal.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:

		return fv.Int() == currentFielVal.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:

		return fv.Uint() == currentFielVal.Uint()

	case reflect.Float32, reflect.Float64:

		return fv.Float() == currentFielVal.Float()
	case reflect.Slice, reflect.Map, reflect.Array:

		return int64(fv.Len()) == int64(currentFielVal.Len())
	case reflect.Struct:

		if fv.Type() == reflect.TypeOf(time.Time{}) {

			if currentFielVal.Type() != reflect.TypeOf(time.Time{}) {
				panic("Bad Top Level field type")
			}

			t := currentFielVal.Interface().(time.Time)
			fieldTime := field.(time.Time)

			return fieldTime.Equal(t)
		}
	}

	panic(fmt.Sprintf("Bad field type %T", field))
}

func isEq(top interface{}, current interface{}, field interface{}, param string) bool {

	st := reflect.ValueOf(field)

	switch st.Kind() {

	case reflect.String:

		return st.String() == param

	case reflect.Slice, reflect.Map, reflect.Array:
		p := asInt(param)

		return int64(st.Len()) == p

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p := asInt(param)

		return st.Int() == p

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p := asUint(param)

		return st.Uint() == p

	case reflect.Float32, reflect.Float64:
		p := asFloat(param)

		return st.Float() == p
	}

	panic(fmt.Sprintf("Bad field type %T", field))
}

func isBase64(top interface{}, current interface{}, field interface{}, param string) bool {
	return matchesRegex(base64Regex, field)
}

func isURI(top interface{}, current interface{}, field interface{}, param string) bool {

	st := reflect.ValueOf(field)

	switch st.Kind() {

	case reflect.String:
		_, err := url.ParseRequestURI(field.(string))

		return err == nil
	}

	panic(fmt.Sprintf("Bad field type %T", field))
}

func isURL(top interface{}, current interface{}, field interface{}, param string) bool {
	st := reflect.ValueOf(field)

	switch st.Kind() {

	case reflect.String:
		url, err := url.ParseRequestURI(field.(string))

		if err != nil {
			return false
		}

		if len(url.Scheme) == 0 {
			return false
		}

		return err == nil
	}

	panic(fmt.Sprintf("Bad field type %T", field))
}

func isEmail(top interface{}, current interface{}, field interface{}, param string) bool {
	return matchesRegex(emailRegex, field)
}

func isHsla(top interface{}, current interface{}, field interface{}, param string) bool {
	return matchesRegex(hslaRegex, field)
}

func isHsl(top interface{}, current interface{}, field interface{}, param string) bool {
	return matchesRegex(hslRegex, field)
}

func isRgba(top interface{}, current interface{}, field interface{}, param string) bool {
	return matchesRegex(rgbaRegex, field)
}

func isRgb(top interface{}, current interface{}, field interface{}, param string) bool {
	return matchesRegex(rgbRegex, field)
}

func isHexcolor(top interface{}, current interface{}, field interface{}, param string) bool {
	return matchesRegex(hexcolorRegex, field)
}

func isHexadecimal(top interface{}, current interface{}, field interface{}, param string) bool {
	return matchesRegex(hexadecimalRegex, field)
}

func isNumber(top interface{}, current interface{}, field interface{}, param string) bool {
	return matchesRegex(numberRegex, field)
}

func isNumeric(top interface{}, current interface{}, field interface{}, param string) bool {
	return matchesRegex(numericRegex, field)
}

func isAlphanum(top interface{}, current interface{}, field interface{}, param string) bool {
	return matchesRegex(alphaNumericRegex, field)
}

func isAlpha(top interface{}, current interface{}, field interface{}, param string) bool {
	return matchesRegex(alphaRegex, field)
}

func hasValue(top interface{}, current interface{}, field interface{}, param string) bool {

	st := reflect.ValueOf(field)

	switch st.Kind() {
	case reflect.Invalid:
		return false
	case reflect.Slice, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Chan, reflect.Func:
		return !st.IsNil()
	case reflect.Array:
		return field != reflect.Zero(reflect.TypeOf(field)).Interface()
	default:
		return field != nil && field != reflect.Zero(reflect.TypeOf(field)).Interface()
	}
}

func isGteField(top interface{}, current interface{}, field interface{}, param string) bool {

	if current == nil {
		panic("struct not passed for cross validation")
	}

	currentVal := reflect.ValueOf(current)

	if currentVal.Kind() == reflect.Ptr && !currentVal.IsNil() {
		currentVal = reflect.ValueOf(currentVal.Elem().Interface())
	}

	var currentFielVal reflect.Value

	switch currentVal.Kind() {

	case reflect.Struct:

		if currentVal.Type() == reflect.TypeOf(time.Time{}) {
			currentFielVal = currentVal
			break
		}

		f := currentVal.FieldByName(param)

		if f.Kind() == reflect.Invalid {
			panic(fmt.Sprintf("Field \"%s\" not found in struct", param))
		}

		currentFielVal = f

	default:

		currentFielVal = currentVal
	}

	if currentFielVal.Kind() == reflect.Ptr && !currentFielVal.IsNil() {

		currentFielVal = reflect.ValueOf(currentFielVal.Elem().Interface())
	}

	fv := reflect.ValueOf(field)

	switch fv.Kind() {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:

		return fv.Int() >= currentFielVal.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:

		return fv.Uint() >= currentFielVal.Uint()

	case reflect.Float32, reflect.Float64:

		return fv.Float() >= currentFielVal.Float()

	case reflect.Struct:

		if fv.Type() == reflect.TypeOf(time.Time{}) {

			if currentFielVal.Type() != reflect.TypeOf(time.Time{}) {
				panic("Bad Top Level field type")
			}

			t := currentFielVal.Interface().(time.Time)
			fieldTime := field.(time.Time)

			return fieldTime.After(t) || fieldTime.Equal(t)
		}
	}

	panic(fmt.Sprintf("Bad field type %T", field))
}

func isGtField(top interface{}, current interface{}, field interface{}, param string) bool {

	if current == nil {
		panic("struct not passed for cross validation")
	}

	currentVal := reflect.ValueOf(current)

	if currentVal.Kind() == reflect.Ptr && !currentVal.IsNil() {
		currentVal = reflect.ValueOf(currentVal.Elem().Interface())
	}

	var currentFielVal reflect.Value

	switch currentVal.Kind() {

	case reflect.Struct:

		if currentVal.Type() == reflect.TypeOf(time.Time{}) {
			currentFielVal = currentVal
			break
		}

		f := currentVal.FieldByName(param)

		if f.Kind() == reflect.Invalid {
			panic(fmt.Sprintf("Field \"%s\" not found in struct", param))
		}

		currentFielVal = f

	default:

		currentFielVal = currentVal
	}

	if currentFielVal.Kind() == reflect.Ptr && !currentFielVal.IsNil() {

		currentFielVal = reflect.ValueOf(currentFielVal.Elem().Interface())
	}

	fv := reflect.ValueOf(field)

	switch fv.Kind() {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:

		return fv.Int() > currentFielVal.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:

		return fv.Uint() > currentFielVal.Uint()

	case reflect.Float32, reflect.Float64:

		return fv.Float() > currentFielVal.Float()

	case reflect.Struct:

		if fv.Type() == reflect.TypeOf(time.Time{}) {

			if currentFielVal.Type() != reflect.TypeOf(time.Time{}) {
				panic("Bad Top Level field type")
			}

			t := currentFielVal.Interface().(time.Time)
			fieldTime := field.(time.Time)

			return fieldTime.After(t)
		}
	}

	panic(fmt.Sprintf("Bad field type %T", field))
}

func isGte(top interface{}, current interface{}, field interface{}, param string) bool {

	st := reflect.ValueOf(field)

	switch st.Kind() {

	case reflect.String:
		p := asInt(param)

		return int64(utf8.RuneCountInString(st.String())) >= p

	case reflect.Slice, reflect.Map, reflect.Array:
		p := asInt(param)

		return int64(st.Len()) >= p

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p := asInt(param)

		return st.Int() >= p

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p := asUint(param)

		return st.Uint() >= p

	case reflect.Float32, reflect.Float64:
		p := asFloat(param)

		return st.Float() >= p

	case reflect.Struct:

		if st.Type() == reflect.TypeOf(time.Time{}) {

			now := time.Now().UTC()
			t := field.(time.Time)

			return t.After(now) || t.Equal(now)
		}
	}

	panic(fmt.Sprintf("Bad field type %T", field))
}

func isGt(top interface{}, current interface{}, field interface{}, param string) bool {

	st := reflect.ValueOf(field)

	switch st.Kind() {

	case reflect.String:
		p := asInt(param)

		return int64(utf8.RuneCountInString(st.String())) > p

	case reflect.Slice, reflect.Map, reflect.Array:
		p := asInt(param)

		return int64(st.Len()) > p

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p := asInt(param)

		return st.Int() > p

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p := asUint(param)

		return st.Uint() > p

	case reflect.Float32, reflect.Float64:
		p := asFloat(param)

		return st.Float() > p
	case reflect.Struct:

		if st.Type() == reflect.TypeOf(time.Time{}) {

			return field.(time.Time).After(time.Now().UTC())
		}
	}

	panic(fmt.Sprintf("Bad field type %T", field))
}

// length tests whether a variable's length is equal to a given
// value. For strings it tests the number of characters whereas
// for maps and slices it tests the number of items.
func hasLengthOf(top interface{}, current interface{}, field interface{}, param string) bool {

	st := reflect.ValueOf(field)

	switch st.Kind() {

	case reflect.String:
		p := asInt(param)

		return int64(utf8.RuneCountInString(st.String())) == p

	case reflect.Slice, reflect.Map, reflect.Array:
		p := asInt(param)

		return int64(st.Len()) == p

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p := asInt(param)

		return st.Int() == p

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p := asUint(param)

		return st.Uint() == p

	case reflect.Float32, reflect.Float64:
		p := asFloat(param)

		return st.Float() == p
	}

	panic(fmt.Sprintf("Bad field type %T", field))
}

// min tests whether a variable value is larger or equal to a given
// number. For number types, it's a simple lesser-than test; for
// strings it tests the number of characters whereas for maps
// and slices it tests the number of items.
func hasMinOf(top interface{}, current interface{}, field interface{}, param string) bool {

	return isGte(top, current, field, param)
}

func isLteField(top interface{}, current interface{}, field interface{}, param string) bool {

	if current == nil {
		panic("struct not passed for cross validation")
	}

	currentVal := reflect.ValueOf(current)

	if currentVal.Kind() == reflect.Ptr && !currentVal.IsNil() {
		currentVal = reflect.ValueOf(currentVal.Elem().Interface())
	}

	var currentFielVal reflect.Value

	switch currentVal.Kind() {

	case reflect.Struct:

		if currentVal.Type() == reflect.TypeOf(time.Time{}) {
			currentFielVal = currentVal
			break
		}

		f := currentVal.FieldByName(param)

		if f.Kind() == reflect.Invalid {
			panic(fmt.Sprintf("Field \"%s\" not found in struct", param))
		}

		currentFielVal = f

	default:

		currentFielVal = currentVal
	}

	if currentFielVal.Kind() == reflect.Ptr && !currentFielVal.IsNil() {

		currentFielVal = reflect.ValueOf(currentFielVal.Elem().Interface())
	}

	fv := reflect.ValueOf(field)

	switch fv.Kind() {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:

		return fv.Int() <= currentFielVal.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:

		return fv.Uint() <= currentFielVal.Uint()

	case reflect.Float32, reflect.Float64:

		return fv.Float() <= currentFielVal.Float()

	case reflect.Struct:

		if fv.Type() == reflect.TypeOf(time.Time{}) {

			if currentFielVal.Type() != reflect.TypeOf(time.Time{}) {
				panic("Bad Top Level field type")
			}

			t := currentFielVal.Interface().(time.Time)
			fieldTime := field.(time.Time)

			return fieldTime.Before(t) || fieldTime.Equal(t)
		}
	}

	panic(fmt.Sprintf("Bad field type %T", field))
}

func isLtField(top interface{}, current interface{}, field interface{}, param string) bool {

	if current == nil {
		panic("struct not passed for cross validation")
	}

	currentVal := reflect.ValueOf(current)

	if currentVal.Kind() == reflect.Ptr && !currentVal.IsNil() {
		currentVal = reflect.ValueOf(currentVal.Elem().Interface())
	}

	var currentFielVal reflect.Value

	switch currentVal.Kind() {

	case reflect.Struct:

		if currentVal.Type() == reflect.TypeOf(time.Time{}) {
			currentFielVal = currentVal
			break
		}

		f := currentVal.FieldByName(param)

		if f.Kind() == reflect.Invalid {
			panic(fmt.Sprintf("Field \"%s\" not found in struct", param))
		}

		currentFielVal = f

	default:

		currentFielVal = currentVal
	}

	if currentFielVal.Kind() == reflect.Ptr && !currentFielVal.IsNil() {

		currentFielVal = reflect.ValueOf(currentFielVal.Elem().Interface())
	}

	fv := reflect.ValueOf(field)

	switch fv.Kind() {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:

		return fv.Int() < currentFielVal.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:

		return fv.Uint() < currentFielVal.Uint()

	case reflect.Float32, reflect.Float64:

		return fv.Float() < currentFielVal.Float()

	case reflect.Struct:

		if fv.Type() == reflect.TypeOf(time.Time{}) {

			if currentFielVal.Type() != reflect.TypeOf(time.Time{}) {
				panic("Bad Top Level field type")
			}

			t := currentFielVal.Interface().(time.Time)
			fieldTime := field.(time.Time)

			return fieldTime.Before(t)
		}
	}

	panic(fmt.Sprintf("Bad field type %T", field))
}

func isLte(top interface{}, current interface{}, field interface{}, param string) bool {

	st := reflect.ValueOf(field)

	switch st.Kind() {

	case reflect.String:
		p := asInt(param)

		return int64(utf8.RuneCountInString(st.String())) <= p

	case reflect.Slice, reflect.Map, reflect.Array:
		p := asInt(param)

		return int64(st.Len()) <= p

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p := asInt(param)

		return st.Int() <= p

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p := asUint(param)

		return st.Uint() <= p

	case reflect.Float32, reflect.Float64:
		p := asFloat(param)

		return st.Float() <= p

	case reflect.Struct:

		if st.Type() == reflect.TypeOf(time.Time{}) {

			now := time.Now().UTC()
			t := field.(time.Time)

			return t.Before(now) || t.Equal(now)
		}
	}

	panic(fmt.Sprintf("Bad field type %T", field))
}

func isLt(top interface{}, current interface{}, field interface{}, param string) bool {

	st := reflect.ValueOf(field)

	switch st.Kind() {

	case reflect.String:
		p := asInt(param)

		return int64(utf8.RuneCountInString(st.String())) < p

	case reflect.Slice, reflect.Map, reflect.Array:
		p := asInt(param)

		return int64(st.Len()) < p

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p := asInt(param)

		return st.Int() < p

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p := asUint(param)

		return st.Uint() < p

	case reflect.Float32, reflect.Float64:
		p := asFloat(param)

		return st.Float() < p

	case reflect.Struct:

		if st.Type() == reflect.TypeOf(time.Time{}) {

			return field.(time.Time).Before(time.Now().UTC())
		}
	}

	panic(fmt.Sprintf("Bad field type %T", field))
}

// max tests whether a variable value is lesser than a given
// value. For numbers, it's a simple lesser-than test; for
// strings it tests the number of characters whereas for maps
// and slices it tests the number of items.
func hasMaxOf(top interface{}, current interface{}, field interface{}, param string) bool {

	return isLte(top, current, field, param)
}

// asInt retuns the parameter as a int64
// or panics if it can't convert
func asInt(param string) int64 {

	i, err := strconv.ParseInt(param, 0, 64)
	panicIf(err)

	return i
}

// asUint returns the parameter as a uint64
// or panics if it can't convert
func asUint(param string) uint64 {

	i, err := strconv.ParseUint(param, 0, 64)
	panicIf(err)

	return i
}

// asFloat returns the parameter as a float64
// or panics if it can't convert
func asFloat(param string) float64 {

	i, err := strconv.ParseFloat(param, 64)
	panicIf(err)

	return i
}

func panicIf(err error) {
	if err != nil {
		panic(err.Error())
	}
}
