package json

import (
	"encoding/json"

	"github.com/goasana/framework/encoder"
)

func init()  {
	e := NewEncoder()
	encoder.Register(e.String(), e)
}

type jsonEncoder struct{}


func Encode(v interface{}, hasIndent bool) ([]byte, error) {
	if hasIndent {
		return json.MarshalIndent(v, "", " ")
	}

	return json.Marshal(v)
}

func (j jsonEncoder) Encode(v interface{}, hasIndent ...bool) ([]byte, error) {
	return Encode(v, len(hasIndent) > 0 && hasIndent[0])
}

func Decode(d []byte, v interface{}) error {
	return json.Unmarshal(d, v)
}

func (j jsonEncoder) Decode(d []byte, v interface{}) error {
	return Decode(d, v)
}

func (j jsonEncoder) String() string {
	return "json"
}

func NewEncoder() encoder.Encoder {
	return jsonEncoder{}
}