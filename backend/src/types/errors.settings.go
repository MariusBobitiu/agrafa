package types

import "errors"

var (
	ErrInvalidInstanceSettingKey   = errors.New("invalid instance setting key")
	ErrInvalidInstanceSettingValue = errors.New("invalid instance setting value")
)
