package helpers

import (
	"os"
	"strconv"
	"strings"
)

// envListSeparator splits a single environment variable into a list, the way
// Laravel's config files split comma-separated values.
const envListSeparator = ","

func Env(key string, fallback any) any {
	if value, ok := os.LookupEnv(key); ok {
		switch fallback.(type) {
		case int:
			if v, err := strconv.Atoi(value); err == nil {
				return v
			}
		case bool:
			if v, err := strconv.ParseBool(value); err == nil {
				return v
			}
		case float64:
			if v, err := strconv.ParseFloat(value, 64); err == nil {
				return v
			}
		case string:
			return value
		}
	}
	return fallback
}

/*
 * EnvSlice reads a comma-separated environment variable as a list.
 *
 * Env switches on the fallback's type and has no []string case, so a slice
 * fallback is returned even when the variable is set. Rather than widen that
 * type switch — and with it the panic-capable assertion every caller has to
 * write — list-valued settings get their own concretely typed reader.
 *
 * An empty or whitespace-only value yields an empty, non-nil slice rather than
 * the fallback: "set to nothing" is a deliberate choice by whoever wrote the
 * .env, and for APP_TRUSTED_PROXIES it is the safe one.
 *
 * @param key      The environment variable to read.
 * @param fallback The value used when the variable is not set at all.
 * @return []string The parsed list, with blank entries dropped.
 */
func EnvSlice(key string, fallback []string) []string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	parts := strings.Split(value, envListSeparator)
	list := make([]string, 0, len(parts))

	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			list = append(list, trimmed)
		}
	}

	return list
}
