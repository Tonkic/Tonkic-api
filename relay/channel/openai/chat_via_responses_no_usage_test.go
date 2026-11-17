package openai

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponsesToChatCreatedOnlyEOFDoesNotEstimatePrompt(t *testing.T) {
	old := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = old })
	body := "data: {\"type\":\"response.created\",\"response\":{\"status\":\"in_progress\",\"model\":\"gpt-4o\"}}\n\n"
	c, _, resp, info := newResponsesChatTestContext(t, body, true)
	info.SetEstimatePromptTokens(144609)
	usage, apiErr := OaiResponsesToChatStreamHandler(c, info, resp)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	require.Zero(t, usage.PromptTokens)
	require.Zero(t, usage.CompletionTokens)
	require.Zero(t, usage.TotalTokens)
}

func TestResponsesToChatNativeZeroOutputUsageKeepsCache(t *testing.T) {
	old := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = old })
	body := strings.Join([]string{
		`data: {"type":"response.created","response":{"status":"in_progress","model":"gpt-4o"}}`,
		`data: {"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":144609,"output_tokens":0,"total_tokens":144609,"input_tokens_details":{"cached_tokens":144000}}}}`,
		`data: [DONE]`, "",
	}, "\n")
	c, _, resp, info := newResponsesChatTestContext(t, body, true)
	info.SetEstimatePromptTokens(144609)
	usage, apiErr := OaiResponsesToChatStreamHandler(c, info, resp)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	require.Equal(t, 144609, usage.PromptTokens)
	require.Zero(t, usage.CompletionTokens)
	require.Equal(t, 144000, usage.PromptTokensDetails.CachedTokens)
}

func TestResponsesToChatToolOutputWithoutUsageStillBills(t *testing.T) {
	old := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = old })
	body := strings.Join([]string{
		`data: {"type":"response.created","response":{"status":"in_progress","model":"gpt-4o"}}`,
		`data: {"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","id":"fc_1","call_id":"call_1","name":"lookup"}}`,
		`data: {"type":"response.function_call_arguments.delta","output_index":0,"delta":"{\"q\":\"x\"}"}`, "",
	}, "\n")
	c, _, resp, info := newResponsesChatTestContext(t, body, true)
	info.SetEstimatePromptTokens(100)
	usage, apiErr := OaiResponsesToChatStreamHandler(c, info, resp)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	require.Equal(t, 100, usage.PromptTokens)
	require.Positive(t, usage.CompletionTokens)
}

func TestResponsesToChatErrorEventsDoNotSynthesizeSuccess(t *testing.T) {
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	for _, tc := range []struct {
		name, event, message, code string
	}{
		{"flat", `{"type":"error","code":"offline_rejected","message":"synthetic rejection","param":"input"}`, "synthetic rejection", "offline_rejected"},
		{"nested", `{"type":"error","error":{"type":"invalid_request_error","code":"offline_rejected","message":"synthetic rejection","param":"input"}}`, "synthetic rejection", "offline_rejected"},
		{"nested_without_type", `{"type":"error","error":{"code":"offline_rejected","message":"synthetic rejection"}}`, "synthetic rejection", "offline_rejected"},
		{"response_error", `{"type":"response.error","error":{"code":"offline_rejected","message":"synthetic rejection"}}`, "synthetic rejection", "offline_rejected"},
		{"failed_without_type", `{"type":"response.failed","response":{"status":"failed","error":{"code":"offline_rejected","message":"synthetic rejection"}}}`, "synthetic rejection", "offline_rejected"},
		{"missing_details", `{"type":"error"}`, "responses stream error: error", "bad_response"},
	} {
		for _, buffered := range []bool{false, true} {
			mode := "stream"
			if buffered {
				mode = "buffered"
			}
			t.Run(tc.name+"/"+mode, func(t *testing.T) {
				body := "data: {\"type\":\"response.created\",\"response\":{\"status\":\"in_progress\",\"model\":\"gpt-4o\"}}\n\n" + "data: " + tc.event + "\n\n"
				c, recorder, resp, info := newResponsesChatTestContext(t, body, !buffered)
				info.SetEstimatePromptTokens(144415)
				var usage *dto.Usage
				var apiErr *types.NewAPIError
				if buffered {
					usage, apiErr = OaiResponsesToChatBufferedStreamHandler(c, info, resp)
				} else {
					usage, apiErr = OaiResponsesToChatStreamHandler(c, info, resp)
				}
				require.NotNil(t, apiErr, "upstream error must not become an empty successful response")
				assert.Nil(t, usage, "no delivered output must not synthesize estimated billable usage")
				written := c.Writer.Written()
				assert.Equal(t, !buffered, written, "response.created commits the streaming role-only frame")
				assert.Equal(t, written, types.IsSkipRetryError(apiErr), "only committed responses must disable retries")
				if written {
					assert.Equal(t, "stop", service.DecideRelayRetry(c, apiErr, 1).Action, "committed streams must not retry")
				}
				require.NoError(t, helper.WriteRelayError(c, info.RelayFormat, apiErr))
				if written {
					assert.Contains(t, recorder.Body.String(), `data: {"error":`)
				} else {
					assert.Equal(t, apiErr.StatusCode, recorder.Code)
					assert.Contains(t, recorder.Header().Get("Content-Type"), "application/json")
				}
				oaiErr := apiErr.ToOpenAIError()
				assert.EqualValues(t, tc.code, oaiErr.Code)
				assert.Contains(t, oaiErr.Message, tc.message)
				assert.NotEmpty(t, oaiErr.Type)
				assert.Equal(t, "failed", info.StreamStatus.ResponseOutcome())
				assert.NotContains(t, recorder.Body.String(), `"finish_reason":"stop"`)
				assert.NotContains(t, recorder.Body.String(), "data: [DONE]")
			})
		}
	}
}

