package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupCompensationCampaignFixture(t *testing.T, expiresAt int64) (*CompensationCampaign, *User) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&CompensationCampaign{}, &CompensationClaim{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&CompensationClaim{}).Error)
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&CompensationCampaign{}).Error)

	user := &User{Username: "compensation-user-" + t.Name(), AffCode: "comp-" + t.Name(), Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(user).Error)
	campaign := &CompensationCampaign{
		Code:        "august-compensation",
		Name:        "August compensation",
		Quota:       2500000,
		Enabled:     true,
		ExpiresTime: expiresAt,
	}
	require.NoError(t, DB.Create(campaign).Error)
	t.Cleanup(func() {
		DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&CompensationClaim{})
		DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&CompensationCampaign{})
		DB.Delete(&User{}, user.Id)
	})
	return campaign, user
}

func TestClaimCompensationCreditsUserOnlyOnce(t *testing.T) {
	campaign, user := setupCompensationCampaignFixture(t, common.GetTimestamp()+3600)

	quota, err := ClaimCompensation(campaign.Code, user.Id)
	require.NoError(t, err)
	assert.Equal(t, campaign.Quota, quota)

	_, err = ClaimCompensation(campaign.Code, user.Id)
	require.ErrorIs(t, err, ErrCompensationAlreadyClaimed)

	var updated User
	require.NoError(t, DB.First(&updated, user.Id).Error)
	assert.Equal(t, campaign.Quota, updated.Quota)
}

func TestClaimCompensationRejectsExpiredCampaign(t *testing.T) {
	campaign, user := setupCompensationCampaignFixture(t, common.GetTimestamp()-1)

	_, err := ClaimCompensation(campaign.Code, user.Id)
	require.ErrorIs(t, err, ErrCompensationUnavailable)

	var updated User
	require.NoError(t, DB.First(&updated, user.Id).Error)
	assert.Zero(t, updated.Quota)
}

func TestClaimCompensationRejectsQuotaOverflow(t *testing.T) {
	campaign, user := setupCompensationCampaignFixture(t, common.GetTimestamp()+3600)
	require.NoError(t, DB.Model(user).Update("quota", common.MaxQuota-campaign.Quota+1).Error)

	_, err := ClaimCompensation(campaign.Code, user.Id)
	require.ErrorIs(t, err, ErrCompensationUnavailable)

	var claimCount int64
	require.NoError(t, DB.Model(&CompensationClaim{}).Where("campaign_id = ?", campaign.Id).Count(&claimCount).Error)
	assert.Zero(t, claimCount)
}
