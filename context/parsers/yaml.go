package parsers

import (
	"fmt"
	"net/http"

	"gopkg.in/yaml.v2"
)

// YAML provider
const YAML Provider = "yaml"

type yamlParser struct{}

func (yamlParser) Name() Provider {
	return YAML
}

// Parse decode http request encoded in yaml
func (yamlParser) Parse(req *http.Request, obj interface{}) error {
	if req == nil || req.Body == nil {
		return fmt.Errorf("invalid request")
	}

	decoder := yaml.NewDecoder(req.Body)
	return decoder.Decode(obj)
}

// ParseBody decode Body encoded in yaml
func (yamlParser) ParseBody(body []byte, obj interface{}) error {
	return yaml.Unmarshal(body, obj)
}

func init() {
	Register(&yamlParser{})
}
