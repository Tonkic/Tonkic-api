package service

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRunRiskScanCreatesCaseAndProtectsSuccessfulTopup(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:risk-scan?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}, &model.TopUp{}, &model.CompensationClaim{}, &model.RiskCase{}, &model.RiskCaseUser{}, &model.RiskAllowlist{}))
	oldDB, oldLogDB := model.DB, model.LOG_DB
	model.DB, model.LOG_DB = db, db
	t.Cleanup(func() { model.DB, model.LOG_DB = oldDB, oldLogDB })

	setting := operation_setting.GetRiskSetting()
	oldSetting := *setting
	*setting = operation_setting.RiskSetting{Enabled: true, AutoBanEnabled: true, ScanIntervalMinutes: 5, LookbackDays: 7, AutoBanScore: 100, MinimumCategories: 2}
	t.Cleanup(func() { *setting = oldSetting })
	now := time.Now().Unix()
	users := []model.User{{Username: "paid", AffCode: "paid-code", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, CreatedAt: now, AuthVersion: 1}, {Username: "bonus", AffCode: "bonus-code", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, CreatedAt: now, AuthVersion: 1}}
	require.NoError(t, db.Create(&users).Error)
	require.NoError(t, db.Create(&model.TopUp{UserId: users[0].Id, Status: common.TopUpStatusSuccess}).Error)
	require.NoError(t, db.Create(&model.CompensationClaim{CampaignId: 1, UserId: users[1].Id, ClaimedTime: now}).Error)
	for _, user := range users {
		for range 10 {
			require.NoError(t, db.Create(&model.Log{UserId: user.Id, Type: model.LogTypeConsume, Ip: "203.0.113.9", CreatedAt: now}).Error)
		}
	}

	summary, err := RunRiskScan(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, summary.CasesUpdated)
	assert.Zero(t, summary.UsersBanned, "one successful top-up protects the entire associated group")
	var caseCount int64
	require.NoError(t, db.Model(&model.RiskCase{}).Count(&caseCount).Error)
	assert.Equal(t, int64(1), caseCount)
}

func TestRunRiskScanDoesNotBanAnIgnoredCase(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:risk-ignored?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}, &model.TopUp{}, &model.CompensationClaim{}, &model.RiskCase{}, &model.RiskCaseUser{}, &model.RiskAllowlist{}))
	oldDB, oldLogDB := model.DB, model.LOG_DB
	oldRedisEnabled := common.RedisEnabled
	model.DB, model.LOG_DB, common.RedisEnabled = db, db, false
	t.Cleanup(func() { model.DB, model.LOG_DB, common.RedisEnabled = oldDB, oldLogDB, oldRedisEnabled })
	setting := operation_setting.GetRiskSetting()
	oldSetting := *setting
	*setting = operation_setting.RiskSetting{Enabled: true, AutoBanEnabled: false, ScanIntervalMinutes: 5, LookbackDays: 7, AutoBanScore: 90, MinimumCategories: 2}
	t.Cleanup(func() { *setting = oldSetting })
	now := time.Now().Unix()
	users := []model.User{{Username: "ignored-a", AffCode: "ignored-a", Role: 1, Status: common.UserStatusEnabled, CreatedAt: now, AuthVersion: 1}, {Username: "ignored-b", AffCode: "ignored-b", Role: 1, Status: common.UserStatusEnabled, CreatedAt: now, AuthVersion: 1}}
	require.NoError(t, db.Create(&users).Error)
	for _, user := range users {
		for range 10 {
			require.NoError(t, db.Create(&model.Log{UserId: user.Id, Type: model.LogTypeConsume, Ip: "203.0.113.10", CreatedAt: now}).Error)
		}
	}
	_, err = RunRiskScan(context.Background())
	require.NoError(t, err)
	var riskCase model.RiskCase
	require.NoError(t, db.First(&riskCase).Error)
	require.NoError(t, model.ResolveRiskCase(riskCase.ID, 1, model.RiskCaseStatusIgnored, "false positive"))
	setting.AutoBanEnabled = true
	_, err = RunRiskScan(context.Background())
	require.NoError(t, err)
	var statuses []int
	require.NoError(t, db.Model(&model.User{}).Order("id").Pluck("status", &statuses).Error)
	assert.Equal(t, []int{common.UserStatusEnabled, common.UserStatusEnabled}, statuses)
}
