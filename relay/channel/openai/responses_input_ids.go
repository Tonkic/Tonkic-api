package openai

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const responsesInputItemIDLimit = 64

var responsesInputItemIDPrefixes = map[string]string{
	"message":                 "msg",
	"reasoning":               "rs",
	"function_call":           "fc",
	"custom_tool_call":        "ctc",
	"custom_tool_call_output": "ctco",
}

// sanitizeResponsesInputItemIDs keeps Responses history IDs compatible when a
// request is retried against a stricter OpenAI/Codex-compatible upstream.
func sanitizeResponsesInputItemIDs(input json.RawMessage) json.RawMessage {
	parsed := gjson.ParseBytes(input)
	if !parsed.IsArray() {
		return input
	}

	items := parsed.Array()
	occupied := make(map[string]struct{}, len(items))
	for _, item := range items {
		idResult := item.Get("id")
		if idResult.Type != gjson.String {
			continue
		}
		id := idResult.String()
		if normalizeResponsesInputItemID(item.Get("type").String(), id) == id && len([]rune(id)) <= responsesInputItemIDLimit {
			occupied[id] = struct{}{}
		}
	}

	changed := false
	var rebuilt bytes.Buffer
	rebuilt.WriteByte('[')
	for index, item := range items {
		if index > 0 {
			rebuilt.WriteByte(',')
		}

		raw := []byte(item.Raw)
		idResult := item.Get("id")
		if idResult.Type == gjson.String {
			originalID := idResult.String()
			id := normalizeResponsesInputItemID(item.Get("type").String(), originalID)
			if len([]rune(id)) > responsesInputItemIDLimit {
				id = shortenResponsesInputItemID(id, 0)
			}
			if id != originalID {
				for attempt := 1; ; attempt++ {
					if _, exists := occupied[id]; !exists {
						break
					}
					id = shortenResponsesInputItemID(normalizeResponsesInputItemID(item.Get("type").String(), originalID), attempt)
				}
				if updated, err := sjson.SetBytes(raw, "id", id); err == nil {
					raw = updated
					changed = true
					occupied[id] = struct{}{}
				}
			}
		}
		rebuilt.Write(raw)
	}
	rebuilt.WriteByte(']')

	if !changed {
		return input
	}
	return json.RawMessage(rebuilt.Bytes())
}

func normalizeResponsesInputItemID(itemType, id string) string {
	prefix, ok := responsesInputItemIDPrefixes[itemType]
	if !ok || id == "" || strings.HasPrefix(id, prefix) {
		return id
	}
	return prefix + "_" + id
}

func shortenResponsesInputItemID(id string, attempt int) string {
	runes := []rune(id)
	if len(runes) <= responsesInputItemIDLimit && attempt == 0 {
		return id
	}

	hashInput := id
	if attempt > 0 {
		hashInput += "\x00" + strconv.Itoa(attempt)
	}
	sum := sha256.Sum256([]byte(hashInput))
	suffix := "_" + hex.EncodeToString(sum[:8])
	prefixLength := responsesInputItemIDLimit - len(suffix)
	if len(runes) < prefixLength {
		prefixLength = len(runes)
	}
	return string(runes[:prefixLength]) + suffix
}
