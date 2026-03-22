package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
)

func mustJSONMap(value any) (map[string]any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func readJSONMap(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read json file: %w", err)
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse json file: %w", err)
	}
	asMap, ok := raw.(map[string]any)
	if !ok {
		return nil, errors.New("json payload must be an object")
	}
	return asMap, nil
}

func readJSONArray(path string) ([]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read json file: %w", err)
	}
	var raw []any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse json file: %w", err)
	}
	return raw, nil
}

func mergeMaps(base map[string]any, overlay map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range base {
		result[key] = value
	}
	for key, value := range overlay {
		result[key] = value
	}
	return result
}

func parseIdentifier(id string) (int, bool) {
	value, err := strconv.Atoi(strings.TrimSpace(id))
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
}

func isScalar(value any) bool {
	switch value.(type) {
	case nil, string, bool, float64, int, int64, json.Number:
		return true
	default:
		return false
	}
}

func asStringSlice(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func asIntSlice(raw string) ([]int, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		value, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("parse int list %q: %w", raw, err)
		}
		out = append(out, value)
	}
	return out, nil
}

func appendQuery(rawURL string, values url.Values) string {
	if len(values) == 0 {
		return rawURL
	}
	if strings.Contains(rawURL, "?") {
		return rawURL + "&" + values.Encode()
	}
	return rawURL + "?" + values.Encode()
}

func cloneMap(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func deriveChangedFields(current map[string]any, desired map[string]any) map[string]any {
	changed := map[string]any{}
	for key, value := range desired {
		if reflect.DeepEqual(normaliseJSONValue(current[key]), normaliseJSONValue(value)) {
			continue
		}
		changed[key] = value
	}
	return changed
}

func normaliseJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := map[string]any{}
		for key, item := range typed {
			out[key] = normaliseJSONValue(item)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, normaliseJSONValue(item))
		}
		return out
	case float64:
		if float64(int64(typed)) == typed {
			return int64(typed)
		}
		return typed
	default:
		return typed
	}
}
