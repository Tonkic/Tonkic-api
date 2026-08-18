package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func ListRiskCases(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	cases, total, err := model.ListRiskCases(strings.TrimSpace(c.Query("status")), pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(cases)
	common.ApiSuccess(c, pageInfo)
}

func RunRiskScan(c *gin.Context) {
	task, created, err := service.EnqueueSystemTask(model.SystemTaskTypeRiskScan, nil)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"task": task, "created": created}})
}

type riskActionRequest struct {
	Resolution string `json:"resolution"`
}

func IgnoreRiskCase(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid risk case id")
		return
	}
	var request riskActionRequest
	_ = common.DecodeJson(c.Request.Body, &request)
	if err := model.ResolveRiskCase(id, c.GetInt("id"), model.RiskCaseStatusIgnored, strings.TrimSpace(request.Resolution)); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func BanRiskCase(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid risk case id")
		return
	}
	riskCase, err := model.GetRiskCase(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if riskCase.Status == model.RiskCaseStatusBanned {
		common.ApiSuccess(c, nil)
		return
	}
	for _, member := range riskCase.Users {
		var user model.User
		if err := model.DB.First(&user, member.UserID).Error; err != nil {
			common.ApiError(c, err)
			return
		}
		if user.Role >= common.RoleAdminUser {
			common.ApiErrorMsg(c, "administrator accounts cannot be risk-banned")
			return
		}
		allowed, err := model.RiskValueAllowed("user", strconv.Itoa(member.UserID), common.GetTimestamp())
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if allowed {
			common.ApiErrorMsg(c, "risk case contains an allowlisted user")
			return
		}
	}
	userIDs := make([]int, 0, len(riskCase.Users))
	for _, member := range riskCase.Users {
		userIDs = append(userIDs, member.UserID)
	}
	if err := model.ApplyRiskBanCase(id, userIDs, c.GetInt("id"), "manually banned by administrator"); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func RevertRiskCase(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid risk case id")
		return
	}
	riskCase, err := model.GetRiskCase(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if riskCase.Status != model.RiskCaseStatusBanned {
		common.ApiSuccess(c, nil)
		return
	}
	if err := model.RevertRiskBanCase(id, c.GetInt("id"), "risk ban reverted by administrator"); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
