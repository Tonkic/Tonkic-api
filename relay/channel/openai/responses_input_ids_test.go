package openai

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestSanitizeResponsesInputItemIDsNormalizesCustomToolOutput(t *testing.T) {
	input := json.RawMessage(`[
		{"type":"custom_tool_call_output","id":"fc_01a01e21-8754-70a1-8aca-ed23425d8e92","call_id":"call_1","output":"ok"},
		{"type":"custom_tool_call_output","id":"ctco_existing","call_id":"call_2","output":"ok"},
		{"type":"function_call_output","id":"fc_output","call_id":"call_3","output":"ok"}
	]`)

	got := sanitizeResponsesInputItemIDs(input)

	require.Equal(t, "ctco_fc_01a01e21-8754-70a1-8aca-ed23425d8e92", gjson.GetBytes(got, "0.id").String())
	require.Equal(t, "call_1", gjson.GetBytes(got, "0.call_id").String())
	require.Equal(t, "ctco_existing", gjson.GetBytes(got, "1.id").String())
	require.Equal(t, "fc_output", gjson.GetBytes(got, "2.id").String())
	require.JSONEq(t, string(got), string(sanitizeResponsesInputItemIDs(got)))
}

func TestSanitizeResponsesInputItemIDsLeavesNonArrayUnchanged(t *testing.T) {
	for _, input := range []json.RawMessage{
		json.RawMessage(`"plain prompt"`),
		json.RawMessage(`{"type":"message","id":"wrong"}`),
		json.RawMessage(`not-json`),
	} {
		require.Equal(t, string(input), string(sanitizeResponsesInputItemIDs(input)))
	}
}

func TestConvertOpenAIResponsesRequestSanitizesInputIDs(t *testing.T) {
	request := dto.OpenAIResponsesRequest{
		Model: "gpt-test",
		Input: json.RawMessage(`[{"type":"custom_tool_call_output","id":"fc_bad","call_id":"call_1","output":"ok"}]`),
	}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, nil, request)
	require.NoError(t, err)
	got := converted.(dto.OpenAIResponsesRequest)
	require.Equal(t, "ctco_fc_bad", gjson.GetBytes(got.Input, "0.id").String())
	require.Equal(t, "call_1", gjson.GetBytes(got.Input, "0.call_id").String())
}

func TestSanitizeResponsesInputItemIDsCapsGeneratedIDs(t *testing.T) {
	input := json.RawMessage(`[{"type":"custom_tool_call_output","id":"` + strings.Repeat("x", 80) + `"}]`)
	got := sanitizeResponsesInputItemIDs(input)
	id := gjson.GetBytes(got, "0.id").String()

	require.True(t, strings.HasPrefix(id, "ctco_"))
	require.LessOrEqual(t, len([]rune(id)), responsesInputItemIDLimit)
}
