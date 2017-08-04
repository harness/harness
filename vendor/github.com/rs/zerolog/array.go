package zerolog

import (
	"sync"
	"time"

	"github.com/rs/zerolog/internal/json"
)

var arrayPool = &sync.Pool{
	New: func() interface{} {
		return &Array{
			buf: make([]byte, 0, 500),
		}
	},
}

type Array struct {
	buf []byte
}

// Arr creates an array to be added to an Event or Context.
func Arr() *Array {
	a := arrayPool.Get().(*Array)
	a.buf = a.buf[:0]
	return a
}

func (*Array) MarshalZerologArray(*Array) {
}

func (a *Array) write(dst []byte) []byte {
	if len(a.buf) == 0 {
		dst = append(dst, `[]`...)
	} else {
		a.buf[0] = '['
		dst = append(append(dst, a.buf...), ']')
	}
	arrayPool.Put(a)
	return dst
}

// Object marshals an object that implement the LogObjectMarshaler
// interface and append it to the array.
func (a *Array) Object(obj LogObjectMarshaler) *Array {
	a.buf = append(a.buf, ',')
	e := Dict()
	obj.MarshalZerologObject(e)
	e.buf = append(e.buf, '}')
	a.buf = append(a.buf, e.buf...)
	return a
}

// Str append the val as a string to the array.
func (a *Array) Str(val string) *Array {
	a.buf = json.AppendString(append(a.buf, ','), val)
	return a
}

// Bytes append the val as a string to the array.
func (a *Array) Bytes(val []byte) *Array {
	a.buf = json.AppendBytes(append(a.buf, ','), val)
	return a
}

// Err append the err as a string to the array.
func (a *Array) Err(err error) *Array {
	a.buf = json.AppendError(append(a.buf, ','), err)
	return a
}

// Bool append the val as a bool to the array.
func (a *Array) Bool(b bool) *Array {
	a.buf = json.AppendBool(append(a.buf, ','), b)
	return a
}

// Int append i as a int to the array.
func (a *Array) Int(i int) *Array {
	a.buf = json.AppendInt(append(a.buf, ','), i)
	return a
}

// Int8 append i as a int8 to the array.
func (a *Array) Int8(i int8) *Array {
	a.buf = json.AppendInt8(append(a.buf, ','), i)
	return a
}

// Int16 append i as a int16 to the array.
func (a *Array) Int16(i int16) *Array {
	a.buf = json.AppendInt16(append(a.buf, ','), i)
	return a
}

// Int32 append i as a int32 to the array.
func (a *Array) Int32(i int32) *Array {
	a.buf = json.AppendInt32(append(a.buf, ','), i)
	return a
}

// Int64 append i as a int64 to the array.
func (a *Array) Int64(i int64) *Array {
	a.buf = json.AppendInt64(append(a.buf, ','), i)
	return a
}

// Uint append i as a uint to the array.
func (a *Array) Uint(i uint) *Array {
	a.buf = json.AppendUint(append(a.buf, ','), i)
	return a
}

// Uint8 append i as a uint8 to the array.
func (a *Array) Uint8(i uint8) *Array {
	a.buf = json.AppendUint8(append(a.buf, ','), i)
	return a
}

// Uint16 append i as a uint16 to the array.
func (a *Array) Uint16(i uint16) *Array {
	a.buf = json.AppendUint16(append(a.buf, ','), i)
	return a
}

// Uint32 append i as a uint32 to the array.
func (a *Array) Uint32(i uint32) *Array {
	a.buf = json.AppendUint32(append(a.buf, ','), i)
	return a
}

// Uint64 append i as a uint64 to the array.
func (a *Array) Uint64(i uint64) *Array {
	a.buf = json.AppendUint64(append(a.buf, ','), i)
	return a
}

// Float32 append f as a float32 to the array.
func (a *Array) Float32(f float32) *Array {
	a.buf = json.AppendFloat32(append(a.buf, ','), f)
	return a
}

// Float64 append f as a float64 to the array.
func (a *Array) Float64(f float64) *Array {
	a.buf = json.AppendFloat64(append(a.buf, ','), f)
	return a
}

// Time append t formated as string using zerolog.TimeFieldFormat.
func (a *Array) Time(t time.Time) *Array {
	a.buf = json.AppendTime(append(a.buf, ','), t, TimeFieldFormat)
	return a
}

// Dur append d to the array.
func (a *Array) Dur(d time.Duration) *Array {
	a.buf = json.AppendDuration(append(a.buf, ','), d, DurationFieldUnit, DurationFieldInteger)
	return a
}

// Interface append i marshaled using reflection.
func (a *Array) Interface(i interface{}) *Array {
	if obj, ok := i.(LogObjectMarshaler); ok {
		return a.Object(obj)
	}
	a.buf = json.AppendInterface(append(a.buf, ','), i)
	return a
}
