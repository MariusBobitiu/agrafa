package utils

import (
	"encoding/json"
	"strings"
	"unicode"
)

func NormalizeJSON(payload json.RawMessage) []byte {
	if len(payload) == 0 {
		return []byte("{}")
	}

	return payload
}

func NormalizeRequiredString(value string) string {
	return strings.TrimSpace(value)
}

func BuildSlug(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return ""
	}

	var builder strings.Builder
	lastWasSeparator := false

	for _, char := range normalized {
		switch {
		case unicode.IsLetter(char) || unicode.IsDigit(char):
			builder.WriteRune(char)
			lastWasSeparator = false
		case !lastWasSeparator && builder.Len() > 0:
			builder.WriteByte('-')
			lastWasSeparator = true
		}
	}

	return strings.Trim(builder.String(), "-")
}
