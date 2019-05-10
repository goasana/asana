package parsers

import (
	"bytes"
	"io"
	"net/http"

	"github.com/ugorji/go/codec"
)

const MSGPACK Provider = "msgpack"

type msgpackParser struct{}

func (msgpackParser) Name() Provider {
	return MSGPACK
}

// Parse decode http request encoded in msgpack
func (msgpackParser) Parse(req *http.Request, obj interface{}) error {
	return decodeMsgPack(req.Body, obj)
}

// ParseBody decode Body encoded in msgpack
func (msgpackParser) ParseBody(body []byte, obj interface{}) error {
	return decodeMsgPack(bytes.NewReader(body), obj)
}

func decodeMsgPack(r io.Reader, obj interface{}) error {
	cdc := new(codec.MsgpackHandle)
	return codec.NewDecoder(r, cdc).Decode(&obj)
}

func init() {
	Register(&msgpackParser{})
}
