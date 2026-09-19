package simple

import (
	"encoding/base64"
	"encoding/json"
	"github.com/torr9522/n6-ui/database"
	"github.com/torr9522/n6-ui/database/model"
	n5model "github.com/torr9522/n6-ui/database/model/n5"
	"github.com/torr9522/n6-ui/logger"
	"github.com/torr9522/n6-ui/util/common"
	coreservice "github.com/torr9522/n6-ui/web/service"
	n5service "github.com/torr9522/n6-ui/web/service/n5"
	"io"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

const (
	simpleRuleRemarkPrefix        = "n5-simple|"
	simpleExecutionRemarkPrefix   = "n5-simple-exec|"
	simpleExecutionRemarkVersion  = 1
	simpleExecutionRemarkMaxBytes = 128 * 1024
	simpleRuleIDPrefix            = "simple-rule:"
	legacySimpleRuleIDPrefix      = "legacy-simple-rule:"

	simpleTrafficAll          = "all"
	simpleTrafficGroup        = "group"
	simpleTrafficAI           = "ai"
	simpleTrafficGame         = "game"
	simpleTrafficStreaming    = "streaming"
	simpleTrafficCustomDomain = "custom-domain"
)

type inboundManager interface {
	GetInbound(id int) (*model.Inbound, error)
	GetAllInbounds() ([]*model.Inbound, error)
}

type trafficPolicyManager interface {
	Create(policy *n5model.TrafficPolicy) (*n5model.TrafficPolicy, error)
	GetPolicy(id int) (*n5model.TrafficPolicy, error)
	List() ([]*n5model.TrafficPolicy, error)
	UpdatePolicy(policy *n5model.TrafficPolicy) (*n5model.TrafficPolicy, error)
	DeletePolicy(id int) error
	EnablePolicy(id int) (*n5model.TrafficPolicy, error)
	DisablePolicy(id int) (*n5model.TrafficPolicy, error)
	AddRule(rule *n5model.TrafficPolicyRule) (*n5model.TrafficPolicyRule, error)
	ListRules(policyId int) ([]*n5model.TrafficPolicyRule, error)
	UpdateRule(rule *n5model.TrafficPolicyRule) (*n5model.TrafficPolicyRule, error)
	DeleteRule(ruleId int) error
	ListBindings() ([]*n5model.TrafficPolicyBinding, error)
	BindInboundPolicy(inboundId int, policyId int) (*n5model.TrafficPolicyBinding, error)
}

type trafficTemplateManager interface {
	List() ([]*n5service.TrafficTemplateSummary, error)
	Create(req *n5service.TrafficTemplateCreateRequest) (*n5service.TrafficTemplateCreateResult, error)
}

type trafficRuleGroupManager interface {
	ListGroups() ([]*TrafficRuleGroup, error)
	ListGroupOptions() ([]*TrafficRuleGroupOption, error)
	GetGroup(id int) (*TrafficRuleGroup, error)
}

type SimpleRule struct {
	Id              int    `json:"id"`
	RuleId          string `json:"ruleId"`
	PolicyId        int    `json:"policyId"`
	InboundId       int    `json:"inboundId"`
	InboundName     string `json:"inboundName"`
	InboundTag      string `json:"inboundTag"`
	TrafficType     string `json:"trafficType"`
	TrafficLabel    string `json:"trafficLabel"`
	GroupId         int    `json:"groupId"`
	GroupName       string `json:"groupName"`
	GroupType       string `json:"groupType"`
	CustomDomain    string `json:"customDomain"`
	EgressId        int    `json:"egressId"`
	EgressName      string `json:"egressName"`
	EgressTag       string `json:"egressTag"`
	Status          string `json:"status"`
	Enabled         bool   `json:"enabled"`
	RuleCount       int    `json:"ruleCount"`
	CreatedAt       int64  `json:"createdAt"`
	UpdatedAt       int64  `json:"updatedAt"`
	PolicyName      string `json:"policyName"`
	PolicyRemark    string `json:"policyRemark"`
	DefaultTargetId int    `json:"defaultTargetId"`
}

type SimpleRuleListResult struct {
	Rules        []*SimpleRule             `json:"rules"`
	Inbounds     []*SimpleInboundOption    `json:"inbounds"`
	Egresses     []*SimpleRuleEgressOption `json:"egresses"`
	Groups       []*TrafficRuleGroupOption `json:"groups"`
	TrafficTypes []*SimpleTrafficOption    `json:"trafficTypes"`
}

type SimpleInboundOption struct {
	Id      int    `json:"id"`
	Name    string `json:"name"`
	Tag     string `json:"tag"`
	Enabled bool   `json:"enabled"`
}

type SimpleRuleEgressOption struct {
	Id      int    `json:"id"`
	Name    string `json:"name"`
	Tag     string `json:"tag"`
	Address string `json:"address"`
	Enabled bool   `json:"enabled"`
}

type SimpleTrafficOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

type CreateSimpleRuleRequest struct {
	InboundId    int    `json:"inboundId" form:"inboundId"`
	TrafficType  string `json:"trafficType" form:"trafficType"`
	GroupId      int    `json:"groupId" form:"groupId"`
	EgressId     int    `json:"egressId" form:"egressId"`
	CustomDomain string `json:"customDomain" form:"customDomain"`
}

type SimpleRuleConflictPreview struct {
	HasConflict bool                  `json:"hasConflict"`
	Warning     string                `json:"warning"`
	Conflicts   []*SimpleRuleConflict `json:"conflicts"`
}

type SimpleRuleConflict struct {
	CurrentLabel       string `json:"currentLabel"`
	CurrentMatchLabel  string `json:"currentMatchLabel"`
	CurrentValue       string `json:"currentValue"`
	ExistingLabel      string `json:"existingLabel"`
	ExistingMatchLabel string `json:"existingMatchLabel"`
	ExistingValue      string `json:"existingValue"`
	Relation           string `json:"relation"`
	PriorityDirection  string `json:"priorityDirection"`
	Message            string `json:"message"`
}

type simpleExecutionRemark struct {
	Version int                    `json:"version"`
	Items   []*simpleExecutionItem `json:"items"`
}

type simpleExecutionItem struct {
	TrafficType  string `json:"trafficType"`
	GroupId      int    `json:"groupId,omitempty"`
	GroupName    string `json:"groupName,omitempty"`
	GroupType    string `json:"groupType,omitempty"`
	CustomDomain string `json:"customDomain,omitempty"`
	RuleIDs      []int  `json:"ruleIds,omitempty"`
}

type simpleExecutionRemarkStats struct {
	RawJSONBytes     int
	Base64Bytes      int
	TotalRemarkBytes int
}

type simpleRuleRemark struct {
	TrafficType  string
	GroupId      int
	GroupName    string
	GroupType    string
	CustomDomain string
}

type simplePolicyContext struct {
	Policy     *n5model.TrafficPolicy
	Binding    *n5model.TrafficPolicyBinding
	Inbound    *model.Inbound
	Rules      []*n5model.TrafficPolicyRule
	RuleMap    map[int]*n5model.TrafficPolicyRule
	ExecRemark *simpleExecutionRemark
	LegacyMeta *simpleRuleRemark
}

type RuleService struct {
	inboundService   inboundManager
	egressService    egressManager
	policyService    trafficPolicyManager
	templateService  trafficTemplateManager
	groupService     trafficRuleGroupManager
	mutationTestHook func(stage string) error
}

func NewRuleService() *RuleService {
	return &RuleService{
		inboundService:  &coreservice.InboundService{},
		egressService:   &n5service.EgressService{},
		policyService:   &n5service.TrafficPolicyService{},
		templateService: &n5service.TrafficTemplateService{},
		groupService:    NewTrafficRuleGroupService(),
	}
}

func (s *RuleService) getInboundService() inboundManager {
	if s.inboundService != nil {
		return s.inboundService
	}
	return &coreservice.InboundService{}
}

func (s *RuleService) getEgressService() egressManager {
	if s.egressService != nil {
		return s.egressService
	}
	return &n5service.EgressService{}
}

func (s *RuleService) getPolicyService() trafficPolicyManager {
	if s.policyService != nil {
		return s.policyService
	}
	return &n5service.TrafficPolicyService{}
}

func (s *RuleService) getTemplateService() trafficTemplateManager {
	if s.templateService != nil {
		return s.templateService
	}
	return &n5service.TrafficTemplateService{}
}

func (s *RuleService) getGroupService() trafficRuleGroupManager {
	if s.groupService != nil {
		return s.groupService
	}
	return NewTrafficRuleGroupService()
}

const (
	simpleMutationHookAfterEnsurePolicy = "after-ensure-policy"
	simpleMutationHookAfterAppendRules  = "after-append-rules"
	simpleMutationHookAfterPolicyUpdate = "after-policy-update"
	simpleMutationHookAfterRuleUpdate   = "after-rule-update"
	simpleMutationHookAfterRuleDelete   = "after-rule-delete"
)

func (s *RuleService) withSimpleRuleTransaction(fn func(*RuleService) (*SimpleRule, error)) (*SimpleRule, error) {
	policyService, ok := s.getPolicyService().(*n5service.TrafficPolicyService)
	if !ok {
		return fn(s)
	}

	var result *SimpleRule
	err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		txService := *s
		txService.policyService = policyService.WithDB(tx)
		item, err := fn(&txService)
		if err != nil {
			return err
		}
		result = item
		return nil
	})
	return result, err
}

func (s *RuleService) withSimpleRuleErrorTransaction(fn func(*RuleService) error) error {
	_, err := s.withSimpleRuleTransaction(func(txService *RuleService) (*SimpleRule, error) {
		return nil, fn(txService)
	})
	return err
}

func (s *RuleService) runSimpleMutationHook(stage string) error {
	if s.mutationTestHook == nil {
		return nil
	}
	return s.mutationTestHook(stage)
}

