package n5

import (
	"github.com/torr9522/n6-ui/database"
	legacyModel "github.com/torr9522/n6-ui/database/model"
	n5model "github.com/torr9522/n6-ui/database/model/n5"
	"strings"
	"testing"
)

func createTrafficTestInbound(t *testing.T, port int, tag string) *legacyModel.Inbound {
	t.Helper()

	inbound := &legacyModel.Inbound{
		UserId:         1,
		Remark:         tag,
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           port,
		Protocol:       legacyModel.Socks,
		Settings:       `{"auth":"noauth","udp":false,"ip":"127.0.0.1"}`,
		StreamSettings: `{}`,
		Tag:            tag,
		Sniffing:       `{}`,
	}
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatalf("create inbound failed: %v", err)
	}
	return inbound
}

func createTrafficTestEgress(t *testing.T, svc *EgressService, name string) *n5model.Egress {
	t.Helper()

	egress, err := svc.Create(&n5model.Egress{
		Name:         name,
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: freedomOutboundJSON(),
	})
	if err != nil {
		t.Fatalf("create egress failed: %v", err)
	}
	return egress
}

func trafficInboundRoutingMatchersAndTargets(t *testing.T, routing map[string]interface{}, inboundTag string) []string {
	t.Helper()
	items, _ := routing["rules"].([]interface{})
	result := make([]string, 0)
	for _, item := range items {
		rule, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		inboundTags, _ := rule["inboundTag"].([]interface{})
		if len(inboundTags) == 0 || inboundTags[0] != inboundTag {
			continue
		}
		matcher := "*"
		if domains, _ := rule["domain"].([]interface{}); len(domains) > 0 {
			matcher, _ = domains[0].(string)
		}
		if ips, _ := rule["ip"].([]interface{}); len(ips) > 0 {
			matcher, _ = ips[0].(string)
		}
		tag, _ := rule["outboundTag"].(string)
		if tag == "" {
			tag, _ = rule["balancerTag"].(string)
		}
		result = append(result, matcher+"=>"+tag)
	}
	return result
}

