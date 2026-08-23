package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetPublicModelStatusReturnsAllActiveGroupsWithoutCountsOrChannels(t *testing.T) {
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
			WindowHours        int     `json:"window_hours"`
			OverallSuccessRate float64 `json:"overall_success_rate"`
			Models             []struct {
				ModelName    string  `json:"model_name"`
				SuccessRate  float64 `json:"success_rate"`
				HourlySeries []struct {
					Ts          int64    `json:"ts"`
					SuccessRate *float64 `json:"success_rate"`
				} `json:"hourly_series"`
			} `json:"models"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
	require.Equal(t, http.StatusOK, response.Code)
	require.True(t, payload.Success)
	assert.Equal(t, 24, payload.Data.WindowHours)
	require.Len(t, payload.Data.Models, 1)
	assert.Equal(t, "example-model", payload.Data.Models[0].ModelName)
	assert.InDelta(t, 40, payload.Data.OverallSuccessRate, 0.01)
	assert.InDelta(t, 40, payload.Data.Models[0].SuccessRate, 0.01)
	require.Len(t, payload.Data.Models[0].HourlySeries, 24)
	for index := 1; index < len(payload.Data.Models[0].HourlySeries); index++ {
		assert.Equal(t, int64(3600), payload.Data.Models[0].HourlySeries[index].Ts-payload.Data.Models[0].HourlySeries[index-1].Ts)
	}
	latest := payload.Data.Models[0].HourlySeries[len(payload.Data.Models[0].HourlySeries)-1]
	require.NotNil(t, latest.SuccessRate)
	assert.InDelta(t, 40, *latest.SuccessRate, 0.01)
	assert.NotContains(t, response.Body.String(), `"request_count"`)
	assert.NotContains(t, response.Body.String(), `"groups"`)
	assert.NotContains(t, response.Body.String(), `"channel_id"`)
	assert.NotContains(t, response.Body.String(), `"channel_name"`)
}

func TestGetPublicModelStatusExcludesConfiguredModels(t *testing.T) {
	initModelListColumnNames(t)
	previousDB := model.DB
	previousMainDatabaseType := common.MainDatabaseType()
	previousLogDatabaseType := common.LogDatabaseType()
	previousExcludedModels := operation_setting.GetMonitorSetting().ExcludedAutoTestModels
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.PerfMetric{}))
	model.DB = db
	operation_setting.GetMonitorSetting().ExcludedAutoTestModels = []string{"glm-5.2", "grok-4.5*", "gpt-image-2"}
	t.Cleanup(func() {
		model.DB = previousDB
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		operation_setting.GetMonitorSetting().ExcludedAutoTestModels = previousExcludedModels
	})

	now := time.Now().Unix()
	for _, modelName := range []string{"gpt-5.6-sol", "glm-5.2", "grok-4.5-long", "gpt-image-2"} {
		require.NoError(t, db.Create(&model.PerfMetric{
			ModelName: modelName, Group: "default", BucketTs: now - 60,
			RequestCount: 1, SuccessCount: 1, TotalLatencyMs: 1000,
		}).Error)
	}

	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/status/models", nil)

	GetPublicModelStatus(context)

	var payload struct {
		Data struct {
			Models []struct {
				ModelName string `json:"model_name"`
			} `json:"models"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
	require.Equal(t, http.StatusOK, response.Code)
	require.Len(t, payload.Data.Models, 1)
	assert.Equal(t, "gpt-5.6-sol", payload.Data.Models[0].ModelName)
}
