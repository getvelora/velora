// Package env provides small helpers for reading process environment variables
// with default fallbacks. It exists so internal packages share one
// implementation instead of carrying private copies.
package env

import "os"

// OrDefault returns the value of the named environment variable, or fallback
// when the variable is unset or empty.
func OrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
