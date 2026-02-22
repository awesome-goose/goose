package string

import "strings"

func IsValidHTTPMethod(value string) bool {
	switch strings.ToUpper(value) {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD":
		return true
	default:
		return false
	}
}

// splitPath splits a path into segments, trimming leading/trailing slashes
func SplitPath(path string) []string {
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	if len(path) > 0 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	return Split(path, '/')
}

// split is a helper to split a string by a rune
func Split(s string, sep rune) []string {
	var res []string
	start := 0
	for i, c := range s {
		if c == sep {
			if start < i {
				res = append(res, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		res = append(res, s[start:])
	}
	return res
}
