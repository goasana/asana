package proto

import (
	"github.com/goasana/config/encoder"
	"github.com/gogo/protobuf/proto"
)

func init() {
	e := NewEncoder()
	encoder.Register(e.String(), e)
}

type protoEncoder struct{}

func Encode(v interface{}) ([]byte, error) {
	return proto.Marshal(v.(proto.Message))
}

func (p protoEncoder) Encode(v interface{}, hasIndent ...bool) ([]byte, error) {
	return Encode(v)
}

func Decode(d []byte, v interface{}) error {
	return proto.Unmarshal(d, v.(proto.Message))
}

func (p protoEncoder) Decode(d []byte, v interface{}) error {
	return Decode(d, v)
}

func (p protoEncoder) String() string {
	return encoder.PROTO
}

func NewEncoder() encoder.Encoder {
	return protoEncoder{}
}
