package simple

import (
	"strings"
	"testing"
	"x-ui/database"
	"x-ui/database/model"
	n5model "x-ui/database/model/n5"
	"x-ui/util/common"
	n5service "x-ui/web/service/n5"
	n5templates "x-ui/web/service/n5/templates"
)

func TestTrafficRuleGroupServiceCreateBuiltinAndNoMerge(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewTrafficRuleGroupService()
	groups, err := svc.ListGroups()
	if err != nil {
		t.Fatalf("list builtin groups failed: %v", err)
	}
	group := findGroupByType(groups, simpleTrafficAI)
	if group == nil {
		t.Fatal("expected builtin ai group")
	}
	if group.GroupType != simpleTrafficAI {
		t.Fatalf("unexpected group: %#v", group)
	}
	if group.RuleCount == 0 {
		t.Fatalf("expected builtin group rules: %#v", group)
	}

	fragments, err := (&n5service.XrayExtService{}).GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("generate routing fragments failed: %v", err)
	}
	rules, _ := fragments["rules"].([]interface{})
	if len(rules) != 0 {
		t.Fatalf("expected no routing rules for definition-only group, got %#v", fragments)
	}
}

func TestTrafficRuleGroupServiceAddDeleteRuleAndDisable(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewTrafficRuleGroupService()
	group, err := svc.CreateGroup(&CreateTrafficRuleGroupRequest{
		GroupType: simpleTrafficCustom,
		Name:      "custom domains",
	})
	if err != nil {
		t.Fatalf("create custom group failed: %v", err)
	}
	if group.RuleCount != 0 {
		t.Fatalf("expected empty custom group: %#v", group)
	}

	rule, err := svc.AddDomainRule(&AddTrafficRuleDomainRequest{
		GroupId: group.Id,
		Domain:  "full:api64.ipify.org",
	})
	if err != nil {
		t.Fatalf("add domain rule failed: %v", err)
	}
	if rule.DisplayValue != "full:api64.ipify.org" {
		t.Fatalf("unexpected rule display value: %#v", rule)
	}

	fetched, err := svc.GetGroup(group.Id)
	if err != nil {
		t.Fatalf("get group failed: %v", err)
	}
	if len(fetched.Rules) != 1 {
		t.Fatalf("unexpected fetched rules: %#v", fetched)
	}

	disabled, err := svc.DisableGroup(group.Id)
	if err != nil {
		t.Fatalf("disable group failed: %v", err)
	}
	if disabled.Enabled {
		t.Fatalf("expected disabled group: %#v", disabled)
	}

	if err := svc.DeleteDomainRule(group.Id, rule.Id); err != nil {
		t.Fatalf("delete domain rule failed: %v", err)
	}
	fetched, err = svc.GetGroup(group.Id)
	if err != nil {
		t.Fatalf("get group after delete failed: %v", err)
	}
	if len(fetched.Rules) != 0 {
		t.Fatalf("expected no rules after delete: %#v", fetched)
	}
}

func TestTrafficRuleGroupServiceUpdateDomainRule(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewTrafficRuleGroupService()
	group, err := svc.CreateGroup(&CreateTrafficRuleGroupRequest{
		GroupType: simpleTrafficCustom,
		Name:      "custom domains",
	})
	if err != nil {
		t.Fatalf("create custom group failed: %v", err)
	}
	rule, err := svc.AddDomainRule(&AddTrafficRuleDomainRequest{
		GroupId: group.Id,
		Domain:  "domain:example.com",
	})
	if err != nil {
		t.Fatalf("add domain rule failed: %v", err)
	}

	updated, err := svc.UpdateDomainRule(rule.Id, &UpdateTrafficRuleDomainRequest{
		GroupId: group.Id,
		Domain:  "full:api.ipify.org",
	})
	if err != nil {
		t.Fatalf("update domain rule failed: %v", err)
	}
	if updated.MatchMode != "exact" || updated.MatchValue != "api.ipify.org" || updated.DisplayValue != "full:api.ipify.org" {
		t.Fatalf("unexpected updated rule: %#v", updated)
	}

	fetched, err := svc.GetGroup(group.Id)
	if err != nil {
		t.Fatalf("get group failed: %v", err)
	}
	if len(fetched.Rules) != 1 || fetched.Rules[0].MatchMode != "exact" || fetched.Rules[0].MatchValue != "api.ipify.org" {
		t.Fatalf("unexpected fetched rules: %#v", fetched.Rules)
	}
}