func TestResponsesToChatErrorAfterOutputKeepsUsageAndReportsFailure(t *testing.T) {
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	for _, tc := range []struct {
		name, output, failure      string
		prompt, completion, cached int
	}{
		{
			name:    "text_then_flat_error",
			output:  `data: {"type":"response.output_text.delta","delta":"delivered text"}`,
			failure: `data: {"type":"error","code":"offline_rejected","message":"synthetic rejection"}`,
			prompt:  100,
		},
		{
			name:    "text_then_failed_with_usage",
			output:  `data: {"type":"response.output_text.delta","delta":"delivered text"}`,
			failure: `data: {"type":"response.failed","response":{"status":"failed","error":{"type":"server_error","code":"offline_rejected","message":"synthetic rejection"},"usage":{"input_tokens":123,"output_tokens":7,"total_tokens":130,"input_tokens_details":{"cached_tokens":100}}}}`,
			prompt:  123, completion: 7, cached: 100,
		},
		{
			name:    "tool_then_flat_error",
			output:  "data: {\"type\":\"response.output_item.added\",\"output_index\":0,\"item\":{\"type\":\"function_call\",\"id\":\"fc_offline\",\"call_id\":\"call_offline\",\"name\":\"lookup\"}}\n\ndata: {\"type\":\"response.function_call_arguments.delta\",\"output_index\":0,\"delta\":\"{\\\"q\\\":\\\"x\\\"}\"}",
			failure: `data: {"type":"error","code":"offline_rejected","message":"synthetic rejection"}`,
			prompt:  100,
		},
	} {
		for _, claude := range []bool{false, true} {
			mode := "openai"
			if claude {
				mode = "claude"
			}
			t.Run(tc.name+"/"+mode, func(t *testing.T) {
				body := "data: {\"type\":\"response.created\",\"response\":{\"status\":\"in_progress\",\"model\":\"gpt-4o\"}}\n\n" + tc.output + "\n\n" + tc.failure + "\n\n"
				c, recorder, resp, info := newResponsesChatTestContext(t, body, true)
				if claude {
					info.RelayFormat = types.RelayFormatClaude
				}
				info.SetEstimatePromptTokens(100)
				usage, apiErr := OaiResponsesToChatStreamHandler(c, info, resp)
				require.Nil(t, apiErr, "delivered output must retain the existing settlement path")
				require.NotNil(t, usage)
				assert.Equal(t, tc.prompt, usage.PromptTokens)
				if tc.completion > 0 {
					assert.Equal(t, tc.completion, usage.CompletionTokens)
				} else {
					assert.Positive(t, usage.CompletionTokens)
				}
				assert.Equal(t, tc.cached, usage.PromptTokensDetails.CachedTokens)
				assert.Equal(t, "failed", info.StreamStatus.ResponseOutcome())
				out := recorder.Body.String()
				if claude {
					assert.Regexp(t, `(?m)^event: ?error\r?$`, out, "Claude failures must use the error event")
					assert.Contains(t, out, `"type":"error"`)
				} else {
					assert.Contains(t, out, `data: {"error":`, "an error after delivered SSE must itself be a framed SSE event")
					assert.Contains(t, out, `"code":"offline_rejected"`)
				}
				assert.Contains(t, out, "synthetic rejection")
				assert.NotContains(t, out, `"finish_reason":"stop"`)
				assert.NotContains(t, out, "data: [DONE]")
			})
		}
	}
}

func TestResponsesToChatFirstErrorUsesJSONWhenNotStarted(t *testing.T) {
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })
	for _, format := range []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude} {
		t.Run(string(format), func(t *testing.T) {
			body := "event: error\ndata: {\"type\":\"error\",\"code\":\"offline_rejected\",\"message\":\"synthetic first event rejection\"}\n\n"
			c, recorder, resp, info := newResponsesChatTestContext(t, body, true)
			info.RelayFormat = format
			info.SetEstimatePromptTokens(144415)
			usage, apiErr := OaiResponsesToChatStreamHandler(c, info, resp)
			require.NotNil(t, apiErr)
			assert.Nil(t, usage, "a rejected request must not synthesize local token usage")
			assert.False(t, c.Writer.Written())
			assert.False(t, types.IsSkipRetryError(apiErr), "an uncommitted response retains normal retry policy")
			require.NoError(t, helper.WriteRelayError(c, info.RelayFormat, apiErr))
			assert.Equal(t, apiErr.StatusCode, recorder.Code)
			assert.Contains(t, recorder.Header().Get("Content-Type"), "application/json")
			assert.NotContains(t, recorder.Header().Get("Content-Type"), "text/event-stream")
			assert.Contains(t, recorder.Body.String(), "synthetic first event rejection")
			assert.NotContains(t, recorder.Body.String(), "[DONE]")
			assert.NotContains(t, recorder.Body.String(), `"finish_reason":"stop"`)
		})
	}
}
