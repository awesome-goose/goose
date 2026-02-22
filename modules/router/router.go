package router

import "github.com/awesome-goose/goose/types"

type staticRouter struct {
	routes types.Routes
}

func (s *staticRouter) Routes() (types.Routes, error) {
	return s.routes, nil
}
