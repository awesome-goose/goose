package output

import (
	"net/http"
	"net/url"
)

// RedirectOutput represents an HTTP redirect response
// Similar to Laravel's redirect() helper
type RedirectOutput struct {
	url         string
	code        int
	headers     map[string]string
	contentType string
	flash       map[string]any // For session flash data
	withInput   bool           // Flash old input
}

// RedirectOutputOption configures a RedirectOutput
type RedirectOutputOption func(*RedirectOutput)

// WithFlash adds flash data to the redirect
func WithFlash(key string, value any) RedirectOutputOption {
	return func(r *RedirectOutput) {
		r.flash[key] = value
	}
}

// WithFlashData adds multiple flash items
func WithFlashData(data map[string]any) RedirectOutputOption {
	return func(r *RedirectOutput) {
		for k, v := range data {
			r.flash[k] = v
		}
	}
}

// WithInput flashes old input (for form re-population)
func WithInput() RedirectOutputOption {
	return func(r *RedirectOutput) {
		r.withInput = true
	}
}

// WithRedirectHeaders adds custom headers
func WithRedirectHeaders(headers map[string]string) RedirectOutputOption {
	return func(r *RedirectOutput) {
		for k, v := range headers {
			r.headers[k] = v
		}
	}
}

// Redirect creates a redirect response (302 Found by default)
func Redirect(targetURL string, opts ...RedirectOutputOption) *RedirectOutput {
	r := &RedirectOutput{
		url:         targetURL,
		code:        http.StatusFound, // 302
		headers:     make(map[string]string),
		contentType: "text/html; charset=utf-8",
		flash:       make(map[string]any),
		withInput:   false,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// RedirectPermanent creates a permanent redirect (301 Moved Permanently)
func RedirectPermanent(targetURL string, opts ...RedirectOutputOption) *RedirectOutput {
	r := Redirect(targetURL, opts...)
	r.code = http.StatusMovedPermanently // 301
	return r
}

// RedirectTemporary creates a temporary redirect (307 Temporary Redirect)
// Unlike 302, this preserves the HTTP method
func RedirectTemporary(targetURL string, opts ...RedirectOutputOption) *RedirectOutput {
	r := Redirect(targetURL, opts...)
	r.code = http.StatusTemporaryRedirect // 307
	return r
}

// RedirectPermanentPreserve creates a permanent redirect that preserves method (308)
func RedirectPermanentPreserve(targetURL string, opts ...RedirectOutputOption) *RedirectOutput {
	r := Redirect(targetURL, opts...)
	r.code = http.StatusPermanentRedirect // 308
	return r
}

// RedirectToRoute redirects to a named route (placeholder - requires router integration)
func RedirectToRoute(routeName string, params map[string]string, opts ...RedirectOutputOption) *RedirectOutput {
	// This would need router integration to resolve route names
	// For now, just use the route name as the URL
	return Redirect(routeName, opts...)
}

// Back creates a redirect back to the previous page
// Uses HTTP Referer header; fallback is "/"
func Back(fallback ...string) *RedirectOutput {
	fb := "/"
	if len(fallback) > 0 {
		fb = fallback[0]
	}
	r := &RedirectOutput{
		url:         fb, // Will be overridden by referer in handler
		code:        http.StatusFound,
		headers:     make(map[string]string),
		contentType: "text/html; charset=utf-8",
		flash:       make(map[string]any),
		withInput:   false,
	}
	r.headers["X-Redirect-Back"] = "true"
	r.headers["X-Redirect-Back-Fallback"] = fb
	return r
}

// Away redirects to an external URL with security headers
func Away(externalURL string) *RedirectOutput {
	r := Redirect(externalURL)
	r.headers["Referrer-Policy"] = "no-referrer"
	return r
}

// Secure creates a redirect that ensures HTTPS
func Secure(path string) *RedirectOutput {
	// Parse and ensure HTTPS
	u, err := url.Parse(path)
	if err != nil {
		return Redirect(path)
	}
	u.Scheme = "https"
	return Redirect(u.String())
}

// Data returns the redirect body (empty for redirects, browser follows Location header)
func (r *RedirectOutput) Data() any {
	// Return a simple HTML body for clients that don't follow redirects
	return []byte(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta http-equiv="refresh" content="0;url=` + r.url + `">
    <title>Redirecting...</title>
</head>
<body>
    <p>Redirecting to <a href="` + r.url + `">` + r.url + `</a>...</p>
</body>
</html>`)
}

// Code returns the HTTP status code
func (r *RedirectOutput) Code() int {
	return r.code
}

// Headers returns headers including Location
func (r *RedirectOutput) Headers() map[string]string {
	headers := make(map[string]string)
	for k, v := range r.headers {
		headers[k] = v
	}
	headers["Location"] = r.url
	return headers
}

// ContentType returns the content type
func (r *RedirectOutput) ContentType() string {
	return r.contentType
}

// URL returns the redirect destination
func (r *RedirectOutput) URL() string {
	return r.url
}

// IsPermanent returns true for 301/308 redirects
func (r *RedirectOutput) IsPermanent() bool {
	return r.code == http.StatusMovedPermanently || r.code == http.StatusPermanentRedirect
}

// Flash returns the flash data
func (r *RedirectOutput) Flash() map[string]any {
	return r.flash
}

// ShouldFlashInput returns whether old input should be flashed
func (r *RedirectOutput) ShouldFlashInput() bool {
	return r.withInput
}

// With adds flash data (chainable)
func (r *RedirectOutput) With(key string, value any) *RedirectOutput {
	r.flash[key] = value
	return r
}

// WithErrors adds validation errors to flash (chainable)
func (r *RedirectOutput) WithErrors(errors map[string][]string) *RedirectOutput {
	r.flash["errors"] = errors
	return r
}

// WithSuccess adds a success message to flash
func (r *RedirectOutput) WithSuccess(message string) *RedirectOutput {
	r.flash["success"] = message
	return r
}

// WithError adds an error message to flash
func (r *RedirectOutput) WithError(message string) *RedirectOutput {
	r.flash["error"] = message
	return r
}

// WithWarning adds a warning message to flash
func (r *RedirectOutput) WithWarning(message string) *RedirectOutput {
	r.flash["warning"] = message
	return r
}

// WithInfo adds an info message to flash
func (r *RedirectOutput) WithInfo(message string) *RedirectOutput {
	r.flash["info"] = message
	return r
}

// SetURL changes the redirect URL
func (r *RedirectOutput) SetURL(url string) *RedirectOutput {
	r.url = url
	return r
}

// SetCode changes the status code
func (r *RedirectOutput) SetCode(code int) *RedirectOutput {
	r.code = code
	return r
}

// WithOldInput flashes old input for form re-population
func (r *RedirectOutput) WithOldInput() *RedirectOutput {
	r.withInput = true
	return r
}
