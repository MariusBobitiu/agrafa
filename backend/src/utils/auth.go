package utils

import (
	"net/mail"
	"strings"

	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

func NormalizeEmail(value string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(value))
	if email == "" {
		return "", types.ErrInvalidEmail
	}

	if _, err := mail.ParseAddress(email); err != nil {
		return "", types.ErrInvalidEmail
	}

	return email, nil
}

func BuildDefaultProjectName(name string) string {
	return name + " Workspace"
}

func BuildDefaultProjectSlug(name, userID string) string {
	base := BuildSlug(name)
	if base == "" {
		base = "workspace"
	}

	suffix := userID
	if cut := strings.IndexByte(suffix, '_'); cut >= 0 && cut < len(suffix)-1 {
		suffix = suffix[cut+1:]
	}
	if len(suffix) > 8 {
		suffix = suffix[:8]
	}

	return base + "-" + suffix
}

func OptionalTrimmed(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}