func TestTrafficPolicyServiceManagePolicyRuleAndBinding(t *testing.T) {
	initTestDB(t)

	egressSvc := &EgressService{}
	policySvc := &TrafficPolicyService{}

	egressA := createTrafficTestEgress(t, egressSvc, "manage-egress-a")
	egressB := createTrafficTestEgress(t, egressSvc, "manage-egress-b")
	inbound := createTrafficTestInbound(t, 34101, "manage-inbound")

	policy, err := policySvc.Create(&n5model.TrafficPolicy{
		Name:              "manage-policy",
		Enabled:           true,
		DefaultTargetType: targetTypeEgress,
		DefaultTargetId:   egressA.Id,
	})
	if err != nil {
		t.Fatalf("create policy failed: %v", err)
	}
	ruleA, err := policySvc.AddRule(&n5model.TrafficPolicyRule{
		PolicyId:   policy.Id,
		RuleType:   ruleTypeDomain,
		MatchMode:  domainModeExact,
		MatchValue: "a.example.com",
		TargetType: targetTypeEgress,
		TargetId:   egressA.Id,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("create rule a failed: %v", err)
	}
	ruleB, err := policySvc.AddRule(&n5model.TrafficPolicyRule{
		PolicyId:   policy.Id,
		RuleType:   ruleTypeDomain,
		MatchMode:  domainModeSuffix,
		MatchValue: "example.org",
		TargetType: targetTypeEgress,
		TargetId:   egressB.Id,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("create rule b failed: %v", err)
	}

	updatedPolicy, err := policySvc.UpdatePolicy(&n5model.TrafficPolicy{
		Id:                policy.Id,
		Name:              "manage-policy-updated",
		Remark:            "phase35c",
		Enabled:           true,
		DefaultTargetType: targetTypeEgress,
		DefaultTargetId:   egressB.Id,
	})
	if err != nil {
		t.Fatalf("update policy failed: %v", err)
	}
	if updatedPolicy.Name != "manage-policy-updated" || updatedPolicy.DefaultTargetId != egressB.Id {
		t.Fatalf("unexpected updated policy: %#v", updatedPolicy)
	}

	disabledPolicy, err := policySvc.DisablePolicy(policy.Id)
	if err != nil {
		t.Fatalf("disable policy failed: %v", err)
	}
	if disabledPolicy.Enabled {
		t.Fatalf("expected policy disabled: %#v", disabledPolicy)
	}
	enabledPolicy, err := policySvc.EnablePolicy(policy.Id)
	if err != nil {
		t.Fatalf("enable policy failed: %v", err)
	}
	if !enabledPolicy.Enabled {
		t.Fatalf("expected policy enabled: %#v", enabledPolicy)
	}

	updatedRule, err := policySvc.UpdateRule(&n5model.TrafficPolicyRule{
		Id:         ruleA.Id,
		RuleType:   ruleTypeDomain,
		MatchMode:  domainModeKeyword,
		MatchValue: "updated",
		TargetType: targetTypeEgress,
		TargetId:   egressB.Id,
		SortOrder:  2,
	})
	if err != nil {
		t.Fatalf("update rule failed: %v", err)
	}
	if updatedRule.MatchMode != domainModeKeyword || updatedRule.MatchValue != "updated" || updatedRule.TargetId != egressB.Id {
		t.Fatalf("unexpected updated rule: %#v", updatedRule)
	}

	disabledRule, err := policySvc.DisableRule(ruleB.Id)
	if err != nil {
		t.Fatalf("disable rule failed: %v", err)
	}
	if disabledRule.Enabled {
		t.Fatalf("expected rule disabled: %#v", disabledRule)
	}
	enabledRule, err := policySvc.EnableRule(ruleB.Id)
	if err != nil {
		t.Fatalf("enable rule failed: %v", err)
	}
	if !enabledRule.Enabled {
		t.Fatalf("expected rule enabled: %#v", enabledRule)
	}

	if err := policySvc.ReorderRules(policy.Id, []int{ruleB.Id, ruleA.Id}); err != nil {
		t.Fatalf("reorder rules failed: %v", err)
	}
	rules, err := policySvc.ListRules(policy.Id)
	if err != nil {
		t.Fatalf("list rules failed: %v", err)
	}
	if len(rules) != 2 || rules[0].Id != ruleB.Id || rules[1].Id != ruleA.Id {
		t.Fatalf("unexpected rule order: %#v", rules)
	}

	binding, err := policySvc.BindInboundPolicy(inbound.Id, policy.Id)
	if err != nil {
		t.Fatalf("bind policy failed: %v", err)
	}
	if binding.InboundId != inbound.Id || binding.PolicyId != policy.Id {
		t.Fatalf("unexpected binding: %#v", binding)
	}
	if err := policySvc.UnbindInboundPolicy(inbound.Id); err != nil {
		t.Fatalf("unbind policy failed: %v", err)
	}
	bindings, err := policySvc.ListBindings()
	if err != nil {
		t.Fatalf("list bindings failed: %v", err)
	}
	if len(bindings) != 0 {
		t.Fatalf("expected no bindings after unbind: %#v", bindings)
	}

	policyB, err := policySvc.Create(&n5model.TrafficPolicy{
		Name:              "manage-policy-b",
		Enabled:           true,
		DefaultTargetType: targetTypeEgress,
		DefaultTargetId:   egressA.Id,
	})
	if err != nil {
		t.Fatalf("create second policy failed: %v", err)
	}
	if _, err := policySvc.RebindInboundPolicy(inbound.Id, policy.Id); err != nil {
		t.Fatalf("rebind to policy a failed: %v", err)
	}
	if _, err := policySvc.RebindInboundPolicy(inbound.Id, policyB.Id); err != nil {
		t.Fatalf("rebind to policy b failed: %v", err)
	}
	bindings, err = policySvc.ListBindings()
	if err != nil {
		t.Fatalf("list bindings after rebind failed: %v", err)
	}
	if len(bindings) != 1 || bindings[0].InboundId != inbound.Id || bindings[0].PolicyId != policyB.Id {
		t.Fatalf("expected one effective binding after rebind: %#v", bindings)
	}

	if err := policySvc.DeletePolicy(policy.Id); err != nil {
		t.Fatalf("delete policy failed: %v", err)
	}
	var policyCount int64
	if err := database.GetDB().Model(&n5model.TrafficPolicy{}).Where("id = ?", policy.Id).Count(&policyCount).Error; err != nil {
		t.Fatalf("count deleted policy failed: %v", err)
	}
	if policyCount != 0 {
		t.Fatalf("expected deleted policy count 0, got %d", policyCount)
	}
	var inboundCount int64
	if err := database.GetDB().Model(&legacyModel.Inbound{}).Where("id = ?", inbound.Id).Count(&inboundCount).Error; err != nil {
		t.Fatalf("count legacy inbound failed: %v", err)
	}
	if inboundCount != 1 {
		t.Fatalf("expected legacy inbound unchanged, got %d", inboundCount)
	}
}

func TestTrafficPolicyDisableExcludesRulesFromXrayFragments(t *testing.T) {
	initTestDB(t)

	egressSvc := &EgressService{}
	policySvc := &TrafficPolicyService{}
	extSvc := &XrayExtService{}

	egress := createTrafficTestEgress(t, egressSvc, "fragment-egress")
	inbound := createTrafficTestInbound(t, 34111, "fragment-inbound")

	policy, err := policySvc.Create(&n5model.TrafficPolicy{
		Name:              "fragment-policy",
		Enabled:           true,
		DefaultTargetType: targetTypeEgress,
		DefaultTargetId:   egress.Id,
	})
	if err != nil {
		t.Fatalf("create policy failed: %v", err)
	}
	rule, err := policySvc.AddRule(&n5model.TrafficPolicyRule{
		PolicyId:   policy.Id,
		RuleType:   ruleTypeDomain,
		MatchMode:  domainModeExact,
		MatchValue: "fragment.example.com",
		TargetType: targetTypeEgress,
		TargetId:   egress.Id,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("create rule failed: %v", err)
	}
	if _, err := policySvc.BindInboundPolicy(inbound.Id, policy.Id); err != nil {
		t.Fatalf("bind policy failed: %v", err)
	}

	routing, err := extSvc.GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("generate routing failed: %v", err)
	}
	rules := routing["rules"].([]interface{})
	if len(rules) != 2 {
		t.Fatalf("expected 2 routing rules before disable, got %d", len(rules))
	}

	if _, err := policySvc.DisableRule(rule.Id); err != nil {
		t.Fatalf("disable rule failed: %v", err)
	}
	routing, err = extSvc.GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("generate routing after rule disable failed: %v", err)
	}
	rules = routing["rules"].([]interface{})
	if len(rules) != 1 {
		t.Fatalf("expected only default routing rule after rule disable, got %d", len(rules))
	}

	if _, err := policySvc.EnableRule(rule.Id); err != nil {
		t.Fatalf("enable rule failed: %v", err)
	}
	if _, err := policySvc.DisablePolicy(policy.Id); err != nil {
		t.Fatalf("disable policy failed: %v", err)
	}
	routing, err = extSvc.GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("generate routing after policy disable failed: %v", err)
	}
	rules = routing["rules"].([]interface{})
	if len(rules) != 0 {
		t.Fatalf("expected no routing rules after policy disable, got %d", len(rules))
	}
}

