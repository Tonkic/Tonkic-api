package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

type riskEvidence struct {
	SharedIP       string `json:"shared_ip"`
	RequestCount   int    `json:"request_count"`
	RecentRegister bool   `json:"recent_register"`
	Invited        bool   `json:"invited"`
	Compensated    bool   `json:"compensated"`
	NoTopup        bool   `json:"no_topup"`
}

type RiskScanSummary struct {
	GroupsDetected int `json:"groups_detected"`
	CasesUpdated   int `json:"cases_updated"`
	UsersBanned    int `json:"users_banned"`
}

// RunRiskScan groups recent API activity by trusted client IP. An IP is only a
// candidate generator: automatic action additionally requires registration or
// invitation/compensation evidence, request volume, no successful top-up and
// the configured multi-category threshold.
func RunRiskScan(ctx context.Context) (RiskScanSummary, error) {
	setting := operation_setting.GetRiskSetting()
	summary := RiskScanSummary{}
	if !setting.Enabled {
		return summary, nil
	}
	cutoff := time.Now().Add(-time.Duration(setting.LookbackDays) * 24 * time.Hour).Unix()
	type aggregate struct {
		IP      string
		UserID  int
		Count   int
		FirstAt int64
		LastAt  int64
	}
	rows := make([]aggregate, 0)
	if err := model.LOG_DB.Model(&model.Log{}).
		Select("ip, user_id, COUNT(*) AS count, MIN(created_at) AS first_at, MAX(created_at) AS last_at").
		Where("created_at >= ? AND type IN ? AND ip <> '' AND user_id > 0", cutoff, []int{model.LogTypeConsume, model.LogTypeError}).
		Group("ip, user_id").Having("COUNT(*) >= ?", 3).Scan(&rows).Error; err != nil {
		return summary, err
	}
	byIP := make(map[string][]aggregate)
	for _, row := range rows {
		byIP[row.IP] = append(byIP[row.IP], row)
	}
	for ip, members := range byIP {
		if err := ctx.Err(); err != nil {
			return summary, err
		}
		ipAllowed, err := model.RiskValueAllowed("ip", ip, common.GetTimestamp())
		if err != nil {
			return summary, err
		}
		if len(members) < 2 || ipAllowed {
			continue
		}
		summary.GroupsDetected++
		ids := make([]int, 0, len(members))
		for _, member := range members {
			ids = append(ids, member.UserID)
		}
		var users []model.User
		if err := model.DB.Where("id IN ?", ids).Find(&users).Error; err != nil {
			return summary, err
		}
		topup, err := model.SuccessfulTopupUsers(ids)
		if err != nil {
			return summary, err
		}
		compensated, err := model.CompensationUsers(ids)
		if err != nil {
			return summary, err
		}
		userByID := make(map[int]model.User)
		for _, user := range users {
			userByID[user.Id] = user
		}
		caseUsers := make([]model.RiskCaseUser, 0, len(members))
		totalScore, categories := 0, 1
		firstSeen, lastSeen := members[0].FirstAt, members[0].LastAt
		hasRegistration, hasBenefit := false, false
		for _, member := range members {
			user := userByID[member.UserID]
			recent := user.CreatedAt >= cutoff
			invited := user.InviterId > 0
			hasRegistration = hasRegistration || recent || invited
			hasBenefit = hasBenefit || compensated[user.Id]
			score := 15
			if member.Count >= 10 {
				score += 10
			}
			if recent {
				score += 10
			}
			if invited {
				score += 25
			}
			if compensated[user.Id] {
				score += 15
			}
			if !topup[user.Id] {
				score += 10
			} else {
				score -= 20
			}
			evidence, _ := common.Marshal(riskEvidence{SharedIP: ip, RequestCount: member.Count, RecentRegister: recent, Invited: invited, Compensated: compensated[user.Id], NoTopup: !topup[user.Id]})
			caseUsers = append(caseUsers, model.RiskCaseUser{UserID: user.Id, Score: score, EvidenceJSON: string(evidence)})
			totalScore += score
			if member.FirstAt < firstSeen {
				firstSeen = member.FirstAt
			}
			if member.LastAt > lastSeen {
				lastSeen = member.LastAt
			}
		}
		if hasRegistration {
			categories++
		}
		if hasBenefit {
			categories++
		}
		if categories < 2 || totalScore < 40 {
			continue
		}
		sort.Ints(ids)
		// The signature must survive process restarts. It excludes the raw IP and
		// hashes only sorted internal user IDs.
		signature := common.Sha1([]byte(fmt.Sprintf("users:%v", ids)))
		reasons := []string{"shared API IP"}
		if hasRegistration {
			reasons = append(reasons, "rapid registration or invitation")
		}
		if hasBenefit {
			reasons = append(reasons, "subsidy use without successful top-up")
		}
		riskCase := &model.RiskCase{Signature: signature, Status: model.RiskCaseStatusOpen, Score: totalScore, Categories: categories, ReasonSummary: strings.Join(reasons, "; "), FirstSeen: firstSeen, LastSeen: lastSeen}
		skipped, err := model.UpsertRiskCase(riskCase, caseUsers)
		if err != nil {
			return summary, err
		}
		if skipped {
			continue
		}
		summary.CasesUpdated++
		if !setting.AutoBanEnabled || totalScore < setting.AutoBanScore || categories < setting.MinimumCategories {
			continue
		}
		protected := false
		for _, id := range ids {
			allowed, err := model.RiskValueAllowed("user", fmt.Sprint(id), common.GetTimestamp())
			if err != nil {
				return summary, err
			}
			if topup[id] || allowed || userByID[id].Role >= common.RoleAdminUser {
				protected = true
				break
			}
		}
		if protected {
			continue
		}
		if err := model.ApplyRiskBanCase(riskCase.ID, ids, 0, "automatically banned by risk policy"); err != nil {
			return summary, err
		}
		for _, id := range ids {
			if userByID[id].Status != common.UserStatusRiskBanned {
				summary.UsersBanned++
			}
		}
	}
	return summary, nil
}
