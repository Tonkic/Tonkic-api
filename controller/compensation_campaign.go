package controller

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

var compensationCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,63}$`)

type compensationCampaignRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Quota       int    `json:"quota"`
	Enabled     *bool  `json:"enabled"`
	ExpiresTime int64  `json:"expires_time"`
}

func ListCompensationCampaigns(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.ListCompensationCampaigns(pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func CreateCompensationCampaign(c *gin.Context) {
	var request compensationCampaignRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	request.Code = strings.ToLower(strings.TrimSpace(request.Code))
	request.Name = strings.TrimSpace(request.Name)
	if !compensationCodePattern.MatchString(request.Code) || request.Name == "" || len(request.Name) > 100 ||
		request.Quota <= 0 || request.Quota > common.MaxQuota || request.ExpiresTime <= common.GetTimestamp() {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Invalid campaign settings"})
		return
	}
	enabled := true
	if request.Enabled != nil {
		enabled = *request.Enabled
	}
	campaign := model.CompensationCampaign{
		Code: request.Code, Name: request.Name, Description: request.Description,
		Quota: request.Quota, Enabled: enabled, CreatedTime: common.GetTimestamp(), ExpiresTime: request.ExpiresTime,
	}
	if err := model.CreateCompensationCampaign(&campaign); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "compensation_campaign.create", map[string]interface{}{"id": campaign.Id, "code": campaign.Code, "quota": campaign.Quota})
	common.ApiSuccess(c, campaign)
}

func UpdateCompensationCampaign(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Invalid campaign ID"})
		return
	}
	var request compensationCampaignRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	request.Name = strings.TrimSpace(request.Name)
	if request.Name == "" || len(request.Name) > 100 || request.Quota <= 0 ||
		request.Quota > common.MaxQuota || request.ExpiresTime <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Invalid campaign settings"})
		return
	}
	updates := map[string]interface{}{
		"name": request.Name, "description": request.Description,
		"quota": request.Quota, "expires_time": request.ExpiresTime,
	}
	if request.Enabled != nil {
		updates["enabled"] = *request.Enabled
	}
	if err := model.UpdateCompensationCampaign(id, updates); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "compensation_campaign.update", map[string]interface{}{"id": id})
	common.ApiSuccess(c, nil)
}

func GetCompensationCampaign(c *gin.Context) {
	campaign, err := model.GetCompensationCampaign(c.Param("code"), c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, campaign)
}

func ClaimCompensationCampaign(c *gin.Context) {
	quota, err := model.ClaimCompensation(c.Param("code"), c.GetInt("id"))
	if err != nil {
		message := "Unable to claim this compensation"
		if errors.Is(err, model.ErrCompensationAlreadyClaimed) {
			message = "You have already claimed this compensation"
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": message})
		return
	}
	common.ApiSuccess(c, gin.H{"quota": quota})
}
