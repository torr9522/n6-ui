package n5

import (
	"testing"
	"x-ui/database"
	legacyModel "x-ui/database/model"
	n5model "x-ui/database/model/n5"
)

func TestTrafficTemplateServiceListAndPreview(t *testing.T) {
	initTestDB(t)

	svc := &TrafficTemplateService{}
	items, err := svc.List()
	if err != nil {
		t.Fatalf("list templates failed: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("unexpected template count: %d", len(items))
	}

	preview, err := svc.Preview("ai")
	if err != nil {
		t.Fatalf("preview ai template failed: %v", err)
	}
	if preview.Name != "ai" || len(preview.Rules) == 0 {
		t.Fatalf("unexpected ai preview: %#v", preview)
	}
	if preview.Rules[0].RuleType != ruleTypeDomain {
		t.Fatalf("unexpected ai preview rule type: %#v", preview.Rules[0])
	}
}

func TestTrafficTemplateServiceCreatePolicyRulesAndBinding(t *testing.T) {
	initTestDB(t)

	egressSvc := &EgressService{}
	templateSvc := &TrafficTemplateService{}

	egress, err := egressSvc.Create(&n5model.Egress{
		Name:         "template-egress",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: freedomOutboundJSON(),
	})
	if err != nil {
		t.Fatalf("create egress failed: %v", err)
	}

	inbound := &legacyModel.Inbound{
		UserId:         1,
		Remark:         "template-inbound",
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           34001,
		Protocol:       legacyModel.Socks,
		Settings:       `{"auth":"noauth","udp":false,"ip":"127.0.0.1"}`,
		StreamSettings: `{}`,
		Tag:            "template-inbound-tag",
		Sniffing:       `{}`,
	}
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatalf("create inbound failed: %v", err)
	}

	result, err := templateSvc.Create(&TrafficTemplateCreateRequest{
		TemplateName: "ai",
		PolicyName:   "AI Template Policy",
		InboundId:    inbound.Id,
		TargetType:   targetTypeEgress,
		TargetId:     egress.Id,
	})
	if err != nil {
		t.Fatalf("create traffic template failed: %v", err)
	}

	if result.Policy == nil || result.Policy.Name != "AI Template Policy" {
		t.Fatalf("unexpected created policy: %#v", result.Policy)
	}
	if result.Binding == nil || result.Binding.InboundId != inbound.Id || result.Binding.PolicyId != result.Policy.Id {
		t.Fatalf("unexpected created binding: %#v", result.Binding)
	}
	if len(result.Rules) != 12 {
		t.Fatalf("unexpected created rule count: %d", len(result.Rules))
	}
	for index, rule := range result.Rules {
		if rule.PolicyId != result.Policy.Id {
			t.Fatalf("unexpected rule policy id: %#v", rule)
		}
		if rule.TargetType != targetTypeEgress || rule.TargetId != egress.Id {
			t.Fatalf("unexpected rule target: %#v", rule)
		}
		if rule.SortOrder != index+1 {
			t.Fatalf("unexpected rule sort order: %#v", rule)
		}
	}

	storedRules := make([]*n5model.TrafficPolicyRule, 0)
	if err := database.GetDB().
		Model(&n5model.TrafficPolicyRule{}).
		Where("policy_id = ?", result.Policy.Id).
		Order("sort_order asc, id asc").
		Find(&storedRules).Error; err != nil {
		t.Fatalf("query stored rules failed: %v", err)
	}
	if len(storedRules) != len(result.Rules) {
		t.Fatalf("unexpected stored rule count: %d", len(storedRules))
	}
}
