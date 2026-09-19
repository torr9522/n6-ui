package service

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"x-ui/database"
	legacyModel "x-ui/database/model"
	n5model "x-ui/database/model/n5"
	n5service "x-ui/web/service/n5"
)

func initServiceTestDB(t *testing.T) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "service-test.db")
	if err := database.InitDB(dbPath); err != nil {
		t.Fatalf("init db failed: %v", err)
	}
}

func TestGetXrayConfigWithN5DisabledKeepsLegacyConfig(t *testing.T) {
	initServiceTestDB(t)

	egressService := &n5service.EgressService{}
	if _, err := egressService.Create(&n5model.Egress{
		Name:         "disabled-egress",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: `{"protocol":"freedom","settings":{}}`,
	}); err != nil {
		t.Fatalf("create egress failed: %v", err)
	}

	settingService := &SettingService{}
	if err := settingService.setString("n5XrayExtensionEnable", "false"); err != nil {
		t.Fatalf("disable n5 xray extension failed: %v", err)
	}

	xrayService := &XrayService{}
	cfg, mergeResult, err := xrayService.getXrayConfigWithMeta()
	if err != nil {
		t.Fatalf("get xray config failed: %v", err)
	}
	if mergeResult != nil {
		t.Fatalf("expected nil merge result when n5 extension is disabled: %#v", mergeResult)
	}

	outbounds := parseOutboundTags(t, cfg.OutboundConfigs)
	if len(outbounds) != 2 {
		t.Fatalf("unexpected outbound count: %d", len(outbounds))
	}
	for _, tag := range outbounds {
		if strings.HasPrefix(tag, "n5-") {
			t.Fatalf("unexpected n5 outbound tag when extension is disabled: %s", tag)
		}
	}

	var historyCount int64
	if err := database.GetDB().Model(&n5model.XrayConfigHistory{}).Count(&historyCount).Error; err != nil {
		t.Fatalf("count config history failed: %v", err)
	}
	if historyCount != 0 {
		t.Fatalf("unexpected config history count: %d", historyCount)
	}
}

func TestGetXrayConfigWithN5EnabledAndNoBindingDoesNotChangeLegacyRouting(t *testing.T) {
	initServiceTestDB(t)

	egressService := &n5service.EgressService{}
	egress, err := egressService.Create(&n5model.Egress{
		Name:         "enabled-egress",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: `{"protocol":"freedom","settings":{}}`,
	})
	if err != nil {
		t.Fatalf("create egress failed: %v", err)
	}

	settingService := &SettingService{}
	if err := settingService.setString("n5XrayExtensionEnable", "true"); err != nil {
		t.Fatalf("enable n5 xray extension failed: %v", err)
	}

	xrayService := &XrayService{}
	cfg, mergeResult, err := xrayService.getXrayConfigWithMeta()
	if err != nil {
		t.Fatalf("get xray config failed: %v", err)
	}
	if mergeResult == nil {
		t.Fatal("expected merge result when n5 extension is enabled")
	}
	if mergeResult.OutboundCount != 1 || mergeResult.RoutingRuleCount != 0 || mergeResult.BindingCount != 0 {
		t.Fatalf("unexpected merge result summary: %#v", mergeResult)
	}

	outbounds := parseOutboundTags(t, cfg.OutboundConfigs)
	if len(outbounds) != 3 {
		t.Fatalf("unexpected outbound count: %d", len(outbounds))
	}
	if outbounds[2] != egress.Tag {
		t.Fatalf("unexpected appended outbound tag: %s", outbounds[2])
	}

	routing := parseRoutingConfig(t, cfg.RouterConfig)
	rules, _ := routing["rules"].([]interface{})
	if len(rules) != 3 {
		t.Fatalf("unexpected routing rule count: %d", len(rules))
	}
	for _, item := range rules {
		rule := item.(map[string]interface{})
		if outboundTag, ok := rule["outboundTag"].(string); ok && strings.HasPrefix(outboundTag, "n5-") {
			t.Fatalf("unexpected n5 routing rule without binding: %#v", rule)
		}
	}
}

