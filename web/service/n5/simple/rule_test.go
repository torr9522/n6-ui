package simple

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"testing"
	"x-ui/database"
	"x-ui/database/model"
	n5model "x-ui/database/model/n5"
	n5service "x-ui/web/service/n5"
)

func TestSimpleRuleServiceAllAndAICanCoexist(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, err := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	if err != nil {
		t.Fatalf("get ai group failed: %v", err)
	}
	sg := createTestRuleEgress(t, "sg-egress")
	us := createTestRuleEgress(t, "us-egress")
	inbound := createTestRuleInbound(t, 33001, "coexist-inbound")

	allRule, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{
		InboundId:   inbound.Id,
		TrafficType: simpleTrafficAll,
		EgressId:    sg.Id,
	})
	if err != nil {
		t.Fatalf("create all rule failed: %v", err)
	}
	aiRule, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{
		InboundId: inbound.Id,
		GroupId:   aiGroup.Id,
		EgressId:  us.Id,
	})
	if err != nil {
		t.Fatalf("create ai rule failed: %v", err)
	}

	list := mustListSimpleRules(t, svc)
	if len(list.Rules) != 2 {
		t.Fatalf("unexpected rule count: %d", len(list.Rules))
	}
	if findAllRule(list.Rules, inbound.Id) == nil || findGroupRule(list.Rules, inbound.Id, simpleTrafficAI) == nil {
		t.Fatalf("expected all + ai rows, got %#v", list.Rules)
	}

	ctx := mustLoadSimplePolicyContext(t, svc, inbound.Id)
	if ctx.Policy.DefaultTargetId != sg.Id {
		t.Fatalf("unexpected default target: %#v", ctx.Policy)
	}
	if ctx.ExecRemark == nil || len(ctx.ExecRemark.Items) != 1 {
		t.Fatalf("unexpected execution items: %#v", ctx.ExecRemark)
	}
	item := ctx.ExecRemark.Items[0]
	if item.GroupId != aiGroup.Id || item.GroupType != simpleTrafficAI {
		t.Fatalf("unexpected exec item: %#v", item)
	}
	if len(item.RuleIDs) != aiGroup.RuleCount {
		t.Fatalf("unexpected ai snapshot size: %d", len(item.RuleIDs))
	}

	fragments, err := (&n5service.XrayExtService{}).GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("generate routing fragments failed: %v", err)
	}
	assertSimpleRuleFragment(t, fragments, inbound.Tag, us.Tag, "domain:openai.com", true)
	assertDefaultRoute(t, fragments, inbound.Tag, sg.Tag, true)
	last := lastInboundRule(t, fragments, inbound.Tag)
	if last["outboundTag"] != sg.Tag {
		t.Fatalf("expected default route last, got %#v", last)
	}
	if allRule.RuleId == aiRule.RuleId {
		t.Fatalf("rule ids should be unique: %#v %#v", allRule, aiRule)
	}
}

func TestSimpleRuleServiceAllAIAndGameCanCoexist(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	gameGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficGame)
	sg := createTestRuleEgress(t, "sg-egress")
	us := createTestRuleEgress(t, "us-egress")
	jp := createTestRuleEgress(t, "jp-egress")
	inbound := createTestRuleInbound(t, 33002, "multi-group-inbound")

	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficAll, EgressId: sg.Id}); err != nil {
		t.Fatalf("create all failed: %v", err)
	}
	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: aiGroup.Id, EgressId: us.Id}); err != nil {
		t.Fatalf("create ai failed: %v", err)
	}
	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: gameGroup.Id, EgressId: jp.Id}); err != nil {
		t.Fatalf("create game failed: %v", err)
	}

	list := mustListSimpleRules(t, svc)
	if len(list.Rules) != 3 {
		t.Fatalf("unexpected rule count: %d", len(list.Rules))
	}
	if findAllRule(list.Rules, inbound.Id) == nil || findGroupRule(list.Rules, inbound.Id, simpleTrafficAI) == nil || findGroupRule(list.Rules, inbound.Id, simpleTrafficGame) == nil {
		t.Fatalf("expected all + ai + game rows, got %#v", list.Rules)
	}

	ctx := mustLoadSimplePolicyContext(t, svc, inbound.Id)
	if len(ctx.ExecRemark.Items) != 2 {
		t.Fatalf("unexpected exec item count: %#v", ctx.ExecRemark)
	}
	if len(ctx.Rules) != aiGroup.RuleCount+gameGroup.RuleCount {
		t.Fatalf("unexpected merged rule count: %d", len(ctx.Rules))
	}
}

func TestSimpleRuleServiceTransactionalCreateRollback(t *testing.T) {
	initSimpleTestDB(t)

	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	egress := createTestRuleEgress(t, "tx-create-egress")
	inboundA := createTestRuleInbound(t, 33201, "tx-create-a")
	inboundB := createTestRuleInbound(t, 33202, "tx-create-b")
	if _, err := NewRuleService().CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inboundB.Id, TrafficType: simpleTrafficAll, EgressId: egress.Id}); err != nil {
		t.Fatalf("create inbound b baseline failed: %v", err)
	}

	before := captureSimpleRuleDBState(t)
	fail := errors.New("injected create failure")
	svc := NewRuleService()
	svc.mutationTestHook = func(stage string) error {
		if stage == simpleMutationHookAfterEnsurePolicy {
			return fail
		}
		return nil
	}
	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inboundA.Id, TrafficType: simpleTrafficAll, EgressId: egress.Id}); !errors.Is(err, fail) {
		t.Fatalf("unexpected create policy rollback error: %v", err)
	}
	assertSimpleRuleDBStateEqual(t, before)

	svc = NewRuleService()
	svc.mutationTestHook = func(stage string) error {
		if stage == simpleMutationHookAfterAppendRules {
			return fail
		}
		return nil
	}
	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inboundA.Id, GroupId: aiGroup.Id, EgressId: egress.Id}); !errors.Is(err, fail) {
		t.Fatalf("unexpected create rule rollback error: %v", err)
	}
	assertSimpleRuleDBStateEqual(t, before)

	ctxB := mustLoadSimplePolicyContext(t, NewRuleService(), inboundB.Id)
	if ctxB.Policy.DefaultTargetId != egress.Id || len(ctxB.ExecRemark.Items) != 0 {
		t.Fatalf("inbound b changed after inbound a rollback: %#v %#v", ctxB.Policy, ctxB.ExecRemark)
	}
}

func TestSimpleRuleServiceTransactionalUpdateRollback(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	egressA := createTestRuleEgress(t, "tx-update-a")
	egressB := createTestRuleEgress(t, "tx-update-b")
	inbound := createTestRuleInbound(t, 33203, "tx-update-inbound")

	allRule, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficAll, EgressId: egressA.Id})
	if err != nil {
		t.Fatalf("create all failed: %v", err)
	}
	customRule, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{
		InboundId:    inbound.Id,
		TrafficType:  simpleTrafficCustomDomain,
		CustomDomain: "full:tx.example.com",
		EgressId:     egressA.Id,
	})
	if err != nil {
		t.Fatalf("create custom failed: %v", err)
	}

	before := captureSimpleRuleDBState(t)
	fail := errors.New("injected update failure")
	failingSvc := NewRuleService()
	failingSvc.mutationTestHook = func(stage string) error {
		if stage == simpleMutationHookAfterPolicyUpdate {
			return fail
		}
		return nil
	}
	if _, err := failingSvc.UpdateSimpleRule(allRule.RuleId, &CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficAll, EgressId: egressB.Id}); !errors.Is(err, fail) {
		t.Fatalf("unexpected update all rollback error: %v", err)
	}
	assertSimpleRuleDBStateEqual(t, before)

	failingSvc = NewRuleService()
	failingSvc.mutationTestHook = func(stage string) error {
		if stage == simpleMutationHookAfterRuleUpdate {
			return fail
		}
		return nil
	}
	if _, err := failingSvc.UpdateSimpleRule(customRule.RuleId, &CreateSimpleRuleRequest{
		InboundId:    inbound.Id,
		TrafficType:  simpleTrafficCustomDomain,
		CustomDomain: "domain:changed.example.com",
		EgressId:     egressB.Id,
	}); !errors.Is(err, fail) {
		t.Fatalf("unexpected update custom rollback error: %v", err)
	}
	assertSimpleRuleDBStateEqual(t, before)
}

