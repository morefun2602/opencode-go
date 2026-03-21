package runtime

import (
	"encoding/json"
	"strings"
)

func validateStructuredOutput(text, schemaPath string) error {
	if strings.TrimSpace(schemaPath) == "" {
		return nil
	}
	s := strings.TrimSpace(text)
	if !json.Valid([]byte(s)) {
		return errStructuredOutput
	}
	return nil
}
