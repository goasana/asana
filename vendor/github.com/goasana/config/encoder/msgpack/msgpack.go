package msgpack

import (
	"github.com/goasana/config/encoder"
	"github.com/vmihailenco/msgpack"
)

func init() {
	e := NewEncoder()
	encoder.Register(e.String(), e)
}

type msgPackEncoder struct{}

func Encode(v interface{}) ([]byte, error) {
	return msgpack.Marshal(v)
}

func (j msgPackEncoder) Encode(v interface{}, hasIndent ...bool) ([]byte, error) {
	return Encode(v)
}

func Decode(d []byte, v interface{}) error {
	return msgpack.Unmarshal(d, v)
}

func (j msgPackEncoder) Decode(d []byte, v interface{}) error {
	return Decode(d, v)
}

func (j msgPackEncoder) String() string {
	return encoder.MSGPACK
}

func NewEncoder() encoder.Encoder {
	return msgPackEncoder{}
}
