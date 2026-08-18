package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type RiskSetting struct {
	Enabled             bool `json:"enabled"`
	AutoBanEnabled      bool `json:"auto_ban_enabled"`
	ScanIntervalMinutes int  `json:"scan_interval_minutes"`
	LookbackDays        int  `json:"lookback_days"`
	AutoBanScore        int  `json:"auto_ban_score"`
	MinimumCategories   int  `json:"minimum_categories"`
}

var riskSetting = RiskSetting{
	Enabled: false, AutoBanEnabled: false, ScanIntervalMinutes: 5,
	LookbackDays: 7, AutoBanScore: 100, MinimumCategories: 2,
}

func init() { config.GlobalConfig.Register("risk_setting", &riskSetting) }

func GetRiskSetting() *RiskSetting {
	if riskSetting.ScanIntervalMinutes < 1 {
		riskSetting.ScanIntervalMinutes = 5
	}
	if riskSetting.LookbackDays < 1 || riskSetting.LookbackDays > 30 {
		riskSetting.LookbackDays = 7
	}
	if riskSetting.AutoBanScore < 90 {
		riskSetting.AutoBanScore = 100
	}
	if riskSetting.MinimumCategories < 2 {
		riskSetting.MinimumCategories = 2
	}
	return &riskSetting
}
