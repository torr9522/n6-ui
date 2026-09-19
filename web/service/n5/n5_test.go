package n5

import (
	"encoding/json"
	"github.com/torr9522/n6-ui/database"
	legacyModel "github.com/torr9522/n6-ui/database/model"
	n5model "github.com/torr9522/n6-ui/database/model/n5"
	"github.com/torr9522/n6-ui/util/common"
	"path/filepath"
	"strings"
	"testing"
)

func initTestDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	if err := database.InitDB(dbPath); err != nil {
		t.Fatalf("init db failed: %v", err)
	}
	return dbPath
}

func freedomOutboundJSON() string {
	return `{"protocol":"freedom","settings":{}}`
}

type stubEgressTestRunner struct {
	result *egressTestExecution
	err    error
}

func (r *stubEgressTestRunner) Run(egress *n5model.Egress) (*egressTestExecution, error) {
	return r.result, r.err
}

func TestEgressServiceCreateGeneratesStableTag(t *testing.T) {
	initTestDB(t)

	svc := &EgressService{}
	egress, err := svc.Create(&n5model.Egress{
		Name:         "test-egress",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: freedomOutboundJSON(),
	})
	if err != nil {
		t.Fatalf("create egress failed: %v", err)
	}
	if egress.Tag != "n5-egress-0000000001" {
		t.Fatalf("unexpected tag: %s", egress.Tag)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(egress.OutboundJSON), &parsed); err != nil {
		t.Fatalf("unmarshal outbound json failed: %v", err)
	}
	if parsed["tag"] != egress.Tag {
		t.Fatalf("outbound tag not normalized: %#v", parsed["tag"])
	}
	if parsed["protocol"] != "freedom" {
		t.Fatalf("outbound protocol not normalized: %#v", parsed["protocol"])
	}
}

func TestEgressServiceRejectsInvalidConfig(t *testing.T) {
	initTestDB(t)

	svc := &EgressService{}
	_, err := svc.Create(&n5model.Egress{
		Name:         "bad-egress",
		Protocol:     "invalid-protocol",
		Enabled:      true,
		OutboundJSON: `{"protocol":"invalid-protocol","settings":{}}`,
	})
	if err == nil {
		t.Fatal("expected invalid outbound config error")
	}
}

