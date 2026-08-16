package controller

import (
	"net/http"
	"strconv"
	"time"

	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

func GetPublicModelStatus(c *gin.Context) {
	activeGroups := lo.Keys(ratio_setting.GetGroupRatioCopy())
	result, err := perfmetrics.QuerySummaryAll(24, activeGroups)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Unable to load model status",
		})
		return
	}

	var totalRequests int64
	var weightedSuccessRate float64
	models := make([]gin.H, 0, len(result.Models))
	for _, item := range result.Models {
		hourlySeries := make([]gin.H, 0, len(item.HourlySeries))
		for _, point := range item.HourlySeries {
			hourlySeries = append(hourlySeries, gin.H{
				"ts": point.Ts, "success_rate": point.SuccessRate,
			})
		}
		models = append(models, gin.H{
			"model_name":           item.ModelName,
			"avg_latency_ms":       item.AvgLatencyMs,
			"success_rate":         item.SuccessRate,
			"avg_tps":              item.AvgTps,
			"recent_success_rates": item.RecentSuccessRates,
			"hourly_series":        hourlySeries,
		})
		totalRequests += item.RequestCount
		weightedSuccessRate += item.SuccessRate * float64(item.RequestCount)
	}
	overallSuccessRate := 0.0
	if totalRequests > 0 {
		overallSuccessRate = weightedSuccessRate / float64(totalRequests)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"generated_at":         time.Now().Unix(),
			"window_hours":         24,
			"overall_success_rate": overallSuccessRate,
			"models":               models,
		},
	})
}

func GetPerfMetricsSummary(c *gin.Context) {
	hours := 24
	if rawHours := c.Query("hours"); rawHours != "" {
		if parsed, err := strconv.Atoi(rawHours); err == nil {
			hours = parsed
		}
	}

	activeGroups := append(lo.Keys(ratio_setting.GetGroupRatioCopy()), "auto")
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

	result, err := perfmetrics.Query(perfmetrics.QueryParams{
		Model: modelName,
		Group: c.Query("group"),
		Hours: hours,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	result.Groups = filterActiveGroups(result.Groups)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func filterActiveGroups(groups []perfmetrics.GroupResult) []perfmetrics.GroupResult {
	activeRatios := ratio_setting.GetGroupRatioCopy()
	return lo.Filter(groups, func(g perfmetrics.GroupResult, _ int) bool {
		_, ok := activeRatios[g.Group]
		return ok || g.Group == "auto"
	})
}
