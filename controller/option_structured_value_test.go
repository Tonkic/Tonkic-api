package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateOptionRejectsStructuredValue(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "object", value: `{"enabled":true}`},
		{name: "array", value: `[true]`},
		{name: "null", value: `null`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(response)
			context.Request = httptest.NewRequest(
				http.MethodPut,
				"/api/option/",
				strings.NewReader(`{"key":"risk_setting","value":`+test.value+`}`),
			)

			UpdateOption(context)

			assert.Equal(t, http.StatusBadRequest, response.Code)
			var payload struct {
				Success bool `json:"success"`
			}
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
			assert.False(t, payload.Success)
		})
	}
}
