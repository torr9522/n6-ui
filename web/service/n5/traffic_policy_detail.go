package n5

import (
	"x-ui/database"
	legacyModel "x-ui/database/model"
	n5model "x-ui/database/model/n5"
	"x-ui/util/common"
)

type TrafficPolicyBindingDetail struct {
	Id          int    `json:"id"`
	InboundId   int    `json:"inboundId"`
	InboundTag  string `json:"inboundTag"`
	InboundName string `json:"inboundName"`
	Enabled     bool   `json:"enabled"`
}

type TrafficPolicyDetail struct {
	Policy       *n5model.TrafficPolicy        `json:"policy"`
	Rules        []*n5model.TrafficPolicyRule  `json:"rules"`
	Bindings     []*TrafficPolicyBindingDetail `json:"bindings"`
	RuleCount    int                           `json:"ruleCount"`
	BindingCount int                           `json:"bindingCount"`
}

type TrafficPolicyDetailService struct {
	policyService TrafficPolicyService
}

func (s *TrafficPolicyDetailService) Get(id int) (*TrafficPolicyDetail, error) {
	if id <= 0 {
		return nil, common.NewError("invalid policy id")
	}

	policy, err := s.policyService.GetPolicy(id)
	if err != nil {
		return nil, err
	}
	rules, err := s.policyService.ListRules(id)
	if err != nil {
		return nil, err
	}
	bindings, err := s.policyService.ListBindingsByPolicy(id)
	if err != nil {
		return nil, err
	}

	bindingDetails := make([]*TrafficPolicyBindingDetail, 0, len(bindings))
	for _, binding := range bindings {
		item := &TrafficPolicyBindingDetail{
			Id:        binding.Id,
			InboundId: binding.InboundId,
			Enabled:   binding.Enabled,
		}
		inbound := &legacyModel.Inbound{}
		if err := database.GetDB().Model(&legacyModel.Inbound{}).Where("id = ?", binding.InboundId).First(inbound).Error; err == nil {
			item.InboundTag = inbound.Tag
			item.InboundName = inbound.Remark
		}
		bindingDetails = append(bindingDetails, item)
	}

	return &TrafficPolicyDetail{
		Policy:       policy,
		Rules:        rules,
		Bindings:     bindingDetails,
		RuleCount:    len(rules),
		BindingCount: len(bindingDetails),
	}, nil
}
