// Package reader parses change sets and provides config values
package reader

import (
	"time"

	"github.com/goasana/config/source"
)

// Reader is an interface for merging changesets
type Reader interface {
	Merge(...*source.ChangeSet) (*source.ChangeSet, error)
	Values(*source.ChangeSet) (Values, error)
	String() string
}

// Values is returned by the reader
type Values interface {
	Bytes() []byte
	Get(path ...string) Value
	Map() map[string]interface{}
	Scan(v interface{}) error
}

// Value represents a value of any type
type Value interface {
	Bool(def ...bool) bool
	Int(def ...int) int
	Int8(def ...int8) int8
	Int32(def ...int32) int32
	Int64(def ...int64) int64
	String(def ...string) string
	Float64(def ...float64) float64
	Float32(def ...float32) float32
	Duration(def ...time.Duration) time.Duration
	StringSlice(def ...[]string) []string
	StringMap(def ...map[string]string) map[string]string
	Scan(val interface{}) error
	Bytes() []byte
}
