package json

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	simple "github.com/bitly/go-simplejson"
	"github.com/goasana/config/reader"
	"github.com/goasana/config/source"
)

type jsonValues struct {
	ch *source.ChangeSet
	sj *simple.Json
}

type jsonValue struct {
	*simple.Json
}

func newValues(ch *source.ChangeSet) (reader.Values, error) {
	sj := simple.New()
	data, _ := reader.ReplaceEnvVars(ch.Data)
	if err := sj.UnmarshalJSON(data); err != nil {
		sj.SetPath(nil, string(ch.Data))
	}
	return &jsonValues{ch, sj}, nil
}

func (j *jsonValues) Get(path ...string) reader.Value {
	return &jsonValue{j.getPath(path...)}
}

func (j *jsonValues) getPath(branch ...string) *simple.Json {
	jin := j.sj
	for _, p := range branch {
		jin = j.get(p, jin)
	}

	return jin
}

func (j *jsonValues) get(key string, sj *simple.Json) *simple.Json  {
	m, _ := sj.Map()

	for k := range m {
		if strings.ToLower(k) == strings.ToLower(key) {
			key = k
			break
		}
	}

	return sj.Get(key)
}


func (j *jsonValues) Del(path ...string) {
	// delete the tree?
	if len(path) == 0 {
		j.sj = simple.New()
		return
	}

	if len(path) == 1 {
		j.sj.Del(path[0])
		return
	}

	vals := j.getPath(path[:len(path)-1]...)
	vals.Del(path[len(path)-1])
	j.sj.SetPath(path[:len(path)-1], vals.Interface())
	return
}

func (j *jsonValues) Set(val interface{}, path ...string) {
	j.sj.SetPath(path, val)
}

func (j *jsonValues) Bytes() []byte {
	b, _ := j.sj.MarshalJSON()
	return b
}

func (j *jsonValues) Map() map[string]interface{} {
	m, _ := j.sj.Map()
	return m
}

func (j *jsonValues) Scan(v interface{}) error {
	b, err := j.sj.MarshalJSON()
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func (j *jsonValues) String() string {
	return "json"
}

func (j *jsonValue) Bool(def ...bool) bool {
	b, err := j.Json.Bool()
	if err == nil {
		return b
	}

	vDef := false

	if len(def) > 0 {
		vDef = def[0]
	}

	str, ok := j.Interface().(string)
	if !ok {
		return vDef
	}

	b, err = strconv.ParseBool(str)
	if err != nil {
		return vDef
	}

	return b
}

func (j *jsonValue) Int(def ...int) int {
	i, err := j.Json.Int()
	if err == nil {
		return i
	}

	vDef := 0
	if len(def) > 0 {
		vDef = def[0]
	}
	str, ok := j.Interface().(string)
	if !ok {
		return vDef
	}

	i, err = strconv.Atoi(str)
	if err != nil {
		return vDef
	}

	return i
}

func (j *jsonValue) Int64(def ...int64) int64 {
	i, err := j.Json.Int64()
	if err == nil {
		return i
	}

	var vDef int64 = 0
	if len(def) > 0 {
		vDef = def[0]
	}

	str, ok := j.Interface().(string)
	if !ok {
		return vDef
	}

	i, err = strconv.ParseInt(str, 10, 0)
	if err != nil {
		return vDef
	}

	return i
}

func (j *jsonValue) Int32(def ...int32) int32 {
	var vDef int32 = 0
	if len(def) > 0 {
		vDef = def[0]
	}
	return int32(j.Int64(int64(vDef)))
}

func (j *jsonValue) Int8(def ...int8) int8 {
	var vDef int8 = 0

	if len(def) > 0 {
		vDef = def[0]
	}
	return int8(j.Int64(int64(vDef)))
}

func (j *jsonValue) String(def ...string) string {
	vDef := ""

	if len(def) > 0 {
		vDef = def[0]
	}

	return j.Json.MustString(vDef)
}

func (j *jsonValue) Float64(def ...float64) float64 {
	f, err := j.Json.Float64()
	if err == nil {
		return f
	}

	vDef := 0.0

	if len(def) > 0 {
		vDef = def[0]
	}

	str, ok := j.Interface().(string)
	if !ok {
		return vDef
	}

	f, err = strconv.ParseFloat(str, 64)
	if err != nil {
		return vDef
	}

	return f
}

func (j *jsonValue) Float32(def ...float32) float32 {
	var vDef float32 = 0.0

	if len(def) > 0 {
		vDef = def[0]
	}

	return float32(j.Float64(float64(vDef)))
}

func (j *jsonValue) Duration(def ...time.Duration) time.Duration {
	v, err := j.Json.String()

	vDef := time.Duration(0)
	if len(def) > 0 {
		vDef = def[0]
	}
	if err != nil {
		return vDef
	}

	value, err := time.ParseDuration(v)
	if err != nil {
		return vDef
	}

	return value
}

func (j *jsonValue) StringSlice(def ...[]string) []string {
	v, err := j.Json.String()
	if err == nil {
		sl := strings.Split(v, ",")
		if len(sl) > 1 {
			return sl
		}
	}

	var vDef []string
	if len(def) > 0 {
		vDef = def[0]
	}
	if err != nil {
		return vDef
	}

	return j.Json.MustStringArray(vDef)
}

func (j *jsonValue) StringMap(def ...map[string]string) map[string]string {
	m, err := j.Json.Map()

	vDef := map[string]string{}
	if len(def) > 0 {
		vDef = def[0]
	}
	if err != nil {
		return vDef
	}

	res := map[string]string{}

	for k, v := range m {
		res[k] = fmt.Sprintf("%v", v)
	}

	return res
}

func (j *jsonValue) Scan(v interface{}) error {
	b, err := j.Json.MarshalJSON()
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func (j *jsonValue) Bytes() []byte {
	b, err := j.Json.Bytes()
	if err != nil {
		// try return marshalled
		b, err = j.Json.MarshalJSON()
		if err != nil {
			return []byte{}
		}
		return b
	}
	return b
}