func TestSimpleRuleServiceTransactionalDeleteRollback(t *testing.T) {
	initSimpleTestDB(t)

	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	svc := NewRuleService()
	egress := createTestRuleEgress(t, "tx-delete-egress")
	inbound := createTestRuleInbound(t, 33204, "tx-delete-inbound")

	allRule, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficAll, EgressId: egress.Id})
	if err != nil {
		t.Fatalf("create all failed: %v", err)
	}
	aiRule, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: aiGroup.Id, EgressId: egress.Id})
	if err != nil {
		t.Fatalf("create ai failed: %v", err)
	}

	before := captureSimpleRuleDBState(t)
	fail := errors.New("injected delete failure")
	failingSvc := NewRuleService()
	failingSvc.mutationTestHook = func(stage string) error {
		if stage == simpleMutationHookAfterPolicyUpdate {
			return fail
		}
		return nil
	}
	if err := failingSvc.DeleteSimpleRule(allRule.RuleId); !errors.Is(err, fail) {
		t.Fatalf("unexpected delete all rollback error: %v", err)
	}
	assertSimpleRuleDBStateEqual(t, before)

	failingSvc = NewRuleService()
	failingSvc.mutationTestHook = func(stage string) error {
		if stage == simpleMutationHookAfterRuleDelete {
			return fail
		}
		return nil
	}
	if err := failingSvc.DeleteSimpleRule(aiRule.RuleId); !errors.Is(err, fail) {
		t.Fatalf("unexpected delete ai rollback error: %v", err)
	}
	assertSimpleRuleDBStateEqual(t, before)
}

func TestSimpleRuleServiceTransactionalNormalMutationMatrix(t *testing.T) {
	initSimpleTestDB(t)

	ruleSvc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	customGroup, err := groupSvc.CreateGroup(&CreateTrafficRuleGroupRequest{GroupType: simpleTrafficCustom, Name: "tx-custom-group"})
	if err != nil {
		t.Fatalf("create custom group failed: %v", err)
	}
	if _, err := groupSvc.AddDomainRule(&AddTrafficRuleDomainRequest{GroupId: customGroup.Id, Domain: "tx-group.example.com"}); err != nil {
		t.Fatalf("add custom group rule failed: %v", err)
	}
	customGroup, err = groupSvc.GetGroup(customGroup.Id)
	if err != nil {
		t.Fatalf("reload custom group failed: %v", err)
	}
	egressA := createTestRuleEgress(t, "tx-normal-a")
	egressB := createTestRuleEgress(t, "tx-normal-b")
	inbound := createTestRuleInbound(t, 33205, "tx-normal-inbound")

	allRule, err := ruleSvc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficAll, EgressId: egressA.Id})
	if err != nil {
		t.Fatalf("create all failed: %v", err)
	}
	customRule, err := ruleSvc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficCustomDomain, CustomDomain: "full:tx-normal.example.com", EgressId: egressA.Id})
	if err != nil {
		t.Fatalf("create direct custom failed: %v", err)
	}
	groupRule, err := ruleSvc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: customGroup.Id, EgressId: egressA.Id})
	if err != nil {
		t.Fatalf("create custom group execution failed: %v", err)
	}
	aiRule, err := ruleSvc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: aiGroup.Id, EgressId: egressA.Id})
	if err != nil {
		t.Fatalf("create ai execution failed: %v", err)
	}

	if _, err := ruleSvc.UpdateSimpleRule(allRule.RuleId, &CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficAll, EgressId: egressB.Id}); err != nil {
		t.Fatalf("update all failed: %v", err)
	}
	updatedCustomRule, err := ruleSvc.UpdateSimpleRule(customRule.RuleId, &CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficCustomDomain, CustomDomain: "domain:tx-normal.example.com", EgressId: egressB.Id})
	if err != nil {
		t.Fatalf("update direct custom failed: %v", err)
	}
	customRule = updatedCustomRule
	if _, err := ruleSvc.UpdateSimpleRule(groupRule.RuleId, &CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: customGroup.Id, EgressId: egressB.Id}); err != nil {
		t.Fatalf("update custom group execution failed: %v", err)
	}
	if _, err := ruleSvc.UpdateSimpleRule(aiRule.RuleId, &CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: aiGroup.Id, EgressId: egressB.Id}); err != nil {
		t.Fatalf("update ai execution failed: %v", err)
	}

	for _, ruleID := range []string{customRule.RuleId, groupRule.RuleId, aiRule.RuleId, allRule.RuleId} {
		if err := ruleSvc.DeleteSimpleRule(ruleID); err != nil {
			t.Fatalf("delete simple rule %s failed: %v", ruleID, err)
		}
	}
	if list := mustListSimpleRules(t, ruleSvc); len(list.Rules) != 0 {
		t.Fatalf("expected empty simple rule list after deletes: %#v", list.Rules)
	}
}