func TestGetXrayConfigWithN5EnabledAndBindingAddsMergeFragments(t *testing.T) {
	initServiceTestDB(t)

	egressService := &n5service.EgressService{}
	policyService := &n5service.TrafficPolicyService{}

	egress, err := egressService.Create(&n5model.Egress{
		Name:         "bound-egress",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: `{"protocol":"freedom","settings":{}}`,
	})
	if err != nil {
		t.Fatalf("create egress failed: %v", err)
	}

	inbound := &legacyModel.Inbound{
		UserId:         1,
		Remark:         "bound-inbound",
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           33001,
		Protocol:       legacyModel.Socks,
		Settings:       `{"auth":"noauth","udp":false,"ip":"127.0.0.1"}`,
		StreamSettings: `{}`,
		Tag:            "bound-inbound-tag",
		Sniffing:       `{}`,
	}
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatalf("create inbound failed: %v", err)
	}

	policy, err := policyService.Create(&n5model.TrafficPolicy{
		Name:    "bound-policy",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("create policy failed: %v", err)
	}
	if _, err := policyService.AddRule(&n5model.TrafficPolicyRule{
		PolicyId:   policy.Id,
		RuleType:   "domain",
		MatchMode:  "exact",
		MatchValue: "bound.example.com",
		TargetType: "egress",
		TargetId:   egress.Id,
		Enabled:    true,
	}); err != nil {
		t.Fatalf("add policy rule failed: %v", err)
	}
	if _, err := policyService.BindInboundPolicy(inbound.Id, policy.Id); err != nil {
		t.Fatalf("bind inbound policy failed: %v", err)
	}

	settingService := &SettingService{}
	if err := settingService.setString("n5XrayExtensionEnable", "true"); err != nil {
		t.Fatalf("enable n5 xray extension failed: %v", err)
	}

	xrayService := &XrayService{}
	cfg, mergeResult, err := xrayService.getXrayConfigWithMeta()
	if err != nil {
		t.Fatalf("get xray config failed: %v", err)
	}
	if mergeResult == nil {
		t.Fatal("expected merge result when n5 extension is enabled")
	}
	if mergeResult.BaseConfigHash == "" || mergeResult.ExtensionConfigHash == "" || mergeResult.ConfigHash == "" {
		t.Fatalf("expected merge hashes: %#v", mergeResult)
	}

	routing := parseRoutingConfig(t, cfg.RouterConfig)
	rules, _ := routing["rules"].([]interface{})
	found := false
	for _, item := range rules {
		rule := item.(map[string]interface{})
		outboundTag, _ := rule["outboundTag"].(string)
		if outboundTag != egress.Tag {
			continue
		}
		inboundTags, ok := rule["inboundTag"].([]interface{})
		if !ok || len(inboundTags) != 1 || inboundTags[0].(string) != inbound.Tag {
			continue
		}
		found = true
	}
	if !found {
		t.Fatalf("expected bound N5 routing rule in merged config: %#v", routing["rules"])
	}

	history := &n5model.XrayConfigHistory{}
	if err := database.GetDB().Model(&n5model.XrayConfigHistory{}).Where("id = ?", mergeResult.HistoryID).First(history).Error; err != nil {
		t.Fatalf("query config history failed: %v", err)
	}
	if history.BaseConfigHash != mergeResult.BaseConfigHash || history.ExtensionConfigHash != mergeResult.ExtensionConfigHash || history.ConfigHash != mergeResult.ConfigHash {
		t.Fatalf("unexpected config history hashes: %#v", history)
	}
}

func parseOutboundTags(t *testing.T, raw []byte) []string {
	t.Helper()
	outbounds := make([]map[string]interface{}, 0)
	if err := json.Unmarshal(raw, &outbounds); err != nil {
		t.Fatalf("unmarshal outbounds failed: %v", err)
	}

	tags := make([]string, 0, len(outbounds))
	for _, outbound := range outbounds {
		if tag, ok := outbound["tag"].(string); ok {
			tags = append(tags, tag)
			continue
		}
		tags = append(tags, "")
	}
	return tags
}

func parseRoutingConfig(t *testing.T, raw []byte) map[string]interface{} {
	t.Helper()
	routing := make(map[string]interface{})
	if err := json.Unmarshal(raw, &routing); err != nil {
		t.Fatalf("unmarshal routing failed: %v", err)
	}
	return routing
}
