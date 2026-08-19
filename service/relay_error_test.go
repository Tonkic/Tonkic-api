package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relaykit/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldRetryRelayErrorSpecificChannelSkipsChannelError(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("specific_channel_id", "1")
	err := types.NewError(errors.New("channel failed"), types.ErrorCodeChannelNoAvailableKey)

	if ShouldRetryRelayError(c, err, 1) {
		t.Fatal("specific channel channel error should not retry")
	}
}

func TestShouldRetryRelayErrorStopsAfterClientDisconnect(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	requestContext, cancel := context.WithCancel(context.Background())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil).WithContext(requestContext)
	cancel()
	require.ErrorIs(t, c.Request.Context().Err(), context.Canceled)

	err := types.NewError(errors.New("upstream failed"), types.ErrorCodeChannelNoAvailableKey)

	assert.False(t, ShouldRetryRelayError(c, err, 1))
}

func TestShouldRetryRelayErrorStopsAfterResponseWasWritten(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Status(http.StatusOK)
	c.Writer.WriteHeaderNow()
	require.True(t, c.Writer.Written())

	err := types.NewErrorWithStatusCode(errors.New("upstream failed after streaming started"), types.ErrorCodeBadResponseStatusCode, http.StatusServiceUnavailable)

	assert.False(t, ShouldRetryRelayError(c, err, 1))
}

func TestShouldRetryRelayErrorAllowsRetryAfterWebSocketUpgradeBeforeModelOutput(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Status(http.StatusSwitchingProtocols)
	c.Writer.WriteHeaderNow()
	require.True(t, c.Writer.Written())

	err := types.NewErrorWithStatusCode(errors.New("upstream websocket dial failed"), types.ErrorCodeBadResponseStatusCode, http.StatusServiceUnavailable)

	assert.True(t, ShouldRetryRelayError(c, err, 1))
}

func TestShouldRetryRelayErrorAllowsGatewayTimeoutForAutoCrossGroupRetry(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(c, constant.ContextKeyTokenGroup, "auto")
	common.SetContextKey(c, constant.ContextKeyTokenCrossGroupRetry, true)
	err := types.NewErrorWithStatusCode(errors.New("upstream gateway timeout"), types.ErrorCodeBadResponseStatusCode, http.StatusGatewayTimeout)

	assert.True(t, ShouldRetryRelayError(c, err, 1))

	common.SetContextKey(c, constant.ContextKeyTokenCrossGroupRetry, false)
	assert.False(t, ShouldRetryRelayError(c, err, 1))
}

func TestShouldRetryRelayErrorAllowsTransientUpstreamStatuses(t *testing.T) {
	for _, statusCode := range []int{
		http.StatusTooManyRequests,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
	} {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			err := types.NewErrorWithStatusCode(errors.New("transient upstream failure"), types.ErrorCodeBadResponseStatusCode, statusCode)

			assert.True(t, ShouldRetryRelayError(c, err, 1))
		})
	}
}
