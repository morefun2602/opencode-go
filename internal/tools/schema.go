package tools

import (
	"fmt"
	"strings"
)

type ValidationError struct {
	Tool   string
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return fmt.Sprintf("tool %q validation failed: %s", e.Tool, e.Reason)
	}
	return fmt.Sprintf("tool %q validation failed for %q: %s", e.Tool, e.Field, e.Reason)
}

func validateArgs(toolName string, schema map[string]any, args map[string]any) error {
	if len(schema) == 0 {
		return nil
	}
	if args == nil {
		args = map[string]any{}
	}

	// Shorthand schema: {"path":"string","offset":"number"}
	if _, ok := schema["type"]; !ok {
		for field, t := range schema {
			ts, ok := t.(string)
			if !ok {
				continue
			}
			if v, exists := args[field]; exists && !matchesType(ts, v) {
				return &ValidationError{Tool: toolName, Field: field, Reason: fmt.Sprintf("expected %s", ts)}
			}
		}
		return nil
	}

	objType, _ := schema["type"].(string)
	if objType != "" && objType != "object" {
		return nil
	}

	required := toStringSlice(schema["required"])
	for _, field := range required {
		if _, ok := args[field]; !ok {
			return &ValidationError{Tool: toolName, Field: field, Reason: "missing required field"}
		}
	}

	props, _ := schema["properties"].(map[string]any)
	for field, def := range props {
		val, exists := args[field]
		if !exists {
			continue
		}
		if err := validateField(toolName, field, def, val); err != nil {
			return err
		}
	}
	return nil
}

func validateField(toolName, field string, def any, val any) error {
	typed, ok := def.(map[string]any)
	if !ok {
		return nil
	}
	if t, _ := typed["type"].(string); t != "" && !matchesType(t, val) {
		return &ValidationError{Tool: toolName, Field: field, Reason: fmt.Sprintf("expected %s", t)}
	}
	if enumRaw, ok := typed["enum"]; ok {
		if !isInEnum(val, enumRaw) {
			return &ValidationError{Tool: toolName, Field: field, Reason: "not in enum"}
		}
	}
	if t, _ := typed["type"].(string); t == "array" {
		if items, ok := typed["items"]; ok {
			if arr, ok := val.([]any); ok {
				for i, item := range arr {
					if err := validateArrayItem(toolName, field, i, items, item); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func validateArrayItem(toolName, field string, idx int, def any, val any) error {
	typed, ok := def.(map[string]any)
	if !ok {
		return nil
	}
	if t, _ := typed["type"].(string); t != "" && !matchesType(t, val) {
		return &ValidationError{Tool: toolName, Field: fmt.Sprintf("%s[%d]", field, idx), Reason: fmt.Sprintf("expected %s", t)}
	}
	if t, _ := typed["type"].(string); t == "object" {
		obj, ok := val.(map[string]any)
		if !ok {
			return &ValidationError{Tool: toolName, Field: fmt.Sprintf("%s[%d]", field, idx), Reason: "expected object"}
		}
		required := toStringSlice(typed["required"])
		for _, req := range required {
			if _, exists := obj[req]; !exists {
				return &ValidationError{Tool: toolName, Field: fmt.Sprintf("%s[%d].%s", field, idx, req), Reason: "missing required field"}
			}
		}
		props, _ := typed["properties"].(map[string]any)
		for key, d := range props {
			if v, exists := obj[key]; exists {
				if err := validateField(toolName, fmt.Sprintf("%s[%d].%s", field, idx, key), d, v); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func matchesType(t string, val any) bool {
	switch strings.ToLower(t) {
	case "string":
		_, ok := val.(string)
		return ok
	case "integer":
		switch n := val.(type) {
		case float64:
			return n == float64(int64(n))
		case int, int8, int16, int32, int64:
			return true
		default:
			return false
		}
	case "number":
		switch val.(type) {
		case float64, float32, int, int8, int16, int32, int64:
			return true
		default:
			return false
		}
	case "boolean", "bool":
		_, ok := val.(bool)
		return ok
	case "array":
		_, ok := val.([]any)
		return ok
	case "object":
		_, ok := val.(map[string]any)
		return ok
	default:
		return true
	}
}

func isInEnum(val any, enumRaw any) bool {
	switch e := enumRaw.(type) {
	case []any:
		for _, candidate := range e {
			if fmt.Sprint(candidate) == fmt.Sprint(val) {
				return true
			}
		}
	case []string:
		for _, candidate := range e {
			if candidate == fmt.Sprint(val) {
				return true
			}
		}
	}
	return false
}

func toStringSlice(raw any) []string {
	switch x := raw.(type) {
	case []string:
		return x
	case []any:
		out := make([]string, 0, len(x))
		for _, v := range x {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
