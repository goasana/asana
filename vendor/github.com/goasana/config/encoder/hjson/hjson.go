package hjson

import (
	"github.com/goasana/config/encoder"
	"github.com/hjson/hjson-go"
)

func init() {
	e := NewEncoder()
	encoder.Register(e.String(), e)
}

type hJsonEncoder struct{}

func Encode(v interface{}, hasIndent bool) ([]byte, error) {
	if hasIndent {
		return hjson.MarshalWithOptions(v, hjson.DefaultOptions())
	}
	return hjson.Marshal(v)
}

func (j hJsonEncoder) Encode(v interface{}, hasIndent ...bool) ([]byte, error) {
	return Encode(v, len(hasIndent) > 0 && hasIndent[0])
}

func Decode(d []byte, v interface{}) error {
	return hjson.Unmarshal(d, v)
}

func (j hJsonEncoder) Decode(d []byte, v interface{}) error {
	return Decode(d, v)
}

func (j hJsonEncoder) String() string {
	return encoder.HJSON
}

func NewEncoder() encoder.Encoder {
	return hJsonEncoder{}
}
