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
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}, &model.SecurityEvent{}, &model.TopUp{}, &model.CompensationClaim{}, &model.RiskCase{}, &model.RiskCaseUser{}, &model.RiskAllowlist{}))
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
			require.NoError(t, db.Create(&model.Log{UserId: user.Id, Type: model.LogTypeConsume, Ip: "8.8.8.9", CreatedAt: now}).Error)
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
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}, &model.SecurityEvent{}, &model.TopUp{}, &model.CompensationClaim{}, &model.RiskCase{}, &model.RiskCaseUser{}, &model.RiskAllowlist{}))
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
			require.NoError(t, db.Create(&model.Log{UserId: user.Id, Type: model.LogTypeConsume, Ip: "8.8.4.10", CreatedAt: now}).Error)
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

func TestRunRiskScanUsesRegistrationAndUserAgentEvidence(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:risk-registration?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}, &model.SecurityEvent{}, &model.TopUp{}, &model.CompensationClaim{}, &model.RiskCase{}, &model.RiskCaseUser{}, &model.RiskAllowlist{}))
	oldDB, oldLogDB := model.DB, model.LOG_DB
	model.DB, model.LOG_DB = db, db
	t.Cleanup(func() { model.DB, model.LOG_DB = oldDB, oldLogDB })
	setting := operation_setting.GetRiskSetting()
	oldSetting := *setting
	*setting = operation_setting.RiskSetting{Enabled: true, AutoBanEnabled: false, ScanIntervalMinutes: 5, LookbackDays: 7, AutoBanScore: 100, MinimumCategories: 1}
	t.Cleanup(func() { *setting = oldSetting })
	now := time.Now().Unix()
	users := []model.User{{Username: "register-a", AffCode: "register-a", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, CreatedAt: now, AuthVersion: 1}, {Username: "register-b", AffCode: "register-b", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, CreatedAt: now, AuthVersion: 1}}
	require.NoError(t, db.Create(&users).Error)
	for _, user := range users {
		require.NoError(t, db.Create(&model.SecurityEvent{UserID: user.Id, EventType: "register", IP: "1.1.1.9", UserAgentHash: "same-browser", CreatedAt: now}).Error)
		require.NoError(t, db.Create(&model.SecurityEvent{UserID: user.Id, EventType: "login", IP: "1.1.1.9", UserAgentHash: "same-browser", CreatedAt: now + 1}).Error)
	}
	summary, err := RunRiskScan(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, summary.CasesUpdated)
	var members []model.RiskCaseUser
	require.NoError(t, db.Order("user_id").Find(&members).Error)
	assert.Len(t, members, 2)
	assert.Contains(t, members[0].EvidenceJSON, "registration_count")
	assert.Contains(t, members[0].EvidenceJSON, "shared_user_agent")
}

func TestNormalizeRiskAssociationIP(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		ok    bool
	}{
		{name: "public IPv4", input: " 8.8.8.8 ", want: "8.8.8.8", ok: true},
		{name: "public IPv4 with port", input: "8.8.4.4:443", want: "8.8.4.4", ok: true},
		{name: "public IPv6", input: "2001:4860:4860::8888", want: "2001:4860:4860::8888", ok: true},
		{name: "IPv4 loopback", input: "127.0.0.1"},
		{name: "IPv6 loopback", input: "::1"},
		{name: "RFC1918 10", input: "10.1.2.3"},
		{name: "RFC1918 172", input: "172.16.0.1"},
		{name: "RFC1918 192", input: "192.168.1.1"},
		{name: "IPv6 ULA", input: "fc00::1"},
		{name: "IPv4 link local", input: "169.254.1.1"},
		{name: "IPv6 link local", input: "fe80::1"},
		{name: "IPv4 unspecified", input: "0.0.0.0"},
		{name: "IPv6 unspecified", input: "::"},
		{name: "CGNAT", input: "100.64.0.1"},
		{name: "IPv4 documentation", input: "192.0.2.1"},
		{name: "IPv6 documentation", input: "2001:db8::1"},
		{name: "mapped loopback", input: "::ffff:127.0.0.1"},
		{name: "invalid", input: "not-an-ip"},
		{name: "empty", input: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := normalizeRiskAssociationIP(tt.input)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRunRiskScanIgnoresNonPublicAssociationIPs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:risk-private-ips?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}, &model.SecurityEvent{}, &model.TopUp{}, &model.CompensationClaim{}, &model.RiskCase{}, &model.RiskCaseUser{}, &model.RiskAllowlist{}))
	oldDB, oldLogDB := model.DB, model.LOG_DB
	model.DB, model.LOG_DB = db, db
	t.Cleanup(func() { model.DB, model.LOG_DB = oldDB, oldLogDB })
	setting := operation_setting.GetRiskSetting()
	oldSetting := *setting
	*setting = operation_setting.RiskSetting{Enabled: true, AutoBanEnabled: true, ScanIntervalMinutes: 5, LookbackDays: 7, AutoBanScore: 100, MinimumCategories: 1}
	t.Cleanup(func() { *setting = oldSetting })
	now := time.Now().Unix()
	users := []model.User{{Username: "local-a", AffCode: "local-a", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, CreatedAt: now, AuthVersion: 1}, {Username: "local-b", AffCode: "local-b", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, CreatedAt: now, AuthVersion: 1}}
	require.NoError(t, db.Create(&users).Error)
	for _, user := range users {
		for range 10 {
			require.NoError(t, db.Create(&model.Log{UserId: user.Id, Type: model.LogTypeConsume, Ip: "127.0.0.1", CreatedAt: now}).Error)
		}
		require.NoError(t, db.Create(&model.SecurityEvent{UserID: user.Id, EventType: "login", IP: "192.168.1.10", UserAgentHash: "shared-local-browser", CreatedAt: now}).Error)
	}
	summary, err := RunRiskScan(context.Background())
	require.NoError(t, err)
	assert.Zero(t, summary.GroupsDetected)
	assert.Zero(t, summary.CasesUpdated)
	assert.Zero(t, summary.UsersBanned)
	var caseCount int64
	require.NoError(t, db.Model(&model.RiskCase{}).Count(&caseCount).Error)
	assert.Zero(t, caseCount)
}