func TestSimpleRuleServiceRejectsDisabledBuiltinCreate(t *testing.T) {
	initSimpleTestDB(t)

	ruleSvc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	egressA := createTestRuleEgress(t, "disabled-builtin-a")
	egressB := createTestRuleEgress(t, "disabled-builtin-b")

	for index, groupType := range []string{simpleTrafficAI, simpleTrafficGame, simpleTrafficStreaming} {
		group, err := mustGetBuiltinGroup(groupSvc, groupType)
		if err != nil {
			t.Fatalf("get builtin %s failed: %v", groupType, err)
		}
		if _, err := groupSvc.DisableGroup(group.Id); err != nil {
			t.Fatalf("disable builtin %s failed: %v", groupType, err)
		}
		inbound := createTestRuleInbound(t, 33301+index, "disabled-builtin-"+groupType)
		_, err = ruleSvc.CreateSimpleRule(&CreateSimpleRuleRequest{
			InboundId:   inbound.Id,
			TrafficType: groupType,
			EgressId:    egressA.Id,
		})
		if err == nil || !strings.Contains(err.Error(), simpleTrafficLabel(groupType)+"已停用") {
			t.Fatalf("unexpected disabled builtin create error for %s: %v", groupType, err)
		}
	}

	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	viaIDInbound := createTestRuleInbound(t, 33311, "disabled-builtin-via-id")
	_, err := ruleSvc.CreateSimpleRule(&CreateSimpleRuleRequest{
		InboundId: viaIDInbound.Id,
		GroupId:   aiGroup.Id,
		EgressId:  egressA.Id,
	})
	if err == nil || !strings.Contains(err.Error(), "AI分流已停用") {
		t.Fatalf("unexpected disabled builtin create via group id error: %v", err)
	}

	allInbound := createTestRuleInbound(t, 33312, "disabled-builtin-all")
	if _, err := ruleSvc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: allInbound.Id, TrafficType: simpleTrafficAll, EgressId: egressA.Id}); err != nil {
		t.Fatalf("ALL should not be blocked by disabled builtin: %v", err)
	}
	customInbound := createTestRuleInbound(t, 33313, "disabled-builtin-direct")
	if _, err := ruleSvc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: customInbound.Id, TrafficType: simpleTrafficCustomDomain, CustomDomain: "full:disabled-builtin.example.com", EgressId: egressA.Id}); err != nil {
		t.Fatalf("direct custom should not be blocked by disabled builtin: %v", err)
	}
	customGroup, err := groupSvc.CreateGroup(&CreateTrafficRuleGroupRequest{GroupType: simpleTrafficCustom, Name: "disabled-builtin-custom"})
	if err != nil {
		t.Fatalf("create custom group failed: %v", err)
	}
	if _, err := groupSvc.AddDomainRule(&AddTrafficRuleDomainRequest{GroupId: customGroup.Id, Domain: "disabled-custom-group.example.com"}); err != nil {
		t.Fatalf("add custom group rule failed: %v", err)
	}
	customGroupInbound := createTestRuleInbound(t, 33314, "disabled-builtin-custom-group")
	if _, err := ruleSvc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: customGroupInbound.Id, GroupId: customGroup.Id, EgressId: egressA.Id}); err != nil {
		t.Fatalf("custom group should not be blocked by disabled builtin: %v", err)
	}

	if _, err := groupSvc.EnableGroup(aiGroup.Id); err != nil {
		t.Fatalf("enable ai group failed: %v", err)
	}
	enabledInbound := createTestRuleInbound(t, 33315, "enabled-builtin")
	aiRule, err := ruleSvc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: enabledInbound.Id, TrafficType: simpleTrafficAI, EgressId: egressA.Id})
	if err != nil {
		t.Fatalf("AI create should pass after re-enable: %v", err)
	}
	before := mustExecutionItemSnapshot(t, ruleSvc, enabledInbound.Id, simpleTrafficAI)
	if _, err := groupSvc.DisableGroup(aiGroup.Id); err != nil {
		t.Fatalf("disable ai group after snapshot failed: %v", err)
	}
	updated, err := ruleSvc.UpdateSimpleRule(aiRule.RuleId, &CreateSimpleRuleRequest{InboundId: enabledInbound.Id, GroupId: aiGroup.Id, EgressId: egressB.Id})
	if err != nil {
		t.Fatalf("existing AI snapshot target update should pass after source disabled: %v", err)
	}
	if updated.EgressId != egressB.Id {
		t.Fatalf("expected updated snapshot target: %#v", updated)
	}
	after := mustExecutionItemSnapshot(t, ruleSvc, enabledInbound.Id, simpleTrafficAI)
	if strings.Join(before, ",") != strings.Join(after, ",") {
		t.Fatalf("existing snapshot changed after disabled builtin target update: before=%v after=%v", before, after)
	}
}

func TestSimpleRuleServiceRejectsDuplicateAll(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	sg := createTestRuleEgress(t, "sg-egress")
	hk := createTestRuleEgress(t, "hk-egress")
	inbound := createTestRuleInbound(t, 33003, "duplicate-all-inbound")

	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficAll, EgressId: sg.Id}); err != nil {
		t.Fatalf("create all failed: %v", err)
	}
	_, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficAll, EgressId: hk.Id})
	if err == nil || !strings.Contains(err.Error(), "已存在默认出口") {
		t.Fatalf("unexpected duplicate all error: %v", err)
	}
}

func TestSimpleRuleServiceRejectsDuplicateAI(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	us := createTestRuleEgress(t, "us-egress")
	sg := createTestRuleEgress(t, "sg-egress")
	inbound := createTestRuleInbound(t, 33004, "duplicate-ai-inbound")

	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: aiGroup.Id, EgressId: us.Id}); err != nil {
		t.Fatalf("create ai failed: %v", err)
	}
	_, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: aiGroup.Id, EgressId: sg.Id})
	if err == nil || !strings.Contains(err.Error(), "已存在该分流规则") {
		t.Fatalf("unexpected duplicate ai error: %v", err)
	}
}

func TestSimpleRuleServiceDeleteAIKeepsAll(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	sg := createTestRuleEgress(t, "sg-egress")
	us := createTestRuleEgress(t, "us-egress")
	inbound := createTestRuleInbound(t, 33005, "delete-ai-inbound")

	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficAll, EgressId: sg.Id}); err != nil {
		t.Fatalf("create all failed: %v", err)
	}
	aiRule, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: aiGroup.Id, EgressId: us.Id})
	if err != nil {
		t.Fatalf("create ai failed: %v", err)
	}
	if err := svc.DeleteSimpleRule(aiRule.RuleId); err != nil {
		t.Fatalf("delete ai failed: %v", err)
	}

	list := mustListSimpleRules(t, svc)
	if len(list.Rules) != 1 || findAllRule(list.Rules, inbound.Id) == nil {
		t.Fatalf("unexpected rules after delete ai: %#v", list.Rules)
	}
	ctx := mustLoadSimplePolicyContext(t, svc, inbound.Id)
	if ctx.Policy.DefaultTargetId != sg.Id || len(ctx.Rules) != 0 || len(ctx.ExecRemark.Items) != 0 {
		t.Fatalf("unexpected context after delete ai: %#v %#v", ctx.Policy, ctx.ExecRemark)
	}
}

func TestSimpleRuleServiceDeleteAllKeepsAI(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	sg := createTestRuleEgress(t, "sg-egress")
	us := createTestRuleEgress(t, "us-egress")
	inbound := createTestRuleInbound(t, 33006, "delete-all-inbound")

	allRule, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficAll, EgressId: sg.Id})
	if err != nil {
		t.Fatalf("create all failed: %v", err)
	}
	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: aiGroup.Id, EgressId: us.Id}); err != nil {
		t.Fatalf("create ai failed: %v", err)
	}
	if err := svc.DeleteSimpleRule(allRule.RuleId); err != nil {
		t.Fatalf("delete all failed: %v", err)
	}

	list := mustListSimpleRules(t, svc)
	if len(list.Rules) != 1 || findGroupRule(list.Rules, inbound.Id, simpleTrafficAI) == nil {
		t.Fatalf("unexpected rules after delete all: %#v", list.Rules)
	}
	ctx := mustLoadSimplePolicyContext(t, svc, inbound.Id)
	if ctx.Policy.DefaultTargetId != 0 || len(ctx.Rules) != aiGroup.RuleCount || len(ctx.ExecRemark.Items) != 1 {
		t.Fatalf("unexpected context after delete all: %#v %#v", ctx.Policy, ctx.ExecRemark)
	}
}

func TestSimpleRuleServiceUpdateAllKeepsAISnapshot(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	sg := createTestRuleEgress(t, "sg-egress")
	us := createTestRuleEgress(t, "us-egress")
	hk := createTestRuleEgress(t, "hk-egress")
	inbound := createTestRuleInbound(t, 33007, "update-all-inbound")

	allRule, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficAll, EgressId: sg.Id})
	if err != nil {
		t.Fatalf("create all failed: %v", err)
	}
	aiRule, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: aiGroup.Id, EgressId: us.Id})
	if err != nil {
		t.Fatalf("create ai failed: %v", err)
	}
	before := mustExecutionItemSnapshot(t, svc, inbound.Id, simpleTrafficAI)

	if _, err := svc.UpdateSimpleRule(allRule.RuleId, &CreateSimpleRuleRequest{
		InboundId:   inbound.Id,
		TrafficType: simpleTrafficAll,
		EgressId:    hk.Id,
	}); err != nil {
		t.Fatalf("update all failed: %v", err)
	}

	after := mustExecutionItemSnapshot(t, svc, inbound.Id, simpleTrafficAI)
	if strings.Join(before, ",") != strings.Join(after, ",") {
		t.Fatalf("ai snapshot changed after all update: before=%v after=%v", before, after)
	}
	list := mustListSimpleRules(t, svc)
	allRow := findAllRule(list.Rules, inbound.Id)
	aiRow := findRuleByRuleID(list.Rules, aiRule.RuleId)
	if allRow == nil || allRow.EgressId != hk.Id {
		t.Fatalf("unexpected all row after update: %#v", allRow)
	}
	if aiRow == nil || aiRow.EgressId != us.Id {
		t.Fatalf("unexpected ai row after update: %#v", aiRow)
	}
}

