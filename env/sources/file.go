package sources

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/awesome-goose/goose/types"
	"github.com/awesome-goose/goose/utils/path"
)

type fileEnvSource struct{}

func NewFileEnvSource() *fileEnvSource {
	return &fileEnvSource{}
}

// Environment variables that steer the .env file (read from the real
// environment only).
const (
	// EnvFileVar names the .env file to read instead of <app root>/.env. "-"
	// means "no file, on purpose": nothing is read and nothing is reported.
	EnvFileVar = "GOOSE_ENV_FILE"
	// EnvFileRequiredVar, set to true, makes a missing or unreadable .env
	// stop the process at start-up instead of running on defaults.
	EnvFileRequiredVar = "GOOSE_ENV_FILE_REQUIRED"
)

// Seams for tests.
var (
	stderr io.Writer = os.Stderr
	exit             = os.Exit
)

// Load reads the .env file and populates the Env store.
//
// The file is GOOSE_ENV_FILE when set, else <app root>/.env (utils/path). A
// missing file is no longer silent: a binary whose app root resolved to the
// wrong directory (built under another module's go.mod, say) would otherwise
// run on default settings, database name included, without a word. Load
// says on stderr which root it used and why. A GOOSE_ENV_FILE that is
// missing, or any missing file with GOOSE_ENV_FILE_REQUIRED=true, ends the
// process (exit status 78, EX_CONFIG). Deployments that configure through
// the real environment only set GOOSE_ENV_FILE=- to skip the file quietly.
func (v *fileEnvSource) Load(env types.Env) {
	envPath, explicit, why := locateEnvFile()
	if envPath == "" {
		return
	}
	file, err := os.Open(envPath)
	if err != nil {
		required := explicit || strings.EqualFold(os.Getenv(EnvFileRequiredVar), "true")
		if required {
			fmt.Fprintf(stderr, "goose: cannot read the env file %s (%s): %v. Set APP_ROOT or %s to the right file.\n", envPath, why, err, EnvFileVar)
			exit(78)
			return
		}
		fmt.Fprintf(stderr, "goose: warning: no .env at %s (app root from %s); running on the process environment and defaults only. Set APP_ROOT or %s to point at it, %s=- if that is intended, or %s=true to refuse to start without it.\n",
			envPath, why, EnvFileVar, EnvFileVar, EnvFileRequiredVar)
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
					setFromFile(env, pendingKey, pendingValue)
					pendingKey, pendingValue = "", ""
				}
				continue
			}
			// End multiline if not quoted and no trailing backslash
			if !strings.HasSuffix(trimmed, "\\") {
				inMultiline = false
				pendingValue = expandVars(pendingValue, env)
				setFromFile(env, pendingKey, pendingValue)
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
		setFromFile(env, key, value)
	}
}

// locateEnvFile is the file Load reads, whether it was named explicitly, and
// how it was chosen; "" means read nothing.
func locateEnvFile() (envPath string, explicit bool, why string) {
	if f, ok := os.LookupEnv(EnvFileVar); ok && f != "" {
		if f == "-" {
			return "", false, ""
		}
		return f, true, EnvFileVar
	}
	root, how, err := path.ResolveAppRoot()
	if err != nil {
		fmt.Fprintf(stderr, "goose: warning: the app root could not be resolved (%v); no .env read. Set APP_ROOT or %s.\n", err, EnvFileVar)
		if strings.EqualFold(os.Getenv(EnvFileRequiredVar), "true") {
			exit(78)
		}
		return "", false, ""
	}
	return filepath.Join(root, ".env"), false, how
}

// setFromFile stores a value read from the .env file, but only when the real
// OS environment does not already define key. A real environment variable —
// set by a deployment platform, CI secret, or a developer's shell — always
// wins over the file, even when it is explicitly set to "": os.LookupEnv
// reports that as present, so a deliberate "disable this" is never treated as
// an absence for the file to fill in.
func setFromFile(env types.Env, key, value string) {
	if _, had := os.LookupEnv(key); had {
		return
	}
	env.Set(key, value)
	os.Setenv(key, value)
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