func TestRunRiskScanRetainsDeletedAnchorAndBansLiveLinkedUser(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:risk-deleted-anchor?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}, &model.SecurityEvent{}, &model.TopUp{}, &model.CompensationClaim{}, &model.RiskCase{}, &model.RiskCaseUser{}, &model.RiskAllowlist{}, &model.UserSession{}, &model.Token{}))
	oldDB, oldLogDB, oldRedis := model.DB, model.LOG_DB, common.RedisEnabled
	model.DB, model.LOG_DB, common.RedisEnabled = db, db, false
	t.Cleanup(func() { model.DB, model.LOG_DB, common.RedisEnabled = oldDB, oldLogDB, oldRedis })
	setting := operation_setting.GetRiskSetting()
	oldSetting := *setting
	*setting = operation_setting.RiskSetting{Enabled: true, AutoBanEnabled: true, ScanIntervalMinutes: 5, LookbackDays: 7, AutoBanScore: 100, MinimumCategories: 2}
	t.Cleanup(func() { *setting = oldSetting })
	now := time.Now().Unix()
	users := []model.User{
		{Username: "deleted-anchor", AffCode: "deleted-anchor", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, CreatedAt: now - 3600, AuthVersion: 1},
		{Username: "new-linked", AffCode: "new-linked", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, CreatedAt: now, AuthVersion: 1},
	}
	require.NoError(t, db.Create(&users).Error)
	users[1].InviterId = users[0].Id
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", users[1].Id).Update("inviter_id", users[0].Id).Error)
	require.NoError(t, db.Delete(&users[0]).Error)
	for _, user := range users {
		for range 10 {
			require.NoError(t, db.Create(&model.Log{UserId: user.Id, Type: model.LogTypeConsume, Ip: "8.8.8.77", CreatedAt: now}).Error)
		}
		require.NoError(t, db.Create(&model.SecurityEvent{UserID: user.Id, EventType: "register", IP: "8.8.8.77", UserAgentHash: "same-browser", CreatedAt: now}).Error)
		require.NoError(t, db.Create(&model.SecurityEvent{UserID: user.Id, EventType: "login", IP: "8.8.8.77", UserAgentHash: "same-browser", CreatedAt: now + 1}).Error)
	}

	summary, err := RunRiskScan(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, summary.CasesUpdated)
	assert.Equal(t, 1, summary.UsersBanned)
	var members []model.RiskCaseUser
	require.NoError(t, db.Order("user_id").Find(&members).Error)
	assert.Len(t, members, 2)
	var live model.User
	require.NoError(t, db.Unscoped().First(&live, users[1].Id).Error)
	assert.Equal(t, common.UserStatusRiskBanned, live.Status)
	var anchor model.User
	require.NoError(t, db.Unscoped().First(&anchor, users[0].Id).Error)
	assert.NotZero(t, anchor.DeletedAt)
}