func TestSimpleRuleServiceUpdateAITargetKeepsSnapshot(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	us := createTestRuleEgress(t, "us-egress")
	jp := createTestRuleEgress(t, "jp-egress")
	inbound := createTestRuleInbound(t, 33008, "update-ai-inbound")

	aiRule, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: aiGroup.Id, EgressId: us.Id})
	if err != nil {
		t.Fatalf("create ai failed: %v", err)
	}
	before := mustExecutionItemSnapshot(t, svc, inbound.Id, simpleTrafficAI)

	if _, err := svc.UpdateSimpleRule(aiRule.RuleId, &CreateSimpleRuleRequest{
		InboundId:   inbound.Id,
		TrafficType: simpleTrafficGroup,
		GroupId:     aiGroup.Id,
		EgressId:    jp.Id,
	}); err != nil {
		t.Fatalf("update ai failed: %v", err)
	}

	after := mustExecutionItemSnapshot(t, svc, inbound.Id, simpleTrafficAI)
	if strings.Join(before, ",") != strings.Join(after, ",") {
		t.Fatalf("ai snapshot changed after target update: before=%v after=%v", before, after)
	}
	ctx := mustLoadSimplePolicyContext(t, svc, inbound.Id)
	for _, id := range ctx.ExecRemark.Items[0].RuleIDs {
		rule := ctx.RuleMap[id]
		if rule.TargetId != jp.Id {
			t.Fatalf("unexpected rule target after ai update: %#v", rule)
		}
	}
}

func TestSimpleRuleServiceSourceGroupChangeDoesNotAlterHistoricalSnapshot(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	us := createTestRuleEgress(t, "us-egress")
	inboundA := createTestRuleInbound(t, 33009, "snapshot-a")
	inboundB := createTestRuleInbound(t, 33010, "snapshot-b")

	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inboundA.Id, GroupId: aiGroup.Id, EgressId: us.Id}); err != nil {
		t.Fatalf("create inboundA ai failed: %v", err)
	}
	before := mustExecutionItemSnapshot(t, svc, inboundA.Id, simpleTrafficAI)
	if len(before) != aiGroup.RuleCount {
		t.Fatalf("unexpected initial snapshot count: %d", len(before))
	}

	if _, err := groupSvc.AddDomainRule(&AddTrafficRuleDomainRequest{
		GroupId: aiGroup.Id,
		Domain:  "full:snapshot-test.example",
	}); err != nil {
		t.Fatalf("add source domain failed: %v", err)
	}
	updatedGroup, err := groupSvc.GetGroup(aiGroup.Id)
	if err != nil {
		t.Fatalf("reload ai group failed: %v", err)
	}
	if updatedGroup.RuleCount != aiGroup.RuleCount+1 {
		t.Fatalf("unexpected updated group count: %d", updatedGroup.RuleCount)
	}

	afterOld := mustExecutionItemSnapshot(t, svc, inboundA.Id, simpleTrafficAI)
	if strings.Join(before, ",") != strings.Join(afterOld, ",") {
		t.Fatalf("historical snapshot changed: before=%v after=%v", before, afterOld)
	}
	if containsSnapshotValue(afterOld, "snapshot-test.example") {
		t.Fatalf("historical snapshot should not contain new domain: %v", afterOld)
	}

	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inboundB.Id, GroupId: aiGroup.Id, EgressId: us.Id}); err != nil {
		t.Fatalf("create inboundB ai failed: %v", err)
	}
	afterNew := mustExecutionItemSnapshot(t, svc, inboundB.Id, simpleTrafficAI)
	if len(afterNew) != updatedGroup.RuleCount {
		t.Fatalf("unexpected new snapshot count: %d", len(afterNew))
	}
	if !containsSnapshotValue(afterNew, "snapshot-test.example") {
		t.Fatalf("new snapshot should contain latest source domain: %v", afterNew)
	}
}

func TestSimpleRuleServiceDeleteLastItemCleansPolicyAndBinding(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	us := createTestRuleEgress(t, "us-egress")
	inbound := createTestRuleInbound(t, 33011, "cleanup-inbound")

	aiRule, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: aiGroup.Id, EgressId: us.Id})
	if err != nil {
		t.Fatalf("create ai failed: %v", err)
	}
	if err := svc.DeleteSimpleRule(aiRule.RuleId); err != nil {
		t.Fatalf("delete ai failed: %v", err)
	}

	ctx, err := svc.loadSimplePolicyContextByInbound(inbound.Id)
	if err != nil {
		t.Fatalf("load context after cleanup failed: %v", err)
	}
	if ctx != nil {
		t.Fatalf("expected context to be cleaned up, got %#v", ctx)
	}
}

func TestSimpleRuleServiceLegacySingleRuleRemainsCompatible(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	us := createTestRuleEgress(t, "us-egress")
	sg := createTestRuleEgress(t, "sg-egress")
	jp := createTestRuleEgress(t, "jp-egress")
	inbound := createTestRuleInbound(t, 33012, "legacy-inbound")

	legacyPolicyID := createLegacySimpleAI(t, inbound, us.Id)
	list := mustListSimpleRules(t, svc)
	if len(list.Rules) != 1 {
		t.Fatalf("unexpected legacy list count: %d", len(list.Rules))
	}
	legacyRow := list.Rules[0]
	if !strings.HasPrefix(legacyRow.RuleId, legacySimpleRuleIDPrefix) {
		t.Fatalf("expected legacy rule id, got %#v", legacyRow)
	}

	if _, err := svc.UpdateSimpleRule(legacyRow.RuleId, &CreateSimpleRuleRequest{
		InboundId:   inbound.Id,
		TrafficType: legacyRow.TrafficType,
		GroupId:     aiGroup.Id,
		EgressId:    jp.Id,
	}); err != nil {
		t.Fatalf("update legacy ai failed: %v", err)
	}
	ctx := mustLoadSimplePolicyContext(t, svc, inbound.Id)
	if ctx.Policy.Id != legacyPolicyID || ctx.LegacyMeta == nil {
		t.Fatalf("expected legacy policy to remain before conversion: %#v", ctx)
	}
	for _, rule := range ctx.Rules {
		if rule.TargetId != jp.Id {
			t.Fatalf("unexpected legacy target after update: %#v", rule)
		}
	}

	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{
		InboundId:   inbound.Id,
		TrafficType: simpleTrafficAll,
		EgressId:    sg.Id,
	}); err != nil {
		t.Fatalf("add all to legacy policy failed: %v", err)
	}
	ctx = mustLoadSimplePolicyContext(t, svc, inbound.Id)
	if ctx.Policy.Id != legacyPolicyID || ctx.ExecRemark == nil || ctx.LegacyMeta != nil {
		t.Fatalf("expected legacy policy conversion in-place, got %#v", ctx)
	}
	if ctx.Policy.DefaultTargetId != sg.Id || len(ctx.ExecRemark.Items) != 1 {
		t.Fatalf("unexpected converted context: %#v %#v", ctx.Policy, ctx.ExecRemark)
	}
	list = mustListSimpleRules(t, svc)
	if len(list.Rules) != 2 {
		t.Fatalf("unexpected rule count after legacy conversion: %d", len(list.Rules))
	}
}

