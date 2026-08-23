package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOaiResponsesHandlerNormalizesErrorInSuccessfulResponse(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(
			`{"error":{"message":"Upstream service temporarily unavailable","type":"server_error"}}`,
		)),
	}

	_, apiErr := OaiResponsesHandler(c, &relaycommon.RelayInfo{}, resp)

	require.NotNil(t, apiErr)
	require.Equal(t, http.StatusBadGateway, apiErr.StatusCode)
	require.Contains(t, apiErr.Error(), "temporarily unavailable")
}

func TestOaiResponsesCompactionHandlerNormalizesErrorInSuccessfulResponse(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(
			`{"error":{"message":"Upstream service temporarily unavailable","type":"server_error"}}`,
		)),
	}

	_, apiErr := OaiResponsesCompactionHandler(c, resp)

	require.NotNil(t, apiErr)
	require.Equal(t, http.StatusBadGateway, apiErr.StatusCode)
	require.Contains(t, apiErr.Error(), "temporarily unavailable")
}

func TestUpstreamErrorStatusPreservesNonSuccessStatus(t *testing.T) {
	require.Equal(t, http.StatusServiceUnavailable, upstreamErrorStatus(http.StatusServiceUnavailable))
}
