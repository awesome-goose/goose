package cli

import (
	"os"
	"strings"

	"github.com/awesome-goose/goose/types"
)

type Request struct {
	raw []string

	name  string
	args  []string
	flags map[string]string

	paramsCache map[string]string
}

func NewRequest() *Request {
	r := &Request{}

	rawArgs := os.Args

	r.raw = rawArgs
	r.name = rawArgs[0]
	r.args = []string{}
	r.flags = map[string]string{}

	i := 1
	for i < len(rawArgs) {
		arg := rawArgs[i]

		if strings.HasPrefix(arg, "--") {
			// --flag or --flag=value
			if strings.Contains(arg, "=") {
				parts := strings.SplitN(arg[2:], "=", 2)
				r.flags[parts[0]] = parts[1]
			} else {
				flagName := arg[2:]
				nextIsValue := i+1 < len(rawArgs) && !strings.HasPrefix(rawArgs[i+1], "-")
				if nextIsValue {
					r.flags[flagName] = rawArgs[i+1]
					i++
				} else {
					r.flags[flagName] = "true"
				}
			}
		} else if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			// -f or -f value
			flagName := arg[1:]
			nextIsValue := i+1 < len(rawArgs) && !strings.HasPrefix(rawArgs[i+1], "-")
			if nextIsValue {
				r.flags[flagName] = rawArgs[i+1]
				i++
			} else {
				r.flags[flagName] = "true"
			}
		} else {
			r.args = append(r.args, arg)
		}

		i++
	}

	return r
}

func (r *Request) Headers() map[string][]string {
	return map[string][]string{}
}

func (r *Request) Method() types.Method {
	return types.GET
}

func (r *Request) Paths() []string {
	if len(r.args) == 0 {
		return []string{"/"}
	}
	return r.Args()
}

func (r *Request) Queries() map[string]string {
	return r.Flags()
}

func (r *Request) Body() ([]byte, error) {
	return nil, nil
}

func (r *Request) Name() string {
	return r.name
}

func (r *Request) Args() []string {
	return r.args
}

func (r *Request) Flags() map[string]string {
	return r.flags
}

func (r *Request) Flag(name string) (string, bool) {
	val, ok := r.flags[name]
	return val, ok
}

func (r *Request) HasFlag(name string) bool {
	_, ok := r.flags[name]
	return ok
}

func (r *Request) Params() map[string]string {
	return r.paramsCache
}

func (r *Request) PopulateParams(params map[string]string) {
	r.paramsCache = params
}