func TestSimpleRemarkKindsAreStrictlySeparated(t *testing.T) {
	validExec, _, err := buildSimpleExecutionRemark(&simpleExecutionRemark{
		Version: simpleExecutionRemarkVersion,
		Items: []*simpleExecutionItem{
			{TrafficType: simpleTrafficGroup, GroupId: 1, GroupType: simpleTrafficAI, RuleIDs: []int{1}},
		},
	})
	if err != nil {
		t.Fatalf("build exec remark failed: %v", err)
	}

	cases := []struct {
		name       string
		remark     string
		wantNew    bool
		wantLegacy bool
		wantOrd    bool
	}{
		{name: "new exec", remark: validExec, wantNew: true},
		{name: "legacy", remark: "n5-simple|type=group|groupId=1|groupName=AI%E5%88%86%E6%B5%81|groupType=ai", wantLegacy: true},
		{name: "ordinary", remark: "my custom policy", wantOrd: true},
		{name: "simple lookalike", remark: "n5-simple-test", wantOrd: true},
		{name: "prefixed exec text", remark: "foo n5-simple-exec|xxx", wantOrd: true},
		{name: "empty", remark: "", wantOrd: true},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNewSimpleExecutionRemark(tt.remark); got != tt.wantNew {
				t.Fatalf("isNewSimpleExecutionRemark(%q)=%v want %v", tt.remark, got, tt.wantNew)
			}
			if got := isLegacySimpleRemark(tt.remark); got != tt.wantLegacy {
				t.Fatalf("isLegacySimpleRemark(%q)=%v want %v", tt.remark, got, tt.wantLegacy)
			}
			if got := isOrdinaryPolicyRemark(tt.remark); got != tt.wantOrd {
				t.Fatalf("isOrdinaryPolicyRemark(%q)=%v want %v", tt.remark, got, tt.wantOrd)
			}
		})
	}
}

func TestDecodeSimpleExecutionRemarkFailuresAreSafe(t *testing.T) {
	cases := []struct {
		name   string
		remark string
	}{
		{name: "empty payload", remark: "n5-simple-exec|"},
		{name: "invalid base64", remark: "n5-simple-exec|@@@@"},
		{name: "invalid json", remark: "n5-simple-exec|" + base64.RawURLEncoding.EncodeToString([]byte(`{"version":1,"items":[`))},
		{name: "unsupported version", remark: "n5-simple-exec|" + base64.RawURLEncoding.EncodeToString([]byte(`{"version":2,"items":[]}`))},
		{name: "duplicate item", remark: "n5-simple-exec|" + base64.RawURLEncoding.EncodeToString([]byte(`{"version":1,"items":[{"trafficType":"group","groupId":1,"groupType":"ai","ruleIds":[1]},{"trafficType":"group","groupId":1,"groupType":"ai","ruleIds":[2]}]}`))},
		{name: "invalid rule id", remark: "n5-simple-exec|" + base64.RawURLEncoding.EncodeToString([]byte(`{"version":1,"items":[{"trafficType":"group","groupId":1,"groupType":"ai","ruleIds":[0]}]}`))},
		{name: "wrong rule id type", remark: "n5-simple-exec|" + base64.RawURLEncoding.EncodeToString([]byte(`{"version":1,"items":[{"trafficType":"group","groupId":1,"groupType":"ai","ruleIds":["x"]}]}`))},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := decodeSimpleExecutionRemark(tt.remark); err == nil {
				t.Fatalf("expected decode error for %s", tt.name)
			}
			if _, ok := parseSimpleExecutionRemark(tt.remark); ok {
				t.Fatalf("parseSimpleExecutionRemark should fail for %s", tt.name)
			}
		})
	}
}

func TestSimpleExecutionRemarkEncodeDecodeRoundTrip(t *testing.T) {
	remark := &simpleExecutionRemark{
		Version: simpleExecutionRemarkVersion,
		Items: []*simpleExecutionItem{
			{TrafficType: simpleTrafficGroup, GroupId: 1, GroupName: "AI分流", GroupType: simpleTrafficAI, RuleIDs: []int{1, 2, 3}},
			{TrafficType: simpleTrafficCustomDomain, CustomDomain: "full:api.ipify.org", RuleIDs: []int{4}},
		},
	}

	encoded, stats, err := buildSimpleExecutionRemark(remark)
	if err != nil {
		t.Fatalf("buildSimpleExecutionRemark failed: %v", err)
	}
	if stats.TotalRemarkBytes != len(encoded) {
		t.Fatalf("unexpected total bytes: %#v", stats)
	}

	decoded, err := decodeSimpleExecutionRemark(encoded)
	if err != nil {
		t.Fatalf("decodeSimpleExecutionRemark failed: %v", err)
	}
	if decoded.Version != simpleExecutionRemarkVersion || len(decoded.Items) != 2 {
		t.Fatalf("unexpected decoded remark: %#v", decoded)
	}
	if decoded.Items[0].GroupId != 1 || decoded.Items[1].CustomDomain != "full:api.ipify.org" {
		t.Fatalf("unexpected decoded items: %#v", decoded.Items)
	}
}

func TestSimpleExecutionRemarkSizeProfiles(t *testing.T) {
	cases := []struct {
		name           string
		itemCount      int
		ruleIDsPerItem int
	}{
		{name: "normal", itemCount: 3, ruleIDsPerItem: 12},
		{name: "10x100", itemCount: 10, ruleIDsPerItem: 100},
		{name: "20x500", itemCount: 20, ruleIDsPerItem: 500},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			encoded, stats, err := buildSimpleExecutionRemark(buildSyntheticExecutionRemark(tt.itemCount, tt.ruleIDsPerItem))
			if err != nil {
				t.Fatalf("buildSimpleExecutionRemark failed: %v", err)
			}
			if encoded == "" || stats.RawJSONBytes <= 0 || stats.Base64Bytes <= 0 || stats.TotalRemarkBytes <= 0 {
				t.Fatalf("unexpected size stats: %#v", stats)
			}
			t.Logf("%s raw=%d base64=%d total=%d", tt.name, stats.RawJSONBytes, stats.Base64Bytes, stats.TotalRemarkBytes)
		})
	}
}

func TestSimpleExecutionRemarkOversizeIsRejected(t *testing.T) {
	_, _, err := buildSimpleExecutionRemark(buildSyntheticExecutionRemark(60, 2000))
	if err == nil || !strings.Contains(err.Error(), "元数据过大") {
		t.Fatalf("unexpected oversize error: %v", err)
	}
}

