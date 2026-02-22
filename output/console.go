package output

import (
	"bytes"
	"fmt"
	"strings"
)

// ANSI color codes for terminal output
const (
	ColorReset   = "\033[0m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorWhite   = "\033[37m"
	ColorGray    = "\033[90m"

	// Bright variants
	ColorBrightRed     = "\033[91m"
	ColorBrightGreen   = "\033[92m"
	ColorBrightYellow  = "\033[93m"
	ColorBrightBlue    = "\033[94m"
	ColorBrightMagenta = "\033[95m"
	ColorBrightCyan    = "\033[96m"
	ColorBrightWhite   = "\033[97m"

	// Background colors
	BgRed     = "\033[41m"
	BgGreen   = "\033[42m"
	BgYellow  = "\033[43m"
	BgBlue    = "\033[44m"
	BgMagenta = "\033[45m"
	BgCyan    = "\033[46m"
	BgWhite   = "\033[47m"

	// Text styles
	StyleBold      = "\033[1m"
	StyleDim       = "\033[2m"
	StyleItalic    = "\033[3m"
	StyleUnderline = "\033[4m"
	StyleBlink     = "\033[5m"
	StyleReverse   = "\033[7m"
	StyleHidden    = "\033[8m"
	StyleStrike    = "\033[9m"
)

// ConsoleOutput represents CLI output with formatting support
type ConsoleOutput struct {
	content     []byte
	code        int
	headers     map[string]string
	contentType string
}

// ConsoleOutputOption configures a ConsoleOutput
type ConsoleOutputOption func(*ConsoleOutput)

// Console creates a basic console output
func Console(content string) *ConsoleOutput {
	return &ConsoleOutput{
		content:     []byte(content),
		code:        0,
		headers:     make(map[string]string),
		contentType: "text/plain",
	}
}

// ConsoleWithCode creates console output with exit code
func ConsoleWithCode(content string, code int) *ConsoleOutput {
	return &ConsoleOutput{
		content:     []byte(content),
		code:        code,
		headers:     make(map[string]string),
		contentType: "text/plain",
	}
}

// Info creates an info message (cyan)
func Info(message string) *ConsoleOutput {
	return Console(fmt.Sprintf("%s%s INFO %s %s\n", ColorCyan, StyleBold, ColorReset, message))
}

// Notice creates a notice message (blue)
func Notice(message string) *ConsoleOutput {
	return Console(fmt.Sprintf("%s%s NOTICE %s %s\n", ColorBlue, StyleBold, ColorReset, message))
}

// Warning creates a warning message (yellow)
func Warning(message string) *ConsoleOutput {
	return Console(fmt.Sprintf("%s%s WARNING %s %s\n", ColorYellow, StyleBold, ColorReset, message))
}

// ConsoleError creates an error message (red)
func ConsoleError(message string) *ConsoleOutput {
	return ConsoleWithCode(fmt.Sprintf("%s%s ERROR %s %s\n", ColorRed, StyleBold, ColorReset, message), 1)
}

// ConsoleSuccess creates a success message (green)
func ConsoleSuccess(message string) *ConsoleOutput {
	return Console(fmt.Sprintf("%s%s SUCCESS %s %s\n", ColorGreen, StyleBold, ColorReset, message))
}

// Line creates a simple line output
func Line(message string) *ConsoleOutput {
	return Console(message + "\n")
}

// NewLine creates an empty line
func NewLine() *ConsoleOutput {
	return Console("\n")
}

// Comment creates a comment-styled output (gray)
func Comment(message string) *ConsoleOutput {
	return Console(fmt.Sprintf("%s// %s%s\n", ColorGray, message, ColorReset))
}

// Question creates a question-styled output
func Question(message string) *ConsoleOutput {
	return Console(fmt.Sprintf("%s%s ? %s %s", ColorBrightCyan, StyleBold, ColorReset, message))
}

// TableHeader represents a table column header
type TableHeader struct {
	Label string
	Width int
	Align string // "left", "center", "right"
}

// Table creates a formatted table output
func Table(headers []string, rows [][]string) *ConsoleOutput {
	var buf bytes.Buffer

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Add padding
	for i := range widths {
		widths[i] += 2
	}

	// Build separator
	separator := "+"
	for _, w := range widths {
		separator += strings.Repeat("-", w) + "+"
	}

	// Print table
	buf.WriteString(separator + "\n")

	// Headers
	buf.WriteString("|")
	for i, h := range headers {
		buf.WriteString(fmt.Sprintf("%s%s%s%s|",
			StyleBold, ColorBrightWhite,
			padString(h, widths[i], "center"),
			ColorReset))
	}
	buf.WriteString("\n")
	buf.WriteString(separator + "\n")

	// Rows
	for _, row := range rows {
		buf.WriteString("|")
		for i, cell := range row {
			if i < len(widths) {
				buf.WriteString(padString(cell, widths[i], "left") + "|")
			}
		}
		buf.WriteString("\n")
	}
	buf.WriteString(separator + "\n")

	return Console(buf.String())
}

// List creates a bulleted list output
func List(items []string, style ...string) *ConsoleOutput {
	var buf bytes.Buffer
	bullet := "•"
	if len(style) > 0 {
		bullet = style[0]
	}

	for _, item := range items {
		buf.WriteString(fmt.Sprintf("  %s%s%s %s\n", ColorCyan, bullet, ColorReset, item))
	}

	return Console(buf.String())
}

// NumberedList creates a numbered list output
func NumberedList(items []string) *ConsoleOutput {
	var buf bytes.Buffer

	for i, item := range items {
		buf.WriteString(fmt.Sprintf("  %s%d.%s %s\n", ColorCyan, i+1, ColorReset, item))
	}

	return Console(buf.String())
}

// Tree creates a tree-like structure output
func Tree(root string, children []string) *ConsoleOutput {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("%s%s%s\n", ColorBrightWhite, root, ColorReset))

	for i, child := range children {
		if i == len(children)-1 {
			buf.WriteString(fmt.Sprintf("└── %s\n", child))
		} else {
			buf.WriteString(fmt.Sprintf("├── %s\n", child))
		}
	}

	return Console(buf.String())
}