func TestTrafficRuleGroupServiceUpdateDomainRuleProtectsOwnershipAndValidates(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewTrafficRuleGroupService()
	groupA, err := svc.CreateGroup(&CreateTrafficRuleGroupRequest{GroupType: simpleTrafficCustom, Name: "group A"})
	if err != nil {
		t.Fatalf("create group A failed: %v", err)
	}
	groupB, err := svc.CreateGroup(&CreateTrafficRuleGroupRequest{GroupType: simpleTrafficCustom, Name: "group B"})
	if err != nil {
		t.Fatalf("create group B failed: %v", err)
	}
	rule, err := svc.AddDomainRule(&AddTrafficRuleDomainRequest{GroupId: groupA.Id, Domain: "domain:example.com"})
	if err != nil {
		t.Fatalf("add domain rule failed: %v", err)
	}

	if _, err := svc.UpdateDomainRule(rule.Id, &UpdateTrafficRuleDomainRequest{
		GroupId: groupB.Id,
		Domain:  "domain:openai.com",
	}); err == nil {
		t.Fatal("expected cross-group update to fail")
	}
	if _, err := svc.UpdateDomainRule(rule.Id, &UpdateTrafficRuleDomainRequest{
		GroupId: groupA.Id,
		Domain:  "https://example.com/path",
	}); err == nil || !strings.Contains(err.Error(), "不要包含") {
		t.Fatalf("expected invalid domain error, got %v", err)
	}
	if _, err := svc.UpdateDomainRule(rule.Id, &UpdateTrafficRuleDomainRequest{
		GroupId: groupA.Id,
		Domain:  "regexp:[",
	}); err == nil || !strings.Contains(err.Error(), "invalid regexp") {
		t.Fatalf("expected invalid regexp error, got %v", err)
	}

	fetched, err := svc.GetGroup(groupA.Id)
	if err != nil {
		t.Fatalf("get group failed: %v", err)
	}
	if len(fetched.Rules) != 1 || fetched.Rules[0].MatchValue != "example.com" {
		t.Fatalf("failed update should not mutate rule: %#v", fetched.Rules)
	}
}

func TestTrafficRuleGroupServiceSnapshotOnlyAffectsExecPolicy(t *testing.T) {
	initSimpleTestDB(t)

	groupSvc := NewTrafficRuleGroupService()
	group, err := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	if err != nil {
		t.Fatalf("get builtin group failed: %v", err)
	}

	inbound := &model.Inbound{
		UserId:         1,
		Remark:         "snapshot-inbound",
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           33101,
		Protocol:       model.Socks,
		Settings:       `{"auth":"noauth","udp":false,"ip":"127.0.0.1"}`,
		StreamSettings: `{}`,
		Tag:            "snapshot-inbound-tag",
		Sniffing:       `{}`,
	}
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatalf("create inbound failed: %v", err)
	}

	egress, err := (&n5service.EgressService{}).Create(&n5model.Egress{
		Name:         "snapshot-egress",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: `{"protocol":"freedom","settings":{}}`,
	})
	if err != nil {
		t.Fatalf("create egress failed: %v", err)
	}

	ruleSvc := NewRuleService()
	if _, err := ruleSvc.CreateSimpleRule(&CreateSimpleRuleRequest{
		InboundId: inbound.Id,
		GroupId:   group.Id,
		EgressId:  egress.Id,
	}); err != nil {
		t.Fatalf("create snapshot exec rule failed: %v", err)
	}

	fragments, err := (&n5service.XrayExtService{}).GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("generate routing fragments failed: %v", err)
	}
	assertSimpleRuleFragment(t, fragments, inbound.Tag, egress.Tag, "domain:openai.com", true)
}

