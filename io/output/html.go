package output

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/awesome-goose/goose/errors"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var titleCaser = cases.Title(language.English)

func titleCase(s string) string { return titleCaser.String(s) }

// HTMLOutput represents an HTML response rendered from templates.
// Supports layouts, partials, and data binding.
type HTMLOutput struct {
	template    string            // Main template name or path
	layout      string            // Optional layout template
	data        any               // Data to pass to the template
	code        int               // HTTP status code
	headers     map[string]string // Custom headers
	contentType string            // Content type
	funcs       template.FuncMap  // Custom template functions
	baseDir     string            // Base directory for templates
	fallbackDir string            // Fallback directory if template not found in baseDir
	partials    []string          // Additional partial templates to include
}

// HTMLOutputOption configures an HTMLOutput
type HTMLOutputOption func(*HTMLOutput)

// WithLayout wraps the template in a layout
func WithLayout(layout string) HTMLOutputOption {
	return func(h *HTMLOutput) {
		h.layout = layout
	}
}

// WithHTMLHeaders adds custom headers
func WithHTMLHeaders(headers map[string]string) HTMLOutputOption {
	return func(h *HTMLOutput) {
		for k, v := range headers {
			h.headers[k] = v
		}
	}
}

// WithFuncs adds custom template functions
func WithFuncs(funcs template.FuncMap) HTMLOutputOption {
	return func(h *HTMLOutput) {
		for k, v := range funcs {
			h.funcs[k] = v
		}
	}
}

// WithBaseDir sets the base directory for templates
func WithBaseDir(dir string) HTMLOutputOption {
	return func(h *HTMLOutput) {
		h.baseDir = dir
	}
}

// WithPartials includes additional partial templates
func WithPartials(partials ...string) HTMLOutputOption {
	return func(h *HTMLOutput) {
		h.partials = append(h.partials, partials...)
	}
}

// WithHTMLCode sets a custom HTTP status code
func WithHTMLCode(code int) HTMLOutputOption {
	return func(h *HTMLOutput) {
		h.code = code
	}
}

// WithFallbackDir sets a fallback directory for templates
func WithFallbackDir(dir string) HTMLOutputOption {
	return func(h *HTMLOutput) {
		h.fallbackDir = dir
	}
}

// NewHTMLOutput creates a basic HTML output (legacy constructor)
func NewHTMLOutput(
	scripts []string,
	styles []string,
	templates []string,
	data []any,
	pipes map[string]any,
	code int,
) *HTMLOutput {
	return &HTMLOutput{
		template:    "",
		layout:      "",
		data:        map[string]any{"scripts": scripts, "styles": styles, "data": data, "pipes": pipes},
		code:        code,
		headers:     make(map[string]string),
		contentType: "text/html; charset=utf-8",
		funcs:       defaultTemplateFuncs(),
		baseDir:     "templates",
		partials:    templates,
	}
}

