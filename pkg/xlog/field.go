package xlog

import (
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type (
	// Field is a structured log field.
	Field = zapcore.Field
	// Opt is a zap option.
	Opt = zap.Option
)

var (
	String     = zap.String
	Strings    = zap.Strings
	Any        = zap.Any
	Int        = zap.Int
	Int64      = zap.Int64
	Int32      = zap.Int32
	Uint       = zap.Uint
	Float64    = zap.Float64
	Bool       = zap.Bool
	Duration   = zap.Duration
	Durationp  = zap.Durationp
	Time       = zap.Time
	ByteString = zap.ByteString
	Object     = zap.Object
	Namespace  = zap.Namespace
	Reflect    = zap.Reflect
	Skip       = zap.Skip()
)

// FieldCost returns a millisecond cost field.
func FieldCost(value time.Duration) Field {
	return String("cost", fmt.Sprintf("%.3f", float64(value.Round(time.Microsecond))/float64(time.Millisecond)))
}

// FieldErr returns an error field.
func FieldErr(err error) Field { return zap.Error(err) }

// FieldExtra returns a JSON encoded extra field.
func FieldExtra(value map[string]any) Field {
	v, err := json.Marshal(value)
	if err != nil {
		return String("extra", "{}")
	}
	return String("extra", string(v))
}
