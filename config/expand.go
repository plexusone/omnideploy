package config

import (
	"os"
	"regexp"
)

// envVarPattern matches "$$" (an escaped literal "$") or "${VAR}"/
// "${VAR:-default}" references, per the shell-style expansion documented
// in docs/configuration/schema.md.
var envVarPattern = regexp.MustCompile(`\$\$|\$\{([A-Za-z_][A-Za-z0-9_]*)(?::-([^}]*))?\}`)

// ExpandEnv performs the shell-style environment variable expansion
// documented in docs/configuration/schema.md on raw config file bytes:
// ${VAR} and ${VAR:-default} are replaced with the named environment
// variable's value (or the default when unset or empty), and "$$" is an
// escaped literal "$". Applied to the raw bytes before YAML/JSON parsing
// so it works uniformly across every field, not just ones with dedicated
// expansion logic.
func ExpandEnv(data []byte) []byte {
	return envVarPattern.ReplaceAllFunc(data, func(match []byte) []byte {
		if string(match) == "$$" {
			return []byte("$")
		}
		sub := envVarPattern.FindSubmatch(match)
		name := string(sub[1])
		def := sub[2]
		if v, ok := os.LookupEnv(name); ok && v != "" {
			return []byte(v)
		}
		return def
	})
}
