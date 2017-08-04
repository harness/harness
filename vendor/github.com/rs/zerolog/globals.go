package zerolog

import "time"
import "sync/atomic"

var (
	// TimestampFieldName is the field name used for the timestamp field.
	TimestampFieldName = "time"

	// LevelFieldName is the field name used for the level field.
	LevelFieldName = "level"

	// MessageFieldName is the field name used for the message field.
	MessageFieldName = "message"

	// ErrorFieldName is the field name used for error fields.
	ErrorFieldName = "error"

	// SampleFieldName is the name of the field used to report sampling.
	SampleFieldName = "sample"

	// TimeFieldFormat defines the time format of the Time field type.
	// If set to an empty string, the time is formatted as an UNIX timestamp
	// as integer.
	TimeFieldFormat = time.RFC3339

	// TimestampFunc defines the function called to generate a timestamp.
	TimestampFunc = time.Now

	// DurationFieldUnit defines the unit for time.Duration type fields added
	// using the Dur method.
	DurationFieldUnit = time.Millisecond

	// DurationFieldInteger renders Dur fields as integer instead of float if
	// set to true.
	DurationFieldInteger = false
)

var (
	gLevel          = new(uint32)
	disableSampling = new(uint32)
)

// SetGlobalLevel sets the global override for log level. If this
// values is raised, all Loggers will use at least this value.
//
// To globally disable logs, set GlobalLevel to Disabled.
func SetGlobalLevel(l Level) {
	atomic.StoreUint32(gLevel, uint32(l))
}

func globalLevel() Level {
	return Level(atomic.LoadUint32(gLevel))
}

// DisableSampling will disable sampling in all Loggers if true.
func DisableSampling(v bool) {
	var i uint32
	if v {
		i = 1
	}
	atomic.StoreUint32(disableSampling, i)
}

func samplingDisabled() bool {
	return atomic.LoadUint32(disableSampling) == 1
}
