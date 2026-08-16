package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

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