func TestListSimpleRulesSkipsCorruptedExecMetadataSafely(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	inbound := createTestRuleInbound(t, 33013, "corrupted-list-inbound")
	policySvc := &n5service.TrafficPolicyService{}
	policy, err := policySvc.Create(&n5model.TrafficPolicy{
		Name:    "corrupted-list-policy",
		Remark:  "n5-simple-exec|broken",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("create policy failed: %v", err)
	}
	if _, err := policySvc.BindInboundPolicy(inbound.Id, policy.Id); err != nil {
		t.Fatalf("bind policy failed: %v", err)
	}

	list, err := svc.ListSimpleRules()
	if err != nil {
		t.Fatalf("ListSimpleRules should not fail on corrupted metadata: %v", err)
	}
	if len(list.Rules) != 0 {
		t.Fatalf("expected corrupted metadata to be skipped, got %#v", list.Rules)
	}
}

func TestDeleteSimpleRuleRejectsCrossPolicyRuleIDTamper(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	us := createTestRuleEgress(t, "us-egress")
	jp := createTestRuleEgress(t, "jp-egress")
	inboundA := createTestRuleInbound(t, 33014, "tamper-a")
	inboundB := createTestRuleInbound(t, 33015, "tamper-b")

	aiRuleA, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inboundA.Id, GroupId: aiGroup.Id, EgressId: us.Id})
	if err != nil {
		t.Fatalf("create ai A failed: %v", err)
	}
	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inboundB.Id, GroupId: aiGroup.Id, EgressId: jp.Id}); err != nil {
		t.Fatalf("create ai B failed: %v", err)
	}

	ctxA := mustLoadSimplePolicyContext(t, svc, inboundA.Id)
	ctxB := mustLoadSimplePolicyContext(t, svc, inboundB.Id)
	ctxA.ExecRemark.Items[0].RuleIDs = []int{ctxB.ExecRemark.Items[0].RuleIDs[0]}
	remarkValue, _, err := buildSimpleExecutionRemark(ctxA.ExecRemark)
	if err != nil {
		t.Fatalf("build tampered remark failed: %v", err)
	}
	if _, err := (&n5service.TrafficPolicyService{}).UpdatePolicy(&n5model.TrafficPolicy{
		Id:                ctxA.Policy.Id,
		Name:              ctxA.Policy.Name,
		Remark:            remarkValue,
		Enabled:           ctxA.Policy.Enabled,
		DefaultTargetType: ctxA.Policy.DefaultTargetType,
		DefaultTargetId:   ctxA.Policy.DefaultTargetId,
	}); err != nil {
		t.Fatalf("persist tampered remark failed: %v", err)
	}

	if err := svc.DeleteSimpleRule(aiRuleA.RuleId); err == nil {
		t.Fatal("expected delete to reject cross-policy rule id tamper")
	}
	var countA, countB int64
	if err := database.GetDB().Model(&n5model.TrafficPolicyRule{}).Where("policy_id = ?", ctxA.Policy.Id).Count(&countA).Error; err != nil {
		t.Fatalf("count policy A rules failed: %v", err)
	}
	if err := database.GetDB().Model(&n5model.TrafficPolicyRule{}).Where("policy_id = ?", ctxB.Policy.Id).Count(&countB).Error; err != nil {
		t.Fatalf("count policy B rules failed: %v", err)
	}
	if countA == 0 || countB == 0 {
		t.Fatalf("tampered delete should not remove rules: countA=%d countB=%d", countA, countB)
	}
}

func TestParseCustomDomainRule(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantMode    string
		wantValue   string
		wantDisplay string
	}{
		{name: "bare domain defaults to suffix match", input: "openai.com", wantMode: "suffix", wantValue: "openai.com", wantDisplay: "domain:openai.com"},
		{name: "explicit domain keeps suffix match", input: "domain:openai.com", wantMode: "suffix", wantValue: "openai.com", wantDisplay: "domain:openai.com"},
		{name: "explicit full keeps exact match", input: "full:openai.com", wantMode: "exact", wantValue: "openai.com", wantDisplay: "full:openai.com"},
		{name: "keyword keeps keyword match", input: "keyword:openai", wantMode: "keyword", wantValue: "openai", wantDisplay: "keyword:openai"},
		{name: "regexp keeps regexp match", input: `regexp:^(.+\.)?openai\.com$`, wantMode: "regexp", wantValue: `^(.+\.)?openai\.com$`, wantDisplay: `regexp:^(.+\.)?openai\.com$`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMode, gotValue, gotDisplay, err := parseCustomDomainRule(tt.input)
			if err != nil {
				t.Fatalf("parseCustomDomainRule(%q) returned error: %v", tt.input, err)
			}
			if gotMode != tt.wantMode || gotValue != tt.wantValue || gotDisplay != tt.wantDisplay {
				t.Fatalf("parseCustomDomainRule(%q) = (%q, %q, %q), want (%q, %q, %q)",
					tt.input, gotMode, gotValue, gotDisplay, tt.wantMode, tt.wantValue, tt.wantDisplay)
			}
		})
	}
}

func TestParseCustomDomainRuleRejectsInvalidDomainInput(t *testing.T) {
	tests := []string{
		"",
		"   ",
		"https://example.com/path",
		"http://example.com",
		"example.com/path",
		"example.com:443",
		"example .com",
		"keyword:bad value",
		"regexp:[",
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			if _, _, _, err := parseCustomDomainRule(input); err == nil {
				t.Fatalf("expected parseCustomDomainRule(%q) to fail", input)
			}
		})
	}
}

func TestSimpleRuleServiceUpdateDirectCustomDomainChangesDomainAndKeepsSnapshot(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	sg := createTestRuleEgress(t, "sg-egress")
	us := createTestRuleEgress(t, "us-egress")
	inbound := createTestRuleInbound(t, 33140, "direct-custom-update-inbound")

	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: aiGroup.Id, EgressId: us.Id}); err != nil {
		t.Fatalf("create ai snapshot failed: %v", err)
	}
	before := mustExecutionItemSnapshot(t, svc, inbound.Id, simpleTrafficAI)

	customRule, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{
		InboundId:    inbound.Id,
		TrafficType:  simpleTrafficCustomDomain,
		CustomDomain: "domain:example.com",
		EgressId:     us.Id,
	})
	if err != nil {
		t.Fatalf("create direct custom rule failed: %v", err)
	}
	updatedRule, err := svc.UpdateSimpleRule(customRule.RuleId, &CreateSimpleRuleRequest{
		InboundId:    inbound.Id,
		TrafficType:  simpleTrafficCustomDomain,
		CustomDomain: "full:iana.org",
		EgressId:     sg.Id,
	})
	if err != nil {
		t.Fatalf("update direct custom rule failed: %v", err)
	}

	after := mustExecutionItemSnapshot(t, svc, inbound.Id, simpleTrafficAI)
	if strings.Join(before, ",") != strings.Join(after, ",") {
		t.Fatalf("ai snapshot changed after direct custom update: before=%v after=%v", before, after)
	}
	list := mustListSimpleRules(t, svc)
	updated := findRuleByRuleID(list.Rules, updatedRule.RuleId)
	if updated == nil || updated.CustomDomain != "full:iana.org" || updated.EgressId != sg.Id {
		t.Fatalf("unexpected updated custom row: %#v", updated)
	}
	fragments, err := (&n5service.XrayExtService{}).GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("generate routing fragments failed: %v", err)
	}
	assertSimpleRuleFragment(t, fragments, inbound.Tag, sg.Tag, "full:iana.org", true)
	assertSimpleRuleFragment(t, fragments, inbound.Tag, us.Tag, "domain:example.com", false)
	assertSimpleRuleFragment(t, fragments, inbound.Tag, us.Tag, "domain:openai.com", true)
}

func TestSimpleRuleServiceUpdateDirectCustomDomainRejectsDuplicate(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	egress := createTestRuleEgress(t, "egress")
	inbound := createTestRuleInbound(t, 33141, "direct-custom-duplicate-inbound")

	first, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{
		InboundId:    inbound.Id,
		TrafficType:  simpleTrafficCustomDomain,
		CustomDomain: "domain:example.com",
		EgressId:     egress.Id,
	})
	if err != nil {
		t.Fatalf("create first custom rule failed: %v", err)
	}
	second, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{
		InboundId:    inbound.Id,
		TrafficType:  simpleTrafficCustomDomain,
		CustomDomain: "domain:openai.com",
		EgressId:     egress.Id,
	})
	if err != nil {
		t.Fatalf("create second custom rule failed: %v", err)
	}
	if _, err := svc.UpdateSimpleRule(second.RuleId, &CreateSimpleRuleRequest{
		InboundId:    inbound.Id,
		TrafficType:  simpleTrafficCustomDomain,
		CustomDomain: "domain:example.com",
		EgressId:     egress.Id,
	}); err == nil || !strings.Contains(err.Error(), "已存在该分流规则") {
		t.Fatalf("expected duplicate custom update error, got %v; first=%s", err, first.RuleId)
	}
}

