package config

import (
	"strings"

	"brale-core/internal/pkg/errclass"
)

const validationScope errclass.Scope = "config"
const validationReason = "invalid_config"

func validationErrorf(format string, args ...any) error {
	return errclass.ValidationErrorf(validationScope, validationReason, format, args...)
}

func validateCanonicalSymbol(field, raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", validationErrorf("%s is required", field)
	}
	if trimmed != NormalizeSymbol(trimmed) {
		return "", validationErrorf("%s must be canonical symbol (e.g. BTCUSDT)", field)
	}
	return trimmed, nil
}
