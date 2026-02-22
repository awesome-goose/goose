package web

import (
	"io"
	"net/http"

	"github.com/awesome-goose/goose/types"
	str "github.com/awesome-goose/goose/utils/string"
)

type Request struct {
	raw *http.Request

	bodyCache    []byte
	bodyErr      error
	pathsCache   []string
	queriesCache map[string]string
	paramsCache  map[string]string
}

func NewRequest(raw *http.Request) *Request {
	return &Request{raw: raw}
}

func (r *Request) Headers() map[string][]string {
	return r.raw.Header
}

func (r *Request) Method() types.Method {
	return types.Method(r.raw.Method)
}

func (r *Request) Paths() []string {
	if r.pathsCache != nil {
		return r.pathsCache
	}

	path := r.raw.URL.Path
	if path == "" || path == "/" {
		r.pathsCache = []string{}
		return r.pathsCache
	}
	segments := []string{}
	for _, seg := range str.SplitPath(path) {
		if seg != "" {
			segments = append(segments, seg)
		}
	}
	r.pathsCache = segments
	return r.pathsCache
}

func (r *Request) Queries() map[string]string {
	if r.queriesCache != nil {
		return r.queriesCache
	}

	queries := map[string]string{}
	q := r.raw.URL.Query()
	for key, vals := range q {
		if len(vals) > 0 {
			queries[key] = vals[0]
		}
	}
	r.queriesCache = queries
	return r.queriesCache
}

func (r *Request) Body() ([]byte, error) {
	if r.bodyCache != nil || r.bodyErr != nil {
		return r.bodyCache, r.bodyErr
	}
	body, err := io.ReadAll(r.raw.Body)
	r.bodyCache = body
	r.bodyErr = err
	return body, err
}

func (r *Request) Params() map[string]string {
	return r.paramsCache
}

func (r *Request) PopulateParams(params map[string]string) {
	r.paramsCache = params
}