func (s *RuleService) ListSimpleRules() (*SimpleRuleListResult, error) {
	policies, err := s.getPolicyService().List()
	if err != nil {
		return nil, err
	}
	bindings, err := s.getPolicyService().ListBindings()
	if err != nil {
		return nil, err
	}
	inbounds, err := s.getInboundService().GetAllInbounds()
	if err != nil {
		return nil, err
	}
	egresses, err := s.getEgressService().List()
	if err != nil {
		return nil, err
	}
	groups, err := s.getGroupService().ListGroupOptions()
	if err != nil {
		return nil, err
	}

	bindingByPolicy := make(map[int]*n5model.TrafficPolicyBinding, len(bindings))
	for _, binding := range bindings {
		bindingByPolicy[binding.PolicyId] = binding
	}

	inboundByID := make(map[int]*model.Inbound, len(inbounds))
	inboundOptions := make([]*SimpleInboundOption, 0, len(inbounds))
	for _, inbound := range inbounds {
		inboundByID[inbound.Id] = inbound
		inboundOptions = append(inboundOptions, &SimpleInboundOption{
			Id:      inbound.Id,
			Name:    simpleInboundName(inbound),
			Tag:     inbound.Tag,
			Enabled: inbound.Enable,
		})
	}

	egressByID := make(map[int]*n5model.Egress, len(egresses))
	egressOptions := make([]*SimpleRuleEgressOption, 0, len(egresses))
	for _, egress := range egresses {
		egressByID[egress.Id] = egress
		address, port := extractAddressPort(egress.OutboundJSON)
		label := address
		if address != "" && port > 0 {
			label = address + ":" + strconv.Itoa(port)
		}
		egressOptions = append(egressOptions, &SimpleRuleEgressOption{
			Id:      egress.Id,
			Name:    egress.Name,
			Tag:     egress.Tag,
			Address: label,
			Enabled: egress.Enabled,
		})
	}

	groupByID := make(map[int]*TrafficRuleGroupOption, len(groups))
	groupByType := make(map[string]*TrafficRuleGroupOption, len(groups))
	for _, group := range groups {
		groupByID[group.Id] = group
		if group.GroupType != "" {
			groupByType[group.GroupType] = group
		}
	}

	items := make([]*SimpleRule, 0)
	for _, policy := range policies {
		binding := bindingByPolicy[policy.Id]
		if binding == nil {
			continue
		}
		inbound := inboundByID[binding.InboundId]
		if inbound == nil {
			continue
		}

		rules, err := s.getPolicyService().ListRules(policy.Id)
		if err != nil {
			return nil, err
		}
		ruleMap := buildRuleMap(rules)
		enabled := policy.Enabled && binding.Enabled

		if isNewSimpleExecutionRemark(policy.Remark) {
			execRemark, err := decodeSimpleExecutionRemark(policy.Remark)
			if err != nil {
				logger.Warningf("skip corrupted simple exec policy in list, policy=%d err=%v", policy.Id, err)
				continue
			}
			for _, item := range execRemark.Items {
				row, ok := buildSimpleRuleFromExecutionItem(policy, binding, inbound, enabled, item, ruleMap, egressByID)
				if !ok {
					continue
				}
				items = append(items, row)
			}
			if strings.EqualFold(policy.DefaultTargetType, "egress") && policy.DefaultTargetId > 0 {
				egress := egressByID[policy.DefaultTargetId]
				if egress != nil {
					items = append(items, buildDefaultSimpleRule(policy, binding, inbound, enabled, egress))
				}
			}
			continue
		}

		if isLegacySimpleRemark(policy.Remark) {
			meta, ok := parseSimpleRuleRemark(policy.Remark)
			if !ok {
				logger.Warningf("skip corrupted legacy simple policy in list, policy=%d", policy.Id)
				continue
			}
			row, ok := buildLegacySimpleRule(policy, binding, inbound, enabled, meta, rules, ruleMap, egressByID, groupByID, groupByType)
			if !ok {
				continue
			}
			items = append(items, row)
			continue
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].InboundId != items[j].InboundId {
			return items[i].InboundId < items[j].InboundId
		}
		if items[i].TrafficType == simpleTrafficAll && items[j].TrafficType != simpleTrafficAll {
			return false
		}
		if items[i].TrafficType != simpleTrafficAll && items[j].TrafficType == simpleTrafficAll {
			return true
		}
		if items[i].PolicyId != items[j].PolicyId {
			return items[i].PolicyId < items[j].PolicyId
		}
		return items[i].RuleId < items[j].RuleId
	})

	return &SimpleRuleListResult{
		Rules:        items,
		Inbounds:     inboundOptions,
		Egresses:     egressOptions,
		Groups:       groups,
		TrafficTypes: s.buildTrafficTypes(),
	}, nil
}

func (s *RuleService) CreateSimpleRule(req *CreateSimpleRuleRequest) (*SimpleRule, error) {
	return s.withSimpleRuleTransaction(func(txService *RuleService) (*SimpleRule, error) {
		return txService.createSimpleRule(req)
	})
}

func (s *RuleService) createSimpleRule(req *CreateSimpleRuleRequest) (*SimpleRule, error) {
	if req == nil {
		return nil, common.NewError("simple rule request is nil")
	}
	if req.GroupId > 0 {
		req.TrafficType = simpleTrafficGroup
	}
	trafficType := normalizeSimpleTrafficType(req.TrafficType)
	if !isSupportedSimpleTrafficType(trafficType) {
		return nil, common.NewError("unsupported traffic type")
	}
	inbound, err := s.getInboundService().GetInbound(req.InboundId)
	if err != nil {
		return nil, err
	}
	egress, err := s.getEgressService().Get(req.EgressId)
	if err != nil {
		return nil, err
	}

	ctx, err := s.ensureSimpleExecutionPolicy(inbound)
	if err != nil {
		return nil, err
	}
	if err := s.runSimpleMutationHook(simpleMutationHookAfterEnsurePolicy); err != nil {
		return nil, err
	}

	if trafficType == simpleTrafficAll {
		if strings.EqualFold(ctx.Policy.DefaultTargetType, "egress") && ctx.Policy.DefaultTargetId > 0 {
			return nil, common.NewError("该入站已存在默认出口，请编辑现有规则")
		}
		updated, err := s.getPolicyService().UpdatePolicy(&n5model.TrafficPolicy{
			Id:                ctx.Policy.Id,
			Name:              ctx.Policy.Name,
			Remark:            ctx.Policy.Remark,
			Enabled:           ctx.Policy.Enabled,
			DefaultTargetType: "egress",
			DefaultTargetId:   egress.Id,
		})
		if err != nil {
			return nil, err
		}
		if err := s.runSimpleMutationHook(simpleMutationHookAfterPolicyUpdate); err != nil {
			return nil, err
		}
		ctx.Policy = updated
		return buildDefaultSimpleRule(ctx.Policy, ctx.Binding, inbound, ctx.Policy.Enabled && ctx.Binding.Enabled, egress), nil
	}

	var item *simpleExecutionItem
	switch trafficType {
	case simpleTrafficGroup:
		group, err := s.getGroupService().GetGroup(req.GroupId)
		if err != nil {
			return nil, err
		}
		if group == nil || group.Id <= 0 {
			return nil, common.NewError("traffic rule group not found")
		}
		if !group.Enabled {
			if isBuiltinSimpleGroupType(group.GroupType) {
				return nil, disabledBuiltinGroupError(group.GroupType)
			}
			return nil, common.NewError("traffic rule group is disabled")
		}
		item, err = s.buildExecutionItemFromRequest(req, trafficType, group)
		if err != nil {
			return nil, err
		}
		if findExecutionItemByKey(ctx.ExecRemark, simpleExecutionItemKey(item)) != nil {
			return nil, common.NewError("该入站已存在该分流规则，请编辑已有规则")
		}
		item.GroupId = group.Id
		item.GroupName = group.Name
		item.GroupType = group.GroupType
		item.RuleIDs, err = s.appendGroupSnapshotRules(ctx.Policy.Id, egress.Id, group)
		if err != nil {
			return nil, err
		}
		if err := s.runSimpleMutationHook(simpleMutationHookAfterAppendRules); err != nil {
			return nil, err
		}
	case simpleTrafficCustomDomain:
		item, err = s.buildExecutionItemFromRequest(req, trafficType, nil)
		if err != nil {
			return nil, err
		}
		if findExecutionItemByKey(ctx.ExecRemark, simpleExecutionItemKey(item)) != nil {
			return nil, common.NewError("该入站已存在该分流规则，请编辑已有规则")
		}
		matchMode, matchValue, displayValue, err := parseCustomDomainRule(req.CustomDomain)
		if err != nil {
			return nil, err
		}
		rule, err := s.getPolicyService().AddRule(&n5model.TrafficPolicyRule{
			PolicyId:   ctx.Policy.Id,
			RuleType:   "domain",
			MatchMode:  matchMode,
			MatchValue: matchValue,
			TargetType: "egress",
			TargetId:   egress.Id,
			Enabled:    true,
		})
		if err != nil {
			return nil, err
		}
		item.CustomDomain = displayValue
		item.RuleIDs = []int{rule.Id}
		if err := s.runSimpleMutationHook(simpleMutationHookAfterAppendRules); err != nil {
			return nil, err
		}
	case simpleTrafficAI, simpleTrafficGame, simpleTrafficStreaming:
		group, err := s.findBuiltinGroupByType(trafficType)
		if err != nil {
			return nil, err
		}
		if group == nil || group.Id <= 0 {
			return nil, common.NewError("traffic rule group not found")
		}
		if !group.Enabled {
			return nil, disabledBuiltinGroupError(trafficType)
		}
		item, err = s.buildExecutionItemFromRequest(req, trafficType, group)
		if err != nil {
			return nil, err
		}
		if findExecutionItemByKey(ctx.ExecRemark, simpleExecutionItemKey(item)) != nil {
			return nil, common.NewError("该入站已存在该分流规则，请编辑已有规则")
		}
		item.TrafficType = simpleTrafficGroup
		item.GroupId = group.Id
		item.GroupName = group.Name
		item.GroupType = group.GroupType
		item.RuleIDs, err = s.appendGroupSnapshotRules(ctx.Policy.Id, egress.Id, group)
		if err != nil {
			return nil, err
		}
		if err := s.runSimpleMutationHook(simpleMutationHookAfterAppendRules); err != nil {
			return nil, err
		}
	default:
		return nil, common.NewError("unsupported traffic type")
	}

	ctx.ExecRemark.Items = append(ctx.ExecRemark.Items, item)
	if err := s.saveExecutionRemark(ctx.Policy, ctx.ExecRemark); err != nil {
		return nil, err
	}
	ctx.Rules, _ = s.getPolicyService().ListRules(ctx.Policy.Id)
	ctx.RuleMap = buildRuleMap(ctx.Rules)
	row, _ := buildSimpleRuleFromExecutionItem(ctx.Policy, ctx.Binding, inbound, ctx.Policy.Enabled && ctx.Binding.Enabled, item, ctx.RuleMap, map[int]*n5model.Egress{
		egress.Id: egress,
	})
	return row, nil
}

