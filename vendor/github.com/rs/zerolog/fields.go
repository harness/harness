package zerolog

import (
	"sort"
	"time"

	"github.com/rs/zerolog/internal/json"
)

func appendFields(dst []byte, fields map[string]interface{}) []byte {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		dst = json.AppendKey(dst, key)
		switch val := fields[key].(type) {
		case string:
			dst = json.AppendString(dst, val)
		case []byte:
			dst = json.AppendBytes(dst, val)
		case error:
			dst = json.AppendError(dst, val)
		case []error:
			dst = json.AppendErrors(dst, val)
		case bool:
			dst = json.AppendBool(dst, val)
		case int:
			dst = json.AppendInt(dst, val)
		case int8:
			dst = json.AppendInt8(dst, val)
		case int16:
			dst = json.AppendInt16(dst, val)
		case int32:
			dst = json.AppendInt32(dst, val)
		case int64:
			dst = json.AppendInt64(dst, val)
		case uint:
			dst = json.AppendUint(dst, val)
		case uint8:
			dst = json.AppendUint8(dst, val)
		case uint16:
			dst = json.AppendUint16(dst, val)
		case uint32:
			dst = json.AppendUint32(dst, val)
		case uint64:
			dst = json.AppendUint64(dst, val)
		case float32:
			dst = json.AppendFloat32(dst, val)
		case float64:
			dst = json.AppendFloat64(dst, val)
		case time.Time:
			dst = json.AppendTime(dst, val, TimeFieldFormat)
		case time.Duration:
			dst = json.AppendDuration(dst, val, DurationFieldUnit, DurationFieldInteger)
		case []string:
			dst = json.AppendStrings(dst, val)
		case []bool:
			dst = json.AppendBools(dst, val)
		case []int:
			dst = json.AppendInts(dst, val)
		case []int8:
			dst = json.AppendInts8(dst, val)
		case []int16:
			dst = json.AppendInts16(dst, val)
		case []int32:
			dst = json.AppendInts32(dst, val)
		case []int64:
			dst = json.AppendInts64(dst, val)
		case []uint:
			dst = json.AppendUints(dst, val)
		// case []uint8:
		// 	dst = appendUints8(dst, val)
		case []uint16:
			dst = json.AppendUints16(dst, val)
		case []uint32:
			dst = json.AppendUints32(dst, val)
		case []uint64:
			dst = json.AppendUints64(dst, val)
		case []float32:
			dst = json.AppendFloats32(dst, val)
		case []float64:
			dst = json.AppendFloats64(dst, val)
		case []time.Time:
			dst = json.AppendTimes(dst, val, TimeFieldFormat)
		case []time.Duration:
			dst = json.AppendDurations(dst, val, DurationFieldUnit, DurationFieldInteger)
		case nil:
			dst = append(dst, "null"...)
		default:
			dst = json.AppendInterface(dst, val)
		}
	}
	return dst
}