func TestTrafficRuleGroupServiceEnsureBuiltinGroupsFreshDB(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewTrafficRuleGroupService()
	groups, err := svc.ListGroups()
	if err != nil {
		t.Fatalf("list groups failed: %v", err)
	}
	if len(groups) != len(builtinSimpleGroupTypes) {
		t.Fatalf("unexpected builtin group count: %d", len(groups))
	}

	options, err := svc.ListGroupOptions()
	if err != nil {
		t.Fatalf("list group options failed: %v", err)
	}
	if len(options) != len(builtinSimpleGroupTypes) {
		t.Fatalf("unexpected builtin option count: %d", len(options))
	}

	policySvc := &n5service.TrafficPolicyService{}
	policies, err := policySvc.List()
	if err != nil {
		t.Fatalf("list policies failed: %v", err)
	}
	if len(policies) != len(builtinSimpleGroupTypes) {
		t.Fatalf("unexpected policy count: %d", len(policies))
	}

	expected := map[string]string{
		simpleTrafficAI:        "AI分流",
		simpleTrafficGame:      "游戏分流",
		simpleTrafficStreaming: "流媒体分流",
	}
	for _, group := range groups {
		if expected[group.GroupType] != group.Name {
			t.Fatalf("unexpected builtin group: %#v", group)
		}
		if !group.Builtin {
			t.Fatalf("expected builtin group: %#v", group)
		}
		if group.RuleCount == 0 {
			t.Fatalf("expected builtin rules: %#v", group)
		}
		delete(expected, group.GroupType)
	}
	if len(expected) != 0 {
		t.Fatalf("missing builtin groups: %#v", expected)
	}
}

func TestTrafficRuleGroupServiceEnsureBuiltinGroupsIsIdempotentAndPreservesChanges(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewTrafficRuleGroupService()
	if err := svc.EnsureBuiltinGroups(); err != nil {
		t.Fatalf("ensure builtin groups failed: %v", err)
	}
	groups, err := svc.ListGroups()
	if err != nil {
		t.Fatalf("list groups failed: %v", err)
	}

	var aiGroup *TrafficRuleGroup
	for _, group := range groups {
		if group.GroupType == simpleTrafficAI {
			aiGroup = group
			break
		}
	}
	if aiGroup == nil {
		t.Fatal("missing ai builtin group")
	}

	addedRule, err := svc.AddDomainRule(&AddTrafficRuleDomainRequest{
		GroupId: aiGroup.Id,
		Domain:  "accounts.google.com",
	})
	if err != nil {
		t.Fatalf("add domain rule failed: %v", err)
	}
	if addedRule.DisplayValue != "domain:accounts.google.com" {
		t.Fatalf("unexpected added rule: %#v", addedRule)
	}

	if _, err := svc.DisableGroup(aiGroup.Id); err != nil {
		t.Fatalf("disable ai group failed: %v", err)
	}

	if err := svc.EnsureBuiltinGroups(); err != nil {
		t.Fatalf("re-ensure builtin groups failed: %v", err)
	}

	policySvc := &n5service.TrafficPolicyService{}
	policies, err := policySvc.List()
	if err != nil {
		t.Fatalf("list policies failed: %v", err)
	}
	countByType := map[string]int{}
	for _, policy := range policies {
		meta, ok := parseSimpleRuleGroupRemark(policy.Remark)
		if !ok {
			continue
		}
		countByType[meta.GroupType]++
	}
	for _, groupType := range builtinSimpleGroupTypes {
		if countByType[groupType] != 1 {
			t.Fatalf("unexpected policy count for %s: %d", groupType, countByType[groupType])
		}
	}

	aiGroup, err = svc.GetGroup(aiGroup.Id)
	if err != nil {
		t.Fatalf("get ai group failed: %v", err)
	}
	if aiGroup.Enabled {
		t.Fatalf("expected ai group to remain disabled: %#v", aiGroup)
	}
	found := false
	for _, rule := range aiGroup.Rules {
		if rule.DisplayValue == "domain:accounts.google.com" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected added rule to remain after ensure: %#v", aiGroup.Rules)
	}
}