// ProgressBar creates a progress bar visualization
func ProgressBar(current, total int, width int) *ConsoleOutput {
	if width <= 0 {
		width = 50
	}
	if total <= 0 {
		total = 1
	}

	percentage := float64(current) / float64(total)
	filled := int(percentage * float64(width))
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	pct := int(percentage * 100)

	return Console(fmt.Sprintf("\r%s[%s%s%s]%s %3d%% (%d/%d)",
		ColorGray, ColorGreen, bar, ColorGray, ColorReset,
		pct, current, total))
}

// Spinner characters for animated spinner
var SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner creates a spinner frame output (call repeatedly with incrementing frame)
func Spinner(message string, frame int) *ConsoleOutput {
	f := SpinnerFrames[frame%len(SpinnerFrames)]
	return Console(fmt.Sprintf("\r%s%s%s %s", ColorCyan, f, ColorReset, message))
}

// KeyValue creates a key-value pair output
func KeyValue(key, value string) *ConsoleOutput {
	return Console(fmt.Sprintf("  %s%s:%s %s\n", ColorGray, key, ColorReset, value))
}

// KeyValueList creates multiple key-value pairs
func KeyValueList(pairs map[string]string) *ConsoleOutput {
	var buf bytes.Buffer

	// Find max key length for alignment
	maxLen := 0
	for k := range pairs {
		if len(k) > maxLen {
			maxLen = len(k)
		}
	}

	for k, v := range pairs {
		buf.WriteString(fmt.Sprintf("  %s%s%s %s\n",
			ColorGray, padString(k+":", maxLen+1, "right"), ColorReset, v))
	}

	return Console(buf.String())
}