func TestAdvancedTrafficPolicyRuleOrderIsPreserved(t *testing.T) {
	initTestDB(t)

	egressSvc := &EgressService{}
	policySvc := &TrafficPolicyService{}
	extSvc := &XrayExtService{}

	egressA := createTrafficTestEgress(t, egressSvc, "advanced-a")
	egressB := createTrafficTestEgress(t, egressSvc, "advanced-b")
	inbound := createTrafficTestInbound(t, 34112, "advanced-order-inbound")

	policy, err := policySvc.Create(&n5model.TrafficPolicy{
		Name:    "advanced-order-policy",
		Remark:  "ordinary-advanced-policy",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("create policy failed: %v", err)
	}
	if _, err := policySvc.AddRule(&n5model.TrafficPolicyRule{
		PolicyId:   policy.Id,
		RuleType:   ruleTypeDomain,
		MatchMode:  domainModeSuffix,
		MatchValue: "wide.example.com",
		TargetType: targetTypeEgress,
		TargetId:   egressA.Id,
		SortOrder:  1,
		Enabled:    true,
	}); err != nil {
		t.Fatalf("create wide rule failed: %v", err)
	}
	if _, err := policySvc.AddRule(&n5model.TrafficPolicyRule{
		PolicyId:   policy.Id,
		RuleType:   ruleTypeDomain,
		MatchMode:  domainModeExact,
		MatchValue: "api.wide.example.com",
		TargetType: targetTypeEgress,
		TargetId:   egressB.Id,
		SortOrder:  2,
		Enabled:    true,
	}); err != nil {
		t.Fatalf("create exact rule failed: %v", err)
	}
	if _, err := policySvc.BindInboundPolicy(inbound.Id, policy.Id); err != nil {
		t.Fatalf("bind policy failed: %v", err)
	}

	routing, err := extSvc.GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("generate routing failed: %v", err)
	}
	got := trafficInboundRoutingMatchersAndTargets(t, routing, inbound.Tag)
	want := []string{
		"domain:wide.example.com=>" + egressA.Tag,
		"full:api.wide.example.com=>" + egressB.Tag,
	}
	if len(got) != len(want) {
		t.Fatalf("unexpected routing count: got=%v want=%v", got, want)
	}
	for idx := range want {
		if got[idx] != want[idx] {
			t.Fatalf("advanced routing order changed: got=%v want=%v", got, want)
		}
	}
}

