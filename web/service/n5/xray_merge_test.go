package n5

import (
	"encoding/json"
	"testing"
	"x-ui/database"
	legacyModel "x-ui/database/model"
	n5model "x-ui/database/model/n5"
	"x-ui/util/json_util"
	"x-ui/xray"
)

func TestXrayMergeServiceAppendsOutboundsAndRouting(t *testing.T) {
	initTestDB(t)

	egressSvc := &EgressService{}
	poolSvc := &EgressPoolService{}
	policySvc := &TrafficPolicyService{}
	mergeSvc := &XrayMergeService{}

	egress, err := egressSvc.Create(&n5model.Egress{
		Name:         "merge-egress",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: freedomOutboundJSON(),
	})
	if err != nil {
		t.Fatalf("create egress failed: %v", err)
	}

	pool, err := poolSvc.Create(&n5model.EgressPool{
		Name:     "merge-pool",
		Strategy: "random",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("create pool failed: %v", err)
	}
	if _, err := poolSvc.AddMember(pool.Id, egress.Id, 1, 1); err != nil {
		t.Fatalf("add member failed: %v", err)
	}

	inbound := &legacyModel.Inbound{
		UserId:         1,
		Remark:         "merge-inbound",
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           32011,
		Protocol:       legacyModel.Socks,
		Settings:       `{"auth":"noauth","udp":false,"ip":"127.0.0.1"}`,
		StreamSettings: `{}`,
		Tag:            "inbound-merge-tag",
		Sniffing:       `{}`,
	}
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatalf("create inbound failed: %v", err)
	}

	policy, err := policySvc.Create(&n5model.TrafficPolicy{
		Name:              "merge-policy",
		Enabled:           true,
		DefaultTargetType: targetTypePool,
		DefaultTargetId:   pool.Id,
	})
	if err != nil {
		t.Fatalf("create policy failed: %v", err)
	}
	if _, err := policySvc.AddRule(&n5model.TrafficPolicyRule{
		PolicyId:   policy.Id,
		RuleType:   ruleTypeDomain,
		MatchMode:  domainModeExact,
		MatchValue: "merge.example.com",
		TargetType: targetTypeEgress,
		TargetId:   egress.Id,
		Enabled:    true,
	}); err != nil {
		t.Fatalf("add rule failed: %v", err)
	}
	if _, err := policySvc.BindInboundPolicy(inbound.Id, policy.Id); err != nil {
		t.Fatalf("bind policy failed: %v", err)
	}

	base := &xray.Config{
		OutboundConfigs: json_util.RawMessage(`[
			{"protocol":"freedom","settings":{},"tag":"direct"},
			{"protocol":"blackhole","settings":{},"tag":"blocked"}
		]`),
		RouterConfig: json_util.RawMessage(`{
			"rules":[
				{"type":"field","inboundTag":["api"],"outboundTag":"api"}
			]
		}`),
	}

	mergeResult, err := mergeSvc.MergeWithMeta(base)
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	if mergeResult.HistoryID <= 0 {
		t.Fatalf("expected config history id, got %d", mergeResult.HistoryID)
	}
	if mergeResult.BaseConfigHash == "" || mergeResult.ExtensionConfigHash == "" || mergeResult.ConfigHash == "" {
		t.Fatalf("expected merge hashes to be populated: %#v", mergeResult)
	}
	if mergeResult.OutboundCount != 1 || mergeResult.RoutingRuleCount != 2 || mergeResult.BindingCount != 1 {
		t.Fatalf("unexpected merge summary: %#v", mergeResult)
	}

	merged := mergeResult.Config

	outbounds := make([]map[string]interface{}, 0)
	if err := json.Unmarshal(merged.OutboundConfigs, &outbounds); err != nil {
		t.Fatalf("unmarshal merged outbounds failed: %v", err)
	}
	if len(outbounds) != 3 {
		t.Fatalf("unexpected merged outbound count: %d", len(outbounds))
	}
	if outbounds[0]["tag"] != "direct" {
		t.Fatalf("unexpected base outbound order: %#v", outbounds[0]["tag"])
	}
	if outbounds[2]["tag"] != egress.Tag {
		t.Fatalf("unexpected appended outbound tag: %#v", outbounds[2]["tag"])
	}

	routing := make(map[string]interface{})
	if err := json.Unmarshal(merged.RouterConfig, &routing); err != nil {
		t.Fatalf("unmarshal merged routing failed: %v", err)
	}
	rules := routing["rules"].([]interface{})
	if len(rules) != 3 {
		t.Fatalf("unexpected merged rule count: %d", len(rules))
	}
	firstRule := rules[0].(map[string]interface{})
	if firstRule["outboundTag"] != "api" {
		t.Fatalf("unexpected preserved base rule target: %#v", firstRule["outboundTag"])
	}

	mergeRule := rules[1].(map[string]interface{})
	inboundTags := mergeRule["inboundTag"].([]interface{})
	if inboundTags[0].(string) != inbound.Tag {
		t.Fatalf("unexpected merge inbound tag binding: %#v", inboundTags)
	}
	if mergeRule["outboundTag"].(string) != egress.Tag {
		t.Fatalf("unexpected merge outbound target: %#v", mergeRule["outboundTag"])
	}

	defaultRule := rules[2].(map[string]interface{})
	if defaultRule["balancerTag"].(string) != pool.Tag {
		t.Fatalf("unexpected default balancer target: %#v", defaultRule["balancerTag"])
	}

	balancers := routing["balancers"].([]interface{})
	if len(balancers) != 1 {
		t.Fatalf("unexpected merged balancer count: %d", len(balancers))
	}
	balancer := balancers[0].(map[string]interface{})
	if balancer["tag"].(string) != pool.Tag {
		t.Fatalf("unexpected merged balancer tag: %#v", balancer["tag"])
	}

	history := &n5model.XrayConfigHistory{}
	if err := database.GetDB().Model(&n5model.XrayConfigHistory{}).Where("id = ?", mergeResult.HistoryID).First(history).Error; err != nil {
		t.Fatalf("query config history failed: %v", err)
	}
	if history.Source != "n5-merge" {
		t.Fatalf("unexpected config history source: %s", history.Source)
	}
	if history.ApplyStatus != "generated" {
		t.Fatalf("unexpected config history status: %s", history.ApplyStatus)
	}
	if history.BaseConfigHash != mergeResult.BaseConfigHash || history.ExtensionConfigHash != mergeResult.ExtensionConfigHash || history.ConfigHash != mergeResult.ConfigHash {
		t.Fatalf("config history hashes do not match merge result: %#v", history)
	}
}
