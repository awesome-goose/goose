package sql

import (
	"fmt"
	"slices"
	"strings"
)

// BuildQuery constructs a dynamic SQL query from a map of query parameters.
// It supports:
// - Generic search across searchable fields using "query" parameter
// - Date range filters using "from" and "to" for created_at
// - Key-value filters for other parameters
// - Logical joiners (AND/OR) using "joiner" parameter
// - Entity scope query if defined
func BuildQuery(queries map[string]string, searchable []string, mainTable string, scopeQuery string, scopeArgs []any) (string, []any) {
	var (
		queryParts []string
		args       []any
	)

	// Keys to exclude from the query
	excludeKeys := []string{"page", "per_page", "query"}

	// Determine joiner ("and" or "or"), default to "and"
	joiner := strings.ToUpper(strings.TrimSpace(queries["joiner"]))
	if joiner != "AND" && joiner != "OR" {
		joiner = "AND"
	}

	// Handle generic query search across searchable fields
	if query, ok := queries["query"]; ok && len(searchable) > 0 {
		var searchParts []string
		var searchArgs []any
		for _, field := range searchable {
			if strings.Contains(field, ".") {
				parts := strings.Split(field, ".")
				if len(parts) == 2 {
					pluralTable := toSnakeCase(parts[0]) + "s"
					searchParts = append(searchParts, fmt.Sprintf("(%s.%s ILIKE ?)", pluralTable, parts[1]))
					searchArgs = append(searchArgs, "%"+query+"%")
				}
			} else {
				col := field
				if mainTable != "" {
					col = fmt.Sprintf("%s.%s", mainTable, field)
				}
				searchParts = append(searchParts, fmt.Sprintf("(%s ILIKE ?)", col))
				searchArgs = append(searchArgs, "%"+query+"%")
			}
		}
		if len(searchParts) > 0 {
			searchQuery := strings.Join(searchParts, " OR ")
			queryParts = append(queryParts, "("+searchQuery+")")
			args = append(args, searchArgs...)
		}
	}

	// Special handling for 'from' and 'to' for created_at
	from, hasFrom := queries["from"]
	to, hasTo := queries["to"]
	if hasFrom || hasTo {
		var createdAtCond string
		var createdAtArgs []any

		if hasFrom && hasTo {
			createdAtCond = "(created_at BETWEEN ? AND ?)"
			createdAtArgs = append(createdAtArgs, from, to)
		} else if hasFrom {
			createdAtCond = "(created_at >= ?)"
			createdAtArgs = append(createdAtArgs, from)
		} else if hasTo {
			createdAtCond = "(created_at <= ?)"
			createdAtArgs = append(createdAtArgs, to)
		}

		queryParts = append(queryParts, createdAtCond)
		args = append(args, createdAtArgs...)
	}

	// Build the query, excluding specified keys and 'from', 'to'
	// Sort keys for deterministic output
	var keys []string
	for k := range queries {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	if len(queries) > 0 {
		for _, key := range keys {
			value := queries[key]
			if key == "from" || key == "to" || key == "joiner" {
				continue
			}
			// Skip excluded keys
			if slices.Contains(excludeKeys, key) {
				continue
			}

			valLower := strings.ToLower(value)
			var condition string
			var partArgs []any

			// Prefix column with mainTable to avoid ambiguity in JOINs
			col := key
			if mainTable != "" && !strings.Contains(key, ".") {
				col = fmt.Sprintf("%s.%s", mainTable, key)
			}

			switch valLower {
			case "n":
				condition = fmt.Sprintf("(%s IS NULL)", col)
			case "y":
				condition = fmt.Sprintf("(%s IS NOT NULL)", col)
			default:
				condition = fmt.Sprintf("(%s = ?)", col)
				if key == "status" {
					condition = fmt.Sprintf("(LOWER(%s) = LOWER(?))", col)
				}
				partArgs = []any{value}
			}

			queryParts = append(queryParts, condition)
			if len(partArgs) > 0 {
				args = append(args, partArgs...)
			}
		}
	}

	query := strings.Join(queryParts, " "+joiner+" ")

	// Add scope query if defined
	if scopeQuery != "" {
		if query != "" {
			query = fmt.Sprintf("(%s) AND (%s)", query, scopeQuery)
		} else {
			query = scopeQuery
		}
		args = append(args, scopeArgs...)
	}

	return query, args
}
