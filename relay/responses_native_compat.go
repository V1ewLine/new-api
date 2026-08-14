package relay

import (
	"encoding/json"

	"github.com/QuantumNous/new-api/common"
)

// normalizeResponsesContentPartsForNativeCompat rewrites only Responses input
// content blocks into the Chat-style content parts expected by Responses
// implementations backed by ChatCompletionRequest. Tool outputs are normalized
// only when the output itself is a top-level Responses content-block array;
// arbitrary nested payloads remain untouched so a user-owned "type" field is
// never rewritten recursively.
func normalizeResponsesContentPartsForNativeCompat(input json.RawMessage) (json.RawMessage, bool, error) {
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
	changed := normalizeResponsesContentPart(item)
	changed = normalizeResponsesToolOutputContentParts(item) || changed
	content, ok := item["content"]
	if !ok {
		return changed
	}

	switch typed := content.(type) {
	case []any:
		for _, part := range typed {
			if contentPart, ok := part.(map[string]any); ok {
				changed = normalizeResponsesContentPart(contentPart) || changed
			}
		}
	case map[string]any:
		changed = normalizeResponsesContentPart(typed) || changed
	}
	return changed
}

func normalizeResponsesToolOutputContentParts(item map[string]any) bool {
	itemType, ok := item["type"].(string)
	if !ok || (itemType != "function_call_output" && itemType != "custom_tool_call_output") {
		return false
	}

	output, ok := item["output"].([]any)
	if !ok {
		return false
	}

	changed := false
	for _, part := range output {
		if contentPart, ok := part.(map[string]any); ok {
			changed = normalizeResponsesContentPart(contentPart) || changed
		}
	}
	return changed
}

func normalizeResponsesContentPart(part map[string]any) bool {
	partType, ok := part["type"].(string)
	if !ok {
		return false
	}

	switch partType {
	case "input_text", "output_text":
		part["type"] = "text"
		return true
	case "input_image":
		imageURL, ok := part["image_url"]
		if !ok {
			imageURL, ok = part["url"]
		}
		if !ok || imageURL == nil {
			return false
		}

		switch typed := imageURL.(type) {
		case string:
			normalized := map[string]any{"url": typed}
			if detail, exists := part["detail"]; exists {
				normalized["detail"] = detail
			}
			imageURL = normalized
		case map[string]any:
			if detail, exists := part["detail"]; exists {
				if _, hasNestedDetail := typed["detail"]; !hasNestedDetail {
					typed["detail"] = detail
				}
			}
		default:
			return false
		}

		part["type"] = "image_url"
		part["image_url"] = imageURL
		delete(part, "url")
		delete(part, "detail")
		return true
	default:
		return false
	}
}