func (s *RuleService) CheckSimpleRuleConflicts(req *CreateSimpleRuleRequest, editingRuleID string) (*SimpleRuleConflictPreview, error) {
	preview := &SimpleRuleConflictPreview{
		Conflicts: []*SimpleRuleConflict{},
	}
	if req == nil {
		return nil, common.NewError("simple rule request is nil")
	}
	if req.GroupId > 0 {
		req.TrafficType = simpleTrafficGroup
	}
	trafficType := normalizeSimpleTrafficType(req.TrafficType)
	if !isSupportedSimpleTrafficType(trafficType) {
		return nil, common.NewError("unsupported traffic type")
	}
	if trafficType == simpleTrafficAll {
		return preview, nil
	}
	if _, err := s.getInboundService().GetInbound(req.InboundId); err != nil {
		return nil, err
	}
	ctx, err := s.loadSimplePolicyContextByInbound(req.InboundId)
	if err != nil {
		return nil, err
	}
	if ctx == nil || ctx.ExecRemark == nil {
		return preview, nil
	}

	excludeKey := ""
	if strings.TrimSpace(editingRuleID) != "" {
		editCtx, item, legacy, err := s.loadSimpleRuleSelection(editingRuleID)
		if err != nil {
			return nil, err
		}
		if legacy {
			return preview, nil
		}
		if editCtx == nil || editCtx.Inbound == nil || editCtx.Inbound.Id != req.InboundId {
			return nil, common.NewError("editing rule inbound is not supported")
		}
		excludeKey = simpleExecutionItemKey(item)
	}

	candidates, err := s.conflictCandidatesForRequest(req, trafficType)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return preview, nil
	}
	existing := s.conflictCandidatesForContext(ctx, excludeKey)
	for _, current := range candidates {
		for _, item := range existing {
			overlap, relation := simpleRuleOverlap(current, item)
			if !overlap {
				continue
			}
			conflict := buildSimpleRuleConflict(current, item, relation)
			preview.Conflicts = append(preview.Conflicts, conflict)
			if len(preview.Conflicts) >= 5 {
				break
			}
		}
		if len(preview.Conflicts) >= 5 {
			break
		}
	}
	preview.HasConflict = len(preview.Conflicts) > 0
	if preview.HasConflict {
		preview.Warning = buildSimpleRuleConflictWarning(preview.Conflicts)
	}
	return preview, nil
}

func (s *RuleService) UpdateSimpleRule(ruleID string, req *CreateSimpleRuleRequest) (*SimpleRule, error) {
	return s.withSimpleRuleTransaction(func(txService *RuleService) (*SimpleRule, error) {
		return txService.updateSimpleRule(ruleID, req)
	})
}

func (s *RuleService) updateSimpleRule(ruleID string, req *CreateSimpleRuleRequest) (*SimpleRule, error) {
	if strings.TrimSpace(ruleID) == "" {
		return nil, common.NewError("rule id is required")
	}
	if req == nil {
		return nil, common.NewError("simple rule request is nil")
	}
	if req.GroupId > 0 {
		req.TrafficType = simpleTrafficGroup
	}
	egress, err := s.getEgressService().Get(req.EgressId)
	if err != nil {
		return nil, err
	}

	ctx, item, legacy, err := s.loadSimpleRuleSelection(ruleID)
	if err != nil {
		return nil, err
	}
	if req.InboundId > 0 && req.InboundId != ctx.Inbound.Id {
		return nil, common.NewError("editing rule inbound is not supported")
	}

	if legacy {
		return s.updateLegacySimpleRule(ctx, req, egress)
	}

	currentGroup, err := s.resolveExecutionGroup(item)
	if err != nil {
		return nil, err
	}
	requestItem, err := s.buildExecutionItemFromRequest(req, normalizeSimpleTrafficType(req.TrafficType), currentGroup)
	if err != nil {
		return nil, err
	}
	if item.TrafficType == simpleTrafficCustomDomain {
		return s.updateCustomDomainSimpleRule(ctx, item, requestItem, egress)
	}
	if simpleExecutionItemKey(item) != simpleExecutionItemKey(requestItem) {
		return nil, common.NewError("editing rule identity is not supported")
	}

	if item.TrafficType == simpleTrafficAll {
		policy, err := s.getPolicyService().UpdatePolicy(&n5model.TrafficPolicy{
			Id:                ctx.Policy.Id,
			Name:              ctx.Policy.Name,
			Remark:            ctx.Policy.Remark,
			Enabled:           ctx.Policy.Enabled,
			DefaultTargetType: "egress",
			DefaultTargetId:   egress.Id,
		})
		if err != nil {
			return nil, err
		}
		if err := s.runSimpleMutationHook(simpleMutationHookAfterPolicyUpdate); err != nil {
			return nil, err
		}
		return buildDefaultSimpleRule(policy, ctx.Binding, ctx.Inbound, policy.Enabled && ctx.Binding.Enabled, egress), nil
	}

	itemRules, err := getExecutionItemRules(item, ctx.RuleMap)
	if err != nil {
		return nil, err
	}
	for _, rule := range itemRules {
		if _, err := s.getPolicyService().UpdateRule(&n5model.TrafficPolicyRule{
			Id:         rule.Id,
			RuleType:   rule.RuleType,
			MatchMode:  rule.MatchMode,
			MatchValue: rule.MatchValue,
			TargetType: "egress",
			TargetId:   egress.Id,
			SortOrder:  rule.SortOrder,
			Enabled:    rule.Enabled,
		}); err != nil {
			return nil, err
		}
	}
	if err := s.runSimpleMutationHook(simpleMutationHookAfterRuleUpdate); err != nil {
		return nil, err
	}
	ctx.Rules, _ = s.getPolicyService().ListRules(ctx.Policy.Id)
	ctx.RuleMap = buildRuleMap(ctx.Rules)
	row, _ := buildSimpleRuleFromExecutionItem(ctx.Policy, ctx.Binding, ctx.Inbound, ctx.Policy.Enabled && ctx.Binding.Enabled, item, ctx.RuleMap, map[int]*n5model.Egress{
		egress.Id: egress,
	})
	return row, nil
}

func (s *RuleService) updateCustomDomainSimpleRule(ctx *simplePolicyContext, item *simpleExecutionItem, requestItem *simpleExecutionItem, egress *n5model.Egress) (*SimpleRule, error) {
	if ctx == nil || item == nil || requestItem == nil || egress == nil {
		return nil, common.NewError("invalid simple custom rule")
	}
	if item.TrafficType != simpleTrafficCustomDomain || requestItem.TrafficType != simpleTrafficCustomDomain {
		return nil, common.NewError("editing rule identity is not supported")
	}
	newKey := simpleExecutionItemKey(requestItem)
	oldKey := simpleExecutionItemKey(item)
	if newKey != oldKey && findExecutionItemByKey(ctx.ExecRemark, newKey) != nil {
		return nil, common.NewError("该入站已存在该分流规则，请编辑已有规则")
	}
	itemRules, err := getExecutionItemRules(item, ctx.RuleMap)
	if err != nil {
		return nil, err
	}
	if len(itemRules) != 1 {
		return nil, common.NewError("simple execution metadata is corrupted")
	}
	matchMode, matchValue, displayValue, err := parseCustomDomainRule(requestItem.CustomDomain)
	if err != nil {
		return nil, err
	}
	rule := itemRules[0]
	if _, err := s.getPolicyService().UpdateRule(&n5model.TrafficPolicyRule{
		Id:         rule.Id,
		RuleType:   "domain",
		MatchMode:  matchMode,
		MatchValue: matchValue,
		TargetType: "egress",
		TargetId:   egress.Id,
		SortOrder:  rule.SortOrder,
		Enabled:    rule.Enabled,
	}); err != nil {
		return nil, err
	}
	if err := s.runSimpleMutationHook(simpleMutationHookAfterRuleUpdate); err != nil {
		return nil, err
	}
	item.CustomDomain = displayValue
	if err := s.saveExecutionRemark(ctx.Policy, ctx.ExecRemark); err != nil {
		return nil, err
	}
	ctx.Rules, _ = s.getPolicyService().ListRules(ctx.Policy.Id)
	ctx.RuleMap = buildRuleMap(ctx.Rules)
	row, _ := buildSimpleRuleFromExecutionItem(ctx.Policy, ctx.Binding, ctx.Inbound, ctx.Policy.Enabled && ctx.Binding.Enabled, item, ctx.RuleMap, map[int]*n5model.Egress{
		egress.Id: egress,
	})
	return row, nil
}

func (s *RuleService) DeleteSimpleRule(ruleID string) error {
	return s.withSimpleRuleErrorTransaction(func(txService *RuleService) error {
		return txService.deleteSimpleRule(ruleID)
	})
}

func (s *RuleService) deleteSimpleRule(ruleID string) error {
	if strings.TrimSpace(ruleID) == "" {
		return common.NewError("rule id is required")
	}

	ctx, item, legacy, err := s.loadSimpleRuleSelection(ruleID)
	if err != nil {
		return err
	}
	if legacy {
		return s.getPolicyService().DeletePolicy(ctx.Policy.Id)
	}

	if item.TrafficType == simpleTrafficAll {
		if _, err := s.getPolicyService().UpdatePolicy(&n5model.TrafficPolicy{
			Id:                ctx.Policy.Id,
			Name:              ctx.Policy.Name,
			Remark:            ctx.Policy.Remark,
			Enabled:           ctx.Policy.Enabled,
			DefaultTargetType: "",
			DefaultTargetId:   0,
		}); err != nil {
			return err
		}
		if err := s.runSimpleMutationHook(simpleMutationHookAfterPolicyUpdate); err != nil {
			return err
		}
		return s.cleanupEmptyExecutionPolicy(ctx.Policy.Id)
	}

	itemRules, err := getExecutionItemRules(item, ctx.RuleMap)
	if err != nil {
		return err
	}
	for _, rule := range itemRules {
		if err := s.getPolicyService().DeleteRule(rule.Id); err != nil {
			return err
		}
		if err := s.runSimpleMutationHook(simpleMutationHookAfterRuleDelete); err != nil {
			return err
		}
	}
	removeExecutionItem(ctx.ExecRemark, simpleExecutionItemKey(item))
	if len(ctx.ExecRemark.Items) == 0 {
		if err := s.saveExecutionRemark(ctx.Policy, ctx.ExecRemark); err != nil {
			return err
		}
		return s.cleanupEmptyExecutionPolicy(ctx.Policy.Id)
	}
	if err := s.saveExecutionRemark(ctx.Policy, ctx.ExecRemark); err != nil {
		return err
	}
	return s.cleanupEmptyExecutionPolicy(ctx.Policy.Id)
}

