package parsers

import (
	"io/ioutil"
	"net/http"

	"github.com/goasana/config/encoder/proto"
)

const PROTOBUF Provider  = "protobuf"

type protobufParser struct{}

func (protobufParser) Name() Provider {
	return PROTOBUF
}

// Parse decode http request encoded in protobug
func (b protobufParser) Parse(req *http.Request, obj interface{}) error {
	buf, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}
	return b.ParseBody(buf, obj)
}

// ParseBody decode Body encoded in protobuf
func (protobufParser) ParseBody(body []byte, obj interface{}) error {
	if err := proto.Decode(body, obj); err != nil {
		return err
	}
	return nil
}

func init()  {
	Register(&protobufParser{})
}