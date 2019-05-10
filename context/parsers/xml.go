package parsers

import (
	"encoding/xml"
	"fmt"
	"net/http"
)

const XML Provider = "xml"

type xmlParser struct{}

func (xmlParser) Name() Provider {
	return XML
}

// Parse decode http request encoded in xml
func (xmlParser) Parse(req *http.Request, obj interface{}) error {
	if req == nil || req.Body == nil {
		return fmt.Errorf("invalid request")
	}

	decoder := xml.NewDecoder(req.Body)
	return decoder.Decode(obj)
}

// ParseBody decode Body encoded in xml
func (xmlParser) ParseBody(body []byte, obj interface{}) error {
	return xml.Unmarshal(body, obj)
}

func init() {
	Register(&xmlParser{})
}
