package helpers

import (
	"os"
	"strconv"
)

func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

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