// View creates an HTML response from a template with data
func View(templateName string, data any, opts ...HTMLOutputOption) *HTMLOutput {
	h := &HTMLOutput{
		template:    templateName,
		layout:      "base/layout.html",
		data:        data,
		code:        http.StatusOK,
		headers:     make(map[string]string),
		contentType: "text/html; charset=utf-8",
		funcs:       defaultTemplateFuncs(),
		baseDir:     "app/templates",
		fallbackDir: "app/templates",
		partials:    []string{},
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// HTML creates a raw HTML string response
func HTML(content string, opts ...HTMLOutputOption) *HTMLOutput {
	h := &HTMLOutput{
		template:    "",
		layout:      "",
		data:        content,
		code:        http.StatusOK,
		headers:     make(map[string]string),
		contentType: "text/html; charset=utf-8",
		funcs:       defaultTemplateFuncs(),
		baseDir:     "",
		partials:    []string{},
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// defaultTemplateFuncs returns built-in template helper functions
func defaultTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		// String helpers
		"upper":      strings.ToUpper,
		"lower":      strings.ToLower,
		"title":      titleCase,
		"trim":       strings.TrimSpace,
		"contains":   strings.Contains,
		"replace":    strings.ReplaceAll,
		"split":      strings.Split,
		"join":       strings.Join,
		"hasPrefix":  strings.HasPrefix,
		"hasSuffix":  strings.HasSuffix,
		"trimPrefix": strings.TrimPrefix,
		"trimSuffix": strings.TrimSuffix,

		// HTML helpers
		"safe": func(s string) template.HTML {
			return template.HTML(s)
		},
		"safeJS": func(s string) template.JS {
			return template.JS(s)
		},
		"safeCSS": func(s string) template.CSS {
			return template.CSS(s)
		},
		"safeURL": func(s string) template.URL {
			return template.URL(s)
		},
		"safeAttr": func(s string) template.HTMLAttr {
			return template.HTMLAttr(s)
		},

		// Collection helpers
		"first": func(items []any) any {
			if len(items) > 0 {
				return items[0]
			}
			return nil
		},
		"last": func(items []any) any {
			if len(items) > 0 {
				return items[len(items)-1]
			}
			return nil
		},
		"reverse": func(items []any) []any {
			result := make([]any, len(items))
			for i, v := range items {
				result[len(items)-1-i] = v
			}
			return result
		},

		// Conditional helpers
		"default": func(defaultVal, val any) any {
			if val == nil || val == "" || val == 0 || val == false {
				return defaultVal
			}
			return val
		},
		"ternary": func(condition bool, trueVal, falseVal any) any {
			if condition {
				return trueVal
			}
			return falseVal
		},

		// Type conversion
		"toString": func(v any) string {
			return fmt.Sprintf("%v", v)
		},

		// Dict helper for passing multiple values to partials
		"dict": func(pairs ...any) map[string]any {
			d := make(map[string]any, len(pairs)/2)
			for i := 0; i < len(pairs)-1; i += 2 {
				if key, ok := pairs[i].(string); ok {
					d[key] = pairs[i+1]
				}
			}
			return d
		},

		// Slice helper for creating arrays
		"slice": func(items ...any) []any {
			return items
		},

		// Attr helper for HTML attributes
		"attr": func(name string, value any) template.HTMLAttr {
			return template.HTMLAttr(fmt.Sprintf(`%s="%v"`, name, value))
		},

		// Class helper for conditional CSS classes
		"classes": func(classes ...any) string {
			var result []string
			for _, c := range classes {
				switch v := c.(type) {
				case string:
					if v != "" {
						result = append(result, v)
					}
				case map[string]bool:
					for class, include := range v {
						if include {
							result = append(result, class)
						}
					}
				}
			}
			return strings.Join(result, " ")
		},
	}
}

// resolveTemplatePath tries to find a template file in baseDir first, then fallbackDir
func (h *HTMLOutput) resolveTemplatePath(templatePath string) (string, error) {
	// Try baseDir first
	primaryPath := filepath.Join(h.baseDir, templatePath)
	if _, err := os.Stat(primaryPath); err == nil {
		return primaryPath, nil
	}

	// Try fallbackDir if different from baseDir
	if h.fallbackDir != "" && h.fallbackDir != h.baseDir {
		fallbackPath := filepath.Join(h.fallbackDir, templatePath)
		if _, err := os.Stat(fallbackPath); err == nil {
			return fallbackPath, nil
		}
	}

	return "", errors.ErrTemplateNotFound.WithMeta(templatePath)
}

// loadPartialsFromDir loads all .html files from a partials directory
func (h *HTMLOutput) loadPartialsFromDir(tmpl *template.Template, partialsDir string) error {
	primaryPath := filepath.Join(h.baseDir, partialsDir)
	fallbackPath := filepath.Join(h.fallbackDir, partialsDir)

	loadedPartials := make(map[string]bool)

	// Load from primary directory
	if files, err := os.ReadDir(primaryPath); err == nil {
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".html") {
				content, err := os.ReadFile(filepath.Join(primaryPath, file.Name()))
				if err != nil {
					return errors.ErrReadingPartial.WithMeta(file.Name()).WithError(err)
				}
				partialName := strings.TrimSuffix(file.Name(), ".html")
				_, err = tmpl.New(partialName).Parse(string(content))
				if err != nil {
					return errors.ErrParsingPartial.WithMeta(file.Name()).WithError(err)
				}
				loadedPartials[file.Name()] = true
			}
		}
	}

	// Load from fallback directory (only files not already loaded)
	if h.fallbackDir != "" && h.fallbackDir != h.baseDir {
		if files, err := os.ReadDir(fallbackPath); err == nil {
			for _, file := range files {
				if !file.IsDir() && strings.HasSuffix(file.Name(), ".html") && !loadedPartials[file.Name()] {
					content, err := os.ReadFile(filepath.Join(fallbackPath, file.Name()))
					if err != nil {
						return errors.ErrReadingPartial.WithMeta(file.Name()).WithError(err)
					}
					partialName := strings.TrimSuffix(file.Name(), ".html")
					_, err = tmpl.New(partialName).Parse(string(content))
					if err != nil {
						return errors.ErrParsingPartial.WithMeta(file.Name()).WithError(err)
					}
				}
			}
		}
	}

	return nil
}