func (s *RuleService) appendGroupSnapshotRules(policyId int, egressId int, group *TrafficRuleGroup) ([]int, error) {
	if group == nil || group.Id <= 0 {
		return nil, common.NewError("traffic rule group not found")
	}
	if len(group.Rules) == 0 {
		return nil, common.NewError("traffic rule group has no rules")
	}
	created := make([]int, 0, len(group.Rules))
	for _, item := range group.Rules {
		if item == nil || !item.Enabled {
			continue
		}
		rule, err := s.getPolicyService().AddRule(&n5model.TrafficPolicyRule{
			PolicyId:   policyId,
			RuleType:   item.RuleType,
			MatchMode:  item.MatchMode,
			MatchValue: item.MatchValue,
			TargetType: "egress",
			TargetId:   egressId,
			Enabled:    true,
		})
		if err != nil {
			return nil, err
		}
		created = append(created, rule.Id)
	}
	if len(created) == 0 {
		return nil, common.NewError("traffic rule group has no enabled rules")
	}
	return created, nil
}

func (s *RuleService) buildExecutionItemFromRequest(req *CreateSimpleRuleRequest, trafficType string, group *TrafficRuleGroup) (*simpleExecutionItem, error) {
	item := &simpleExecutionItem{TrafficType: normalizeSimpleTrafficType(trafficType)}
	switch item.TrafficType {
	case simpleTrafficAll:
		return item, nil
	case simpleTrafficGroup:
		if req.GroupId <= 0 {
			return nil, common.NewError("traffic rule group is required")
		}
		item.GroupId = req.GroupId
		if group != nil {
			item.GroupName = group.Name
			item.GroupType = group.GroupType
		}
		return item, nil
	case simpleTrafficCustomDomain:
		_, _, displayValue, err := parseCustomDomainRule(req.CustomDomain)
		if err != nil {
			return nil, err
		}
		item.CustomDomain = displayValue
		return item, nil
	case simpleTrafficAI, simpleTrafficGame, simpleTrafficStreaming:
		item.TrafficType = simpleTrafficGroup
		if group != nil {
			item.GroupId = group.Id
			item.GroupName = group.Name
			item.GroupType = group.GroupType
		} else {
			item.GroupType = normalizeSimpleTrafficType(trafficType)
		}
		return item, nil
	default:
		return nil, common.NewError("unsupported traffic type")
	}
}

func (s *RuleService) resolveExecutionGroup(item *simpleExecutionItem) (*TrafficRuleGroup, error) {
	if item == nil || item.TrafficType != simpleTrafficGroup {
		return nil, nil
	}
	if item.GroupId > 0 {
		group, err := s.getGroupService().GetGroup(item.GroupId)
		if err == nil && group != nil && group.Id > 0 {
			return group, nil
		}
	}
	if groupType := normalizeSimpleGroupType(item.GroupType); groupType != "" {
		return s.findBuiltinGroupByType(groupType)
	}
	return nil, nil
}

func (s *RuleService) findBuiltinGroupByType(groupType string) (*TrafficRuleGroup, error) {
	groups, err := s.getGroupService().ListGroups()
	if err != nil {
		return nil, err
	}
	for _, group := range groups {
		if group != nil && group.GroupType == groupType {
			return s.getGroupService().GetGroup(group.Id)
		}
	}
	return nil, common.NewError("traffic rule group not found")
}

func (s *RuleService) ensureSimpleExecutionPolicy(inbound *model.Inbound) (*simplePolicyContext, error) {
	if inbound == nil || inbound.Id <= 0 {
		return nil, common.NewError("invalid inbound")
	}
	ctx, err := s.loadSimplePolicyContextByInbound(inbound.Id)
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		remarkValue, _, err := buildSimpleExecutionRemark(&simpleExecutionRemark{Version: simpleExecutionRemarkVersion})
		if err != nil {
			return nil, err
		}
		policy, err := s.getPolicyService().Create(&n5model.TrafficPolicy{
			Name:    simpleExecutionPolicyName(inbound),
			Remark:  remarkValue,
			Enabled: true,
		})
		if err != nil {
			return nil, err
		}
		binding, err := s.getPolicyService().BindInboundPolicy(inbound.Id, policy.Id)
		if err != nil {
			_ = s.getPolicyService().DeletePolicy(policy.Id)
			return nil, err
		}
		return &simplePolicyContext{
			Policy:     policy,
			Binding:    binding,
			Inbound:    inbound,
			Rules:      []*n5model.TrafficPolicyRule{},
			RuleMap:    map[int]*n5model.TrafficPolicyRule{},
			ExecRemark: &simpleExecutionRemark{Version: simpleExecutionRemarkVersion},
		}, nil
	}
	if ctx.ExecRemark != nil {
		return ctx, nil
	}

	execRemark, err := s.buildExecutionRemarkFromLegacy(ctx.LegacyMeta, ctx.Rules)
	if err != nil {
		return nil, err
	}
	remarkValue, _, err := buildSimpleExecutionRemark(execRemark)
	if err != nil {
		return nil, err
	}
	policy, err := s.getPolicyService().UpdatePolicy(&n5model.TrafficPolicy{
		Id:                ctx.Policy.Id,
		Name:              simpleExecutionPolicyName(inbound),
		Remark:            remarkValue,
		Enabled:           ctx.Policy.Enabled,
		DefaultTargetType: ctx.Policy.DefaultTargetType,
		DefaultTargetId:   ctx.Policy.DefaultTargetId,
	})
	if err != nil {
		return nil, err
	}
	ctx.Policy = policy
	ctx.ExecRemark = execRemark
	ctx.LegacyMeta = nil
	return ctx, nil
}

func (s *RuleService) buildExecutionRemarkFromLegacy(meta *simpleRuleRemark, rules []*n5model.TrafficPolicyRule) (*simpleExecutionRemark, error) {
	remark := &simpleExecutionRemark{
		Version: simpleExecutionRemarkVersion,
		Items:   []*simpleExecutionItem{},
	}
	if meta == nil || meta.TrafficType == simpleTrafficAll {
		return remark, nil
	}

	item := &simpleExecutionItem{
		TrafficType:  meta.TrafficType,
		GroupId:      meta.GroupId,
		GroupName:    meta.GroupName,
		GroupType:    meta.GroupType,
		CustomDomain: meta.CustomDomain,
		RuleIDs:      collectRuleIDs(rules),
	}
	switch meta.TrafficType {
	case simpleTrafficAI, simpleTrafficGame, simpleTrafficStreaming:
		item.TrafficType = simpleTrafficGroup
		item.GroupType = meta.TrafficType
		if item.GroupName == "" {
			item.GroupName = simpleTrafficLabel(meta.TrafficType)
		}
	}
	remark.Items = append(remark.Items, item)
	return remark, nil
}

func (s *RuleService) saveExecutionRemark(policy *n5model.TrafficPolicy, remark *simpleExecutionRemark) error {
	if policy == nil || policy.Id <= 0 {
		return common.NewError("invalid policy")
	}
	if remark == nil {
		remark = &simpleExecutionRemark{Version: simpleExecutionRemarkVersion}
	}
	remarkValue, _, err := buildSimpleExecutionRemark(remark)
	if err != nil {
		return err
	}
	updated, err := s.getPolicyService().UpdatePolicy(&n5model.TrafficPolicy{
		Id:                policy.Id,
		Name:              policy.Name,
		Remark:            remarkValue,
		Enabled:           policy.Enabled,
		DefaultTargetType: policy.DefaultTargetType,
		DefaultTargetId:   policy.DefaultTargetId,
	})
	if err != nil {
		return err
	}
	policy.Remark = updated.Remark
	policy.UpdatedAt = updated.UpdatedAt
	return nil
}

func (s *RuleService) loadSimplePolicyContextByInbound(inboundId int) (*simplePolicyContext, error) {
	if inboundId <= 0 {
		return nil, common.NewError("invalid inbound id")
	}
	bindings, err := s.getPolicyService().ListBindings()
	if err != nil {
		return nil, err
	}
	for _, binding := range bindings {
		if binding.InboundId != inboundId {
			continue
		}
		return s.loadSimplePolicyContextByPolicyID(binding.PolicyId)
	}
	return nil, nil
}

func (s *RuleService) loadSimplePolicyContextByPolicyID(policyId int) (*simplePolicyContext, error) {
	if policyId <= 0 {
		return nil, common.NewError("invalid policy id")
	}
	policy, err := s.getPolicyService().GetPolicy(policyId)
	if err != nil {
		return nil, err
	}
	binding, err := s.findBindingByPolicyID(policyId)
	if err != nil {
		return nil, err
	}
	inbound, err := s.getInboundService().GetInbound(binding.InboundId)
	if err != nil {
		return nil, err
	}
	rules, err := s.getPolicyService().ListRules(policy.Id)
	if err != nil {
		return nil, err
	}
	ctx := &simplePolicyContext{
		Policy:  policy,
		Binding: binding,
		Inbound: inbound,
		Rules:   rules,
		RuleMap: buildRuleMap(rules),
	}
	if isNewSimpleExecutionRemark(policy.Remark) {
		execRemark, err := decodeSimpleExecutionRemark(policy.Remark)
		if err != nil {
			logger.Warningf("simple execution metadata decode failed, policy=%d err=%v", policy.Id, err)
			return nil, common.NewError("simple execution metadata is corrupted")
		}
		ctx.ExecRemark = execRemark
		return ctx, nil
	}
	if isLegacySimpleRemark(policy.Remark) {
		legacyMeta, ok := parseSimpleRuleRemark(policy.Remark)
		if !ok {
			logger.Warningf("legacy simple metadata decode failed, policy=%d", policy.Id)
			return nil, common.NewError("simple execution metadata is corrupted")
		}
		ctx.LegacyMeta = legacyMeta
		return ctx, nil
	}
	return nil, common.NewError("inbound already bound to advanced traffic policy")
}

