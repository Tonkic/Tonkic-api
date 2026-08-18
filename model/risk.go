package model

import (
	"errors"
	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	RiskCaseStatusOpen    = "open"
	RiskCaseStatusIgnored = "ignored"
	RiskCaseStatusBanned  = "banned"
)

type RiskCase struct {
	ID            int            `json:"id"`
	Signature     string         `json:"signature" gorm:"size:191;uniqueIndex"`
	Status        string         `json:"status" gorm:"size:16;index"`
	Score         int            `json:"score" gorm:"index"`
	Categories    int            `json:"categories"`
	ReasonSummary string         `json:"reason_summary" gorm:"type:text"`
	FirstSeen     int64          `json:"first_seen" gorm:"index"`
	LastSeen      int64          `json:"last_seen" gorm:"index"`
	CreatedAt     int64          `json:"created_at" gorm:"autoCreateTime"`
	ReviewedBy    int            `json:"reviewed_by"`
	ReviewedAt    int64          `json:"reviewed_at"`
	Resolution    string         `json:"resolution" gorm:"type:text"`
	Users         []RiskCaseUser `json:"users,omitempty" gorm:"foreignKey:CaseID"`
}

type RiskCaseUser struct {
	CaseID       int    `json:"case_id" gorm:"primaryKey"`
	UserID       int    `json:"user_id" gorm:"primaryKey;index"`
	Username     string `json:"username" gorm:"-"`
	Score        int    `json:"score"`
	EvidenceJSON string `json:"evidence_json" gorm:"type:text"`
	BannedByCase bool   `json:"banned_by_case"`
}

type RiskAllowlist struct {
	ID        int    `json:"id"`
	Type      string `json:"type" gorm:"size:24;uniqueIndex:idx_risk_allowlist_value"`
	Value     string `json:"value" gorm:"size:191;uniqueIndex:idx_risk_allowlist_value"`
	Reason    string `json:"reason" gorm:"size:255"`
	ExpiresAt int64  `json:"expires_at" gorm:"index"`
	CreatedBy int    `json:"created_by"`
	CreatedAt int64  `json:"created_at" gorm:"autoCreateTime"`
}

type SecurityEvent struct {
	ID            int    `json:"id"`
	UserID        int    `json:"user_id" gorm:"index:idx_security_user_time,priority:1"`
	EventType     string `json:"event_type" gorm:"size:16;index"`
	IP            string `json:"ip" gorm:"size:64;index:idx_security_ip_time,priority:1"`
	UserAgentHash string `json:"user_agent_hash" gorm:"size:64"`
	RequestID     string `json:"request_id" gorm:"size:64"`
	CreatedAt     int64  `json:"created_at" gorm:"index:idx_security_user_time,priority:2;index:idx_security_ip_time,priority:2"`
}

func RecordSecurityEvent(event *SecurityEvent) error { return DB.Create(event).Error }

func UpsertRiskCase(riskCase *RiskCase, users []RiskCaseUser) (bool, error) {
	skipped := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		var existing RiskCase
		err := tx.Where("signature = ?", riskCase.Signature).First(&existing).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if err == nil {
			if existing.Status == RiskCaseStatusIgnored {
				skipped = true
				return nil
			}
			riskCase.ID = existing.ID
			riskCase.FirstSeen = existing.FirstSeen
			if err := tx.Model(&existing).Updates(map[string]any{
				"score": riskCase.Score, "categories": riskCase.Categories,
				"reason_summary": riskCase.ReasonSummary, "last_seen": riskCase.LastSeen,
			}).Error; err != nil {
				return err
			}
			if err := tx.Where("case_id = ?", existing.ID).Delete(&RiskCaseUser{}).Error; err != nil {
				return err
			}
		} else if err := tx.Create(riskCase).Error; err != nil {
			return err
		}
		for i := range users {
			users[i].CaseID = riskCase.ID
		}
		return tx.Create(&users).Error
	})
	return skipped, err
}