// Data renders the template and returns the HTML bytes
func (h *HTMLOutput) Data() any {
	// If it's raw HTML string (no template to process)
	if h.template == "" && h.layout == "" {
		if str, ok := h.data.(string); ok {
			return []byte(str)
		}
	}

	// Build template with all partials and layout
	var buf bytes.Buffer
	tmpl := template.New("").Funcs(h.funcs)

	// Auto-load all partials from partials directory
	if err := h.loadPartialsFromDir(tmpl, "partials"); err != nil {
		// Partials are optional, just log the error
		_ = err
	}

	// Parse explicitly specified partials
	for _, partial := range h.partials {
		path, err := h.resolveTemplatePath(partial)
		if err != nil {
			return []byte(fmt.Sprintf("Error resolving partial %s: %v", partial, err))
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return []byte(fmt.Sprintf("Error loading partial %s: %v", partial, err))
		}
		partialName := strings.TrimSuffix(filepath.Base(partial), ".html")
		_, err = tmpl.New(partialName).Parse(string(content))
		if err != nil {
			return []byte(fmt.Sprintf("Error parsing partial %s: %v", partial, err))
		}
	}

	// Parse layout if specified
	if h.layout != "" {
		path, err := h.resolveTemplatePath(h.layout)
		if err != nil {
			return []byte(fmt.Sprintf("Error resolving layout %s: %v", h.layout, err))
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return []byte(fmt.Sprintf("Error loading layout %s: %v", h.layout, err))
		}
		_, err = tmpl.New("layout").Parse(string(content))
		if err != nil {
			return []byte(fmt.Sprintf("Error parsing layout %s: %v", h.layout, err))
		}
	}

	// Parse main template (page)
	if h.template != "" {
		path, err := h.resolveTemplatePath(h.template)
		if err != nil {
			return []byte(fmt.Sprintf("Error resolving template %s: %v", h.template, err))
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return []byte(fmt.Sprintf("Error loading template %s: %v", h.template, err))
		}
		_, err = tmpl.New("content").Parse(string(content))
		if err != nil {
			return []byte(fmt.Sprintf("Error parsing template %s: %v", h.template, err))
		}
	}

	// Execute the appropriate template
	var execTemplate string
	if h.layout != "" {
		execTemplate = "layout"
	} else if h.template != "" {
		execTemplate = "content"
	}

	if execTemplate != "" {
		err := tmpl.ExecuteTemplate(&buf, execTemplate, h.data)
		if err != nil {
			return []byte(fmt.Sprintf("Error executing template: %v", err))
		}
	}

	return buf.Bytes()
}

// Code returns the HTTP status code
func (h *HTMLOutput) Code() int {
	return h.code
}

// Headers returns custom headers
func (h *HTMLOutput) Headers() map[string]string {
	return h.headers
}

// ContentType returns the HTML content type
func (h *HTMLOutput) ContentType() string {
	return h.contentType
}

// SetHeader adds a header
func (h *HTMLOutput) SetHeader(key, value string) *HTMLOutput {
	h.headers[key] = value
	return h
}

// SetData updates the template data
func (h *HTMLOutput) SetData(data any) *HTMLOutput {
	h.data = data
	return h
}

// AddFunc adds a custom template function
func (h *HTMLOutput) AddFunc(name string, fn any) *HTMLOutput {
	h.funcs[name] = fn
	return h
}