func (s *RuleService) loadSimpleRuleSelection(ruleID string) (*simplePolicyContext, *simpleExecutionItem, bool, error) {
	policyId, key, legacy, ok := parseSimpleRuleID(ruleID)
	if !ok {
		if numericID, err := strconv.Atoi(strings.TrimSpace(ruleID)); err == nil && numericID > 0 {
			policyId = numericID
			legacy = true
		} else {
			return nil, nil, false, common.NewError("invalid simple rule id")
		}
	}
	ctx, err := s.loadSimplePolicyContextByPolicyID(policyId)
	if err != nil {
		return nil, nil, false, err
	}
	if legacy {
		if ctx.LegacyMeta != nil {
			return ctx, buildLegacySelectionItem(ctx.LegacyMeta, ctx.Rules), true, nil
		}
		if ctx.ExecRemark != nil && key == simpleTrafficAll && strings.EqualFold(ctx.Policy.DefaultTargetType, "egress") && ctx.Policy.DefaultTargetId > 0 {
			return ctx, &simpleExecutionItem{TrafficType: simpleTrafficAll}, false, nil
		}
		return nil, nil, false, common.NewError("invalid simple rule id")
	}
	if key == simpleTrafficAll {
		if strings.EqualFold(ctx.Policy.DefaultTargetType, "egress") && ctx.Policy.DefaultTargetId > 0 {
			return ctx, &simpleExecutionItem{TrafficType: simpleTrafficAll}, false, nil
		}
		return nil, nil, false, common.NewError("simple rule not found")
	}
	item := findExecutionItemByKey(ctx.ExecRemark, key)
	if item == nil {
		return nil, nil, false, common.NewError("simple rule not found")
	}
	return ctx, item, false, nil
}

func (s *RuleService) updateLegacySimpleRule(ctx *simplePolicyContext, req *CreateSimpleRuleRequest, egress *n5model.Egress) (*SimpleRule, error) {
	if ctx == nil || ctx.Policy == nil || ctx.LegacyMeta == nil {
		return nil, common.NewError("simple rule not found")
	}
	legacyItem := buildLegacySelectionItem(ctx.LegacyMeta, ctx.Rules)
	if legacyItem.TrafficType == simpleTrafficAll {
		policy, err := s.getPolicyService().UpdatePolicy(&n5model.TrafficPolicy{
			Id:                ctx.Policy.Id,
			Name:              ctx.Policy.Name,
			Remark:            ctx.Policy.Remark,
			Enabled:           ctx.Policy.Enabled,
			DefaultTargetType: "egress",
			DefaultTargetId:   egress.Id,
		})
		if err != nil {
			return nil, err
		}
		if err := s.runSimpleMutationHook(simpleMutationHookAfterPolicyUpdate); err != nil {
			return nil, err
		}
		return buildDefaultSimpleRule(policy, ctx.Binding, ctx.Inbound, policy.Enabled && ctx.Binding.Enabled, egress), nil
	}

	for _, rule := range ctx.Rules {
		if _, err := s.getPolicyService().UpdateRule(&n5model.TrafficPolicyRule{
			Id:         rule.Id,
			RuleType:   rule.RuleType,
			MatchMode:  rule.MatchMode,
			MatchValue: rule.MatchValue,
			TargetType: "egress",
			TargetId:   egress.Id,
			SortOrder:  rule.SortOrder,
			Enabled:    rule.Enabled,
		}); err != nil {
			return nil, err
		}
	}
	if err := s.runSimpleMutationHook(simpleMutationHookAfterRuleUpdate); err != nil {
		return nil, err
	}
	ctx.Rules, _ = s.getPolicyService().ListRules(ctx.Policy.Id)
	ctx.RuleMap = buildRuleMap(ctx.Rules)
	row, _ := buildLegacySimpleRule(ctx.Policy, ctx.Binding, ctx.Inbound, ctx.Policy.Enabled && ctx.Binding.Enabled, ctx.LegacyMeta, ctx.Rules, ctx.RuleMap, map[int]*n5model.Egress{
		egress.Id: egress,
	}, map[int]*TrafficRuleGroupOption{}, map[string]*TrafficRuleGroupOption{})
	return row, nil
}

func (s *RuleService) cleanupEmptyExecutionPolicy(policyId int) error {
	ctx, err := s.loadSimplePolicyContextByPolicyID(policyId)
	if err != nil {
		return err
	}
	if ctx.ExecRemark == nil {
		return nil
	}
	if len(ctx.ExecRemark.Items) > 0 {
		return nil
	}
	if strings.EqualFold(ctx.Policy.DefaultTargetType, "egress") && ctx.Policy.DefaultTargetId > 0 {
		return nil
	}
	if len(ctx.Rules) > 0 {
		return nil
	}
	return s.getPolicyService().DeletePolicy(policyId)
}

func (s *RuleService) findBindingByPolicyID(policyId int) (*n5model.TrafficPolicyBinding, error) {
	bindings, err := s.getPolicyService().ListBindings()
	if err != nil {
		return nil, err
	}
	for _, binding := range bindings {
		if binding.PolicyId == policyId {
			return binding, nil
		}
	}
	return nil, common.NewError("traffic policy binding not found")
}

func (s *RuleService) buildTrafficTypes() []*SimpleTrafficOption {
	items := []*SimpleTrafficOption{
		{
			Value:       simpleTrafficAll,
			Label:       simpleTrafficLabel(simpleTrafficAll),
			Description: "该入口的全部流量走指定出口。",
		},
		{
			Value:       simpleTrafficAI,
			Label:       simpleTrafficLabel(simpleTrafficAI),
			Description: "使用内置 AI 分流规则生成执行策略。",
		},
		{
			Value:       simpleTrafficGame,
			Label:       simpleTrafficLabel(simpleTrafficGame),
			Description: "使用内置游戏分流规则生成执行策略。",
		},
		{
			Value:       simpleTrafficStreaming,
			Label:       simpleTrafficLabel(simpleTrafficStreaming),
			Description: "使用内置流媒体分流规则生成执行策略。",
		},
		{
			Value:       simpleTrafficGroup,
			Label:       simpleTrafficLabel(simpleTrafficGroup),
			Description: "从自定义规则组复制规则并生成执行策略。",
		},
		{
			Value:       simpleTrafficCustomDomain,
			Label:       simpleTrafficLabel(simpleTrafficCustomDomain),
			Description: "为单个域名生成简单分流规则。",
		},
	}
	sort.Slice(items, func(i, j int) bool {
		order := map[string]int{
			simpleTrafficAll:          1,
			simpleTrafficAI:           2,
			simpleTrafficGame:         3,
			simpleTrafficStreaming:    4,
			simpleTrafficCustomDomain: 5,
			simpleTrafficGroup:        6,
		}
		return order[items[i].Value] < order[items[j].Value]
	})
	return items
}

func buildSimpleRuleFromExecutionItem(policy *n5model.TrafficPolicy, binding *n5model.TrafficPolicyBinding, inbound *model.Inbound, enabled bool, item *simpleExecutionItem, ruleMap map[int]*n5model.TrafficPolicyRule, egressByID map[int]*n5model.Egress) (*SimpleRule, bool) {
	if policy == nil || binding == nil || inbound == nil || item == nil {
		return nil, false
	}
	egressID, ok := executionItemEgressID(item, ruleMap)
	if !ok {
		return nil, false
	}
	egress := egressByID[egressID]
	if egress == nil {
		return nil, false
	}
	status := "disabled"
	if enabled {
		status = "enabled"
	}
	trafficLabel := simpleTrafficLabel(item.TrafficType)
	groupName := ""
	groupType := normalizeSimpleGroupType(item.GroupType)
	groupId := item.GroupId
	customDomain := item.CustomDomain
	if item.TrafficType == simpleTrafficGroup {
		groupName = item.GroupName
		if groupName == "" {
			groupName = defaultSimpleGroupName(groupType)
		}
		trafficLabel = groupName
	}
	return &SimpleRule{
		Id:              policy.Id,
		RuleId:          buildSimpleRuleID(policy.Id, simpleExecutionItemKey(item)),
		PolicyId:        policy.Id,
		InboundId:       inbound.Id,
		InboundName:     simpleInboundName(inbound),
		InboundTag:      inbound.Tag,
		TrafficType:     item.TrafficType,
		TrafficLabel:    trafficLabel,
		GroupId:         groupId,
		GroupName:       groupName,
		GroupType:       groupType,
		CustomDomain:    customDomain,
		EgressId:        egress.Id,
		EgressName:      egress.Name,
		EgressTag:       egress.Tag,
		Status:          status,
		Enabled:         enabled,
		RuleCount:       len(item.RuleIDs),
		CreatedAt:       policy.CreatedAt,
		UpdatedAt:       policy.UpdatedAt,
		PolicyName:      policy.Name,
		PolicyRemark:    policy.Remark,
		DefaultTargetId: policy.DefaultTargetId,
	}, true
}

func buildDefaultSimpleRule(policy *n5model.TrafficPolicy, binding *n5model.TrafficPolicyBinding, inbound *model.Inbound, enabled bool, egress *n5model.Egress) *SimpleRule {
	status := "disabled"
	if enabled {
		status = "enabled"
	}
	return &SimpleRule{
		Id:              policy.Id,
		RuleId:          buildSimpleRuleID(policy.Id, simpleTrafficAll),
		PolicyId:        policy.Id,
		InboundId:       inbound.Id,
		InboundName:     simpleInboundName(inbound),
		InboundTag:      inbound.Tag,
		TrafficType:     simpleTrafficAll,
		TrafficLabel:    simpleTrafficLabel(simpleTrafficAll),
		EgressId:        egress.Id,
		EgressName:      egress.Name,
		EgressTag:       egress.Tag,
		Status:          status,
		Enabled:         enabled,
		RuleCount:       0,
		CreatedAt:       policy.CreatedAt,
		UpdatedAt:       policy.UpdatedAt,
		PolicyName:      policy.Name,
		PolicyRemark:    policy.Remark,
		DefaultTargetId: policy.DefaultTargetId,
	}
}

