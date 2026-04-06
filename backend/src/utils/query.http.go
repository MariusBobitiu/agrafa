package utils

import (
	"errors"
	"net/http"
	"strconv"
)

func ParseOptionalPositiveInt64Query(r *http.Request, key string) (*int64, error) {
	rawValue := r.URL.Query().Get(key)
	if rawValue == "" {
		return nil, nil
	}

	parsed, err := strconv.ParseInt(rawValue, 10, 64)
	if err != nil || parsed <= 0 {
		return nil, errors.New(key + " must be a positive integer")
	}

	return &parsed, nil
}
