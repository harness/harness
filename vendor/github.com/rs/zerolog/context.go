package zerolog

import (
	"io/ioutil"
	"time"

	"github.com/rs/zerolog/internal/json"
)

// Context configures a new sub-logger with contextual fields.
type Context struct {
	l Logger
}

// Logger returns the logger with the context previously set.
func (c Context) Logger() Logger {
	return c.l
}

// Fields is a helper function to use a map to set fields using type assertion.
func (c Context) Fields(fields map[string]interface{}) Context {
	c.l.context = appendFields(c.l.context, fields)
	return c
}

// Dict adds the field key with the dict to the logger context.
func (c Context) Dict(key string, dict *Event) Context {
	dict.buf = append(dict.buf, '}')
	c.l.context = append(json.AppendKey(c.l.context, key), dict.buf...)
	eventPool.Put(dict)
	return c
}

// Array adds the field key with an array to the event context.
// Use zerolog.Arr() to create the array or pass a type that
// implement the LogArrayMarshaler interface.
func (c Context) Array(key string, arr LogArrayMarshaler) Context {
	c.l.context = json.AppendKey(c.l.context, key)
	if arr, ok := arr.(*Array); ok {
		c.l.context = arr.write(c.l.context)
		return c
	}
	var a *Array
	if aa, ok := arr.(*Array); ok {
		a = aa
	} else {
		a = Arr()
		arr.MarshalZerologArray(a)
	}
	c.l.context = a.write(c.l.context)
	return c
}

// Object marshals an object that implement the LogObjectMarshaler interface.
func (c Context) Object(key string, obj LogObjectMarshaler) Context {
	e := newEvent(levelWriterAdapter{ioutil.Discard}, 0, true)
	e.Object(key, obj)
	e.buf[0] = ',' // A new event starts as an object, we want to embed it.
	c.l.context = append(c.l.context, e.buf...)
	eventPool.Put(e)
	return c
}

// Str adds the field key with val as a string to the logger context.
func (c Context) Str(key, val string) Context {
	c.l.context = json.AppendString(json.AppendKey(c.l.context, key), val)
	return c
}

// Strs adds the field key with val as a string to the logger context.
func (c Context) Strs(key string, vals []string) Context {
	c.l.context = json.AppendStrings(json.AppendKey(c.l.context, key), vals)
	return c
}

// Bytes adds the field key with val as a []byte to the logger context.
func (c Context) Bytes(key string, val []byte) Context {
	c.l.context = json.AppendBytes(json.AppendKey(c.l.context, key), val)
	return c
}

// AnErr adds the field key with err as a string to the logger context.
func (c Context) AnErr(key string, err error) Context {
	if err != nil {
		c.l.context = json.AppendError(json.AppendKey(c.l.context, key), err)
	}
	return c
}

// Errs adds the field key with errs as an array of strings to the logger context.
func (c Context) Errs(key string, errs []error) Context {
	c.l.context = json.AppendErrors(json.AppendKey(c.l.context, key), errs)
	return c
}

// Err adds the field "error" with err as a string to the logger context.
// To customize the key name, change zerolog.ErrorFieldName.
func (c Context) Err(err error) Context {
	if err != nil {
		c.l.context = json.AppendError(json.AppendKey(c.l.context, ErrorFieldName), err)
	}
	return c
}

// Bool adds the field key with val as a bool to the logger context.
func (c Context) Bool(key string, b bool) Context {
	c.l.context = json.AppendBool(json.AppendKey(c.l.context, key), b)
	return c
}

// Bools adds the field key with val as a []bool to the logger context.
func (c Context) Bools(key string, b []bool) Context {
	c.l.context = json.AppendBools(json.AppendKey(c.l.context, key), b)
	return c
}

// Int adds the field key with i as a int to the logger context.
func (c Context) Int(key string, i int) Context {
	c.l.context = json.AppendInt(json.AppendKey(c.l.context, key), i)
	return c
}

// Ints adds the field key with i as a []int to the logger context.
func (c Context) Ints(key string, i []int) Context {
	c.l.context = json.AppendInts(json.AppendKey(c.l.context, key), i)
	return c
}

// Int8 adds the field key with i as a int8 to the logger context.
func (c Context) Int8(key string, i int8) Context {
	c.l.context = json.AppendInt8(json.AppendKey(c.l.context, key), i)
	return c
}

// Ints8 adds the field key with i as a []int8 to the logger context.
func (c Context) Ints8(key string, i []int8) Context {
	c.l.context = json.AppendInts8(json.AppendKey(c.l.context, key), i)
	return c
}

// Int16 adds the field key with i as a int16 to the logger context.
func (c Context) Int16(key string, i int16) Context {
	c.l.context = json.AppendInt16(json.AppendKey(c.l.context, key), i)
	return c
}

// Ints16 adds the field key with i as a []int16 to the logger context.
func (c Context) Ints16(key string, i []int16) Context {
	c.l.context = json.AppendInts16(json.AppendKey(c.l.context, key), i)
	return c
}

// Int32 adds the field key with i as a int32 to the logger context.
func (c Context) Int32(key string, i int32) Context {
	c.l.context = json.AppendInt32(json.AppendKey(c.l.context, key), i)
	return c
}