func buildLegacySimpleRule(policy *n5model.TrafficPolicy, binding *n5model.TrafficPolicyBinding, inbound *model.Inbound, enabled bool, meta *simpleRuleRemark, rules []*n5model.TrafficPolicyRule, ruleMap map[int]*n5model.TrafficPolicyRule, egressByID map[int]*n5model.Egress, groupByID map[int]*TrafficRuleGroupOption, groupByType map[string]*TrafficRuleGroupOption) (*SimpleRule, bool) {
	if policy == nil || binding == nil || inbound == nil || meta == nil {
		return nil, false
	}
	status := "disabled"
	if enabled {
		status = "enabled"
	}
	trafficType := meta.TrafficType
	trafficLabel := simpleTrafficLabel(trafficType)
	groupID := meta.GroupId
	groupName := meta.GroupName
	groupType := meta.GroupType
	customDomain := meta.CustomDomain
	egressID := 0
	if trafficType == simpleTrafficAll {
		egressID = policy.DefaultTargetId
	} else {
		egressID, _ = executionItemEgressID(buildLegacySelectionItem(meta, rules), ruleMap)
	}
	if trafficType == simpleTrafficGroup {
		if group := groupByID[groupID]; group != nil {
			groupName = group.Name
			groupType = group.GroupType
		}
		if groupName == "" && groupType != "" {
			if group := groupByType[groupType]; group != nil {
				groupName = group.Name
				if groupID <= 0 {
					groupID = group.Id
				}
			}
		}
		if groupName != "" {
			trafficLabel = groupName
		}
	}
	if groupType == "" && (trafficType == simpleTrafficAI || trafficType == simpleTrafficGame || trafficType == simpleTrafficStreaming) {
		groupType = trafficType
		if group := groupByType[groupType]; group != nil {
			groupID = group.Id
			groupName = group.Name
			trafficType = simpleTrafficGroup
			trafficLabel = group.Name
		}
	}
	egress := egressByID[egressID]
	if egress == nil {
		return nil, false
	}
	return &SimpleRule{
		Id:              policy.Id,
		RuleId:          buildLegacySimpleRuleID(policy.Id),
		PolicyId:        policy.Id,
		InboundId:       inbound.Id,
		InboundName:     simpleInboundName(inbound),
		InboundTag:      inbound.Tag,
		TrafficType:     trafficType,
		TrafficLabel:    trafficLabel,
		GroupId:         groupID,
		GroupName:       groupName,
		GroupType:       groupType,
		CustomDomain:    customDomain,
		EgressId:        egress.Id,
		EgressName:      egress.Name,
		EgressTag:       egress.Tag,
		Status:          status,
		Enabled:         enabled,
		RuleCount:       len(rules),
		CreatedAt:       policy.CreatedAt,
		UpdatedAt:       policy.UpdatedAt,
		PolicyName:      policy.Name,
		PolicyRemark:    policy.Remark,
		DefaultTargetId: policy.DefaultTargetId,
	}, true
}

func buildLegacySelectionItem(meta *simpleRuleRemark, rules []*n5model.TrafficPolicyRule) *simpleExecutionItem {
	if meta == nil {
		return nil
	}
	return &simpleExecutionItem{
		TrafficType:  meta.TrafficType,
		GroupId:      meta.GroupId,
		GroupName:    meta.GroupName,
		GroupType:    meta.GroupType,
		CustomDomain: meta.CustomDomain,
		RuleIDs:      collectRuleIDs(rules),
	}
}

func buildRuleMap(rules []*n5model.TrafficPolicyRule) map[int]*n5model.TrafficPolicyRule {
	ruleMap := make(map[int]*n5model.TrafficPolicyRule, len(rules))
	for _, rule := range rules {
		ruleMap[rule.Id] = rule
	}
	return ruleMap
}

func collectRuleIDs(rules []*n5model.TrafficPolicyRule) []int {
	items := make([]int, 0, len(rules))
	for _, rule := range rules {
		if rule != nil {
			items = append(items, rule.Id)
		}
	}
	return items
}

func executionItemEgressID(item *simpleExecutionItem, ruleMap map[int]*n5model.TrafficPolicyRule) (int, bool) {
	rules, err := getExecutionItemRules(item, ruleMap)
	if err != nil || len(rules) == 0 {
		return 0, false
	}
	targetID := rules[0].TargetId
	for _, rule := range rules[1:] {
		if rule.TargetId != targetID || !strings.EqualFold(rule.TargetType, rules[0].TargetType) {
			return 0, false
		}
	}
	return targetID, true
}

func simpleExecutionItemKey(item *simpleExecutionItem) string {
	if item == nil {
		return ""
	}
	trafficType := normalizeSimpleTrafficType(item.TrafficType)
	switch trafficType {
	case simpleTrafficAll:
		return simpleTrafficAll
	case simpleTrafficGroup:
		groupType := normalizeSimpleGroupType(item.GroupType)
		if isBuiltinSimpleGroupType(groupType) {
			return "builtin:" + groupType
		}
		if item.GroupId > 0 {
			return "group:" + strconv.Itoa(item.GroupId)
		}
		return "group-name:" + strings.TrimSpace(strings.ToLower(item.GroupName))
	case simpleTrafficAI, simpleTrafficGame, simpleTrafficStreaming:
		return "builtin:" + trafficType
	case simpleTrafficCustomDomain:
		return "custom:" + canonicalCustomDomainKey(item.CustomDomain)
	default:
		return trafficType
	}
}

func findExecutionItemByKey(remark *simpleExecutionRemark, key string) *simpleExecutionItem {
	if remark == nil {
		return nil
	}
	for _, item := range remark.Items {
		if simpleExecutionItemKey(item) == key {
			return item
		}
	}
	return nil
}

func removeExecutionItem(remark *simpleExecutionRemark, key string) {
	if remark == nil {
		return
	}
	items := make([]*simpleExecutionItem, 0, len(remark.Items))
	for _, item := range remark.Items {
		if simpleExecutionItemKey(item) == key {
			continue
		}
		items = append(items, item)
	}
	remark.Items = items
}

func getExecutionItemRules(item *simpleExecutionItem, ruleMap map[int]*n5model.TrafficPolicyRule) ([]*n5model.TrafficPolicyRule, error) {
	if item == nil {
		return nil, common.NewError("simple execution metadata is corrupted")
	}
	if len(item.RuleIDs) == 0 {
		return nil, common.NewError("simple execution metadata is corrupted")
	}
	items := make([]*n5model.TrafficPolicyRule, 0, len(item.RuleIDs))
	for _, ruleID := range item.RuleIDs {
		rule := ruleMap[ruleID]
		if rule == nil {
			return nil, common.NewError("simple execution metadata is corrupted")
		}
		items = append(items, rule)
	}
	return items, nil
}

func canonicalCustomDomainKey(value string) string {
	_, _, displayValue, err := parseCustomDomainRule(value)
	if err != nil {
		return strings.TrimSpace(strings.ToLower(value))
	}
	return strings.TrimSpace(strings.ToLower(displayValue))
}

type simpleRuleConflictCandidate struct {
	ItemKey    string
	ItemLabel  string
	RuleType   string
	MatchMode  string
	MatchValue string
	Priority   int
}

func (s *RuleService) conflictCandidatesForRequest(req *CreateSimpleRuleRequest, trafficType string) ([]*simpleRuleConflictCandidate, error) {
	switch trafficType {
	case simpleTrafficCustomDomain:
		matchMode, matchValue, _, err := parseCustomDomainRule(req.CustomDomain)
		if err != nil {
			return nil, err
		}
		item := &simpleExecutionItem{TrafficType: simpleTrafficCustomDomain, CustomDomain: req.CustomDomain}
		return []*simpleRuleConflictCandidate{{
			ItemKey:    simpleExecutionItemKey(item),
			ItemLabel:  simpleTrafficLabel(simpleTrafficCustomDomain),
			RuleType:   "domain",
			MatchMode:  matchMode,
			MatchValue: matchValue,
			Priority:   simpleBusinessPriority(item, &n5model.TrafficPolicyRule{MatchMode: matchMode}),
		}}, nil
	case simpleTrafficGroup, simpleTrafficAI, simpleTrafficGame, simpleTrafficStreaming:
		var group *TrafficRuleGroup
		var err error
		if trafficType == simpleTrafficGroup {
			group, err = s.getGroupService().GetGroup(req.GroupId)
		} else {
			group, err = s.findBuiltinGroupByType(trafficType)
		}
		if err != nil {
			return nil, err
		}
		if group == nil || group.Id <= 0 {
			return nil, common.NewError("traffic rule group not found")
		}
		item := &simpleExecutionItem{
			TrafficType: simpleTrafficGroup,
			GroupId:     group.Id,
			GroupName:   group.Name,
			GroupType:   group.GroupType,
		}
		label := simpleConflictItemLabel(item)
		candidates := make([]*simpleRuleConflictCandidate, 0, len(group.Rules))
		for _, rule := range group.Rules {
			if rule == nil || !rule.Enabled || rule.RuleType != "domain" {
				continue
			}
			candidates = append(candidates, &simpleRuleConflictCandidate{
				ItemKey:    simpleExecutionItemKey(item),
				ItemLabel:  label,
				RuleType:   rule.RuleType,
				MatchMode:  rule.MatchMode,
				MatchValue: rule.MatchValue,
				Priority:   simpleBusinessPriority(item, &n5model.TrafficPolicyRule{MatchMode: rule.MatchMode}),
			})
		}
		return candidates, nil
	default:
		return []*simpleRuleConflictCandidate{}, nil
	}
}

func (s *RuleService) conflictCandidatesForContext(ctx *simplePolicyContext, excludeKey string) []*simpleRuleConflictCandidate {
	if ctx == nil || ctx.ExecRemark == nil {
		return []*simpleRuleConflictCandidate{}
	}
	candidates := make([]*simpleRuleConflictCandidate, 0)
	for _, item := range ctx.ExecRemark.Items {
		if item == nil {
			continue
		}
		itemKey := simpleExecutionItemKey(item)
		if excludeKey != "" && itemKey == excludeKey {
			continue
		}
		rules, err := getExecutionItemRules(item, ctx.RuleMap)
		if err != nil {
			continue
		}
		label := simpleConflictItemLabel(item)
		for _, rule := range rules {
			if rule == nil || !rule.Enabled || rule.RuleType != "domain" {
				continue
			}
			candidates = append(candidates, &simpleRuleConflictCandidate{
				ItemKey:    itemKey,
				ItemLabel:  label,
				RuleType:   rule.RuleType,
				MatchMode:  rule.MatchMode,
				MatchValue: rule.MatchValue,
				Priority:   simpleBusinessPriority(item, rule),
			})
		}
	}
	return candidates
}