func TestEgressServiceDeleteProtectsReferences(t *testing.T) {
	initTestDB(t)

	egressSvc := &EgressService{}
	policySvc := &TrafficPolicyService{}
	poolSvc := &EgressPoolService{}

	egress, err := egressSvc.Create(&n5model.Egress{
		Name:         "protected-egress",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: freedomOutboundJSON(),
	})
	if err != nil {
		t.Fatalf("create egress failed: %v", err)
	}
	freeEgress, err := egressSvc.Create(&n5model.Egress{
		Name:         "free-egress",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: freedomOutboundJSON(),
	})
	if err != nil {
		t.Fatalf("create free egress failed: %v", err)
	}
	if err := egressSvc.Delete(freeEgress.Id); err != nil {
		t.Fatalf("delete unreferenced egress failed: %v", err)
	}

	assertDeleteRejected := func(name, want string) {
		t.Helper()
		err := egressSvc.Delete(egress.Id)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("%s: unexpected delete error: %v", name, err)
		}
		var count int64
		if err := database.GetDB().Model(&n5model.Egress{}).Where("id = ?", egress.Id).Count(&count).Error; err != nil {
			t.Fatalf("%s: count egress failed: %v", name, err)
		}
		if count != 1 {
			t.Fatalf("%s: referenced egress was deleted", name)
		}
	}

	advancedDefault, err := policySvc.Create(&n5model.TrafficPolicy{
		Name:              "advanced-default",
		Remark:            "ordinary-advanced",
		Enabled:           true,
		DefaultTargetType: targetTypeEgress,
		DefaultTargetId:   egress.Id,
	})
	if err != nil {
		t.Fatalf("create advanced default policy failed: %v", err)
	}
	assertDeleteRejected("advanced default", "默认出口")
	if err := database.GetDB().Delete(&n5model.TrafficPolicy{}, advancedDefault.Id).Error; err != nil {
		t.Fatalf("cleanup advanced default failed: %v", err)
	}

	advancedRulePolicy, err := policySvc.Create(&n5model.TrafficPolicy{Name: "advanced-rule", Remark: "ordinary-advanced", Enabled: true})
	if err != nil {
		t.Fatalf("create advanced rule policy failed: %v", err)
	}
	advancedRule, err := policySvc.AddRule(&n5model.TrafficPolicyRule{
		PolicyId:   advancedRulePolicy.Id,
		RuleType:   ruleTypeDomain,
		MatchMode:  domainModeExact,
		MatchValue: "advanced.example.com",
		TargetType: targetTypeEgress,
		TargetId:   egress.Id,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("create advanced rule failed: %v", err)
	}
	assertDeleteRejected("advanced rule", "出口规则使用")
	if err := database.GetDB().Delete(&n5model.TrafficPolicyRule{}, advancedRule.Id).Error; err != nil {
		t.Fatalf("cleanup advanced rule failed: %v", err)
	}
	if err := database.GetDB().Delete(&n5model.TrafficPolicy{}, advancedRulePolicy.Id).Error; err != nil {
		t.Fatalf("cleanup advanced rule policy failed: %v", err)
	}

	simpleDefault, err := policySvc.Create(&n5model.TrafficPolicy{
		Name:              "simple-default",
		Remark:            "n5-simple-exec|eyJ2ZXJzaW9uIjoxLCJpdGVtcyI6W119",
		Enabled:           true,
		DefaultTargetType: targetTypeEgress,
		DefaultTargetId:   egress.Id,
	})
	if err != nil {
		t.Fatalf("create simple default policy failed: %v", err)
	}
	assertDeleteRejected("simple default", "默认出口")
	if err := database.GetDB().Delete(&n5model.TrafficPolicy{}, simpleDefault.Id).Error; err != nil {
		t.Fatalf("cleanup simple default failed: %v", err)
	}

	simpleRulePolicy, err := policySvc.Create(&n5model.TrafficPolicy{
		Name:    "simple-rules",
		Remark:  "n5-simple-exec|eyJ2ZXJzaW9uIjoxLCJpdGVtcyI6W119",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("create simple rule policy failed: %v", err)
	}
	for _, item := range []struct {
		name  string
		value string
	}{
		{name: "custom group execution", value: "group.example.com"},
		{name: "builtin execution", value: "builtin.example.com"},
	} {
		rule, err := policySvc.AddRule(&n5model.TrafficPolicyRule{
			PolicyId:   simpleRulePolicy.Id,
			RuleType:   ruleTypeDomain,
			MatchMode:  domainModeExact,
			MatchValue: item.value,
			TargetType: targetTypeEgress,
			TargetId:   egress.Id,
			Enabled:    true,
		})
		if err != nil {
			t.Fatalf("create %s rule failed: %v", item.name, err)
		}
		assertDeleteRejected(item.name, "出口规则使用")
		if err := database.GetDB().Delete(&n5model.TrafficPolicyRule{}, rule.Id).Error; err != nil {
			t.Fatalf("cleanup %s rule failed: %v", item.name, err)
		}
	}
	if err := database.GetDB().Delete(&n5model.TrafficPolicy{}, simpleRulePolicy.Id).Error; err != nil {
		t.Fatalf("cleanup simple rule policy failed: %v", err)
	}

	pool, err := poolSvc.Create(&n5model.EgressPool{Name: "protect-pool", Enabled: true})
	if err != nil {
		t.Fatalf("create pool failed: %v", err)
	}
	if _, err := poolSvc.AddMember(pool.Id, egress.Id, 1, 1); err != nil {
		t.Fatalf("add pool member failed: %v", err)
	}
	assertDeleteRejected("pool member", "出口池")
	if err := poolSvc.RemoveMember(pool.Id, egress.Id); err != nil {
		t.Fatalf("remove pool member failed: %v", err)
	}

	if err := database.GetDB().Model(&n5model.EgressPool{}).Where("id = ?", pool.Id).Updates(map[string]interface{}{
		"fallback_type":      targetTypeEgress,
		"fallback_target_id": egress.Id,
	}).Error; err != nil {
		t.Fatalf("update pool fallback failed: %v", err)
	}
	assertDeleteRejected("pool fallback", "fallback")
	if err := database.GetDB().Model(&n5model.EgressPool{}).Where("id = ?", pool.Id).Updates(map[string]interface{}{
		"fallback_type":      "",
		"fallback_target_id": 0,
	}).Error; err != nil {
		t.Fatalf("clear pool fallback failed: %v", err)
	}

	if err := egressSvc.Delete(egress.Id); err != nil {
		t.Fatalf("delete after clearing references failed: %v", err)
	}
	var remaining int64
	if err := database.GetDB().Model(&n5model.Egress{}).Where("id = ?", egress.Id).Count(&remaining).Error; err != nil {
		t.Fatalf("count deleted egress failed: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected egress removed after references cleared, got %d", remaining)
	}
}

func TestEgressPoolServiceMembers(t *testing.T) {
	initTestDB(t)

	egressSvc := &EgressService{}
	poolSvc := &EgressPoolService{}

	egress, err := egressSvc.Create(&n5model.Egress{
		Name:         "pool-egress",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: freedomOutboundJSON(),
	})
	if err != nil {
		t.Fatalf("create egress failed: %v", err)
	}

	pool, err := poolSvc.Create(&n5model.EgressPool{
		Name:     "pool-a",
		Strategy: "random",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("create pool failed: %v", err)
	}
	if pool.Tag != "n5-pool-0000000001" {
		t.Fatalf("unexpected pool tag: %s", pool.Tag)
	}

	member, err := poolSvc.AddMember(pool.Id, egress.Id, 2, 1)
	if err != nil {
		t.Fatalf("add member failed: %v", err)
	}
	if member.Weight != 2 {
		t.Fatalf("unexpected member weight: %d", member.Weight)
	}

	members, err := poolSvc.ListMembers(pool.Id)
	if err != nil {
		t.Fatalf("list members failed: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("unexpected member count: %d", len(members))
	}

	if err := poolSvc.RemoveMember(pool.Id, egress.Id); err != nil {
		t.Fatalf("remove member failed: %v", err)
	}
	members, err = poolSvc.ListMembers(pool.Id)
	if err != nil {
		t.Fatalf("list members after remove failed: %v", err)
	}
	if len(members) != 0 {
		t.Fatalf("unexpected member count after remove: %d", len(members))
	}
}

func TestEgressTestServicePersistsResultAndUpdatesEgress(t *testing.T) {
	initTestDB(t)

	egressSvc := &EgressService{}
	egress, err := egressSvc.Create(&n5model.Egress{
		Name:         "testable-egress",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: freedomOutboundJSON(),
	})
	if err != nil {
		t.Fatalf("create egress failed: %v", err)
	}

	testSvc := &EgressTestService{
		runner: &stubEgressTestRunner{
			result: &egressTestExecution{
				Status:  egressTestStatusSuccess,
				Latency: 123,
				ExitIP:  "203.0.113.5",
			},
		},
	}

	record, err := testSvc.Test(egress.Id)
	if err != nil {
		t.Fatalf("test egress failed: %v", err)
	}
	if record.Status != egressTestStatusSuccess {
		t.Fatalf("unexpected record status: %s", record.Status)
	}
	if record.ExitIP != "203.0.113.5" {
		t.Fatalf("unexpected exit ip: %s", record.ExitIP)
	}
	if record.Latency != 123 {
		t.Fatalf("unexpected latency: %d", record.Latency)
	}

	updated, err := egressSvc.Get(egress.Id)
	if err != nil {
		t.Fatalf("get updated egress failed: %v", err)
	}
	if updated.LastStatus != egressTestStatusSuccess {
		t.Fatalf("unexpected last status: %s", updated.LastStatus)
	}
	if updated.LastExitIP != "203.0.113.5" {
		t.Fatalf("unexpected last exit ip: %s", updated.LastExitIP)
	}
	if updated.LastTestTime == 0 || updated.LastTestAt == 0 {
		t.Fatalf("expected test timestamps to be set: %#v", updated)
	}
	if updated.LastTestLatencyMs != 123 {
		t.Fatalf("unexpected last latency: %d", updated.LastTestLatencyMs)
	}

	var count int64
	if err := database.GetDB().Model(&n5model.EgressTest{}).Where("egress_id = ?", egress.Id).Count(&count).Error; err != nil {
		t.Fatalf("count egress test records failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("unexpected egress test count: %d", count)
	}
}

func TestEgressTestServicePersistsFailureAndPoolSkipsFailedSelector(t *testing.T) {
	initTestDB(t)

	egressSvc := &EgressService{}
	poolSvc := &EgressPoolService{}
	extSvc := &XrayExtService{}

	successEgress, err := egressSvc.Create(&n5model.Egress{
		Name:         "untested-egress",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: freedomOutboundJSON(),
	})
	if err != nil {
		t.Fatalf("create success egress failed: %v", err)
	}
	failedEgress, err := egressSvc.Create(&n5model.Egress{
		Name:         "failed-egress",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: freedomOutboundJSON(),
	})
	if err != nil {
		t.Fatalf("create failed egress failed: %v", err)
	}

	pool, err := poolSvc.Create(&n5model.EgressPool{
		Name:     "pool-filter",
		Strategy: "random",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("create pool failed: %v", err)
	}
	if _, err := poolSvc.AddMember(pool.Id, successEgress.Id, 1, 1); err != nil {
		t.Fatalf("add success member failed: %v", err)
	}
	if _, err := poolSvc.AddMember(pool.Id, failedEgress.Id, 1, 2); err != nil {
		t.Fatalf("add failed member failed: %v", err)
	}

	testSvc := &EgressTestService{
		runner: &stubEgressTestRunner{
			err: common.NewError("tcp probe failed"),
		},
	}
	record, err := testSvc.Test(failedEgress.Id)
	if err != nil {
		t.Fatalf("persist failed test failed: %v", err)
	}
	if record.Status != egressTestStatusFailed {
		t.Fatalf("unexpected failed record status: %s", record.Status)
	}

	routing, err := extSvc.GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("generate routing failed: %v", err)
	}
	balancers := routing["balancers"].([]interface{})
	if len(balancers) != 1 {
		t.Fatalf("unexpected balancer count: %d", len(balancers))
	}
	balancer := balancers[0].(map[string]interface{})
	selectors := balancer["selector"].([]interface{})
	if len(selectors) != 1 {
		t.Fatalf("unexpected selector count: %d", len(selectors))
	}
	if selectors[0].(string) != successEgress.Tag {
		t.Fatalf("unexpected remaining selector: %#v", selectors)
	}
}

func TestEgressLabelServiceCRUDAndBindings(t *testing.T) {
	initTestDB(t)

	egressSvc := &EgressService{}
	labelSvc := &EgressLabelService{}

	egress, err := egressSvc.Create(&n5model.Egress{
		Name:         "label-egress",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: freedomOutboundJSON(),
	})
	if err != nil {
		t.Fatalf("create egress failed: %v", err)
	}

	regionLabel, err := labelSvc.Create(&n5model.EgressLabel{
		Name: "Singapore",
		Type: labelTypeRegion,
	})
	if err != nil {
		t.Fatalf("create region label failed: %v", err)
	}
	usageLabel, err := labelSvc.Create(&n5model.EgressLabel{
		Name: "AI",
		Type: labelTypeUsage,
	})
	if err != nil {
		t.Fatalf("create usage label failed: %v", err)
	}

	updatedRegion, err := labelSvc.Update(&n5model.EgressLabel{
		Id:   regionLabel.Id,
		Name: "Singapore-1",
		Type: labelTypeRegion,
	})
	if err != nil {
		t.Fatalf("update region label failed: %v", err)
	}
	if updatedRegion.Name != "Singapore-1" {
		t.Fatalf("unexpected updated label name: %s", updatedRegion.Name)
	}

	if _, err := labelSvc.Bind(egress.Id, regionLabel.Id); err != nil {
		t.Fatalf("bind region label failed: %v", err)
	}
	if _, err := labelSvc.Bind(egress.Id, usageLabel.Id); err != nil {
		t.Fatalf("bind usage label failed: %v", err)
	}

	labels, err := labelSvc.ListByEgress(egress.Id)
	if err != nil {
		t.Fatalf("list labels by egress failed: %v", err)
	}
	if len(labels) != 2 {
		t.Fatalf("unexpected label count: %d", len(labels))
	}

	if err := labelSvc.Unbind(egress.Id, regionLabel.Id); err != nil {
		t.Fatalf("unbind region label failed: %v", err)
	}
	labels, err = labelSvc.ListByEgress(egress.Id)
	if err != nil {
		t.Fatalf("list labels after unbind failed: %v", err)
	}
	if len(labels) != 1 || labels[0].Id != usageLabel.Id {
		t.Fatalf("unexpected labels after unbind: %#v", labels)
	}

	if err := labelSvc.Delete(usageLabel.Id); err != nil {
		t.Fatalf("delete usage label failed: %v", err)
	}
	allLabels, err := labelSvc.List()
	if err != nil {
		t.Fatalf("list labels failed: %v", err)
	}
	if len(allLabels) != 1 || allLabels[0].Id != regionLabel.Id {
		t.Fatalf("unexpected labels after delete: %#v", allLabels)
	}
}

func TestEgressDetailServiceAggregatesLabelsPoolsAndPolicies(t *testing.T) {
	initTestDB(t)

	egressSvc := &EgressService{}
	labelSvc := &EgressLabelService{}
	poolSvc := &EgressPoolService{}
	policySvc := &TrafficPolicyService{}
	detailSvc := &EgressDetailService{}

	egressA, err := egressSvc.Create(&n5model.Egress{
		Name:         "detail-egress-a",
		Protocol:     "socks",
		Enabled:      true,
		OutboundJSON: `{"protocol":"socks","settings":{"servers":[{"address":"1.1.1.1","port":1080}]}}`,
	})
	if err != nil {
		t.Fatalf("create egress a failed: %v", err)
	}
	egressB, err := egressSvc.Create(&n5model.Egress{
		Name:         "detail-egress-b",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: freedomOutboundJSON(),
	})
	if err != nil {
		t.Fatalf("create egress b failed: %v", err)
	}

	regionLabel, err := labelSvc.Create(&n5model.EgressLabel{Name: "HK", Type: labelTypeRegion})
	if err != nil {
		t.Fatalf("create region label failed: %v", err)
	}
	usageLabel, err := labelSvc.Create(&n5model.EgressLabel{Name: "AI", Type: labelTypeUsage})
	if err != nil {
		t.Fatalf("create usage label failed: %v", err)
	}
	if _, err := labelSvc.Bind(egressA.Id, regionLabel.Id); err != nil {
		t.Fatalf("bind region label failed: %v", err)
	}
	if _, err := labelSvc.Bind(egressA.Id, usageLabel.Id); err != nil {
		t.Fatalf("bind usage label failed: %v", err)
	}
	if _, err := labelSvc.Bind(egressB.Id, usageLabel.Id); err != nil {
		t.Fatalf("bind shared usage label failed: %v", err)
	}

	pool, err := poolSvc.Create(&n5model.EgressPool{
		Name:     "detail-pool",
		Strategy: "random",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("create pool failed: %v", err)
	}
	if _, err := poolSvc.AddMember(pool.Id, egressA.Id, 1, 1); err != nil {
		t.Fatalf("add pool member failed: %v", err)
	}

	policyA, err := policySvc.Create(&n5model.TrafficPolicy{
		Name:              "policy-direct",
		Enabled:           true,
		DefaultTargetType: targetTypeEgress,
		DefaultTargetId:   egressA.Id,
	})
	if err != nil {
		t.Fatalf("create direct policy failed: %v", err)
	}
	if _, err := policySvc.AddRule(&n5model.TrafficPolicyRule{
		PolicyId:   policyA.Id,
		RuleType:   ruleTypeDomain,
		MatchMode:  domainModeExact,
		MatchValue: "example.com",
		TargetType: targetTypeEgress,
		TargetId:   egressA.Id,
		Enabled:    true,
	}); err != nil {
		t.Fatalf("add direct rule failed: %v", err)
	}

	if _, err := policySvc.Create(&n5model.TrafficPolicy{
		Name:              "policy-pool",
		Enabled:           true,
		DefaultTargetType: targetTypePool,
		DefaultTargetId:   pool.Id,
	}); err != nil {
		t.Fatalf("create pool policy failed: %v", err)
	}

	detail, err := detailSvc.Get(egressA.Id)
	if err != nil {
		t.Fatalf("get egress detail failed: %v", err)
	}
	if detail.Address != "1.1.1.1" {
		t.Fatalf("unexpected detail address: %s", detail.Address)
	}
	if len(detail.Labels) != 2 {
		t.Fatalf("unexpected detail label count: %d", len(detail.Labels))
	}
	if len(detail.Pools) != 1 || detail.Pools[0].Id != pool.Id {
		t.Fatalf("unexpected detail pools: %#v", detail.Pools)
	}
	if len(detail.Policies) != 2 {
		t.Fatalf("unexpected detail policies: %#v", detail.Policies)
	}
}

func TestTrafficPolicyAndXrayExtServiceGenerateFragments(t *testing.T) {
	initTestDB(t)

	egressSvc := &EgressService{}
	poolSvc := &EgressPoolService{}
	policySvc := &TrafficPolicyService{}
	extSvc := &XrayExtService{}

	egressA, err := egressSvc.Create(&n5model.Egress{
		Name:         "egress-a",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: freedomOutboundJSON(),
	})
	if err != nil {
		t.Fatalf("create egress a failed: %v", err)
	}
	egressB, err := egressSvc.Create(&n5model.Egress{
		Name:         "egress-b",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: freedomOutboundJSON(),
	})
	if err != nil {
		t.Fatalf("create egress b failed: %v", err)
	}

	pool, err := poolSvc.Create(&n5model.EgressPool{
		Name:     "pool-b",
		Strategy: "random",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("create pool failed: %v", err)
	}
	if _, err := poolSvc.AddMember(pool.Id, egressB.Id, 1, 1); err != nil {
		t.Fatalf("add pool member failed: %v", err)
	}

	db := database.GetDB()
	inbound := &legacyModel.Inbound{
		UserId:         1,
		Remark:         "inbound-a",
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           31001,
		Protocol:       legacyModel.Socks,
		Settings:       `{"auth":"noauth","udp":false,"ip":"127.0.0.1"}`,
		StreamSettings: `{}`,
		Tag:            "inbound-test",
		Sniffing:       `{}`,
	}
	if err := db.Create(inbound).Error; err != nil {
		t.Fatalf("create inbound failed: %v", err)
	}

	policy, err := policySvc.Create(&n5model.TrafficPolicy{
		Name:              "policy-a",
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
		MatchValue: "example.com",
		TargetType: targetTypeEgress,
		TargetId:   egressA.Id,
		Enabled:    true,
	}); err != nil {
		t.Fatalf("add domain rule failed: %v", err)
	}
	if _, err := policySvc.AddRule(&n5model.TrafficPolicyRule{
		PolicyId:   policy.Id,
		RuleType:   ruleTypeIP,
		MatchMode:  ipModeCIDR,
		MatchValue: "1.1.1.0/24",
		TargetType: targetTypePool,
		TargetId:   pool.Id,
		Enabled:    true,
	}); err != nil {
		t.Fatalf("add ip rule failed: %v", err)
	}
	if _, err := policySvc.BindInboundPolicy(inbound.Id, policy.Id); err != nil {
		t.Fatalf("bind inbound policy failed: %v", err)
	}

	outbounds, err := extSvc.GenerateOutboundFragments()
	if err != nil {
		t.Fatalf("generate outbounds failed: %v", err)
	}
	if len(outbounds) != 2 {
		t.Fatalf("unexpected outbound fragment count: %d", len(outbounds))
	}

	routing, err := extSvc.GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("generate routing failed: %v", err)
	}
	balancers, ok := routing["balancers"].([]interface{})
	if !ok || len(balancers) != 1 {
		t.Fatalf("unexpected balancers: %#v", routing["balancers"])
	}
	rules, ok := routing["rules"].([]interface{})
	if !ok || len(rules) != 3 {
		t.Fatalf("unexpected rules: %#v", routing["rules"])
	}

	firstRule := rules[0].(map[string]interface{})
	domainMatchers := firstRule["domain"].([]interface{})
	if domainMatchers[0].(string) != "full:example.com" {
		t.Fatalf("unexpected domain matcher: %#v", domainMatchers[0])
	}
	if firstRule["outboundTag"].(string) != egressA.Tag {
		t.Fatalf("unexpected domain rule target: %#v", firstRule["outboundTag"])
	}

	lastRule := rules[2].(map[string]interface{})
	if lastRule["balancerTag"].(string) != pool.Tag {
		t.Fatalf("unexpected default rule balancer target: %#v", lastRule["balancerTag"])
	}
}

func TestN5StableTagsDoNotPrefixCollide(t *testing.T) {
	egressSvc := &EgressService{}
	poolSvc := &EgressPoolService{}

	egressTags := make([]string, 0)
	for _, id := range []int{1, 2, 10, 11, 100, 999999999} {
		egressTags = append(egressTags, egressSvc.GenerateStableTag(id))
	}
	for i, selector := range egressTags {
		for j, tag := range egressTags {
			if i == j {
				continue
			}
			if strings.HasPrefix(tag, selector) {
				t.Fatalf("egress selector %q incorrectly matches tag %q", selector, tag)
			}
		}
	}

	poolTags := make([]string, 0)
	for _, id := range []int{1, 2, 10, 11, 100, 999999999} {
		poolTags = append(poolTags, poolSvc.GeneratePoolTag(id))
	}
	for i, selector := range poolTags {
		for j, tag := range poolTags {
			if i == j {
				continue
			}
			if strings.HasPrefix(tag, selector) {
				t.Fatalf("pool selector %q incorrectly matches tag %q", selector, tag)
			}
		}
	}
}
