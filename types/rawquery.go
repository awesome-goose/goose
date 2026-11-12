package types

import "net/url"

// RawQuerier is implemented by the requests of the HTTP platforms (api, web and spa): the query
// string exactly as the client sent it, before any decoding. A signature over the query (the SAML
// HTTP-Redirect binding's) is checked against these bytes, which the decoded Queries() map cannot
// give back: the sender's escapes and parameter order are lost.
type RawQuerier interface {
	RawQuery() string
}

// RawQuery is the request's query string as sent, or, on a platform without one, its Queries()
// encoded again (sorted by key).
func RawQuery(req Request) string {
	if r, ok := req.(RawQuerier); ok {
		return r.RawQuery()
	}
	v := url.Values{}
	for k, q := range req.Queries() {
		v.Set(k, q)
	}
	return v.Encode()
}