func TestSimpleRoutingPriorityIgnoresCreationOrder(t *testing.T) {
	cases := []struct {
		name  string
		steps []string
	}{
		{name: "builtin first", steps: []string{"all", "ai", "group", "suffix", "exact"}},
		{name: "direct first", steps: []string{"exact", "suffix", "group", "ai", "all"}},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			initSimpleTestDB(t)

			svc := NewRuleService()
			groupSvc := NewTrafficRuleGroupService()
			aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
			customGroup, err := groupSvc.CreateGroup(&CreateTrafficRuleGroupRequest{
				GroupType: simpleTrafficCustom,
				Name:      "custom group",
			})
			if err != nil {
				t.Fatalf("create custom group failed: %v", err)
			}
			if _, err := groupSvc.AddDomainRule(&AddTrafficRuleDomainRequest{GroupId: customGroup.Id, Domain: "domain:openai.com"}); err != nil {
				t.Fatalf("add custom group domain failed: %v", err)
			}
			customGroup, err = groupSvc.GetGroup(customGroup.Id)
			if err != nil {
				t.Fatalf("reload custom group failed: %v", err)
			}

			all := createTestRuleEgress(t, "all-egress")
			ai := createTestRuleEgress(t, "ai-egress")
			group := createTestRuleEgress(t, "group-egress")
			suffix := createTestRuleEgress(t, "suffix-egress")
			exact := createTestRuleEgress(t, "exact-egress")
			inbound := createTestRuleInbound(t, 33160, "priority-inbound")

			for _, step := range tt.steps {
				switch step {
				case "all":
					if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficAll, EgressId: all.Id}); err != nil {
						t.Fatalf("create all failed: %v", err)
					}
				case "ai":
					if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: aiGroup.Id, EgressId: ai.Id}); err != nil {
						t.Fatalf("create ai failed: %v", err)
					}
				case "group":
					if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: customGroup.Id, EgressId: group.Id}); err != nil {
						t.Fatalf("create custom group failed: %v", err)
					}
				case "suffix":
					if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficCustomDomain, CustomDomain: "domain:openai.com", EgressId: suffix.Id}); err != nil {
						t.Fatalf("create suffix failed: %v", err)
					}
				case "exact":
					if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficCustomDomain, CustomDomain: "full:api.openai.com", EgressId: exact.Id}); err != nil {
						t.Fatalf("create exact failed: %v", err)
					}
				}
			}

			fragments, err := (&n5service.XrayExtService{}).GenerateRoutingFragments()
			if err != nil {
				t.Fatalf("generate routing fragments failed: %v", err)
			}
			got := inboundRoutingMatchersAndTargets(t, fragments, inbound.Tag)
			want := []string{
				"full:api.openai.com=>" + exact.Tag,
				"domain:openai.com=>" + suffix.Tag,
				"domain:openai.com=>" + group.Tag,
				"domain:openai.com=>" + ai.Tag,
			}
			assertRoutingPrefix(t, got, want)
			if got[len(got)-1] != "*=>"+all.Tag {
				t.Fatalf("default route should be last: got=%v", got)
			}
		})
	}
}

func TestSimpleConflictPreviewDetectsOverlapAndPriorityDirection(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	all := createTestRuleEgress(t, "all-egress")
	ai := createTestRuleEgress(t, "ai-egress")
	custom := createTestRuleEgress(t, "custom-egress")
	inbound := createTestRuleInbound(t, 33161, "conflict-preview-inbound")

	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficAll, EgressId: all.Id}); err != nil {
		t.Fatalf("create all failed: %v", err)
	}
	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: aiGroup.Id, EgressId: ai.Id}); err != nil {
		t.Fatalf("create ai failed: %v", err)
	}

	preview, err := svc.CheckSimpleRuleConflicts(&CreateSimpleRuleRequest{
		InboundId:    inbound.Id,
		TrafficType:  simpleTrafficCustomDomain,
		CustomDomain: "full:api.openai.com",
		EgressId:     custom.Id,
	}, "")
	if err != nil {
		t.Fatalf("preview exact/suffix failed: %v", err)
	}
	if !preview.HasConflict || !strings.Contains(preview.Warning, "AI分流") || !strings.Contains(preview.Warning, "当前规则优先级更高") {
		t.Fatalf("unexpected exact/suffix preview: %#v", preview)
	}

	preview, err = svc.CheckSimpleRuleConflicts(&CreateSimpleRuleRequest{
		InboundId:    inbound.Id,
		TrafficType:  simpleTrafficCustomDomain,
		CustomDomain: "keyword:openai",
		EgressId:     custom.Id,
	}, "")
	if err != nil {
		t.Fatalf("preview keyword failed: %v", err)
	}
	if !preview.HasConflict || !strings.Contains(preview.Warning, "关键词规则可能") {
		t.Fatalf("unexpected keyword preview: %#v", preview)
	}

	preview, err = svc.CheckSimpleRuleConflicts(&CreateSimpleRuleRequest{
		InboundId:    inbound.Id,
		TrafficType:  simpleTrafficCustomDomain,
		CustomDomain: "regexp:.*openai.*",
		EgressId:     custom.Id,
	}, "")
	if err != nil {
		t.Fatalf("preview regexp failed: %v", err)
	}
	if !preview.HasConflict || !strings.Contains(preview.Warning, "正则表达式规则") {
		t.Fatalf("unexpected regexp preview: %#v", preview)
	}

	preview, err = svc.CheckSimpleRuleConflicts(&CreateSimpleRuleRequest{
		InboundId:    inbound.Id,
		TrafficType:  simpleTrafficCustomDomain,
		CustomDomain: "domain:example.org",
		EgressId:     custom.Id,
	}, "")
	if err != nil {
		t.Fatalf("preview no conflict failed: %v", err)
	}
	if preview.HasConflict {
		t.Fatalf("unexpected no-conflict preview: %#v", preview)
	}
}

func TestSimpleConflictPreviewExcludesEditedItem(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	egress := createTestRuleEgress(t, "egress")
	inbound := createTestRuleInbound(t, 33162, "conflict-edit-inbound")

	rule, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{
		InboundId:    inbound.Id,
		TrafficType:  simpleTrafficCustomDomain,
		CustomDomain: "domain:example.com",
		EgressId:     egress.Id,
	})
	if err != nil {
		t.Fatalf("create custom failed: %v", err)
	}
	preview, err := svc.CheckSimpleRuleConflicts(&CreateSimpleRuleRequest{
		InboundId:    inbound.Id,
		TrafficType:  simpleTrafficCustomDomain,
		CustomDomain: "domain:example.com",
		EgressId:     egress.Id,
	}, rule.RuleId)
	if err != nil {
		t.Fatalf("preview edit failed: %v", err)
	}
	if preview.HasConflict {
		t.Fatalf("edit item should not conflict with itself: %#v", preview)
	}
}

func createTestRuleEgress(t *testing.T, name string) *n5model.Egress {
	t.Helper()
	egress, err := (&n5service.EgressService{}).Create(&n5model.Egress{
		Name:         name,
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: `{"protocol":"freedom","settings":{}}`,
	})
	if err != nil {
		t.Fatalf("create egress failed: %v", err)
	}
	return egress
}

func createTestRuleInbound(t *testing.T, port int, remark string) *model.Inbound {
	t.Helper()
	inbound := &model.Inbound{
		UserId:         1,
		Remark:         remark,
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           port,
		Protocol:       model.Socks,
		Settings:       `{"auth":"noauth","udp":false,"ip":"127.0.0.1"}`,
		StreamSettings: `{}`,
		Tag:            "tag-" + strconv.Itoa(port),
		Sniffing:       `{}`,
	}
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatalf("create inbound failed: %v", err)
	}
	return inbound
}

