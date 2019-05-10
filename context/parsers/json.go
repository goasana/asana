package parsers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const JSON Provider = "json"

type jsonParser struct{}

func (jsonParser) Name() Provider {
	return JSON
}

// Parse decode http request encoded in json
func (jsonParser) Parse(req *http.Request, obj interface{}) error {
	if req == nil || req.Body == nil {
		return fmt.Errorf("invalid request")
	}

	decoder := json.NewDecoder(req.Body)
	return decoder.Decode(obj)
}

// ParseBody decode Body encoded in json
func (jsonParser) ParseBody(body []byte, obj interface{}) error {
	return json.Unmarshal(body, obj)
}

func init() {
	Register(&jsonParser{})
}
