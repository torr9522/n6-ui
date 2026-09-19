package n5

import (
	"testing"

	n5model "x-ui/database/model/n5"
	ssservice "x-ui/web/service/shadowsocks"
)

func createSS2022AdvancedRoutingEgress(t *testing.T, name string, method string, port int) *n5model.Egress {
	t.Helper()
	key, err := ssservice.Generate2022Key(method)
	if err != nil {
		t.Fatalf("generate SS2022 key failed: %v", err)
	}
	egress, err := (&EgressService{}).Create(&n5model.Egress{
		Name:         name,
		Protocol:     "shadowsocks",
		Enabled:      true,
		OutboundJSON: ss2022OutboundJSON(method, key, "127.0.0.1", port),
	})
	if err != nil {
		t.Fatalf("create SS2022 egress failed: %v", err)
	}
	return egress
}

func findRoutingBalancer(t *testing.T, routing map[string]interface{}, tag string) map[string]interface{} {
	t.Helper()
	balancers, _ := routing["balancers"].([]interface{})
	for _, item := range balancers {
		balancer, ok := item.(map[string]interface{})
		if ok && balancer["tag"] == tag {
			return balancer
		}
	}
	t.Fatalf("balancer %s not found: %#v", tag, routing["balancers"])
	return nil
}

func TestAdvancedSS2022RoutingAndPoolTargets(t *testing.T) {
	initTestDB(t)

	methods := ss2022Methods()
	member := createSS2022AdvancedRoutingEgress(t, "ss2022-pool-member", methods[0], 37900)
	direct := createSS2022AdvancedRoutingEgress(t, "ss2022-direct-target", methods[1], 37901)
	fallback := createSS2022AdvancedRoutingEgress(t, "ss2022-pool-fallback", methods[2], 37902)

	poolSvc := &EgressPoolService{}
	pool, err := poolSvc.Create(&n5model.EgressPool{
		Name:             "ss2022-pool",
		Strategy:         "random",
		FallbackType:     targetTypeEgress,
		FallbackTargetId: fallback.Id,
		Enabled:          true,
	})
	if err != nil {
		t.Fatalf("create SS2022 pool failed: %v", err)
	}
	if _, err := poolSvc.AddMember(pool.Id, member.Id, 1, 1); err != nil {
		t.Fatalf("add SS2022 pool member failed: %v", err)
	}

	policySvc := &TrafficPolicyService{}
	directInbound := createTrafficTestInbound(t, 37910, "ss2022-advanced-direct-inbound")
	directPolicy, err := policySvc.CreatePolicyFromAdvanced(&n5model.TrafficPolicy{
		Name:              "ss2022-advanced-direct",
		Remark:            "ordinary-advanced-ss2022-direct",
		Enabled:           true,
		DefaultTargetType: targetTypeEgress,
		DefaultTargetId:   member.Id,
	})
	if err != nil {
		t.Fatalf("create direct advanced policy failed: %v", err)
	}
	if _, err := policySvc.AddRuleFromAdvanced(&n5model.TrafficPolicyRule{
		PolicyId:   directPolicy.Id,
		RuleType:   ruleTypeDomain,
		MatchMode:  domainModeExact,
		MatchValue: "api.ss2022.example",
		TargetType: targetTypeEgress,
		TargetId:   direct.Id,
		Enabled:    true,
	}); err != nil {
		t.Fatalf("add direct advanced rule failed: %v", err)
	}
	if _, err := policySvc.BindInboundPolicy(directInbound.Id, directPolicy.Id); err != nil {
		t.Fatalf("bind direct advanced policy failed: %v", err)
	}

	poolInbound := createTrafficTestInbound(t, 37911, "ss2022-advanced-pool-inbound")
	poolPolicy, err := policySvc.CreatePolicyFromAdvanced(&n5model.TrafficPolicy{
		Name:              "ss2022-advanced-pool",
		Remark:            "ordinary-advanced-ss2022-pool",
		Enabled:           true,
		DefaultTargetType: targetTypePool,
		DefaultTargetId:   pool.Id,
	})
	if err != nil {
		t.Fatalf("create pool advanced policy failed: %v", err)
	}
	if _, err := policySvc.AddRuleFromAdvanced(&n5model.TrafficPolicyRule{
		PolicyId:   poolPolicy.Id,
		RuleType:   ruleTypeDomain,
		MatchMode:  domainModeSuffix,
		MatchValue: "pool.ss2022.example",
		TargetType: targetTypePool,
		TargetId:   pool.Id,
		Enabled:    true,
	}); err != nil {
		t.Fatalf("add pool advanced rule failed: %v", err)
	}
	if _, err := policySvc.BindInboundPolicy(poolInbound.Id, poolPolicy.Id); err != nil {
		t.Fatalf("bind pool advanced policy failed: %v", err)
	}

	routing, err := (&XrayExtService{}).GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("generate routing fragments failed: %v", err)
	}
	directRoutes := trafficInboundRoutingMatchersAndTargets(t, routing, directInbound.Tag)
	directWant := []string{
		"full:api.ss2022.example=>" + direct.Tag,
		"*=>" + member.Tag,
	}
	if len(directRoutes) != len(directWant) {
		t.Fatalf("unexpected direct advanced routes: got=%v want=%v", directRoutes, directWant)
	}
	for index := range directWant {
		if directRoutes[index] != directWant[index] {
			t.Fatalf("unexpected direct advanced routes: got=%v want=%v", directRoutes, directWant)
		}
	}

	poolRoutes := trafficInboundRoutingMatchersAndTargets(t, routing, poolInbound.Tag)
	poolWant := []string{
		"domain:pool.ss2022.example=>" + pool.Tag,
		"*=>" + pool.Tag,
	}
	if len(poolRoutes) != len(poolWant) {
		t.Fatalf("unexpected pool advanced routes: got=%v want=%v", poolRoutes, poolWant)
	}
	for index := range poolWant {
		if poolRoutes[index] != poolWant[index] {
			t.Fatalf("unexpected pool advanced routes: got=%v want=%v", poolRoutes, poolWant)
		}
	}

	balancer := findRoutingBalancer(t, routing, pool.Tag)
	selectors, _ := balancer["selector"].([]interface{})
	if len(selectors) != 1 || selectors[0] != member.Tag {
		t.Fatalf("unexpected SS2022 pool selectors: %#v", selectors)
	}
	if balancer["fallbackTag"] != fallback.Tag {
		t.Fatalf("unexpected SS2022 pool fallback: %#v", balancer["fallbackTag"])
	}
}
