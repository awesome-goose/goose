package config

import (
	"os"
	"regexp"
)

var (
	envVarRegex = regexp.MustCompile(`\{\{(\w+)\}\}`)
)

// Replaces {{ENV_VAR}} with the environment value
func renderEnv(s string) string {
	return envVarRegex.ReplaceAllStringFunc(s, func(match string) string {
		key := envVarRegex.FindStringSubmatch(match)[1]
		return os.Getenv(key) // TODO: reaplce with env.Get(key, "")
	})
}