// Ints32 adds the field key with i as a []int32 to the logger context.
func (c Context) Ints32(key string, i []int32) Context {
	c.l.context = json.AppendInts32(json.AppendKey(c.l.context, key), i)
	return c
}

// Int64 adds the field key with i as a int64 to the logger context.
func (c Context) Int64(key string, i int64) Context {
	c.l.context = json.AppendInt64(json.AppendKey(c.l.context, key), i)
	return c
}

// Ints64 adds the field key with i as a []int64 to the logger context.
func (c Context) Ints64(key string, i []int64) Context {
	c.l.context = json.AppendInts64(json.AppendKey(c.l.context, key), i)
	return c
}

// Uint adds the field key with i as a uint to the logger context.
func (c Context) Uint(key string, i uint) Context {
	c.l.context = json.AppendUint(json.AppendKey(c.l.context, key), i)
	return c
}

// Uints adds the field key with i as a []uint to the logger context.
func (c Context) Uints(key string, i []uint) Context {
	c.l.context = json.AppendUints(json.AppendKey(c.l.context, key), i)
	return c
}

// Uint8 adds the field key with i as a uint8 to the logger context.
func (c Context) Uint8(key string, i uint8) Context {
	c.l.context = json.AppendUint8(json.AppendKey(c.l.context, key), i)
	return c
}

// Uints8 adds the field key with i as a []uint8 to the logger context.
func (c Context) Uints8(key string, i []uint8) Context {
	c.l.context = json.AppendUints8(json.AppendKey(c.l.context, key), i)
	return c
}

// Uint16 adds the field key with i as a uint16 to the logger context.
func (c Context) Uint16(key string, i uint16) Context {
	c.l.context = json.AppendUint16(json.AppendKey(c.l.context, key), i)
	return c
}

// Uints16 adds the field key with i as a []uint16 to the logger context.
func (c Context) Uints16(key string, i []uint16) Context {
	c.l.context = json.AppendUints16(json.AppendKey(c.l.context, key), i)
	return c
}

// Uint32 adds the field key with i as a uint32 to the logger context.
func (c Context) Uint32(key string, i uint32) Context {
	c.l.context = json.AppendUint32(json.AppendKey(c.l.context, key), i)
	return c
}

// Uints32 adds the field key with i as a []uint32 to the logger context.
func (c Context) Uints32(key string, i []uint32) Context {
	c.l.context = json.AppendUints32(json.AppendKey(c.l.context, key), i)
	return c
}

// Uint64 adds the field key with i as a uint64 to the logger context.
func (c Context) Uint64(key string, i uint64) Context {
	c.l.context = json.AppendUint64(json.AppendKey(c.l.context, key), i)
	return c
}

// Uints64 adds the field key with i as a []uint64 to the logger context.
func (c Context) Uints64(key string, i []uint64) Context {
	c.l.context = json.AppendUints64(json.AppendKey(c.l.context, key), i)
	return c
}

// Float32 adds the field key with f as a float32 to the logger context.
func (c Context) Float32(key string, f float32) Context {
	c.l.context = json.AppendFloat32(json.AppendKey(c.l.context, key), f)
	return c
}

// Floats32 adds the field key with f as a []float32 to the logger context.
func (c Context) Floats32(key string, f []float32) Context {
	c.l.context = json.AppendFloats32(json.AppendKey(c.l.context, key), f)
	return c
}

// Float64 adds the field key with f as a float64 to the logger context.
func (c Context) Float64(key string, f float64) Context {
	c.l.context = json.AppendFloat64(json.AppendKey(c.l.context, key), f)
	return c
}

// Floats64 adds the field key with f as a []float64 to the logger context.
func (c Context) Floats64(key string, f []float64) Context {
	c.l.context = json.AppendFloats64(json.AppendKey(c.l.context, key), f)
	return c
}

// Timestamp adds the current local time as UNIX timestamp to the logger context with the "time" key.
// To customize the key name, change zerolog.TimestampFieldName.
func (c Context) Timestamp() Context {
	if len(c.l.context) > 0 {
		c.l.context[0] = 1
	} else {
		c.l.context = append(c.l.context, 1)
	}
	return c
}

// Time adds the field key with t formated as string using zerolog.TimeFieldFormat.
func (c Context) Time(key string, t time.Time) Context {
	c.l.context = json.AppendTime(json.AppendKey(c.l.context, key), t, TimeFieldFormat)
	return c
}

// Times adds the field key with t formated as string using zerolog.TimeFieldFormat.
func (c Context) Times(key string, t []time.Time) Context {
	c.l.context = json.AppendTimes(json.AppendKey(c.l.context, key), t, TimeFieldFormat)
	return c
}

// Dur adds the fields key with d divided by unit and stored as a float.
func (c Context) Dur(key string, d time.Duration) Context {
	c.l.context = json.AppendDuration(json.AppendKey(c.l.context, key), d, DurationFieldUnit, DurationFieldInteger)
	return c
}

// Durs adds the fields key with d divided by unit and stored as a float.
func (c Context) Durs(key string, d []time.Duration) Context {
	c.l.context = json.AppendDurations(json.AppendKey(c.l.context, key), d, DurationFieldUnit, DurationFieldInteger)
	return c
}

// Interface adds the field key with obj marshaled using reflection.
func (c Context) Interface(key string, i interface{}) Context {
	c.l.context = json.AppendInterface(json.AppendKey(c.l.context, key), i)
	return c
}
