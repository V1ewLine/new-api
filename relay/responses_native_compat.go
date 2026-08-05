package relay

import (
	"encoding/json"

	"github.com/QuantumNous/new-api/common"
)

// normalizeResponsesTextPartsForNativeCompat rewrites only Responses input
// content blocks. It deliberately does not walk tool outputs or arbitrary
// nested payloads, where a user-owned "type" field must remain untouched.
func normalizeResponsesTextPartsForNativeCompat(input json.RawMessage) (json.RawMessage, bool, error) {
	if len(input) == 0 {
		return input, false, nil
	}

	var value any
	if err := common.Unmarshal(input, &value); err != nil {
		return nil, false, err
	}

	changed := false
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if responsesInputItem, ok := item.(map[string]any); ok {
				changed = normalizeResponsesInputItem(responsesInputItem) || changed
			}
		}
	case map[string]any:
		changed = normalizeResponsesInputItem(typed)
	}
	if !changed {
		return input, false, nil
	}

	normalized, err := common.Marshal(value)
	if err != nil {
		return nil, false, err
	}
	return normalized, true, nil
}

func normalizeResponsesInputItem(item map[string]any) bool {
	changed := normalizeResponsesTextPart(item)
	content, ok := item["content"]
	if !ok {
		return changed
	}

	switch typed := content.(type) {
	case []any:
		for _, part := range typed {
			if contentPart, ok := part.(map[string]any); ok {
				changed = normalizeResponsesTextPart(contentPart) || changed
			}
		}
	case map[string]any:
		changed = normalizeResponsesTextPart(typed) || changed
	}
	return changed
}

func normalizeResponsesTextPart(part map[string]any) bool {
	partType, ok := part["type"].(string)
	if !ok || (partType != "input_text" && partType != "output_text") {
		return false
	}
	part["type"] = "text"
	return true
}
