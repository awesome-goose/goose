package types

import "github.com/awesome-goose/goose/utils/props"

type Config interface {
	Dir() string
	Tree() props.Props
	Get(path string) (any, error)
	Set(path string, value any) error
	Export(namespace string, config any) error
	Import(namespace string, in any) error
}