func TestTrafficTemplateCreatedPolicyCanBeEdited(t *testing.T) {
	initTestDB(t)

	egressSvc := &EgressService{}
	policySvc := &TrafficPolicyService{}
	templateSvc := &TrafficTemplateService{}

	egress := createTrafficTestEgress(t, egressSvc, "template-edit-egress")
	inbound := createTrafficTestInbound(t, 34121, "template-edit-inbound")

	result, err := templateSvc.Create(&TrafficTemplateCreateRequest{
		TemplateName: "ai",
		PolicyName:   "Template Edit Policy",
		InboundId:    inbound.Id,
		TargetType:   targetTypeEgress,
		TargetId:     egress.Id,
	})
	if err != nil {
		t.Fatalf("create template policy failed: %v", err)
	}

	updatedPolicy, err := policySvc.UpdatePolicy(&n5model.TrafficPolicy{
		Id:                result.Policy.Id,
		Name:              "Template Edit Policy Updated",
		Remark:            "editable",
		Enabled:           true,
		DefaultTargetType: targetTypeEgress,
		DefaultTargetId:   egress.Id,
	})
	if err != nil {
		t.Fatalf("update template policy failed: %v", err)
	}
	if updatedPolicy.Name != "Template Edit Policy Updated" {
		t.Fatalf("unexpected updated template policy: %#v", updatedPolicy)
	}

	firstRule := result.Rules[0]
	updatedRule, err := policySvc.UpdateRule(&n5model.TrafficPolicyRule{
		Id:         firstRule.Id,
		RuleType:   firstRule.RuleType,
		MatchMode:  domainModeKeyword,
		MatchValue: "openai",
		TargetType: firstRule.TargetType,
		TargetId:   firstRule.TargetId,
		SortOrder:  firstRule.SortOrder,
	})
	if err != nil {
		t.Fatalf("update template rule failed: %v", err)
	}
	if updatedRule.MatchMode != domainModeKeyword || updatedRule.MatchValue != "openai" {
		t.Fatalf("unexpected updated template rule: %#v", updatedRule)
	}
}

