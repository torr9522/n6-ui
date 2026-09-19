package service

import (
	"path/filepath"
	"testing"
	"x-ui/database"
	"x-ui/database/model"
	n5model "x-ui/database/model/n5"
)

func initInboundN5CleanupTestDB(t *testing.T) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "inbound-n5-cleanup.db")
	if err := database.InitDB(dbPath); err != nil {
		t.Fatalf("init db failed: %v", err)
	}
}

func createInboundCleanupTestInbound(t *testing.T, tag string, port int) *model.Inbound {
	t.Helper()
	inbound := &model.Inbound{
		UserId:         1,
		Remark:         tag,
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           port,
		Protocol:       model.Socks,
		Settings:       `{"auth":"noauth","udp":true,"ip":"127.0.0.1"}`,
		StreamSettings: `{}`,
		Tag:            tag,
		Sniffing:       `{}`,
	}
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatalf("create inbound failed: %v", err)
	}
	return inbound
}

func createInboundCleanupPolicy(t *testing.T, inboundId int, remark string) (*n5model.TrafficPolicy, *n5model.TrafficPolicyRule) {
	t.Helper()
	policy := &n5model.TrafficPolicy{
		Name:              "cleanup-policy",
		Remark:            remark,
		Enabled:           true,
		DefaultTargetType: "egress",
		DefaultTargetId:   1,
	}
	if err := database.GetDB().Create(policy).Error; err != nil {
		t.Fatalf("create policy failed: %v", err)
	}
	rule := &n5model.TrafficPolicyRule{
		PolicyId:   policy.Id,
		RuleType:   "domain",
		MatchMode:  "exact",
		MatchValue: "cleanup.example.com",
		TargetType: "egress",
		TargetId:   1,
		SortOrder:  1,
		Enabled:    true,
	}
	if err := database.GetDB().Create(rule).Error; err != nil {
		t.Fatalf("create policy rule failed: %v", err)
	}
	binding := &n5model.TrafficPolicyBinding{
		InboundId: inboundId,
		PolicyId:  policy.Id,
		Enabled:   true,
	}
	if err := database.GetDB().Create(binding).Error; err != nil {
		t.Fatalf("create policy binding failed: %v", err)
	}
	return policy, rule
}

func assertInboundCleanupCount(t *testing.T, table string, query string, args []interface{}, want int64) {
	t.Helper()
	var count int64
	if err := database.GetDB().Table(table).Where(query, args...).Count(&count).Error; err != nil {
		t.Fatalf("count %s failed: %v", table, err)
	}
	if count != want {
		t.Fatalf("unexpected %s count: got %d want %d", table, count, want)
	}
}

func TestInboundServiceDeleteCleansOnlyDeletedInboundSimpleState(t *testing.T) {
	initInboundN5CleanupTestDB(t)

	if err := database.GetDB().Create(&n5model.Egress{
		Id:           1,
		Name:         "cleanup-egress",
		Protocol:     "freedom",
		Tag:          "n5-egress-0000000001",
		Enabled:      true,
		OutboundJSON: `{"protocol":"freedom","tag":"n5-egress-0000000001","settings":{}}`,
	}).Error; err != nil {
		t.Fatalf("create egress failed: %v", err)
	}

	inboundA := createInboundCleanupTestInbound(t, "cleanup-a", 35101)
	inboundB := createInboundCleanupTestInbound(t, "cleanup-b", 35102)
	inboundC := createInboundCleanupTestInbound(t, "cleanup-c", 35103)

	sourceGroup, sourceRule := createInboundCleanupPolicy(t, 0, "n5-simple-rule-group|eyJ0eXBlIjoiY3VzdG9tIn0=")
	if err := database.GetDB().Where("policy_id = ?", sourceGroup.Id).Delete(&n5model.TrafficPolicyBinding{}).Error; err != nil {
		t.Fatalf("remove source group binding failed: %v", err)
	}

	policyA, ruleA := createInboundCleanupPolicy(t, inboundA.Id, "n5-simple-exec|eyJ2ZXJzaW9uIjoxLCJpdGVtcyI6W119")
	policyB, ruleB := createInboundCleanupPolicy(t, inboundB.Id, "n5-simple-exec|eyJ2ZXJzaW9uIjoxLCJpdGVtcyI6W119")
	advancedPolicy, _ := createInboundCleanupPolicy(t, inboundC.Id, "ordinary-advanced")

	if err := (&InboundService{}).DelInbound(inboundA.Id); err != nil {
		t.Fatalf("delete inbound failed: %v", err)
	}

	assertInboundCleanupCount(t, "inbounds", "id = ?", []interface{}{inboundA.Id}, 0)
	assertInboundCleanupCount(t, "n5_traffic_policy_bindings", "inbound_id = ?", []interface{}{inboundA.Id}, 0)
	assertInboundCleanupCount(t, "n5_traffic_policies", "id = ?", []interface{}{policyA.Id}, 0)
	assertInboundCleanupCount(t, "n5_traffic_policy_rules", "policy_id = ?", []interface{}{policyA.Id}, 0)

	assertInboundCleanupCount(t, "inbounds", "id = ?", []interface{}{inboundB.Id}, 1)
	assertInboundCleanupCount(t, "n5_traffic_policy_bindings", "inbound_id = ? and policy_id = ?", []interface{}{inboundB.Id, policyB.Id}, 1)
	assertInboundCleanupCount(t, "n5_traffic_policies", "id = ?", []interface{}{policyB.Id}, 1)
	assertInboundCleanupCount(t, "n5_traffic_policy_rules", "id = ?", []interface{}{ruleB.Id}, 1)

	assertInboundCleanupCount(t, "n5_traffic_policies", "id = ?", []interface{}{sourceGroup.Id}, 1)
	assertInboundCleanupCount(t, "n5_traffic_policy_rules", "id = ?", []interface{}{sourceRule.Id}, 1)
	assertInboundCleanupCount(t, "n5_traffic_policies", "id = ?", []interface{}{advancedPolicy.Id}, 1)
	assertInboundCleanupCount(t, "n5_traffic_policy_bindings", "inbound_id = ? and policy_id = ?", []interface{}{inboundC.Id, advancedPolicy.Id}, 1)
	assertInboundCleanupCount(t, "n5_traffic_policy_rules", "id = ?", []interface{}{ruleA.Id}, 0)

	if err := (&InboundService{}).DelInbound(inboundC.Id); err != nil {
		t.Fatalf("delete advanced-bound inbound failed: %v", err)
	}
	assertInboundCleanupCount(t, "n5_traffic_policy_bindings", "inbound_id = ?", []interface{}{inboundC.Id}, 0)
	assertInboundCleanupCount(t, "n5_traffic_policies", "id = ?", []interface{}{advancedPolicy.Id}, 1)
}
