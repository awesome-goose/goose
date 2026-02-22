package types

type Router interface {
	Routes() (Routes, error)
}

type RouterFinder interface {
	Find(routes Routes, method string, segments []string) (*Route, map[string]string, error)
}