// Box creates a boxed text output
func Box(title string, content []string) *ConsoleOutput {
	var buf bytes.Buffer

	// Find max width
	maxWidth := len(title)
	for _, line := range content {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}
	maxWidth += 4

	// Top border
	buf.WriteString(fmt.Sprintf("╔%s╗\n", strings.Repeat("═", maxWidth)))

	// Title
	if title != "" {
		buf.WriteString(fmt.Sprintf("║ %s%s%s%s ║\n",
			StyleBold, ColorBrightWhite,
			padString(title, maxWidth-2, "center"),
			ColorReset))
		buf.WriteString(fmt.Sprintf("╠%s╣\n", strings.Repeat("═", maxWidth)))
	}

	// Content
	for _, line := range content {
		buf.WriteString(fmt.Sprintf("║ %s ║\n", padString(line, maxWidth-2, "left")))
	}

	// Bottom border
	buf.WriteString(fmt.Sprintf("╚%s╝\n", strings.Repeat("═", maxWidth)))

	return Console(buf.String())
}

// TwoColumns creates a two-column layout
func TwoColumns(left, right []string, leftWidth int) *ConsoleOutput {
	var buf bytes.Buffer

	maxRows := len(left)
	if len(right) > maxRows {
		maxRows = len(right)
	}

	for i := 0; i < maxRows; i++ {
		l := ""
		r := ""
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		buf.WriteString(padString(l, leftWidth, "left") + "  " + r + "\n")
	}

	return Console(buf.String())
}

// Divider creates a horizontal divider
func Divider(char string, width int) *ConsoleOutput {
	if char == "" {
		char = "─"
	}
	if width <= 0 {
		width = 60
	}
	return Console(fmt.Sprintf("%s%s%s\n", ColorGray, strings.Repeat(char, width), ColorReset))
}

// Alert creates an alert-style box
func Alert(alertType string, message string) *ConsoleOutput {
	var color, icon string

	switch alertType {
	case "error", "danger":
		color = ColorRed
		icon = "✖"
	case "warning", "warn":
		color = ColorYellow
		icon = "⚠"
	case "success", "ok":
		color = ColorGreen
		icon = "✔"
	case "info":
		color = ColorCyan
		icon = "ℹ"
	default:
		color = ColorWhite
		icon = "●"
	}

	width := len(message) + 6
	border := strings.Repeat("─", width)

	return Console(fmt.Sprintf(
		"%s┌%s┐%s\n%s│ %s %s │%s\n%s└%s┘%s\n",
		color, border, ColorReset,
		color, icon, padString(message, width-4, "left"), ColorReset,
		color, border, ColorReset,
	))
}

// Data returns the console output content
func (c *ConsoleOutput) Data() any {
	return c.content
}

// Code returns the exit code
func (c *ConsoleOutput) Code() int {
	return c.code
}

// Headers returns empty headers (not used for console)
func (c *ConsoleOutput) Headers() map[string]string {
	return c.headers
}

// ContentType returns the content type
func (c *ConsoleOutput) ContentType() string {
	return c.contentType
}

// Append adds more content to the output
func (c *ConsoleOutput) Append(other *ConsoleOutput) *ConsoleOutput {
	if otherContent, ok := other.Data().([]byte); ok {
		c.content = append(c.content, otherContent...)
	}
	return c
}

// WithExitCode sets the exit code
func (c *ConsoleOutput) WithExitCode(code int) *ConsoleOutput {
	c.code = code
	return c
}

// Helper function to pad strings
func padString(s string, width int, align string) string {
	if len(s) >= width {
		return s[:width]
	}

	padding := width - len(s)
	switch align {
	case "right":
		return strings.Repeat(" ", padding) + s
	case "center":
		left := padding / 2
		right := padding - left
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
	default: // left
		return s + strings.Repeat(" ", padding)
	}
}

// Colorize applies color to text
func Colorize(text string, color string) string {
	return color + text + ColorReset
}

// Bold makes text bold
func Bold(text string) string {
	return StyleBold + text + ColorReset
}

// Dim makes text dim
func Dim(text string) string {
	return StyleDim + text + ColorReset
}

// Underline underlines text
func Underline(text string) string {
	return StyleUnderline + text + ColorReset
}
