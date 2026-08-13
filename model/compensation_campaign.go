package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrCompensationNotFound       = errors.New("compensation campaign not found")
	ErrCompensationUnavailable    = errors.New("compensation campaign is unavailable")
	ErrCompensationAlreadyClaimed = errors.New("compensation campaign already claimed")
)

type CompensationCampaign struct {
	Id          int    `json:"id"`
	Code        string `json:"code" gorm:"size:64;uniqueIndex"`
	Name        string `json:"name" gorm:"size:100;index"`
	Description string `json:"description" gorm:"type:text"`
	Quota       int    `json:"quota"`
	Enabled     bool   `json:"enabled"`
	CreatedTime int64  `json:"created_time" gorm:"bigint"`
	ExpiresTime int64  `json:"expires_time" gorm:"bigint"`
	ClaimCount  int64  `json:"claim_count" gorm:"-:all"`
	Claimed     bool   `json:"claimed" gorm:"-:all"`
}

type CompensationClaim struct {
	Id          int   `json:"id"`
	CampaignId  int   `json:"campaign_id" gorm:"uniqueIndex:idx_campaign_user"`
	UserId      int   `json:"user_id" gorm:"uniqueIndex:idx_campaign_user;index"`
	Quota       int   `json:"quota"`
	ClaimedTime int64 `json:"claimed_time" gorm:"bigint"`
}

func ClaimCompensation(code string, userId int) (int, error) {
	if code == "" || userId <= 0 {
		return 0, ErrCompensationNotFound
	}
	var campaign CompensationCampaign
	now := common.GetTimestamp()
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where("code = ?", code).First(&campaign).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrCompensationNotFound
			}
			return err
		}
		if !campaign.Enabled || campaign.Quota <= 0 || campaign.ExpiresTime <= 0 || campaign.ExpiresTime < now {
			return ErrCompensationUnavailable
		}
		claim := CompensationClaim{
			CampaignId:  campaign.Id,
			UserId:      userId,
			Quota:       campaign.Quota,
			ClaimedTime: now,
		}
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&claim)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrCompensationAlreadyClaimed
		}
		var user User
		if err := lockForUpdate(tx).Select("id", "quota").Where("id = ?", userId).First(&user).Error; err != nil {
			return err
		}
		if user.Quota > common.MaxQuota-campaign.Quota {
			return ErrCompensationUnavailable
		}
		return tx.Model(&User{}).Where("id = ?", userId).
			Update("quota", gorm.Expr("quota + ?", campaign.Quota)).Error
	})
	if err != nil {
		return 0, err
	}
	RecordLog(userId, LogTypeTopup, fmt.Sprintf("领取补偿活动 %s，额度 %s", campaign.Name, logger.LogQuota(campaign.Quota)))
	return campaign.Quota, nil
}

func GetCompensationCampaign(code string, userId int) (*CompensationCampaign, error) {
	var campaign CompensationCampaign
	if err := DB.Where("code = ?", code).First(&campaign).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCompensationNotFound
		}
		return nil, err
	}
	if userId > 0 {
		var count int64
		if err := DB.Model(&CompensationClaim{}).
			Where("campaign_id = ? AND user_id = ?", campaign.Id, userId).
			Count(&count).Error; err != nil {
			return nil, err
		}
		campaign.Claimed = count > 0
	}
	return &campaign, nil
}

func ListCompensationCampaigns(startIdx, num int) ([]CompensationCampaign, int64, error) {
	var campaigns []CompensationCampaign
	var total int64
	if err := DB.Model(&CompensationCampaign{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := DB.Order("id desc").Offset(startIdx).Limit(num).Find(&campaigns).Error; err != nil {
		return nil, 0, err
	}
	for i := range campaigns {
		if err := DB.Model(&CompensationClaim{}).Where("campaign_id = ?", campaigns[i].Id).
			Count(&campaigns[i].ClaimCount).Error; err != nil {
			return nil, 0, err
		}
	}
	if campaigns == nil {
		campaigns = make([]CompensationCampaign, 0)
	}
	return campaigns, total, nil
}

func CreateCompensationCampaign(campaign *CompensationCampaign) error {
	return DB.Create(campaign).Error
}

func UpdateCompensationCampaign(id int, updates map[string]interface{}) error {
	result := DB.Model(&CompensationCampaign{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCompensationNotFound
	}
	return nil
}
