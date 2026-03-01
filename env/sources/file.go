package sources

import (
	"bufio"
	"os"
	"strings"

	"github.com/awesome-goose/goose/types"
	"github.com/awesome-goose/goose/utils/path"
)

type fileEnvSource struct{}

func NewFileEnvSource() *fileEnvSource {
	return &fileEnvSource{}
}

// Load reads the .env file and populates the Env store
// Silently ignores missing .env files
func (v *fileEnvSource) Load(env types.Env) {
	directory, err := path.AppRoot()
	if err != nil {
		return
	}
	envPath := directory + "/.env"
	file, err := os.Open(envPath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var (
		pendingKey     string
		pendingValue   string
		inMultiline    bool
		multilineQuote byte
	)

	for scanner.Scan() {
		line := scanner.Text()

		// Handle multiline continuation
		if inMultiline {
			trimmed := strings.TrimRight(line, "\r\n")
			pendingValue += "\n" + trimmed
			if multilineQuote != 0 {
				// End multiline if closing quote found
				if strings.HasSuffix(trimmed, string(multilineQuote)) {
					pendingValue = pendingValue[:len(pendingValue)-1] // remove closing quote
					inMultiline = false
					pendingValue = expandVars(pendingValue, env)
					env.Set(pendingKey, pendingValue)
					os.Setenv(pendingKey, pendingValue)
					pendingKey, pendingValue = "", ""
				}
				continue
			}
			// End multiline if not quoted and no trailing backslash
			if !strings.HasSuffix(trimmed, "\\") {
				inMultiline = false
				pendingValue = expandVars(pendingValue, env)
				env.Set(pendingKey, pendingValue)
				os.Setenv(pendingKey, pendingValue)
				pendingKey, pendingValue = "", ""
			}
			continue
		}

		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Handle "export KEY=value" syntax
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimPrefix(line, "export ")
			line = strings.TrimSpace(line)
		}

		// Split on first '=' only (handles values containing '=')
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		if key == "" {
			continue // Skip empty keys
		}

		value := parts[1]

		// Multiline quoted value
		if len(value) > 0 && (value[0] == '"' || value[0] == '\'') && !strings.HasSuffix(value, string(value[0])) {
			inMultiline = true
			multilineQuote = value[0]
			pendingKey = key
			pendingValue = value[1:] // remove opening quote
			continue
		}

		// Multiline with trailing backslash
		if strings.HasSuffix(value, "\\") {
			inMultiline = true
			multilineQuote = 0
			pendingKey = key
			pendingValue = strings.TrimSuffix(value, "\\")
			continue
		}

		// Remove surrounding quotes if present
		if len(value) >= 2 {
			first, last := value[0], value[len(value)-1]
			if first == '"' && last == '"' {
				value = value[1 : len(value)-1]
				value = unescapeValue(value)
			} else if first == '\'' && last == '\'' {
				value = value[1 : len(value)-1]
			} else {
				value = strings.TrimSpace(value)
				if idx := strings.Index(value, " #"); idx != -1 {
					value = strings.TrimSpace(value[:idx])
				}
			}
		} else {
			value = strings.TrimSpace(value)
		}

		value = expandVars(value, env)
		env.Set(key, value)
		os.Setenv(key, value)
	}
}

// unescapeValue handles common escape sequences in double-quoted values
func unescapeValue(s string) string {
	replacer := strings.NewReplacer(
		`\"`, `"`,
		`\\`, `\`,
		`\n`, "\n",
		`\r`, "\r",
		`\t`, "\t",
	)
	return replacer.Replace(s)
}

// expandVars replaces ${VAR} with env.Get(VAR)
func expandVars(s string, env types.Env) string {
	varStart := strings.Index(s, "${")
	for varStart != -1 {
		varEnd := strings.Index(s[varStart:], "}")
		if varEnd == -1 {
			break
		}
		varEnd += varStart
		varName := s[varStart+2 : varEnd]
		varValue := env.Get(varName)
		s = s[:varStart] + varValue + s[varEnd+1:]
		varStart = strings.Index(s, "${")
	}
	return s
}
