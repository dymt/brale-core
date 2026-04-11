package llmapp

import "encoding/json"

func normalizeMechanicsInput(raw []byte) []byte {
	if len(raw) == 0 {
		return raw
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return raw
	}
	stripMechanicsLegacyFields(payload)
	normalized, err := json.Marshal(payload)
	if err != nil {
		return raw
	}
	return normalized
}

func stripMechanicsLegacyFields(payload map[string]any) {
	for key, value := range payload {
		switch key {
		case "timestamp", "fear_greed_next_update_sec", "fear_greed_history", "sentiment_by_interval", "bins":
			delete(payload, key)
			continue
		}
		switch typed := value.(type) {
		case map[string]any:
			stripMechanicsLegacyFields(typed)
			if len(typed) == 0 {
				delete(payload, key)
			}
		case []any:
			normalizeMechanicsSlice(typed)
		}
	}
}

func normalizeMechanicsSlice(items []any) {
	for _, item := range items {
		switch typed := item.(type) {
		case map[string]any:
			stripMechanicsLegacyFields(typed)
		case []any:
			normalizeMechanicsSlice(typed)
		}
	}
}