func simpleBusinessPriority(item *simpleExecutionItem, rule *n5model.TrafficPolicyRule) int {
	if item == nil {
		return 100
	}
	trafficType := normalizeSimpleTrafficType(item.TrafficType)
	if trafficType == simpleTrafficCustomDomain {
		if rule == nil {
			return 49
		}
		switch strings.TrimSpace(strings.ToLower(rule.MatchMode)) {
		case "exact":
			return 10
		case "suffix":
			return 20
		case "keyword":
			return 30
		case "regexp":
			return 40
		default:
			return 49
		}
	}
	if trafficType == simpleTrafficGroup {
		switch normalizeSimpleGroupType(item.GroupType) {
		case simpleTrafficCustom, "":
			return 50
		case simpleTrafficAI:
			return 60
		case simpleTrafficGame:
			return 70
		case simpleTrafficStreaming:
			return 80
		default:
			return 50
		}
	}
	switch trafficType {
	case simpleTrafficAI:
		return 60
	case simpleTrafficGame:
		return 70
	case simpleTrafficStreaming:
		return 80
	default:
		return 100
	}
}

func simpleConflictItemLabel(item *simpleExecutionItem) string {
	if item == nil {
		return "已有规则"
	}
	if normalizeSimpleTrafficType(item.TrafficType) == simpleTrafficCustomDomain {
		return simpleTrafficLabel(simpleTrafficCustomDomain)
	}
	if strings.TrimSpace(item.GroupName) != "" {
		return strings.TrimSpace(item.GroupName)
	}
	if groupType := normalizeSimpleGroupType(item.GroupType); groupType != "" {
		return defaultSimpleGroupName(groupType)
	}
	return simpleTrafficLabel(item.TrafficType)
}

func simpleRuleOverlap(a, b *simpleRuleConflictCandidate) (bool, string) {
	if a == nil || b == nil || a.RuleType != "domain" || b.RuleType != "domain" {
		return false, ""
	}
	modeA := strings.TrimSpace(strings.ToLower(a.MatchMode))
	modeB := strings.TrimSpace(strings.ToLower(b.MatchMode))
	valueA := strings.TrimSpace(strings.ToLower(a.MatchValue))
	valueB := strings.TrimSpace(strings.ToLower(b.MatchValue))
	if valueA == "" || valueB == "" {
		return false, ""
	}
	if modeA == "regexp" || modeB == "regexp" {
		return true, "regexp"
	}
	if modeA == "keyword" || modeB == "keyword" {
		if modeA == "keyword" && modeB == "keyword" {
			return strings.Contains(valueA, valueB) || strings.Contains(valueB, valueA), "keyword"
		}
		keyword := valueA
		other := valueB
		if modeB == "keyword" {
			keyword = valueB
			other = valueA
		}
		if strings.Contains(other, keyword) || strings.Contains(keyword, other) {
			return true, "keyword"
		}
		return false, ""
	}
	if modeA == "exact" && modeB == "exact" {
		return valueA == valueB, "exact"
	}
	if modeA == "suffix" && modeB == "suffix" {
		return suffixDomainsOverlap(valueA, valueB), "suffix"
	}
	if modeA == "exact" && modeB == "suffix" {
		return exactMatchesSuffix(valueA, valueB), "exact-suffix"
	}
	if modeA == "suffix" && modeB == "exact" {
		return exactMatchesSuffix(valueB, valueA), "exact-suffix"
	}
	return false, ""
}

func suffixDomainsOverlap(a, b string) bool {
	return exactMatchesSuffix(a, b) || exactMatchesSuffix(b, a)
}

func exactMatchesSuffix(exact, suffix string) bool {
	return exact == suffix || strings.HasSuffix(exact, "."+suffix)
}

func buildSimpleRuleConflict(current, existing *simpleRuleConflictCandidate, relation string) *SimpleRuleConflict {
	conflict := &SimpleRuleConflict{
		CurrentLabel:       current.ItemLabel,
		CurrentMatchLabel:  simpleMatchModeLabel(current.MatchMode),
		CurrentValue:       current.MatchValue,
		ExistingLabel:      existing.ItemLabel,
		ExistingMatchLabel: simpleMatchModeLabel(existing.MatchMode),
		ExistingValue:      existing.MatchValue,
		Relation:           relation,
	}
	switch {
	case current.Priority < existing.Priority:
		conflict.PriorityDirection = "current"
		conflict.Message = "当前规则优先级更高，保存后重叠流量将优先走当前选择的出口。"
	case current.Priority > existing.Priority:
		conflict.PriorityDirection = "existing"
		conflict.Message = "已有规则优先级更高，保存后重叠流量会先命中已有规则。"
	default:
		conflict.PriorityDirection = "same"
		conflict.Message = "两条规则属于同一优先级，系统会按稳定规则顺序匹配。"
	}
	if relation == "keyword" {
		conflict.Message = "关键词规则可能与现有规则重叠。" + conflict.Message
	}
	if relation == "regexp" {
		conflict.Message = "正则表达式规则的覆盖范围无法完全静态判断，请确认优先级。" + conflict.Message
	}
	return conflict
}

func buildSimpleRuleConflictWarning(conflicts []*SimpleRuleConflict) string {
	if len(conflicts) == 0 || conflicts[0] == nil {
		return ""
	}
	first := conflicts[0]
	prefix := "发现规则重叠"
	if len(conflicts) > 1 {
		prefix = "发现 " + strconv.Itoa(len(conflicts)) + " 处规则重叠"
	}
	return prefix + "：" + first.CurrentValue + " 与「" + first.ExistingLabel + "」中的" +
		first.ExistingMatchLabel + " " + first.ExistingValue + " 重叠。" + first.Message
}

func simpleMatchModeLabel(mode string) string {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "exact":
		return "精确域名"
	case "suffix":
		return "域名及子域名"
	case "keyword":
		return "域名关键词"
	case "regexp":
		return "正则表达式"
	default:
		return "域名规则"
	}
}

func buildSimpleExecutionRemark(remark *simpleExecutionRemark) (string, simpleExecutionRemarkStats, error) {
	if remark == nil {
		remark = &simpleExecutionRemark{}
	}
	normalized := &simpleExecutionRemark{
		Version: simpleExecutionRemarkVersion,
		Items:   make([]*simpleExecutionItem, 0, len(remark.Items)),
	}
	seenItems := make(map[string]struct{}, len(remark.Items))
	for _, item := range remark.Items {
		if item == nil {
			continue
		}
		copyValue := *item
		copyValue.TrafficType = normalizeSimpleTrafficType(copyValue.TrafficType)
		copyValue.GroupType = normalizeSimpleGroupType(copyValue.GroupType)
		copyValue.CustomDomain = strings.TrimSpace(copyValue.CustomDomain)
		copyValue.GroupName = strings.TrimSpace(copyValue.GroupName)
		copyValue.RuleIDs = append([]int(nil), copyValue.RuleIDs...)
		if copyValue.TrafficType == simpleTrafficAll {
			return "", simpleExecutionRemarkStats{}, common.NewError("simple execution metadata item is invalid")
		}
		if copyValue.TrafficType == simpleTrafficGroup && copyValue.GroupId <= 0 && copyValue.GroupType == "" {
			return "", simpleExecutionRemarkStats{}, common.NewError("simple execution metadata item is invalid")
		}
		if copyValue.TrafficType == simpleTrafficCustomDomain && copyValue.CustomDomain == "" {
			return "", simpleExecutionRemarkStats{}, common.NewError("simple execution metadata item is invalid")
		}
		if len(copyValue.RuleIDs) == 0 {
			return "", simpleExecutionRemarkStats{}, common.NewError("simple execution metadata item is invalid")
		}
		if err := validatePositiveUniqueRuleIDs(copyValue.RuleIDs); err != nil {
			return "", simpleExecutionRemarkStats{}, err
		}
		key := simpleExecutionItemKey(&copyValue)
		if key == "" {
			return "", simpleExecutionRemarkStats{}, common.NewError("simple execution metadata item is invalid")
		}
		if _, ok := seenItems[key]; ok {
			return "", simpleExecutionRemarkStats{}, common.NewError("simple execution metadata contains duplicate item")
		}
		seenItems[key] = struct{}{}
		normalized.Items = append(normalized.Items, &copyValue)
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", simpleExecutionRemarkStats{}, err
	}
	encoded := base64.RawURLEncoding.EncodeToString(data)
	total := len(simpleExecutionRemarkPrefix) + len(encoded)
	if total > simpleExecutionRemarkMaxBytes {
		return "", simpleExecutionRemarkStats{
			RawJSONBytes:     len(data),
			Base64Bytes:      len(encoded),
			TotalRemarkBytes: total,
		}, common.NewError("Simple 执行元数据过大，请减少规则数量")
	}
	return simpleExecutionRemarkPrefix + encoded, simpleExecutionRemarkStats{
		RawJSONBytes:     len(data),
		Base64Bytes:      len(encoded),
		TotalRemarkBytes: total,
	}, nil
}

func decodeSimpleExecutionRemark(remark string) (*simpleExecutionRemark, error) {
	remark = strings.TrimSpace(remark)
	if !isNewSimpleExecutionRemark(remark) {
		return nil, common.NewError("invalid simple execution remark")
	}
	if len(remark) > simpleExecutionRemarkMaxBytes {
		return nil, common.NewError("simple execution metadata is too large")
	}
	raw := strings.TrimPrefix(remark, simpleExecutionRemarkPrefix)
	if raw == "" {
		return nil, common.NewError("simple execution metadata is empty")
	}
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, common.NewError("simple execution metadata decode failed")
	}
	parsed := &simpleExecutionRemark{}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(parsed); err != nil {
		return nil, common.NewError("simple execution metadata decode failed")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, common.NewError("simple execution metadata decode failed")
	}
	if parsed.Version != simpleExecutionRemarkVersion {
		return nil, common.NewError("unsupported simple execution metadata version")
	}
	if parsed.Items == nil {
		parsed.Items = []*simpleExecutionItem{}
	}
	seenItems := make(map[string]struct{}, len(parsed.Items))
	for _, item := range parsed.Items {
		if item == nil {
			return nil, common.NewError("simple execution metadata contains empty item")
		}
		item.TrafficType = normalizeSimpleTrafficType(item.TrafficType)
		item.GroupType = normalizeSimpleGroupType(item.GroupType)
		item.CustomDomain = strings.TrimSpace(item.CustomDomain)
		item.GroupName = strings.TrimSpace(item.GroupName)
		if err := validatePositiveUniqueRuleIDs(item.RuleIDs); err != nil {
			return nil, err
		}
		if item.TrafficType == simpleTrafficAll {
			return nil, common.NewError("simple execution metadata item is invalid")
		}
		if item.TrafficType == simpleTrafficGroup {
			if item.GroupId <= 0 && item.GroupType == "" {
				return nil, common.NewError("simple execution metadata item is invalid")
			}
		}
		if item.TrafficType == simpleTrafficCustomDomain && item.CustomDomain == "" {
			return nil, common.NewError("simple execution metadata item is invalid")
		}
		if len(item.RuleIDs) == 0 {
			return nil, common.NewError("simple execution metadata item is invalid")
		}
		key := simpleExecutionItemKey(item)
		if key == "" {
			return nil, common.NewError("simple execution metadata item is invalid")
		}
		if _, ok := seenItems[key]; ok {
			return nil, common.NewError("simple execution metadata contains duplicate item")
		}
		seenItems[key] = struct{}{}
	}
	return parsed, nil
}