func TestTrafficRuleGroupServiceEnsureBuiltinGroupsBackfillsMissingOnly(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewTrafficRuleGroupService()
	if err := database.GetDB().Where("1 = 1").Delete(&n5model.TrafficPolicyRule{}).Error; err != nil {
		t.Fatalf("clear policy rules failed: %v", err)
	}
	if err := database.GetDB().Where("1 = 1").Delete(&n5model.TrafficPolicy{}).Error; err != nil {
		t.Fatalf("clear policies failed: %v", err)
	}

	policy := &n5model.TrafficPolicy{
		Name:    defaultSimpleGroupName(simpleTrafficAI),
		Remark:  buildSimpleRuleGroupRemark(simpleTrafficAI),
		Enabled: true,
	}
	if err := database.GetDB().Create(policy).Error; err != nil {
		t.Fatalf("create existing ai policy failed: %v", err)
	}
	if err := database.GetDB().Model(&n5model.TrafficPolicy{}).Where("id = ?", policy.Id).Update("enabled", false).Error; err != nil {
		t.Fatalf("disable existing ai policy failed: %v", err)
	}
	if err := svc.seedGroupRules(policy.Id, simpleTrafficAI); err != nil {
		t.Fatalf("seed ai rules failed: %v", err)
	}
	if err := svc.EnsureBuiltinGroups(); err != nil {
		t.Fatalf("ensure builtin groups failed: %v", err)
	}

	groups, err := svc.ListGroups()
	if err != nil {
		t.Fatalf("list groups failed: %v", err)
	}
	if len(groups) != len(builtinSimpleGroupTypes) {
		t.Fatalf("unexpected group count: %d", len(groups))
	}
	countByType := map[string]int{}
	for _, group := range groups {
		countByType[group.GroupType]++
		if group.GroupType == simpleTrafficAI && group.Enabled {
			t.Fatalf("expected existing ai group to remain disabled: %#v", group)
		}
	}
	for _, groupType := range builtinSimpleGroupTypes {
		if countByType[groupType] != 1 {
			t.Fatalf("unexpected group count for %s: %d", groupType, countByType[groupType])
		}
	}
}

func TestTrafficRuleGroupServiceDeleteProtectsBuiltinOnly(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewTrafficRuleGroupService()
	groups, err := svc.ListGroups()
	if err != nil {
		t.Fatalf("list groups failed: %v", err)
	}
	if len(groups) == 0 {
		t.Fatal("expected builtin groups")
	}
	if err := svc.DeleteGroup(groups[0].Id); err == nil || !strings.Contains(err.Error(), "内置规则组不可删除") {
		t.Fatalf("expected builtin delete protection error, got %v", err)
	}

	custom, err := svc.CreateGroup(&CreateTrafficRuleGroupRequest{
		GroupType: simpleTrafficCustom,
		Name:      "custom domains",
	})
	if err != nil {
		t.Fatalf("create custom group failed: %v", err)
	}
	if err := svc.DeleteGroup(custom.Id); err != nil {
		t.Fatalf("delete custom group failed: %v", err)
	}
}

func TestBuiltinTemplatesContainOnlySuffixDomainRules(t *testing.T) {
	definitions := []*n5templates.Definition{
		n5templates.AI(),
		n5templates.Game(),
		n5templates.Streaming(),
	}

	for _, definition := range definitions {
		if definition == nil {
			t.Fatal("expected builtin template definition")
		}
		if len(definition.Rules) == 0 {
			t.Fatalf("expected rules for template %s", definition.Name)
		}
		seen := make(map[string]bool)
		for _, rule := range definition.Rules {
			if rule.RuleType != "domain" {
				t.Fatalf("unexpected rule type in %s: %#v", definition.Name, rule)
			}
			if rule.MatchMode != "suffix" {
				t.Fatalf("unexpected match mode in %s: %#v", definition.Name, rule)
			}
			if strings.TrimSpace(rule.MatchValue) == "" {
				t.Fatalf("unexpected empty match value in %s", definition.Name)
			}
			if strings.Contains(rule.MatchValue, "geosite:") {
				t.Fatalf("unexpected geosite rule in %s: %#v", definition.Name, rule)
			}
			if strings.HasPrefix(rule.MatchValue, "full:") {
				t.Fatalf("unexpected full matcher in %s: %#v", definition.Name, rule)
			}
			if seen[rule.MatchValue] {
				t.Fatalf("unexpected duplicate rule in %s: %s", definition.Name, rule.MatchValue)
			}
			seen[rule.MatchValue] = true
		}
	}
}

func mustGetBuiltinGroup(svc *TrafficRuleGroupService, groupType string) (*TrafficRuleGroup, error) {
	groups, err := svc.ListGroups()
	if err != nil {
		return nil, err
	}
	group := findGroupByType(groups, groupType)
	if group == nil {
		return nil, common.NewError("builtin traffic rule group not found")
	}
	return group, nil
}

func findGroupByType(groups []*TrafficRuleGroup, groupType string) *TrafficRuleGroup {
	for _, group := range groups {
		if group != nil && group.GroupType == groupType {
			return group
		}
	}
	return nil
}
