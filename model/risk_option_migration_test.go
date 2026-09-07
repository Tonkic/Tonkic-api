package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateLegacyRiskSettingOptionExpandsDottedKeys(t *testing.T) {
	db := useFrontendOptionMigrationDB(t)
	require.NoError(t, db.Create(&Option{
		Key: legacyRiskSettingOptionKey,
		Value: "map[auto_ban_enabled:true auto_ban_score:100 enabled:true " +
			"lookback_days:7 minimum_categories:2 scan_interval_minutes:30]",
	}).Error)

	require.NoError(t, MigrateLegacyRiskSettingOption())

	requireOptionMissing(t, db, legacyRiskSettingOptionKey)
	expected := map[string]string{
		"enabled": "true", "auto_ban_enabled": "true",
		"scan_interval_minutes": "30", "lookback_days": "7",
		"auto_ban_score": "100", "minimum_categories": "2",
	}
	for field, value := range expected {
		assert.Equal(t, value, requireOptionValue(t, db, legacyRiskSettingOptionKey+"."+field))
	}

	require.NoError(t, MigrateLegacyRiskSettingOption())
}

func TestMigrateLegacyRiskSettingOptionKeepsExistingDottedValues(t *testing.T) {
	db := useFrontendOptionMigrationDB(t)
	require.NoError(t, db.Create(&[]Option{
		{Key: legacyRiskSettingOptionKey, Value: "map[auto_ban_enabled:true auto_ban_score:100 enabled:true lookback_days:7 minimum_categories:2 scan_interval_minutes:30]"},
		{Key: "risk_setting.enabled", Value: "false"},
	}).Error)

	require.NoError(t, MigrateLegacyRiskSettingOption())

	assert.Equal(t, "false", requireOptionValue(t, db, "risk_setting.enabled"))
	requireOptionMissing(t, db, legacyRiskSettingOptionKey)
}

func TestMigrateLegacyRiskSettingOptionPreservesMalformedValue(t *testing.T) {
	db := useFrontendOptionMigrationDB(t)
	require.NoError(t, db.Create(&Option{Key: legacyRiskSettingOptionKey, Value: "invalid"}).Error)

	err := MigrateLegacyRiskSettingOption()

	require.Error(t, err)
	assert.Equal(t, "invalid", requireOptionValue(t, db, legacyRiskSettingOptionKey))
}
