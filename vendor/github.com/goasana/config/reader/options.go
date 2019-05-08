package reader

import (
	"github.com/goasana/config/encoder"
	"github.com/goasana/config/encoder/hcl"
	"github.com/goasana/config/encoder/hjson"
	"github.com/goasana/config/encoder/json"
	"github.com/goasana/config/encoder/proto"
	"github.com/goasana/config/encoder/toml"
	"github.com/goasana/config/encoder/xml"
	"github.com/goasana/config/encoder/yaml"
)

type Options struct {
	Encoding map[string]encoder.Encoder
}

type Option func(o *Options)

func NewOptions(opts ...Option) Options {
	options := Options{
		Encoding: map[string]encoder.Encoder{
			"json":  json.NewEncoder(),
			"yaml":  yaml.NewEncoder(),
			"hjson": hjson.NewEncoder(),
			"proto": proto.NewEncoder(),
			"toml":  toml.NewEncoder(),
			"xml":   xml.NewEncoder(),
			"hcl":   hcl.NewEncoder(),
			"yml":   yaml.NewEncoder(),
		},
	}
	for _, o := range opts {
		o(&options)
	}
	return options
}

func WithEncoder(e encoder.Encoder) Option {
	return func(o *Options) {
		if o.Encoding == nil {
			o.Encoding = make(map[string]encoder.Encoder)
		}
		o.Encoding[e.String()] = e
	}
}
