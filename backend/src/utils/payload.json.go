package utils

import "encoding/json"

func MarshalPayloadMap(payload map[string]any) (json.RawMessage, error) {
	if len(payload) == 0 {
		return nil, nil
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(encoded), nil
}
