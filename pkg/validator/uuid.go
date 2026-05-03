package validator

import "regexp"

var (
	// UUIDv4 regex
	uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
)

// IsUUID checks if a string is a valid UUID format.
func IsUUID(s string) bool {
	return uuidRegex.MatchString(s)
}
