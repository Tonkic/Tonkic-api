package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetPublicModelStatusReturnsAllActiveGroupsWithoutCountsOrChannels(t *testing.T) {
	initModelListColumnNames(t)
	previousDB := model.DB
	previousMainDatabaseType := common.MainDatabaseType()
	previousLogDatabaseType := common.LogDatabaseType()
	previousAutoGroups := setting.AutoGroups2JsonString()
	previousGroupRatios := ratio_setting.GroupRatio2JSONString()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.PerfMetric{}))
	model.DB = db
	require.NoError(t, setting.UpdateAutoGroupsByJsonString(`["default","vip"]`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":1,"svip":1}`))
	t.Cleanup(func() {
		model.DB = previousDB
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		require.NoError(t, setting.UpdateAutoGroupsByJsonString(previousAutoGroups))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousGroupRatios))
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
				Group        string  `json:"group"`
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
	require.Len(t, payload.Data.Models, 3)
	groups := lo.Map(payload.Data.Models, func(item struct {
		ModelName    string  `json:"model_name"`
		Group        string  `json:"group"`
		SuccessRate  float64 `json:"success_rate"`
		HourlySeries []struct {
			Ts          int64    `json:"ts"`
			SuccessRate *float64 `json:"success_rate"`
		} `json:"hourly_series"`
	}, _ int) string {
		return item.Group
	})
	assert.ElementsMatch(t, []string{"default", "vip", "svip"}, groups)
	assert.Equal(t, "example-model", payload.Data.Models[0].ModelName)
	assert.InDelta(t, 40, payload.Data.OverallSuccessRate, 0.01)
	require.Len(t, payload.Data.Models[0].HourlySeries, 24)
	for index := 1; index < len(payload.Data.Models[0].HourlySeries); index++ {
		assert.Equal(t, int64(3600), payload.Data.Models[0].HourlySeries[index].Ts-payload.Data.Models[0].HourlySeries[index-1].Ts)
	}
	latest := payload.Data.Models[0].HourlySeries[len(payload.Data.Models[0].HourlySeries)-1]
	require.NotNil(t, latest.SuccessRate)
	require.NotEmpty(t, payload.Data.Models[0].Group)
	assert.NotContains(t, response.Body.String(), `"request_count"`)
	assert.NotContains(t, response.Body.String(), `"channel_id"`)
	assert.NotContains(t, response.Body.String(), `"channel_name"`)
}

func TestGetPublicModelStatusSplitsGPTAutoGroups(t *testing.T) {
	initModelListColumnNames(t)
	previousDB := model.DB
	previousMainDatabaseType := common.MainDatabaseType()
	previousLogDatabaseType := common.LogDatabaseType()
	previousAutoGroups := setting.AutoGroups2JsonString()
	previousGroupRatios := ratio_setting.GroupRatio2JSONString()
	previousUsableGroups := setting.UserUsableGroups2JSONString()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.PerfMetric{}))
	model.DB = db
	require.NoError(t, setting.UpdateAutoGroupsByJsonString(`["0.05羊毛","gpt_sale"]`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"0.05羊毛":0.05,"gpt_sale":0.5,"all-with-team":1}`))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"0.05羊毛":"0.05倍率分组","gpt_sale":"0.1倍率分组"}`))
	t.Cleanup(func() {
		model.DB = previousDB
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		require.NoError(t, setting.UpdateAutoGroupsByJsonString(previousAutoGroups))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousGroupRatios))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(previousUsableGroups))
	})

	now := time.Now().Unix()
	for _, metric := range []model.PerfMetric{
		{ModelName: "gpt-5.6-sol", Group: "0.05羊毛", BucketTs: now - 60, RequestCount: 1, SuccessCount: 1, TotalLatencyMs: 100},
		{ModelName: "gpt-5.6-sol", Group: "gpt_sale", BucketTs: now - 60, RequestCount: 9, SuccessCount: 0, TotalLatencyMs: 900},
		{ModelName: "gpt-5.6-sol", Group: "all-with-team", BucketTs: now - 60, RequestCount: 50, SuccessCount: 50, TotalLatencyMs: 5000},
	} {
		require.NoError(t, db.Create(&metric).Error)
	}

	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/status/models", nil)
	GetPublicModelStatus(context)

	var payload struct {
		Data struct {
			OverallSuccessRate float64 `json:"overall_success_rate"`
			Models             []struct {
				ModelName        string  `json:"model_name"`
				Group            string  `json:"group"`
				GroupDisplayName string  `json:"group_display_name"`
				SuccessRate      float64 `json:"success_rate"`
			} `json:"models"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
	require.Len(t, payload.Data.Models, 2)
	groups := lo.Map(payload.Data.Models, func(item struct {
		ModelName        string  `json:"model_name"`
		Group            string  `json:"group"`
		GroupDisplayName string  `json:"group_display_name"`
		SuccessRate      float64 `json:"success_rate"`
	}, _ int) string {
		return item.Group
	})
	assert.ElementsMatch(t, []string{"0.05羊毛", "gpt_sale"}, groups)
	displayNames := lo.Map(payload.Data.Models, func(item struct {
		ModelName        string  `json:"model_name"`
		Group            string  `json:"group"`
		GroupDisplayName string  `json:"group_display_name"`
		SuccessRate      float64 `json:"success_rate"`
	}, _ int) string {
		return item.GroupDisplayName
	})
	assert.ElementsMatch(t, []string{"0.05倍率分组", "0.1倍率分组"}, displayNames)
	assert.InDelta(t, 10, payload.Data.OverallSuccessRate, 0.01)
	assert.NotContains(t, response.Body.String(), "all-with-team")
	assert.NotContains(t, response.Body.String(), `"request_count"`)
}

func TestGetPublicModelStatusExcludesConfiguredModels(t *testing.T) {
	initModelListColumnNames(t)
	previousDB := model.DB
	previousMainDatabaseType := common.MainDatabaseType()
	previousLogDatabaseType := common.LogDatabaseType()
	previousExcludedModels := operation_setting.GetMonitorSetting().ExcludedAutoTestModels
	previousAutoGroups := setting.AutoGroups2JsonString()
	previousGroupRatios := ratio_setting.GroupRatio2JSONString()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.PerfMetric{}))
	model.DB = db
	require.NoError(t, setting.UpdateAutoGroupsByJsonString(`["default"]`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1}`))
	operation_setting.GetMonitorSetting().ExcludedAutoTestModels = []string{"glm-5.2", "grok-4.5*", "gpt-image-2"}
	t.Cleanup(func() {
		model.DB = previousDB
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		require.NoError(t, setting.UpdateAutoGroupsByJsonString(previousAutoGroups))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousGroupRatios))
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
