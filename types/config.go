package types

import "github.com/awesome-goose/goose/utils/props"

type Config interface {
	Dir() string
	Tree() props.Props
	Get(path string) string
	Set(path string, value any)
	Export(namespace string, config any) error
	Import(namespace string, in any) error

	GetWithDefault(key, defaultValue string) string
	GetInt(key string) int
	GetBool(key string) bool
	GetFloat(key string) float64
}