func ListRiskCases(status string, offset, limit int) ([]RiskCase, int64, error) {
	query := DB.Model(&RiskCase{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	cases := make([]RiskCase, 0)
	err := query.Preload("Users").Order("score DESC, last_seen DESC").Offset(offset).Limit(limit).Find(&cases).Error
	if err != nil {
		return nil, 0, err
	}
	for i := range cases {
		for j := range cases[i].Users {
			DB.Model(&User{}).Where("id = ?", cases[i].Users[j].UserID).Pluck("username", &cases[i].Users[j].Username)
		}
	}
	return cases, total, nil
}

func ResolveRiskCase(id, reviewer int, status, resolution string) error {
	return DB.Model(&RiskCase{}).Where("id = ?", id).Updates(map[string]any{
		"status": status, "reviewed_by": reviewer, "reviewed_at": common.GetTimestamp(), "resolution": resolution,
	}).Error
}

func GetRiskCase(id int) (*RiskCase, error) {
	var riskCase RiskCase
	if err := DB.Preload("Users").First(&riskCase, id).Error; err != nil {
		return nil, err
	}
	return &riskCase, nil
}

func ApplyRiskBanCase(caseID int, userIDs []int, reviewer int, resolution string) error {
	updated := make([]User, 0, len(userIDs))
	err := DB.Transaction(func(tx *gorm.DB) error {
		for _, userID := range userIDs {
			var user User
			if err := lockForUpdate(tx).First(&user, userID).Error; err != nil {
				return err
			}
			if user.Role >= common.RoleAdminUser {
				return errors.New("administrator accounts cannot be risk-banned")
			}
			if user.Status == common.UserStatusRiskBanned {
				continue
			}
			if user.Status != common.UserStatusEnabled {
				return errors.New("only enabled accounts can be risk-banned")
			}
			user.Status = common.UserStatusRiskBanned
			if err := user.UpdateWithTx(tx, false); err != nil {
				return err
			}
			if err := tx.Model(&RiskCaseUser{}).Where("case_id = ? AND user_id = ?", caseID, userID).Update("banned_by_case", true).Error; err != nil {
				return err
			}
			updated = append(updated, user)
		}
		return tx.Model(&RiskCase{}).Where("id = ?", caseID).Updates(map[string]any{"status": RiskCaseStatusBanned, "reviewed_by": reviewer, "reviewed_at": common.GetTimestamp(), "resolution": resolution}).Error
	})
	if err != nil {
		return err
	}
	for _, user := range updated {
		if err := updateUserCache(user); err != nil {
			return err
		}
		if _, err := RevokeAllUserSessions(user.Id, "risk_banned"); err != nil {
			return err
		}
		if err := InvalidateUserTokensCache(user.Id); err != nil {
			return err
		}
	}
	return nil
}

func RevertRiskBanCase(caseID int, reviewer int, resolution string) error {
	updated := make([]User, 0)
	err := DB.Transaction(func(tx *gorm.DB) error {
		var members []RiskCaseUser
		if err := tx.Where("case_id = ? AND banned_by_case = ?", caseID, true).Find(&members).Error; err != nil {
			return err
		}
		for _, member := range members {
			var user User
			if err := lockForUpdate(tx).First(&user, member.UserID).Error; err != nil {
				return err
			}
			if user.Status != common.UserStatusRiskBanned {
				continue
			}
			user.Status = common.UserStatusEnabled
			if err := user.UpdateWithTx(tx, false); err != nil {
				return err
			}
			updated = append(updated, user)
		}
		return tx.Model(&RiskCase{}).Where("id = ?", caseID).Updates(map[string]any{"status": RiskCaseStatusIgnored, "reviewed_by": reviewer, "reviewed_at": common.GetTimestamp(), "resolution": resolution}).Error
	})
	if err != nil {
		return err
	}
	for _, user := range updated {
		if err := updateUserCache(user); err != nil {
			return err
		}
		if _, err := RevokeAllUserSessions(user.Id, "risk_ban_reverted"); err != nil {
			return err
		}
		if err := InvalidateUserTokensCache(user.Id); err != nil {
			return err
		}
	}
	return nil
}

func RiskValueAllowed(kind, value string, now int64) (bool, error) {
	var count int64
	err := DB.Model(&RiskAllowlist{}).Where("type = ? AND value = ? AND (expires_at = 0 OR expires_at > ?)", kind, value, now).Count(&count).Error
	return count > 0, err
}

func SuccessfulTopupUsers(userIDs []int) (map[int]bool, error) {
	result := make(map[int]bool)
	if len(userIDs) == 0 {
		return result, nil
	}
	var ids []int
	if err := DB.Model(&TopUp{}).Where("user_id IN ? AND status = ?", userIDs, common.TopUpStatusSuccess).Distinct("user_id").Pluck("user_id", &ids).Error; err != nil {
		return nil, err
	}
	for _, id := range ids {
		result[id] = true
	}
	return result, nil
}

func CompensationUsers(userIDs []int) (map[int]bool, error) {
	result := make(map[int]bool)
	if len(userIDs) == 0 {
		return result, nil
	}
	var ids []int
	if err := DB.Model(&CompensationClaim{}).Where("user_id IN ?", userIDs).Distinct("user_id").Pluck("user_id", &ids).Error; err != nil {
		return nil, err
	}
	for _, id := range ids {
		result[id] = true
	}
	return result, nil
}
