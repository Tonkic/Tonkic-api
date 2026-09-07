package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

const legacyRiskSettingOptionKey = "risk_setting"

var riskSettingOptionKeys = []string{
	"enabled",
	"auto_ban_enabled",
	"scan_interval_minutes",
	"lookback_days",
	"auto_ban_score",
	"minimum_categories",
}

// MigrateLegacyRiskSettingOption repairs the rc.24.4 form payload, which was
// persisted as one Go-formatted map instead of six dotted option keys.
func MigrateLegacyRiskSettingOption() error {
	if DB == nil {
		return errors.New("database is not initialized")
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		var legacy Option
		if err := tx.Where(&Option{Key: legacyRiskSettingOptionKey}).First(&legacy).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return fmt.Errorf("read legacy risk setting: %w", err)
		}

		values, err := parseLegacyRiskSettingValue(legacy.Value)
		if err != nil {
			return fmt.Errorf("parse legacy risk setting: %w", err)
		}
		for _, field := range riskSettingOptionKeys {
			key := legacyRiskSettingOptionKey + "." + field
			var target Option
			findErr := tx.Where(&Option{Key: key}).First(&target).Error
			if findErr == nil {
				continue
			}
			if !errors.Is(findErr, gorm.ErrRecordNotFound) {
				return fmt.Errorf("read target option %s: %w", key, findErr)
			}
			if err := tx.Create(&Option{Key: key, Value: values[field]}).Error; err != nil {
				return fmt.Errorf("write target option %s: %w", key, err)
			}
		}
		if err := tx.Delete(&legacy).Error; err != nil {
			return fmt.Errorf("delete legacy risk setting: %w", err)
		}
		return nil
	})
}

func parseLegacyRiskSettingValue(value string) (map[string]string, error) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "map[") || !strings.HasSuffix(value, "]") {
		return nil, errors.New("unsupported legacy value format")
	}

	parsed := make(map[string]string, len(riskSettingOptionKeys))
	for _, item := range strings.Fields(strings.TrimSuffix(strings.TrimPrefix(value, "map["), "]")) {
		key, raw, found := strings.Cut(item, ":")
		if !found {
			return nil, fmt.Errorf("invalid item %q", item)
		}
		parsed[key] = raw
	}
	for _, key := range riskSettingOptionKeys {
		raw, found := parsed[key]
		if !found {
			return nil, fmt.Errorf("missing %s", key)
		}
		if key == "enabled" || key == "auto_ban_enabled" {
			if _, err := strconv.ParseBool(raw); err != nil {
				return nil, fmt.Errorf("invalid %s: %w", key, err)
			}
			continue
		}
		if _, err := strconv.Atoi(raw); err != nil {
			return nil, fmt.Errorf("invalid %s: %w", key, err)
		}
	}
	return parsed, nil
}