func parseSimpleExecutionRemark(remark string) (*simpleExecutionRemark, bool) {
	parsed, err := decodeSimpleExecutionRemark(remark)
	return parsed, err == nil
}

func buildSimpleRuleRemark(trafficType string, customDomain string) string {
	values := []string{
		simpleRuleRemarkPrefix + "type=" + normalizeSimpleTrafficType(trafficType),
	}
	if strings.TrimSpace(customDomain) != "" {
		values = append(values, "value="+url.QueryEscape(strings.TrimSpace(customDomain)))
	}
	return strings.Join(values, "|")
}

func parseSimpleRuleRemark(remark string) (*simpleRuleRemark, bool) {
	remark = strings.TrimSpace(remark)
	if !isLegacySimpleRemark(remark) {
		return nil, false
	}
	meta := &simpleRuleRemark{}
	parts := strings.Split(remark, "|")
	for _, part := range parts {
		if strings.HasPrefix(part, simpleRuleRemarkPrefix) {
			part = strings.TrimPrefix(part, simpleRuleRemarkPrefix)
		}
		if strings.HasPrefix(part, "type=") {
			meta.TrafficType = normalizeSimpleTrafficType(strings.TrimPrefix(part, "type="))
		}
		if strings.HasPrefix(part, "groupId=") {
			groupID, err := strconv.Atoi(strings.TrimPrefix(part, "groupId="))
			if err == nil && groupID > 0 {
				meta.GroupId = groupID
			}
		}
		if strings.HasPrefix(part, "groupName=") {
			value, err := url.QueryUnescape(strings.TrimPrefix(part, "groupName="))
			if err == nil {
				meta.GroupName = value
			}
		}
		if strings.HasPrefix(part, "groupType=") {
			value, err := url.QueryUnescape(strings.TrimPrefix(part, "groupType="))
			if err == nil {
				meta.GroupType = normalizeSimpleGroupType(value)
			}
		}
		if strings.HasPrefix(part, "value=") {
			value, err := url.QueryUnescape(strings.TrimPrefix(part, "value="))
			if err == nil {
				meta.CustomDomain = value
			}
		}
	}
	if meta.TrafficType == simpleTrafficGroup {
		if meta.GroupId <= 0 && meta.GroupType == "" {
			return nil, false
		}
		if meta.GroupName == "" {
			meta.GroupName = "group-" + strconv.Itoa(meta.GroupId)
		}
		return meta, true
	}
	if !isSupportedSimpleTrafficType(meta.TrafficType) {
		return nil, false
	}
	return meta, true
}

func isNewSimpleExecutionRemark(remark string) bool {
	return strings.HasPrefix(strings.TrimSpace(remark), simpleExecutionRemarkPrefix)
}

func isLegacySimpleRemark(remark string) bool {
	return strings.HasPrefix(strings.TrimSpace(remark), simpleRuleRemarkPrefix)
}

func isOrdinaryPolicyRemark(remark string) bool {
	return !isNewSimpleExecutionRemark(remark) && !isLegacySimpleRemark(remark)
}

func validatePositiveUniqueRuleIDs(values []int) error {
	seen := make(map[int]struct{}, len(values))
	for _, value := range values {
		if value <= 0 {
			return common.NewError("simple execution metadata contains invalid rule id")
		}
		if _, ok := seen[value]; ok {
			return common.NewError("simple execution metadata contains duplicate rule id")
		}
		seen[value] = struct{}{}
	}
	return nil
}

func buildSimpleRuleID(policyId int, key string) string {
	return simpleRuleIDPrefix + strconv.Itoa(policyId) + ":" + base64.RawURLEncoding.EncodeToString([]byte(key))
}

func buildLegacySimpleRuleID(policyId int) string {
	return legacySimpleRuleIDPrefix + strconv.Itoa(policyId)
}

func parseSimpleRuleID(ruleID string) (int, string, bool, bool) {
	ruleID = strings.TrimSpace(ruleID)
	if strings.HasPrefix(ruleID, legacySimpleRuleIDPrefix) {
		policyID, err := strconv.Atoi(strings.TrimPrefix(ruleID, legacySimpleRuleIDPrefix))
		if err != nil || policyID <= 0 {
			return 0, "", false, false
		}
		return policyID, "", true, true
	}
	if !strings.HasPrefix(ruleID, simpleRuleIDPrefix) {
		return 0, "", false, false
	}
	parts := strings.SplitN(strings.TrimPrefix(ruleID, simpleRuleIDPrefix), ":", 2)
	if len(parts) != 2 {
		return 0, "", false, false
	}
	policyID, err := strconv.Atoi(parts[0])
	if err != nil || policyID <= 0 {
		return 0, "", false, false
	}
	keyData, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, "", false, false
	}
	return policyID, string(keyData), false, true
}

func simpleExecutionPolicyName(inbound *model.Inbound) string {
	name := simpleInboundName(inbound)
	if name == "" {
		name = "inbound-" + strconv.Itoa(inbound.Id)
	}
	return "Simple 执行策略 - " + name
}

func simplePolicyName(inbound *model.Inbound, trafficType string) string {
	name := simpleInboundName(inbound)
	if name == "" {
		name = "inbound-" + strconv.Itoa(inbound.Id)
	}
	return "Simple " + simpleTrafficLabel(trafficType) + " - " + name
}

func simpleGroupPolicyName(inbound *model.Inbound, groupName string) string {
	name := simpleInboundName(inbound)
	if name == "" {
		name = "inbound-" + strconv.Itoa(inbound.Id)
	}
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		groupName = "分流规则"
	}
	return "Simple " + groupName + " - " + name
}

func simpleInboundName(inbound *model.Inbound) string {
	if inbound == nil {
		return ""
	}
	if value := strings.TrimSpace(inbound.Remark); value != "" {
		return value
	}
	if value := strings.TrimSpace(inbound.Tag); value != "" {
		return value
	}
	return "inbound-" + strconv.Itoa(inbound.Id)
}

func simpleTrafficLabel(trafficType string) string {
	switch normalizeSimpleTrafficType(trafficType) {
	case simpleTrafficAll:
		return "全部流量"
	case simpleTrafficAI:
		return "AI分流"
	case simpleTrafficGame:
		return "游戏分流"
	case simpleTrafficStreaming:
		return "流媒体分流"
	case simpleTrafficGroup:
		return "自定义规则组"
	case simpleTrafficCustomDomain:
		return "自定义域名"
	default:
		return trafficType
	}
}

func disabledBuiltinGroupError(groupType string) error {
	return common.NewError(simpleTrafficLabel(normalizeSimpleGroupType(groupType)) + "已停用，请先在分流规则中启用后再使用")
}

func normalizeSimpleTrafficType(trafficType string) string {
	return strings.TrimSpace(strings.ToLower(trafficType))
}

func isSupportedSimpleTrafficType(trafficType string) bool {
	switch normalizeSimpleTrafficType(trafficType) {
	case simpleTrafficAll, simpleTrafficGroup, simpleTrafficAI, simpleTrafficGame, simpleTrafficStreaming, simpleTrafficCustomDomain:
		return true
	default:
		return false
	}
}

func parseCustomDomainRule(raw string) (string, string, string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", "", "", common.NewError("custom domain is required")
	}
	switch {
	case strings.HasPrefix(value, "full:"):
		match := strings.TrimSpace(strings.TrimPrefix(value, "full:"))
		if err := validateSimpleDomainInput(match); err != nil {
			return "", "", "", err
		}
		return "exact", match, "full:" + match, nil
	case strings.HasPrefix(value, "domain:"):
		match := strings.TrimSpace(strings.TrimPrefix(value, "domain:"))
		if err := validateSimpleDomainInput(match); err != nil {
			return "", "", "", err
		}
		return "suffix", match, "domain:" + match, nil
	case strings.HasPrefix(value, "keyword:"):
		match := strings.TrimSpace(strings.TrimPrefix(value, "keyword:"))
		if err := validateSimpleKeywordInput(match); err != nil {
			return "", "", "", err
		}
		return "keyword", match, "keyword:" + match, nil
	case strings.HasPrefix(value, "regexp:"):
		match := strings.TrimSpace(strings.TrimPrefix(value, "regexp:"))
		if match == "" {
			return "", "", "", common.NewError("custom domain is required")
		}
		if _, err := regexp.Compile(match); err != nil {
			return "", "", "", common.NewErrorf("invalid regexp: %v", err)
		}
		return "regexp", match, "regexp:" + match, nil
	default:
		if err := validateSimpleDomainInput(value); err != nil {
			return "", "", "", err
		}
		return "suffix", value, "domain:" + value, nil
	}
}

func validateSimpleDomainInput(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return common.NewError("custom domain is required")
	}
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.Contains(value, "/") {
		return common.NewError("请输入域名，不要包含 http://、https:// 或路径")
	}
	if strings.ContainsAny(value, " \t\r\n") {
		return common.NewError("域名不能包含空格")
	}
	if strings.Contains(value, ":") {
		return common.NewError("域名不能包含端口")
	}
	return nil
}

func validateSimpleKeywordInput(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return common.NewError("custom domain is required")
	}
	if strings.ContainsAny(value, " \t\r\n") {
		return common.NewError("域名关键词不能包含空格")
	}
	return nil
}
