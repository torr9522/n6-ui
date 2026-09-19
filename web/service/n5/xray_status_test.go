package n5

import (
	"testing"
	"x-ui/database"
	"x-ui/database/model"
	n5model "x-ui/database/model/n5"
)

func TestXrayStatusServiceReflectsLatestHistoryAndCounts(t *testing.T) {
	initTestDB(t)

	egressSvc := &EgressService{}
	if _, err := egressSvc.Create(&n5model.Egress{
		Name:         "status-egress",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: freedomOutboundJSON(),
	}); err != nil {
		t.Fatalf("create egress failed: %v", err)
	}

	if err := database.GetDB().Create(&model.Setting{
		Key:   "n5XrayExtensionEnable",
		Value: "true",
	}).Error; err != nil {
		t.Fatalf("create setting failed: %v", err)
	}

	history := &n5model.XrayConfigHistory{
		Source:              "n5-merge",
		BaseConfigHash:      "base-hash",
		ExtensionConfigHash: "extension-hash",
		ConfigHash:          "config-hash",
		ApplyStatus:         "applied",
	}
	if err := database.GetDB().Create(history).Error; err != nil {
		t.Fatalf("create xray config history failed: %v", err)
	}

	statusSvc := &XrayStatusService{}
	status, err := statusSvc.GetStatus()
	if err != nil {
		t.Fatalf("get status failed: %v", err)
	}
	if !status.Enabled {
		t.Fatal("expected enabled status")
	}
	if status.OutboundCount != 1 {
		t.Fatalf("unexpected outbound count: %d", status.OutboundCount)
	}
	if status.RoutingCount != 0 {
		t.Fatalf("unexpected routing count: %d", status.RoutingCount)
	}
	if status.Hash != "config-hash" {
		t.Fatalf("unexpected status hash: %s", status.Hash)
	}
	if status.LastApply == nil {
		t.Fatal("expected last apply to be populated")
	}
	if status.LastApply.Status != "applied" || status.LastApply.BaseConfigHash != "base-hash" || status.LastApply.ExtensionConfigHash != "extension-hash" {
		t.Fatalf("unexpected last apply payload: %#v", status.LastApply)
	}
}
