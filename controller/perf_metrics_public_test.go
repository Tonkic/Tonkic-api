package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetPublicModelStatusReturnsPerGroupAggregateWithoutChannels(t *testing.T) {
	initModelListColumnNames(t)
	previousDB := model.DB
	previousMainDatabaseType := common.MainDatabaseType()
	previousLogDatabaseType := common.LogDatabaseType()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.PerfMetric{}))
	model.DB = db
	t.Cleanup(func() {
		model.DB = previousDB
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
	})

	now := time.Now().Unix()
	require.NoError(t, db.Create(&model.PerfMetric{
		ModelName: "example-model", Group: "default", BucketTs: now - 60,
		RequestCount: 10, SuccessCount: 9, TotalLatencyMs: 15000,
	}).Error)
	require.NoError(t, db.Create(&model.PerfMetric{
		ModelName: "example-model", Group: "vip", BucketTs: now - 60,
		RequestCount: 5, SuccessCount: 5, TotalLatencyMs: 5000,
	}).Error)
	require.NoError(t, db.Create(&model.PerfMetric{
		ModelName: "example-model", Group: "svip", BucketTs: now - 60,
		RequestCount: 20, SuccessCount: 0, TotalLatencyMs: 40000,
	}).Error)

	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/status/models", nil)

	GetPublicModelStatus(context)

	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			WindowHours int `json:"window_hours"`
			Models      []struct {
				ModelName    string  `json:"model_name"`
				SuccessRate  float64 `json:"success_rate"`
				RequestCount int64   `json:"request_count"`
				HourlySeries []struct {
					Ts           int64    `json:"ts"`
					SuccessRate  *float64 `json:"success_rate"`
					RequestCount int64    `json:"request_count"`
				} `json:"hourly_series"`
				Groups []struct {
					Group        string  `json:"group"`
					SuccessRate  float64 `json:"success_rate"`
					RequestCount int64   `json:"request_count"`
				} `json:"groups"`
			} `json:"models"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
	require.Equal(t, http.StatusOK, response.Code)
	require.True(t, payload.Success)
	assert.Equal(t, 24, payload.Data.WindowHours)
	require.Len(t, payload.Data.Models, 1)
	assert.Equal(t, "example-model", payload.Data.Models[0].ModelName)
	assert.InDelta(t, 93.33, payload.Data.Models[0].SuccessRate, 0.01)
	assert.Equal(t, int64(15), payload.Data.Models[0].RequestCount)
	require.Len(t, payload.Data.Models[0].HourlySeries, 24)
	for index := 1; index < len(payload.Data.Models[0].HourlySeries); index++ {
		assert.Equal(t, int64(3600), payload.Data.Models[0].HourlySeries[index].Ts-payload.Data.Models[0].HourlySeries[index-1].Ts)
	}
	latest := payload.Data.Models[0].HourlySeries[len(payload.Data.Models[0].HourlySeries)-1]
	assert.Equal(t, int64(15), latest.RequestCount)
	require.NotNil(t, latest.SuccessRate)
	assert.InDelta(t, 93.33, *latest.SuccessRate, 0.01)
	require.Len(t, payload.Data.Models[0].Groups, 2)
	assert.ElementsMatch(t, []string{"default", "vip"}, []string{payload.Data.Models[0].Groups[0].Group, payload.Data.Models[0].Groups[1].Group})
	assert.NotContains(t, response.Body.String(), `"svip"`)
	assert.NotContains(t, response.Body.String(), `"channel_id"`)
	assert.NotContains(t, response.Body.String(), `"channel_name"`)
}
