package toml

import (
	"bytes"

	"github.com/BurntSushi/toml"
	"github.com/goasana/config/encoder"
)

func init() {
	e := NewEncoder()
	encoder.Register(e.String(), e)
}

type tomlEncoder struct{}

func Encode(v interface{}) ([]byte, error) {
	b := bytes.NewBuffer(nil)
	defer b.Reset()
	err := toml.NewEncoder(b).Encode(v)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (t tomlEncoder) Encode(v interface{}, hasIndent ...bool) ([]byte, error) {
	return Encode(v)
}

func Decode(d []byte, v interface{}) error {
	return toml.Unmarshal(d, v)
}

func (t tomlEncoder) Decode(d []byte, v interface{}) error {
	return Decode(d, v)
}

func (t tomlEncoder) String() string {
	return encoder.TOML
}

func NewEncoder() encoder.Encoder {
	return tomlEncoder{}
}
