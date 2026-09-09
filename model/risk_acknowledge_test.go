package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupRiskAcknowledgeTest(t *testing.T) {
	t.Helper()
	previousDB := DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&RiskCase{}, &RiskCaseUser{}); err != nil {
		t.Fatalf("migrate risk tables: %v", err)
	}
	DB = db
	t.Cleanup(func() { DB = previousDB })
}

func TestAcknowledgeRiskBanPreservesBanEvidence(t *testing.T) {
	setupRiskAcknowledgeTest(t)
	riskCase := RiskCase{
		Signature:  "auto-ban",
		Status:     RiskCaseStatusBanned,
		Resolution: "automatically banned by risk policy",
	}
	if err := DB.Create(&riskCase).Error; err != nil {
		t.Fatalf("create risk case: %v", err)
	}
	member := RiskCaseUser{CaseID: riskCase.ID, UserID: 42, BannedByCase: true}
	if err := DB.Create(&member).Error; err != nil {
		t.Fatalf("create case member: %v", err)
	}

	if err := AcknowledgeRiskBanCase(riskCase.ID, 7); err != nil {
		t.Fatalf("acknowledge risk ban: %v", err)
	}

	var got RiskCase
	if err := DB.Preload("Users").First(&got, riskCase.ID).Error; err != nil {
		t.Fatalf("load risk case: %v", err)
	}
	if got.Status != RiskCaseStatusAcknowledged {
		t.Fatalf("status = %q, want %q", got.Status, RiskCaseStatusAcknowledged)
	}
	if got.ReviewedBy != 7 || got.ReviewedAt == 0 {
		t.Fatalf("review metadata = (%d, %d), want reviewer 7 and timestamp", got.ReviewedBy, got.ReviewedAt)
	}
	if got.Resolution != "automatically banned by risk policy" {
		t.Fatalf("resolution changed to %q", got.Resolution)
	}
	if len(got.Users) != 1 || !got.Users[0].BannedByCase {
		t.Fatalf("ban evidence was not preserved: %+v", got.Users)
	}
}

func TestUpsertRiskCaseSkipsAcknowledgedHistory(t *testing.T) {
	setupRiskAcknowledgeTest(t)
	existing := RiskCase{
		Signature:     "archived-case",
		Status:        RiskCaseStatusAcknowledged,
		Score:         100,
		ReasonSummary: "original evidence",
	}
	if err := DB.Create(&existing).Error; err != nil {
		t.Fatalf("create risk case: %v", err)
	}
	member := RiskCaseUser{CaseID: existing.ID, UserID: 42, BannedByCase: true}
	if err := DB.Create(&member).Error; err != nil {
		t.Fatalf("create case member: %v", err)
	}

	incoming := RiskCase{Signature: existing.Signature, Status: RiskCaseStatusOpen, Score: 999, ReasonSummary: "new evidence"}
	skipped, err := UpsertRiskCase(&incoming, []RiskCaseUser{{UserID: 99}})
	if err != nil {
		t.Fatalf("upsert acknowledged case: %v", err)
	}
	if !skipped {
		t.Fatal("acknowledged case was not skipped")
	}

	var got RiskCase
	if err := DB.Preload("Users").First(&got, existing.ID).Error; err != nil {
		t.Fatalf("load risk case: %v", err)
	}
	if got.Score != 100 || got.ReasonSummary != "original evidence" || len(got.Users) != 1 || got.Users[0].UserID != 42 || !got.Users[0].BannedByCase {
		t.Fatalf("acknowledged history was modified: %+v", got)
	}
}
