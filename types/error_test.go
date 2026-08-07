package types_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/require"
)

func TestNewAPIErrorSetMessageUpdatesRelayPayload(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		err   *types.NewAPIError
		check func(*testing.T, *types.NewAPIError)
	}{
		{
			name: "OpenAI payload",
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "internal error",
				Type:    "server_error",
				Code:    "server_error",
			}, http.StatusServiceUnavailable),
			check: func(t *testing.T, err *types.NewAPIError) {
				require.Equal(t, "friendly error", err.ToOpenAIError().Message)
			},
		},
		{
			name: "Claude payload",
			err: types.WithClaudeError(types.ClaudeError{
				Message: "internal error",
				Type:    "api_error",
			}, http.StatusServiceUnavailable),
			check: func(t *testing.T, err *types.NewAPIError) {
				require.Equal(t, "friendly error", err.ToClaudeError().Message)
			},
		},
		{
			name: "local payload",
			err:  types.NewError(errors.New("internal error"), types.ErrorCodeBadResponse),
			check: func(t *testing.T, err *types.NewAPIError) {
				require.Equal(t, "friendly error", err.ToOpenAIError().Message)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.err.SetMessage("friendly error")
			require.Equal(t, "friendly error", tc.err.Error())
			tc.check(t, tc.err)
		})
	}
}
