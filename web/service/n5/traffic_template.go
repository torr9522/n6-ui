package n5

import (
	"strings"
	"x-ui/database"
	n5model "x-ui/database/model/n5"
	"x-ui/util/common"
	n5templates "x-ui/web/service/n5/templates"

	"gorm.io/gorm"
)

type TrafficTemplateSummary struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	RuleCount   int    `json:"ruleCount"`
}

type TrafficTemplatePreview struct {
	Name        string             `json:"name"`
	DisplayName string             `json:"displayName"`
	Description string             `json:"description"`
	Rules       []n5templates.Rule `json:"rules"`
}

type TrafficTemplateCreateRequest struct {
	TemplateName string `json:"templateName" form:"templateName"`
	PolicyName   string `json:"policyName" form:"policyName"`
	InboundId    int    `json:"inboundId" form:"inboundId"`
	TargetType   string `json:"targetType" form:"targetType"`
	TargetId     int    `json:"targetId" form:"targetId"`
}

type TrafficTemplateCreateResult struct {
	TemplateName string                        `json:"templateName"`
	Policy       *n5model.TrafficPolicy        `json:"policy"`
	Rules        []*n5model.TrafficPolicyRule  `json:"rules"`
	Binding      *n5model.TrafficPolicyBinding `json:"binding"`
}

type TrafficTemplateService struct {
}

func (s *TrafficTemplateService) List() ([]*TrafficTemplateSummary, error) {
	definitions := n5templates.All()
	items := make([]*TrafficTemplateSummary, 0, len(definitions))
	for _, definition := range definitions {
		items = append(items, &TrafficTemplateSummary{
			Name:        definition.Name,
			DisplayName: definition.DisplayName,
			Description: definition.Description,
			RuleCount:   len(definition.Rules),
		})
	}
	return items, nil
}

func (s *TrafficTemplateService) Preview(name string) (*TrafficTemplatePreview, error) {
	definition, err := s.getTemplate(name)
	if err != nil {
		return nil, err
	}
	rules := make([]n5templates.Rule, 0, len(definition.Rules))
	rules = append(rules, definition.Rules...)
	return &TrafficTemplatePreview{
		Name:        definition.Name,
		DisplayName: definition.DisplayName,
		Description: definition.Description,
		Rules:       rules,
	}, nil
}

func (s *TrafficTemplateService) Create(req *TrafficTemplateCreateRequest) (*TrafficTemplateCreateResult, error) {
	if req == nil {
		return nil, common.NewError("traffic template request is nil")
	}
	definition, err := s.getTemplate(req.TemplateName)
	if err != nil {
		return nil, err
	}
	if req.InboundId <= 0 {
		return nil, common.NewError("inbound id is required")
	}
	if _, err := getInboundByID(req.InboundId); err != nil {
		return nil, err
	}

	targetType := normalizeTargetType(req.TargetType)
	if err := validateTarget(targetType, req.TargetId); err != nil {
		return nil, err
	}

	policyName := strings.TrimSpace(req.PolicyName)
	if policyName == "" {
		policyName = definition.DisplayName
	}

	db := database.GetDB()
	result := &TrafficTemplateCreateResult{
		TemplateName: definition.Name,
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		policy := &n5model.TrafficPolicy{
			Name:              policyName,
			Enabled:           true,
			DefaultTargetType: targetType,
			DefaultTargetId:   req.TargetId,
		}
		if err := tx.Create(policy).Error; err != nil {
			return err
		}
		result.Policy = policy

		rules := make([]*n5model.TrafficPolicyRule, 0, len(definition.Rules))
		for index, item := range definition.Rules {
			ruleType := normalizeRuleType(item.RuleType)
			matchMode := normalizeMatchMode(item.MatchMode)
			matchValue := strings.TrimSpace(item.MatchValue)

			switch ruleType {
			case ruleTypeDomain:
				if err := validateDomainRule(matchMode, matchValue); err != nil {
					return err
				}
			case ruleTypeIP:
				if err := validateIPRule(matchMode, matchValue); err != nil {
					return err
				}
			default:
				return common.NewError("invalid template rule type")
			}

			rule := &n5model.TrafficPolicyRule{
				PolicyId:   policy.Id,
				RuleType:   ruleType,
				MatchMode:  matchMode,
				MatchValue: matchValue,
				TargetType: targetType,
				TargetId:   req.TargetId,
				SortOrder:  index + 1,
				Enabled:    true,
			}
			if err := tx.Create(rule).Error; err != nil {
				return err
			}
			rules = append(rules, rule)
		}
		result.Rules = rules

		if err := tx.Where("inbound_id = ?", req.InboundId).Delete(&n5model.TrafficPolicyBinding{}).Error; err != nil {
			return err
		}
		binding := &n5model.TrafficPolicyBinding{
			InboundId: req.InboundId,
			PolicyId:  policy.Id,
			Enabled:   true,
		}
		if err := tx.Create(binding).Error; err != nil {
			return err
		}
		result.Binding = binding
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *TrafficTemplateService) getTemplate(name string) (*n5templates.Definition, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return nil, common.NewError("template name is required")
	}
	definition := n5templates.Find(name)
	if definition == nil {
		return nil, common.NewError("traffic template not found")
	}
	return definition, nil
}
