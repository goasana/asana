package parsers

import (
	"net/http"
)


type Provider string

type Parser interface {
	ParseBody(body []byte, obj interface{}) error
	Parse(req *http.Request, obj interface{}) error
	Name() Provider
}

var providers = make(map[Provider]Parser)

func Register(b Parser) {
	providers[b.Name()] = b
}

func GetProvider(name Provider) Parser {
	if len(providers) == 0 {
		panic("no providers found")
	}

	if p, ok := providers[name]; ok {
		return p
	}

	for _, p := range providers {
		return p
	}

	return nil
}