func TestTrafficPolicyServiceRejectsRemarkTransitionForSimpleManagedPolicy(t *testing.T) {
	initTestDB(t)

	policySvc := &TrafficPolicyService{}
	policy, err := policySvc.Create(&n5model.TrafficPolicy{
		Name:    "simple-managed",
		Remark:  "n5-simple-exec|eyJ2ZXJzaW9uIjoxLCJpdGVtcyI6W119",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("create policy failed: %v", err)
	}

	_, err = policySvc.UpdatePolicy(&n5model.TrafficPolicy{
		Id:      policy.Id,
		Name:    policy.Name,
		Remark:  "ordinary-remark",
		Enabled: true,
	})
	if err == nil || !strings.Contains(err.Error(), "Simple 出口规则管理") {
		t.Fatalf("unexpected update error: %v", err)
	}
}

func TestTrafficPolicyServiceAdvancedMethodsProtectSimpleManagedState(t *testing.T) {
	initTestDB(t)

	egressSvc := &EgressService{}
	policySvc := &TrafficPolicyService{}
	egress := createTrafficTestEgress(t, egressSvc, "advanced-protect-egress")
	inboundSimple := createTrafficTestInbound(t, 34151, "advanced-protect-simple")
	inboundOrdinary := createTrafficTestInbound(t, 34152, "advanced-protect-ordinary")

	simplePolicy, err := policySvc.Create(&n5model.TrafficPolicy{
		Name:              "simple-managed-policy",
		Remark:            "n5-simple-exec|eyJ2ZXJzaW9uIjoxLCJpdGVtcyI6W119",
		Enabled:           true,
		DefaultTargetType: targetTypeEgress,
		DefaultTargetId:   egress.Id,
	})
	if err != nil {
		t.Fatalf("create simple policy failed: %v", err)
	}
	simpleRule, err := policySvc.AddRule(&n5model.TrafficPolicyRule{
		PolicyId:   simplePolicy.Id,
		RuleType:   ruleTypeDomain,
		MatchMode:  domainModeExact,
		MatchValue: "simple.example.com",
		TargetType: targetTypeEgress,
		TargetId:   egress.Id,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("create simple rule failed: %v", err)
	}
	if _, err := policySvc.BindInboundPolicy(inboundSimple.Id, simplePolicy.Id); err != nil {
		t.Fatalf("bind simple policy failed: %v", err)
	}

	ordinaryPolicy, err := policySvc.CreatePolicyFromAdvanced(&n5model.TrafficPolicy{
		Name:              "ordinary-policy",
		Remark:            "ordinary-advanced",
		Enabled:           true,
		DefaultTargetType: targetTypeEgress,
		DefaultTargetId:   egress.Id,
	})
	if err != nil {
		t.Fatalf("create ordinary policy via advanced failed: %v", err)
	}
	ordinaryRule, err := policySvc.AddRuleFromAdvanced(&n5model.TrafficPolicyRule{
		PolicyId:   ordinaryPolicy.Id,
		RuleType:   ruleTypeDomain,
		MatchMode:  domainModeExact,
		MatchValue: "ordinary.example.com",
		TargetType: targetTypeEgress,
		TargetId:   egress.Id,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("create ordinary rule via advanced failed: %v", err)
	}
	if _, err := policySvc.BindInboundPolicyFromAdvanced(inboundOrdinary.Id, ordinaryPolicy.Id); err != nil {
		t.Fatalf("bind ordinary policy via advanced failed: %v", err)
	}

	expectSimpleManagedReject := func(name string, err error) {
		t.Helper()
		if err == nil || !strings.Contains(err.Error(), "n6-ui 简易出口规则管理") {
			t.Fatalf("%s: expected simple-managed rejection, got %v", name, err)
		}
	}

	_, err = policySvc.CreatePolicyFromAdvanced(&n5model.TrafficPolicy{Name: "fake-simple", Remark: simpleManagedExecRemarkPrefix + "e30=", Enabled: true})
	expectSimpleManagedReject("create fake simple policy", err)
	_, err = policySvc.UpdatePolicyFromAdvanced(&n5model.TrafficPolicy{
		Id:                simplePolicy.Id,
		Name:              "simple-updated",
		Remark:            simplePolicy.Remark,
		Enabled:           true,
		DefaultTargetType: targetTypeEgress,
		DefaultTargetId:   egress.Id,
	})
	expectSimpleManagedReject("update simple policy", err)
	_, err = policySvc.DisablePolicyFromAdvanced(simplePolicy.Id)
	expectSimpleManagedReject("disable simple policy", err)
	_, err = policySvc.EnablePolicyFromAdvanced(simplePolicy.Id)
	expectSimpleManagedReject("enable simple policy", err)
	err = policySvc.DeletePolicyFromAdvanced(simplePolicy.Id)
	expectSimpleManagedReject("delete simple policy", err)
	_, err = policySvc.AddRuleFromAdvanced(&n5model.TrafficPolicyRule{
		PolicyId:   simplePolicy.Id,
		RuleType:   ruleTypeDomain,
		MatchMode:  domainModeExact,
		MatchValue: "new-simple.example.com",
		TargetType: targetTypeEgress,
		TargetId:   egress.Id,
		Enabled:    true,
	})
	expectSimpleManagedReject("add simple rule", err)
	_, err = policySvc.UpdateRuleFromAdvanced(&n5model.TrafficPolicyRule{
		Id:         simpleRule.Id,
		RuleType:   ruleTypeDomain,
		MatchMode:  domainModeKeyword,
		MatchValue: "changed",
		TargetType: targetTypeEgress,
		TargetId:   egress.Id,
	})
	expectSimpleManagedReject("update simple rule", err)
	_, err = policySvc.DisableRuleFromAdvanced(simpleRule.Id)
	expectSimpleManagedReject("disable simple rule", err)
	_, err = policySvc.EnableRuleFromAdvanced(simpleRule.Id)
	expectSimpleManagedReject("enable simple rule", err)
	err = policySvc.DeleteRuleFromAdvanced(simpleRule.Id)
	expectSimpleManagedReject("delete simple rule", err)
	err = policySvc.ReorderRulesFromAdvanced(simplePolicy.Id, []int{simpleRule.Id})
	expectSimpleManagedReject("reorder simple rules", err)
	_, err = policySvc.RebindInboundPolicyFromAdvanced(inboundOrdinary.Id, simplePolicy.Id)
	expectSimpleManagedReject("rebind to simple policy", err)
	_, err = policySvc.RebindInboundPolicyFromAdvanced(inboundSimple.Id, ordinaryPolicy.Id)
	expectSimpleManagedReject("rebind simple inbound", err)
	err = policySvc.UnbindInboundPolicyFromAdvanced(inboundSimple.Id)
	expectSimpleManagedReject("unbind simple inbound", err)

	updatedOrdinaryRule, err := policySvc.UpdateRuleFromAdvanced(&n5model.TrafficPolicyRule{
		Id:         ordinaryRule.Id,
		PolicyId:   ordinaryPolicy.Id,
		RuleType:   ruleTypeDomain,
		MatchMode:  domainModeKeyword,
		MatchValue: "ordinary-updated",
		TargetType: targetTypeEgress,
		TargetId:   egress.Id,
		SortOrder:  1,
	})
	if err != nil {
		t.Fatalf("update ordinary rule via advanced failed: %v", err)
	}
	if updatedOrdinaryRule.MatchValue != "ordinary-updated" {
		t.Fatalf("ordinary rule was not updated: %#v", updatedOrdinaryRule)
	}
	if _, err := policySvc.UpdateRuleFromAdvanced(&n5model.TrafficPolicyRule{
		Id:         simpleRule.Id,
		PolicyId:   ordinaryPolicy.Id,
		RuleType:   ruleTypeDomain,
		MatchMode:  domainModeKeyword,
		MatchValue: "cross-policy",
		TargetType: targetTypeEgress,
		TargetId:   egress.Id,
	}); err == nil || !strings.Contains(err.Error(), "rule not found in policy") {
		t.Fatalf("expected cross-policy rule rejection, got %v", err)
	}
	if err := policySvc.UnbindInboundPolicyFromAdvanced(inboundOrdinary.Id); err != nil {
		t.Fatalf("unbind ordinary inbound via advanced failed: %v", err)
	}

	var simpleRuleCount int64
	if err := database.GetDB().Model(&n5model.TrafficPolicyRule{}).Where("id = ?", simpleRule.Id).Count(&simpleRuleCount).Error; err != nil {
		t.Fatalf("count simple rule failed: %v", err)
	}
	if simpleRuleCount != 1 {
		t.Fatalf("simple rule was mutated or deleted")
	}
	var simpleBindingCount int64
	if err := database.GetDB().Model(&n5model.TrafficPolicyBinding{}).Where("inbound_id = ? and policy_id = ?", inboundSimple.Id, simplePolicy.Id).Count(&simpleBindingCount).Error; err != nil {
		t.Fatalf("count simple binding failed: %v", err)
	}
	if simpleBindingCount != 1 {
		t.Fatalf("simple binding was mutated or deleted")
	}
}
