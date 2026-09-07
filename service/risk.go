package service

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

type riskEvidence struct {
	SharedIP          string `json:"shared_ip"`
	RequestCount      int    `json:"request_count"`
	RegistrationCount int    `json:"registration_count"`
	LoginCount        int    `json:"login_count"`
	APIEventCount     int    `json:"api_event_count"`
	SharedUserAgent   bool   `json:"shared_user_agent"`
	RecentRegister    bool   `json:"recent_register"`
	Invited           bool   `json:"invited"`
	Compensated       bool   `json:"compensated"`
	NoTopup           bool   `json:"no_topup"`
}

type RiskScanSummary struct {
	GroupsDetected int `json:"groups_detected"`
	CasesUpdated   int `json:"cases_updated"`
	UsersBanned    int `json:"users_banned"`
}

// normalizeRiskAssociationIP returns only globally routable client addresses.
// Local, private and special-purpose addresses remain in the source event
// timeline, but must never be used to associate identities across accounts.
func normalizeRiskAssociationIP(value string) (string, bool) {
	normalized := model.NormalizeSecurityIP(value)
	parsed := net.ParseIP(normalized)
	if parsed == nil || common.IsPrivateIP(parsed) {
		return "", false
	}
	return normalized, true
}

// RunRiskScan groups recent API activity and registration/login events by
// trusted client IP. Shared registration/login user-agent hashes are added as
// supporting evidence for the associated user group.
func RunRiskScan(ctx context.Context) (RiskScanSummary, error) {
	setting := operation_setting.GetRiskSetting()
	summary := RiskScanSummary{}
	if !setting.Enabled {
		return summary, nil
	}
	cutoff := time.Now().Add(-time.Duration(setting.LookbackDays) * 24 * time.Hour).Unix()
	type aggregate struct {
		IP                string
		UserID            int
		Count             int
		APIRequestCount   int
		RegistrationCount int
		LoginCount        int
		APIEventCount     int
		FirstAt           int64
		LastAt            int64
		UserAgentHashes   map[string]bool
	}
	byIPUsers := make(map[string]map[int]*aggregate)
	ensureAggregate := func(ip string, userID int) *aggregate {
		members := byIPUsers[ip]
		if members == nil {
			members = make(map[int]*aggregate)
			byIPUsers[ip] = members
		}
		member := members[userID]
		if member == nil {
			member = &aggregate{IP: ip, UserID: userID, UserAgentHashes: make(map[string]bool)}
			members[userID] = member
		}
		return member
	}
	type apiAggregate struct {
		IP      string
		UserID  int
		Count   int
		FirstAt int64
		LastAt  int64
	}
	rows := make([]apiAggregate, 0)
	if err := model.LOG_DB.Model(&model.Log{}).
		Select("ip, user_id, COUNT(*) AS count, MIN(created_at) AS first_at, MAX(created_at) AS last_at").
		Where("created_at >= ? AND type IN ? AND ip <> '' AND user_id > 0", cutoff, []int{model.LogTypeConsume, model.LogTypeError}).
		Group("ip, user_id").Having("COUNT(*) >= ?", 3).Scan(&rows).Error; err != nil {
		return summary, err
	}
	for _, row := range rows {
		ip, usable := normalizeRiskAssociationIP(row.IP)
		if !usable {
			continue
		}
		member := ensureAggregate(ip, row.UserID)
		member.Count += row.Count
		member.APIRequestCount += row.Count
		member.FirstAt = row.FirstAt
		member.LastAt = row.LastAt
	}
	var securityEvents []model.SecurityEvent
	if err := model.DB.Where("created_at >= ? AND event_type IN ? AND user_id > 0", cutoff, []string{"register", "login", "api_request", "request"}).Find(&securityEvents).Error; err != nil {
		return summary, err
	}
	for _, event := range securityEvents {
		ip, usable := normalizeRiskAssociationIP(event.IP)
		if !usable {
			continue
		}
		member := ensureAggregate(ip, event.UserID)
		member.Count++
		if event.EventType == "register" {
			member.RegistrationCount++
		}
		if event.EventType == "login" {
			member.LoginCount++
		}
		if event.EventType == "api_request" || event.EventType == "request" {
			member.APIEventCount++
			member.APIRequestCount++
		}
		if member.FirstAt == 0 || event.CreatedAt < member.FirstAt {
			member.FirstAt = event.CreatedAt
		}
		if event.CreatedAt > member.LastAt {
			member.LastAt = event.CreatedAt
		}
		if event.UserAgentHash != "" {
			member.UserAgentHashes[event.UserAgentHash] = true
		}
	}
	byIP := make(map[string][]aggregate)
	for ip, members := range byIPUsers {
		for _, member := range members {
			byIP[ip] = append(byIP[ip], *member)
		}
	}
	for ip, members := range byIP {
		if err := ctx.Err(); err != nil {
			return summary, err
		}
		if strings.TrimSpace(ip) == "" {
			// Missing client IP is retained in the event timeline but never
			// treated as a shared identity signal.
			continue
		}
		ipAllowed, err := model.RiskValueAllowed("ip", ip, common.GetTimestamp())
		if err != nil {
			return summary, err
		}
		if len(members) < 2 || ipAllowed {
			continue
		}
		sharedUserAgents := make(map[string]bool)
		for _, left := range members {
			for hash := range left.UserAgentHashes {
				for _, right := range members {
					if left.UserID != right.UserID && right.UserAgentHashes[hash] {
						sharedUserAgents[hash] = true
					}
				}
			}
		}
		hasSharedUserAgent := len(sharedUserAgents) > 0
		summary.GroupsDetected++
		ids := make([]int, 0, len(members))
		for _, member := range members {
			ids = append(ids, member.UserID)
		}
		// Keep soft-deleted accounts as evidence anchors.  A deleted account
		// must still connect a newly-created account to the historical risk
		// cluster, even though it is not eligible for another status change.
		var users []model.User
		if err := model.DB.Unscoped().Where("id IN ?", ids).Find(&users).Error; err != nil {
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
		validMembers := make([]aggregate, 0, len(members))
		for _, member := range members {
			if _, ok := userByID[member.UserID]; ok {
				validMembers = append(validMembers, member)
			}
		}
		members = validMembers
		if len(members) < 2 {
			continue
		}
		ids = ids[:0]
		for _, member := range members {
			ids = append(ids, member.UserID)
		}
		for _, member := range members {
			user := userByID[member.UserID]
			recent := user.CreatedAt >= cutoff
			invited := user.InviterId > 0
			hasRegistration = hasRegistration || member.RegistrationCount > 0 || recent || invited
			hasBenefit = hasBenefit || compensated[user.Id]
			score := 15
			if member.APIRequestCount >= 10 {
				score += 10
			}
			if recent {
				score += 10
			}
			if member.RegistrationCount > 0 {
				score += 25
			}
			if invited {
				score += 25
			}
			if hasSharedUserAgent {
				score += 20
			}
			if compensated[user.Id] {
				score += 15
			}
			if !topup[user.Id] {
				score += 10
			} else {
				score -= 20
			}
			evidence, _ := common.Marshal(riskEvidence{SharedIP: ip, RequestCount: member.APIRequestCount, RegistrationCount: member.RegistrationCount, LoginCount: member.LoginCount, APIEventCount: member.APIEventCount, SharedUserAgent: hasSharedUserAgent, RecentRegister: recent, Invited: invited, Compensated: compensated[user.Id], NoTopup: !topup[user.Id]})
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
		if hasSharedUserAgent {
			categories++
		}
		if hasBenefit {
			categories++
		}
		if categories < 1 || totalScore < 40 {
			continue
		}
		sort.Ints(ids)
		// The signature must survive process restarts. It excludes the raw IP and
		// hashes only sorted internal user IDs.
		signature := common.Sha1([]byte(fmt.Sprintf("users:%v", ids)))
		reasons := []string{"shared client IP"}
		if hasRegistration {
			reasons = append(reasons, "rapid registration or invitation")
		}
		if hasSharedUserAgent {
			reasons = append(reasons, "shared registration/login user-agent fingerprint")
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
		// Shared IP is the primary association signal and can form a case by
		// itself; retain the configured score floor before automatic action.
		if !setting.AutoBanEnabled || totalScore < setting.AutoBanScore || categories < setting.MinimumCategories {
			continue
		}
		protected := false
		for _, id := range ids {
			allowed, err := model.RiskValueAllowed("user", fmt.Sprint(id), common.GetTimestamp())
			if err != nil {
				return summary, err
			}
			// A protected member protects the entire association. This keeps
			// automatic action all-or-none instead of banning only one account.
			if topup[id] || allowed || userByID[id].Role >= common.RoleAdminUser {
				protected = true
				break
			}
		}
		if protected {
			continue
		}
		// Apply the ban only to live accounts. Deleted accounts remain in the
		// case for auditability and continue to contribute evidence/score.
		eligibleIDs := make([]int, 0, len(ids))
		for _, id := range ids {
			if user, ok := userByID[id]; ok && user.DeletedAt.Valid {
				continue
			}
			eligibleIDs = append(eligibleIDs, id)
		}
		if len(eligibleIDs) == 0 {
			continue
		}
		if err := model.ApplyRiskBanCase(riskCase.ID, eligibleIDs, 0, "automatically banned by risk policy"); err != nil {
			return summary, err
		}
		for _, id := range eligibleIDs {
			if userByID[id].Status != common.UserStatusRiskBanned {
				summary.UsersBanned++
			}
		}
	}
	return summary, nil
}
