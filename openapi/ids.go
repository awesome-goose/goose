package openapi

import (
	"regexp"
	"strings"
	"unicode"
)

// PathOperationID names an operation by its method and path alone, leaving out the path
// segments named in skip (a host's "api" prefix, a mount's "apps"): GET /api/apps/social/posts/{id}
// with skip "api", "apps" is getSocialPostsById, and POST /api/apps/capital/{type}/charge/{providerId}
// is postCapitalByTypeChargeByProviderid. Each word is title-cased the way Python's str.title()
// does it, so ids match the ones a host already derives in a script. Pass it as Options.OperationID.
func PathOperationID(skip ...string) func(method, path, handler string) string {
	drop := map[string]bool{}
	for _, s := range skip {
		drop[s] = true
	}
	return func(method, path, _ string) string {
		var b strings.Builder
		b.WriteString(method)
		for _, seg := range strings.Split(strings.Trim(path, "/"), "/") {
			if seg == "" || drop[seg] {
				continue
			}
			if strings.HasPrefix(seg, "{") {
				b.WriteString("By" + title(strings.Trim(seg, "{}")))
				continue
			}
			for _, w := range nonWord.Split(seg, -1) {
				if w != "" {
					b.WriteString(title(w))
				}
			}
		}
		return b.String()
	}
}

var nonWord = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// title is Python's str.title(): a letter after a letter is lower case, any other is upper case.
func title(s string) string {
	out := []rune(s)
	prevCased := false
	for i, r := range out {
		if unicode.IsLetter(r) {
			if prevCased {
				out[i] = unicode.ToLower(r)
			} else {
				out[i] = unicode.ToUpper(r)
			}
			prevCased = true
		} else {
			prevCased = false
		}
	}
	return string(out)
}

// Page is the shared shape of a cursor-paged list: one page of items and the cursor of the
// next page, "" (or absent) on the last one. Return it, or name it as an Operation's Response,
// and every list documents the same {items, next} schema, named for its item type: Page[Post]
// is the component PagePost.
type Page[T any] struct {
	Items []T    `json:"items" binding:"required"`
	Next  string `json:"next"`
}