func createLegacySimpleAI(t *testing.T, inbound *model.Inbound, egressID int) int {
	t.Helper()
	result, err := (&n5service.TrafficTemplateService{}).Create(&n5service.TrafficTemplateCreateRequest{
		TemplateName: simpleTrafficAI,
		PolicyName:   simplePolicyName(inbound, simpleTrafficAI),
		InboundId:    inbound.Id,
		TargetType:   "egress",
		TargetId:     egressID,
	})
	if err != nil {
		t.Fatalf("create legacy ai template failed: %v", err)
	}
	if err := database.GetDB().Model(&n5model.TrafficPolicy{}).Where("id = ?", result.Policy.Id).Updates(map[string]interface{}{
		"remark":              buildSimpleRuleRemark(simpleTrafficAI, ""),
		"default_target_type": "",
		"default_target_id":   0,
	}).Error; err != nil {
		t.Fatalf("mark legacy ai policy failed: %v", err)
	}
	return result.Policy.Id
}

func mustListSimpleRules(t *testing.T, svc *RuleService) *SimpleRuleListResult {
	t.Helper()
	list, err := svc.ListSimpleRules()
	if err != nil {
		t.Fatalf("list simple rules failed: %v", err)
	}
	return list
}

func mustLoadSimplePolicyContext(t *testing.T, svc *RuleService, inboundId int) *simplePolicyContext {
	t.Helper()
	ctx, err := svc.loadSimplePolicyContextByInbound(inboundId)
	if err != nil {
		t.Fatalf("load simple policy context failed: %v", err)
	}
	if ctx == nil {
		t.Fatal("expected simple policy context")
	}
	return ctx
}

func mustExecutionItemSnapshot(t *testing.T, svc *RuleService, inboundId int, groupType string) []string {
	t.Helper()
	ctx := mustLoadSimplePolicyContext(t, svc, inboundId)
	for _, item := range ctx.ExecRemark.Items {
		if normalizeSimpleGroupType(item.GroupType) != groupType {
			continue
		}
		values := make([]string, 0, len(item.RuleIDs))
		for _, ruleID := range item.RuleIDs {
			rule := ctx.RuleMap[ruleID]
			if rule != nil {
				values = append(values, rule.MatchValue)
			}
		}
		sort.Strings(values)
		return values
	}
	t.Fatalf("group item not found: %s", groupType)
	return nil
}

func findAllRule(rules []*SimpleRule, inboundId int) *SimpleRule {
	for _, rule := range rules {
		if rule.InboundId == inboundId && rule.TrafficType == simpleTrafficAll {
			return rule
		}
	}
	return nil
}

func findGroupRule(rules []*SimpleRule, inboundId int, groupType string) *SimpleRule {
	for _, rule := range rules {
		if rule.InboundId == inboundId && normalizeSimpleGroupType(rule.GroupType) == groupType {
			return rule
		}
	}
	return nil
}

func findRuleByRuleID(rules []*SimpleRule, ruleID string) *SimpleRule {
	for _, rule := range rules {
		if rule.RuleId == ruleID {
			return rule
		}
	}
	return nil
}

func containsSnapshotValue(values []string, target string) bool {
	for _, value := range values {
		if strings.Contains(value, target) {
			return true
		}
	}
	return false
}

func buildSyntheticExecutionRemark(itemCount int, ruleIDsPerItem int) *simpleExecutionRemark {
	items := make([]*simpleExecutionItem, 0, itemCount)
	for i := 0; i < itemCount; i++ {
		ruleIDs := make([]int, 0, ruleIDsPerItem)
		for j := 1; j <= ruleIDsPerItem; j++ {
			ruleIDs = append(ruleIDs, j)
		}
		items = append(items, &simpleExecutionItem{
			TrafficType: simpleTrafficGroup,
			GroupId:     i + 1,
			GroupName:   "group-" + strconv.Itoa(i+1),
			RuleIDs:     ruleIDs,
		})
	}
	return &simpleExecutionRemark{
		Version: simpleExecutionRemarkVersion,
		Items:   items,
	}
}

func inboundRoutingRules(t *testing.T, fragments map[string]interface{}, inboundTag string) []map[string]interface{} {
	t.Helper()
	rules, _ := fragments["rules"].([]interface{})
	result := make([]map[string]interface{}, 0)
	for _, item := range rules {
		rule, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		inboundTags, _ := rule["inboundTag"].([]interface{})
		if len(inboundTags) == 0 || inboundTags[0] != inboundTag {
			continue
		}
		result = append(result, rule)
	}
	return result
}

func lastInboundRule(t *testing.T, fragments map[string]interface{}, inboundTag string) map[string]interface{} {
	t.Helper()
	rules := inboundRoutingRules(t, fragments, inboundTag)
	if len(rules) == 0 {
		t.Fatalf("no inbound rules found for %s", inboundTag)
	}
	return rules[len(rules)-1]
}

func assertSimpleRuleFragment(t *testing.T, fragments map[string]interface{}, inboundTag string, outboundTag string, domain string, expect bool) {
	t.Helper()
	found := false
	for _, rule := range inboundRoutingRules(t, fragments, inboundTag) {
		domains, _ := rule["domain"].([]interface{})
		if len(domains) == 0 || domains[0] != domain {
			continue
		}
		if tag, _ := rule["outboundTag"].(string); tag == outboundTag {
			found = true
			break
		}
	}
	if expect && !found {
		t.Fatalf("expected fragment not found: inbound=%s outbound=%s domain=%s fragments=%#v", inboundTag, outboundTag, domain, fragments)
	}
	if !expect && found {
		t.Fatalf("unexpected fragment found: inbound=%s outbound=%s domain=%s", inboundTag, outboundTag, domain)
	}
}

func assertDefaultRoute(t *testing.T, fragments map[string]interface{}, inboundTag string, outboundTag string, expect bool) {
	t.Helper()
	found := false
	for _, rule := range inboundRoutingRules(t, fragments, inboundTag) {
		if _, hasDomain := rule["domain"]; hasDomain {
			continue
		}
		if _, hasIP := rule["ip"]; hasIP {
			continue
		}
		if tag, _ := rule["outboundTag"].(string); tag == outboundTag {
			found = true
			break
		}
	}
	if expect && !found {
		t.Fatalf("expected default route not found: inbound=%s outbound=%s", inboundTag, outboundTag)
	}
	if !expect && found {
		t.Fatalf("unexpected default route found: inbound=%s outbound=%s", inboundTag, outboundTag)
	}
}

func inboundRoutingMatchersAndTargets(t *testing.T, fragments map[string]interface{}, inboundTag string) []string {
	t.Helper()
	result := make([]string, 0)
	for _, rule := range inboundRoutingRules(t, fragments, inboundTag) {
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

func assertRoutingPrefix(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) < len(want) {
		t.Fatalf("routing too short: got=%v want-prefix=%v", got, want)
	}
	for index, expected := range want {
		if got[index] != expected {
			t.Fatalf("unexpected routing order at %d: got=%v want-prefix=%v", index, got, want)
		}
	}
}

func captureSimpleRuleDBState(t *testing.T) string {
	t.Helper()
	state := struct {
		Policies []*n5model.TrafficPolicy
		Bindings []*n5model.TrafficPolicyBinding
		Rules    []*n5model.TrafficPolicyRule
	}{}
	if err := database.GetDB().Model(&n5model.TrafficPolicy{}).Order("id asc").Find(&state.Policies).Error; err != nil {
		t.Fatalf("capture policies failed: %v", err)
	}
	if err := database.GetDB().Model(&n5model.TrafficPolicyBinding{}).Order("id asc").Find(&state.Bindings).Error; err != nil {
		t.Fatalf("capture bindings failed: %v", err)
	}
	if err := database.GetDB().Model(&n5model.TrafficPolicyRule{}).Order("id asc").Find(&state.Rules).Error; err != nil {
		t.Fatalf("capture rules failed: %v", err)
	}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal db state failed: %v", err)
	}
	return string(data)
}

func assertSimpleRuleDBStateEqual(t *testing.T, want string) {
	t.Helper()
	if got := captureSimpleRuleDBState(t); got != want {
		t.Fatalf("simple rule DB state changed after rollback:\ngot:  %s\nwant: %s", got, want)
	}
}
