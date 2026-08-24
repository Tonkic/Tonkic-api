package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

func GetPublicModelStatus(c *gin.Context) {
	generatedAt := time.Now().Unix()
	activeGroups := visibleMetricGroups(c, false)
	result, err := perfmetrics.QuerySummaryAll(24, activeGroups)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Unable to load model status",
		})
		return
	}
	enabledPairs, err := model.GetEnabledModelGroups(activeGroups)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Unable to load current model capabilities",
		})
		return
	}

	var totalRequests int64
	var weightedSuccessRate float64
	models := make([]gin.H, 0, len(enabledPairs))
	excludedModels := operation_setting.GetMonitorSetting().ExcludedAutoTestModels
	metricByPair := make(map[string]perfmetrics.GroupSummary)
	for _, item := range result.Models {
		for _, group := range item.Groups {
			metricByPair[publicStatusPairKey(group.Group, item.ModelName)] = group
		}
	}

	for _, pair := range enabledPairs {
		if len(excludeAutomaticChannelTestModels([]string{pair.Model}, excludedModels)) == 0 {
			continue
		}

		group, hasMetrics := metricByPair[publicStatusPairKey(pair.Group, pair.Model)]
		hourlySeries := make([]gin.H, 0, 24)
		var successRate any
		if hasMetrics {
			for _, point := range group.HourlySeries {
				hourlySeries = append(hourlySeries, gin.H{
					"ts": point.Ts, "success_rate": point.SuccessRate,
				})
			}
			successRate = group.SuccessRate
			totalRequests += group.RequestCount
			weightedSuccessRate += group.SuccessRate * float64(group.RequestCount)
		} else {
			hourlySeries = emptyPublicStatusHourlySeries(24, generatedAt)
		}

		models = append(models, gin.H{
			"model_name":           pair.Model,
			"group":                pair.Group,
			"group_display_name":   setting.GetUsableGroupDescription(pair.Group),
			"avg_latency_ms":       group.AvgLatencyMs,
			"success_rate":         successRate,
			"avg_tps":              group.AvgTps,
			"recent_success_rates": group.RecentSuccessRates,
			"hourly_series":        hourlySeries,
		})
	}
	var overallSuccessRate any
	if totalRequests > 0 {
		overallSuccessRate = weightedSuccessRate / float64(totalRequests)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"generated_at":         generatedAt,
			"window_hours":         24,
			"overall_success_rate": overallSuccessRate,
			"models":               models,
		},
	})
}

func publicStatusPairKey(group, modelName string) string {
	return group + "\x00" + modelName
}

func emptyPublicStatusHourlySeries(hours int, endTs int64) []gin.H {
	endHour := endTs - endTs%3600
	startHour := endHour - int64(hours-1)*3600
	series := make([]gin.H, 0, hours)
	for ts := startHour; ts <= endHour; ts += 3600 {
		series = append(series, gin.H{"ts": ts, "success_rate": nil})
	}
	return series
}

func visibleMetricGroups(c *gin.Context, includeAuto bool) []string {
	activeRatios := ratio_setting.GetGroupRatioCopy()
	role := c.GetInt("role")
	groups := make([]string, 0, len(activeRatios)+1)
	if role >= common.RoleAdminUser {
		groups = append(groups, lo.Keys(activeRatios)...)
	} else {
		usable := service.GetUserUsableGroupsForRole(c.GetString("group"), role)
		for group := range activeRatios {
			if _, ok := usable[group]; ok {
				groups = append(groups, group)
			}
		}
	}
	if includeAuto {
		usable := service.GetUserUsableGroupsForRole(c.GetString("group"), role)
		if role >= common.RoleAdminUser {
			groups = append(groups, "auto")
		} else if _, ok := usable["auto"]; ok {
			groups = append(groups, "auto")
		}
	}
	return groups
}

func GetPerfMetricsSummary(c *gin.Context) {
	hours := 24
	if rawHours := c.Query("hours"); rawHours != "" {
		if parsed, err := strconv.Atoi(rawHours); err == nil {
			hours = parsed
		}
	}

	activeGroups := visibleMetricGroups(c, true)
	result, err := perfmetrics.QuerySummaryAll(hours, activeGroups)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func GetPerfMetrics(c *gin.Context) {
	modelName := c.Query("model")
	if modelName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "model is required",
		})
		return
	}

	hours := 24
	if rawHours := c.Query("hours"); rawHours != "" {
		if parsed, err := strconv.Atoi(rawHours); err == nil {
			hours = parsed
		}
	}

	visibleGroups := visibleMetricGroups(c, true)
	requestedGroup := c.Query("group")
	if requestedGroup != "" && !lo.Contains(visibleGroups, requestedGroup) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "group is not available",
		})
		return
	}
	result, err := perfmetrics.Query(perfmetrics.QueryParams{
		Model:         modelName,
		Group:         requestedGroup,
		AllowedGroups: visibleGroups,
		Hours:         hours,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	result.Groups = filterVisibleGroups(result.Groups, visibleGroups)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func filterActiveGroups(groups []perfmetrics.GroupResult) []perfmetrics.GroupResult {
	activeGroups := append(lo.Keys(ratio_setting.GetGroupRatioCopy()), "auto")
	return filterVisibleGroups(groups, activeGroups)
}

func filterVisibleGroups(groups []perfmetrics.GroupResult, visibleGroups []string) []perfmetrics.GroupResult {
	visible := lo.SliceToMap(visibleGroups, func(group string) (string, struct{}) {
		return group, struct{}{}
	})
	return lo.Filter(groups, func(g perfmetrics.GroupResult, _ int) bool {
		_, ok := visible[g.Group]
		return ok
	})
}
