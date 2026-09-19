package simple

import (
	"encoding/json"
	"strings"
	"testing"

	n5model "github.com/torr9522/n6-ui/database/model/n5"
	n5service "github.com/torr9522/n6-ui/web/service/n5"
	ssservice "github.com/torr9522/n6-ui/web/service/shadowsocks"
)

func createSS2022RoutingEgress(t *testing.T, name string, method string, port int) *n5model.Egress {
	t.Helper()
	key, err := ssservice.Generate2022Key(method)
	if err != nil {
		t.Fatalf("generate SS2022 key failed: %v", err)
	}
	outbound, err := json.Marshal(map[string]interface{}{
		"protocol": "shadowsocks",
		"settings": map[string]interface{}{
			"servers": []map[string]interface{}{{
				"address":  "127.0.0.1",
				"port":     port,
				"method":   method,
				"password": key,
			}},
		},
	})
	if err != nil {
		t.Fatalf("marshal SS2022 outbound failed: %v", err)
	}
	egress, err := (&n5service.EgressService{}).Create(&n5model.Egress{
		Name:         name,
		Protocol:     "shadowsocks",
		Enabled:      true,
		OutboundJSON: string(outbound),
	})
	if err != nil {
		t.Fatalf("create SS2022 egress failed: %v", err)
	}
	return egress
}

func assertSS2022TargetOrder(t *testing.T, routes []string, tags []string) {
	t.Helper()
	previousLast := -1
	for _, tag := range tags {
		first := -1
		last := -1
		suffix := "=>" + tag
		for index, route := range routes {
			if strings.HasSuffix(route, suffix) {
				if first < 0 {
					first = index
				}
				last = index
			}
		}
		if first < 0 {
			t.Fatalf("routing target %s not found: %v", tag, routes)
		}
		if first <= previousLast {
			t.Fatalf("routing target order changed at %s: %v", tag, routes)
		}
		previousLast = last
	}
}

func TestSimpleSS2022RoutingTargetsPreserveBusinessPriority(t *testing.T) {
	initSimpleTestDB(t)

	ruleSvc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, err := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	if err != nil {
		t.Fatalf("get AI group failed: %v", err)
	}
	gameGroup, err := mustGetBuiltinGroup(groupSvc, simpleTrafficGame)
	if err != nil {
		t.Fatalf("get Game group failed: %v", err)
	}
	streamingGroup, err := mustGetBuiltinGroup(groupSvc, simpleTrafficStreaming)
	if err != nil {
		t.Fatalf("get Streaming group failed: %v", err)
	}
	customGroup, err := groupSvc.CreateGroup(&CreateTrafficRuleGroupRequest{
		GroupType: simpleTrafficCustom,
		Name:      "SS2022 custom group",
	})
	if err != nil {
		t.Fatalf("create custom group failed: %v", err)
	}
	if _, err := groupSvc.AddDomainRule(&AddTrafficRuleDomainRequest{
		GroupId: customGroup.Id,
		Domain:  "domain:custom.ss2022.example",
	}); err != nil {
		t.Fatalf("add custom group rule failed: %v", err)
	}
	customGroup, err = groupSvc.GetGroup(customGroup.Id)
	if err != nil {
		t.Fatalf("reload custom group failed: %v", err)
	}

	methods := []string{
		ssservice.Method2022Blake3AES128GCM,
		ssservice.Method2022Blake3AES256GCM,
		ssservice.Method2022Blake3ChaCha20Poly1305,
	}
	names := []string{"exact", "suffix", "keyword", "regexp", "custom", "ai", "game", "streaming", "all"}
	egresses := make(map[string]*n5model.Egress, len(names))
	for index, name := range names {
		egresses[name] = createSS2022RoutingEgress(t, "ss2022-"+name, methods[index%len(methods)], 37800+index)
	}
	inbound := createTestRuleInbound(t, 33780, "ss2022-priority-inbound")

	// Deliberately create the rules in reverse business-priority order.
	steps := []struct {
		name string
		req  *CreateSimpleRuleRequest
	}{
		{"all", &CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficAll, EgressId: egresses["all"].Id}},
		{"streaming", &CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: streamingGroup.Id, EgressId: egresses["streaming"].Id}},
		{"game", &CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: gameGroup.Id, EgressId: egresses["game"].Id}},
		{"ai", &CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: aiGroup.Id, EgressId: egresses["ai"].Id}},
		{"custom", &CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: customGroup.Id, EgressId: egresses["custom"].Id}},
		{"regexp", &CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficCustomDomain, CustomDomain: `regexp:^regexp\.ss2022\.example$`, EgressId: egresses["regexp"].Id}},
		{"keyword", &CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficCustomDomain, CustomDomain: "keyword:ss2022-keyword", EgressId: egresses["keyword"].Id}},
		{"suffix", &CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficCustomDomain, CustomDomain: "domain:suffix.ss2022.example", EgressId: egresses["suffix"].Id}},
		{"exact", &CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficCustomDomain, CustomDomain: "full:exact.ss2022.example", EgressId: egresses["exact"].Id}},
	}
	for _, step := range steps {
		if _, err := ruleSvc.CreateSimpleRule(step.req); err != nil {
			t.Fatalf("create %s rule failed: %v", step.name, err)
		}
	}

	fragments, err := (&n5service.XrayExtService{}).GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("generate routing fragments failed: %v", err)
	}
	routes := inboundRoutingMatchersAndTargets(t, fragments, inbound.Tag)
	orderedTags := []string{
		egresses["exact"].Tag,
		egresses["suffix"].Tag,
		egresses["keyword"].Tag,
		egresses["regexp"].Tag,
		egresses["custom"].Tag,
		egresses["ai"].Tag,
		egresses["game"].Tag,
		egresses["streaming"].Tag,
		egresses["all"].Tag,
	}
	assertSS2022TargetOrder(t, routes, orderedTags)
	if routes[len(routes)-1] != "*=>"+egresses["all"].Tag {
		t.Fatalf("ALL target must remain last: %v", routes)
	}
}
