package sources

import (
	"os"
	"strings"

	"github.com/awesome-goose/goose/types"
)

type osEnvSource struct{}

func NewOsEnvSource() *osEnvSource {
	return &osEnvSource{}
}

func (v *osEnvSource) Load(env types.Env) {
	all := os.Environ()
	for _, kv := range all {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			env.Set(parts[0], parts[1])
		}
	}
}
