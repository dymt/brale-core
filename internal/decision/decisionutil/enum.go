package decisionutil

import (
	"encoding/json"
	"fmt"
)

func ParseEnumJSON(data []byte, allowed map[string]struct{}, name string) (string, error) {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return "", err
	}
	if _, ok := allowed[value]; !ok {
		return "", fmt.Errorf("invalid %s: %s", name, value)
	}
	return value, nil
}

func UnmarshalEnumJSON[T ~string](data []byte, allowed map[string]struct{}, name string) (T, error) {
	value, err := ParseEnumJSON(data, allowed, name)
	if err != nil {
		return T(""), err
	}
	return T(value), nil
}

func BuildEnumSet[T ~string](values []T) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[string(value)] = struct{}{}
	}
	return set
}

func EnumStrings[T ~string](values []T) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, string(value))
	}
	return out
}
