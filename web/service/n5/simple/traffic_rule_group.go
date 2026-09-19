package simple

import (
	"github.com/torr9522/n6-ui/database"
	n5model "github.com/torr9522/n6-ui/database/model/n5"
	"github.com/torr9522/n6-ui/util/common"
	n5service "github.com/torr9522/n6-ui/web/service/n5"
	n5templates "github.com/torr9522/n6-ui/web/service/n5/templates"
	"net/url"
	"strconv"
	"strings"
)

const (
	simpleRuleGroupRemarkPrefix = "n5-simple-group|"
	simpleTrafficCustom         = "custom"
)

type TrafficRuleGroup struct {
	Id            int                     `json:"id"`
	Name          string                  `json:"name"`
	GroupType     string                  `json:"groupType"`
	GroupLabel    string                  `json:"groupLabel"`
	KindLabel     string                  `json:"kindLabel"`
	Builtin       bool                    `json:"builtin"`
	Status        string                  `json:"status"`
	Enabled       bool                    `json:"enabled"`
	RuleCount     int                     `json:"ruleCount"`
	SnapshotCount int                     `json:"snapshotCount"`
	DeleteHint    string                  `json:"deleteHint"`
	Rules         []*TrafficRuleGroupRule `json:"rules,omitempty"`
	CreatedAt     int64                   `json:"createdAt"`
	UpdatedAt     int64                   `json:"updatedAt"`
}

type TrafficRuleGroupRule struct {
	Id           int    `json:"id"`
	RuleType     string `json:"ruleType"`
	MatchMode    string `json:"matchMode"`
	MatchValue   string `json:"matchValue"`
	DisplayValue string `json:"displayValue"`
	Enabled      bool   `json:"enabled"`
	SortOrder    int    `json:"sortOrder"`
}

type TrafficRuleGroupOption struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	GroupType string `json:"groupType"`
	RuleCount int    `json:"ruleCount"`
	Enabled   bool   `json:"enabled"`
}

type CreateTrafficRuleGroupRequest struct {
	Name      string `json:"name" form:"name"`
	GroupType string `json:"groupType" form:"groupType"`
}

type UpdateTrafficRuleGroupRequest struct {
	Name string `json:"name" form:"name"`
}

type AddTrafficRuleDomainRequest struct {
	GroupId int    `json:"groupId" form:"groupId"`
	Domain  string `json:"domain" form:"domain"`
}

type UpdateTrafficRuleDomainRequest struct {
	GroupId int    `json:"groupId" form:"groupId"`
	Domain  string `json:"domain" form:"domain"`
}

type TrafficRuleGroupService struct {
	policyService trafficPolicyManager
}

var builtinSimpleGroupTypes = []string{
	simpleTrafficAI,
	simpleTrafficGame,
	simpleTrafficStreaming,
}

func NewTrafficRuleGroupService() *TrafficRuleGroupService {
	return &TrafficRuleGroupService{
		policyService: &n5service.TrafficPolicyService{},
	}
}

func (s *TrafficRuleGroupService) getPolicyService() trafficPolicyManager {
	if s.policyService != nil {
		return s.policyService
	}
	return &n5service.TrafficPolicyService{}
}

func (s *TrafficRuleGroupService) ListGroups() ([]*TrafficRuleGroup, error) {
	if err := s.EnsureBuiltinGroups(); err != nil {
		return nil, err
	}
	policies, err := s.getPolicyService().List()
	if err != nil {
		return nil, err
	}
	items := make([]*TrafficRuleGroup, 0)
	for _, policy := range policies {
		meta, ok := parseSimpleRuleGroupRemark(policy.Remark)
		if !ok {
			continue
		}
		rules, err := s.getPolicyService().ListRules(policy.Id)
		if err != nil {
			return nil, err
		}
		items = append(items, buildTrafficRuleGroup(policy, rules, meta, false))
	}
	return items, nil
}

func (s *TrafficRuleGroupService) ListGroupOptions() ([]*TrafficRuleGroupOption, error) {
	if err := s.EnsureBuiltinGroups(); err != nil {
		return nil, err
	}
	groups, err := s.ListGroups()
	if err != nil {
		return nil, err
	}
	items := make([]*TrafficRuleGroupOption, 0, len(groups))
	for _, group := range groups {
		if !group.Enabled {
			continue
		}
		items = append(items, &TrafficRuleGroupOption{
			Id:        group.Id,
			Name:      group.Name,
			GroupType: group.GroupType,
			RuleCount: group.RuleCount,
			Enabled:   group.Enabled,
		})
	}
	return items, nil
}

func (s *TrafficRuleGroupService) GetGroup(id int) (*TrafficRuleGroup, error) {
	policy, meta, err := s.getGroupPolicy(id)
	if err != nil {
		return nil, err
	}
	rules, err := s.getPolicyService().ListRules(policy.Id)
	if err != nil {
		return nil, err
	}
	return buildTrafficRuleGroup(policy, rules, meta, true), nil
}

func (s *TrafficRuleGroupService) CreateGroup(req *CreateTrafficRuleGroupRequest) (*TrafficRuleGroup, error) {
	if req == nil {
		return nil, common.NewError("traffic rule group request is nil")
	}
	groupType := normalizeSimpleGroupType(req.GroupType)
	if !isSupportedSimpleGroupType(groupType) {
		return nil, common.NewError("unsupported traffic rule group type")
	}
	if err := s.ensureBuiltinGroupAvailable(groupType); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = defaultSimpleGroupName(groupType)
	}
	if name == "" {
		return nil, common.NewError("group name is required")
	}

	policy, err := s.getPolicyService().Create(&n5model.TrafficPolicy{
		Name:    name,
		Remark:  buildSimpleRuleGroupRemark(groupType),
		Enabled: true,
	})
	if err != nil {
		return nil, err
	}

	if err := s.seedGroupRules(policy.Id, groupType); err != nil {
		_ = s.getPolicyService().DeletePolicy(policy.Id)
		return nil, err
	}
	return s.GetGroup(policy.Id)
}

func (s *TrafficRuleGroupService) UpdateGroup(id int, req *UpdateTrafficRuleGroupRequest) (*TrafficRuleGroup, error) {
	if req == nil {
		return nil, common.NewError("traffic rule group request is nil")
	}
	policy, meta, err := s.getGroupPolicy(id)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, common.NewError("group name is required")
	}
	_, err = s.getPolicyService().UpdatePolicy(&n5model.TrafficPolicy{
		Id:                policy.Id,
		Name:              name,
		Remark:            policy.Remark,
		Enabled:           policy.Enabled,
		DefaultTargetType: "",
		DefaultTargetId:   0,
	})
	if err != nil {
		return nil, err
	}
	return s.GetGroupWithMeta(policy.Id, meta)
}

func (s *TrafficRuleGroupService) DeleteGroup(id int) error {
	_, meta, err := s.getGroupPolicy(id)
	if err != nil {
		return err
	}
	if isBuiltinSimpleGroupType(meta.GroupType) {
		return common.NewError("内置规则组不可删除")
	}
	return s.getPolicyService().DeletePolicy(id)
}

func (s *TrafficRuleGroupService) EnsureBuiltinGroups() error {
	for _, groupType := range builtinSimpleGroupTypes {
		exists, err := s.builtinGroupExists(groupType)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		policy, err := s.getPolicyService().Create(&n5model.TrafficPolicy{
			Name:    defaultSimpleGroupName(groupType),
			Remark:  buildSimpleRuleGroupRemark(groupType),
			Enabled: true,
		})
		if err != nil {
			return err
		}
		if err := s.seedGroupRules(policy.Id, groupType); err != nil {
			_ = s.getPolicyService().DeletePolicy(policy.Id)
			return err
		}
	}
	return nil
}

func (s *TrafficRuleGroupService) builtinGroupExists(groupType string) (bool, error) {
	policies, err := s.getPolicyService().List()
	if err != nil {
		return false, err
	}
	for _, policy := range policies {
		meta, ok := parseSimpleRuleGroupRemark(policy.Remark)
		if !ok {
			continue
		}
		if meta.GroupType == groupType {
			return true, nil
		}
	}
	return false, nil
}

func (s *TrafficRuleGroupService) AddDomainRule(req *AddTrafficRuleDomainRequest) (*TrafficRuleGroupRule, error) {
	if req == nil {
		return nil, common.NewError("traffic rule request is nil")
	}
	if _, _, err := s.getGroupPolicy(req.GroupId); err != nil {
		return nil, err
	}
	matchMode, matchValue, _, err := parseCustomDomainRule(req.Domain)
	if err != nil {
		return nil, err
	}

	record := &n5model.TrafficPolicyRule{
		PolicyId:   req.GroupId,
		RuleType:   "domain",
		MatchMode:  matchMode,
		MatchValue: matchValue,
		TargetType: "",
		TargetId:   0,
		SortOrder:  s.nextGroupRuleSortOrder(req.GroupId),
		Enabled:    true,
	}
	if err := database.GetDB().Create(record).Error; err != nil {
		return nil, err
	}
	return toTrafficRuleGroupRule(record), nil
}

func (s *TrafficRuleGroupService) DeleteDomainRule(groupId int, ruleId int) error {
	if ruleId <= 0 {
		return common.NewError("invalid traffic rule id")
	}
	if _, _, err := s.getGroupPolicy(groupId); err != nil {
		return err
	}
	record := &n5model.TrafficPolicyRule{}
	if err := database.GetDB().Where("id = ? and policy_id = ?", ruleId, groupId).First(record).Error; err != nil {
		return err
	}
	return database.GetDB().Delete(&n5model.TrafficPolicyRule{}, ruleId).Error
}

func (s *TrafficRuleGroupService) UpdateDomainRule(ruleId int, req *UpdateTrafficRuleDomainRequest) (*TrafficRuleGroupRule, error) {
	if req == nil {
		return nil, common.NewError("traffic rule request is nil")
	}
	if ruleId <= 0 || req.GroupId <= 0 {
		return nil, common.NewError("invalid traffic rule id")
	}
	if _, _, err := s.getGroupPolicy(req.GroupId); err != nil {
		return nil, err
	}
	record := &n5model.TrafficPolicyRule{}
	if err := database.GetDB().Where("id = ? and policy_id = ?", ruleId, req.GroupId).First(record).Error; err != nil {
		return nil, err
	}
	matchMode, matchValue, _, err := parseCustomDomainRule(req.Domain)
	if err != nil {
		return nil, err
	}
	record.RuleType = "domain"
	record.MatchMode = matchMode
	record.MatchValue = matchValue
	if err := database.GetDB().Save(record).Error; err != nil {
		return nil, err
	}
	return toTrafficRuleGroupRule(record), nil
}

func (s *TrafficRuleGroupService) EnableGroup(id int) (*TrafficRuleGroup, error) {
	if _, _, err := s.getGroupPolicy(id); err != nil {
		return nil, err
	}
	if _, err := s.getPolicyService().EnablePolicy(id); err != nil {
		return nil, err
	}
	return s.GetGroup(id)
}

func (s *TrafficRuleGroupService) DisableGroup(id int) (*TrafficRuleGroup, error) {
	if _, _, err := s.getGroupPolicy(id); err != nil {
		return nil, err
	}
	if _, err := s.getPolicyService().DisablePolicy(id); err != nil {
		return nil, err
	}
	return s.GetGroup(id)
}

func (s *TrafficRuleGroupService) GetGroupWithMeta(id int, meta *simpleRuleGroupRemark) (*TrafficRuleGroup, error) {
	policy, err := s.getPolicyService().GetPolicy(id)
	if err != nil {
		return nil, err
	}
	rules, err := s.getPolicyService().ListRules(id)
	if err != nil {
		return nil, err
	}
	return buildTrafficRuleGroup(policy, rules, meta, true), nil
}

func (s *TrafficRuleGroupService) getGroupPolicy(id int) (*n5model.TrafficPolicy, *simpleRuleGroupRemark, error) {
	if id <= 0 {
		return nil, nil, common.NewError("invalid traffic rule group id")
	}
	policy, err := s.getPolicyService().GetPolicy(id)
	if err != nil {
		return nil, nil, err
	}
	meta, ok := parseSimpleRuleGroupRemark(policy.Remark)
	if !ok {
		return nil, nil, common.NewError("traffic rule group not found")
	}
	return policy, meta, nil
}

func (s *TrafficRuleGroupService) ensureBuiltinGroupAvailable(groupType string) error {
	if groupType == simpleTrafficCustom {
		return nil
	}
	groups, err := s.ListGroups()
	if err != nil {
		return err
	}
	for _, group := range groups {
		if group.GroupType == groupType {
			return common.NewError("builtin traffic rule group already exists")
		}
	}
	return nil
}

func (s *TrafficRuleGroupService) seedGroupRules(policyId int, groupType string) error {
	if groupType == simpleTrafficCustom {
		return nil
	}
	definition := n5templates.Find(groupType)
	if definition == nil {
		return common.NewError("traffic template not found")
	}
	for index, item := range definition.Rules {
		record := &n5model.TrafficPolicyRule{
			PolicyId:   policyId,
			RuleType:   item.RuleType,
			MatchMode:  item.MatchMode,
			MatchValue: item.MatchValue,
			TargetType: "",
			TargetId:   0,
			SortOrder:  index + 1,
			Enabled:    true,
		}
		if err := database.GetDB().Create(record).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *TrafficRuleGroupService) nextGroupRuleSortOrder(policyId int) int {
	record := &n5model.TrafficPolicyRule{}
	if err := database.GetDB().
		Where("policy_id = ?", policyId).
		Order("sort_order desc, id desc").
		First(record).Error; err != nil {
		return 1
	}
	return record.SortOrder + 1
}

type simpleRuleGroupRemark struct {
	GroupType string
}

func buildSimpleRuleGroupRemark(groupType string) string {
	return simpleRuleGroupRemarkPrefix + "type=" + normalizeSimpleGroupType(groupType)
}

func parseSimpleRuleGroupRemark(remark string) (*simpleRuleGroupRemark, bool) {
	remark = strings.TrimSpace(remark)
	if !strings.HasPrefix(remark, simpleRuleGroupRemarkPrefix) {
		return nil, false
	}
	meta := &simpleRuleGroupRemark{}
	parts := strings.Split(remark, "|")
	for _, part := range parts {
		if strings.HasPrefix(part, simpleRuleGroupRemarkPrefix) {
			part = strings.TrimPrefix(part, simpleRuleGroupRemarkPrefix)
		}
		if strings.HasPrefix(part, "type=") {
			meta.GroupType = normalizeSimpleGroupType(strings.TrimPrefix(part, "type="))
		}
	}
	if !isSupportedSimpleGroupType(meta.GroupType) {
		return nil, false
	}
	return meta, true
}

func buildTrafficRuleGroup(policy *n5model.TrafficPolicy, rules []*n5model.TrafficPolicyRule, meta *simpleRuleGroupRemark, includeRules bool) *TrafficRuleGroup {
	snapshotCount := countSimpleRuleSnapshots(policy.Id)
	item := &TrafficRuleGroup{
		Id:            policy.Id,
		Name:          policy.Name,
		GroupType:     meta.GroupType,
		GroupLabel:    defaultSimpleGroupName(meta.GroupType),
		KindLabel:     simpleGroupKindLabel(meta.GroupType),
		Builtin:       isBuiltinSimpleGroupType(meta.GroupType),
		Enabled:       policy.Enabled,
		RuleCount:     len(rules),
		SnapshotCount: snapshotCount,
		CreatedAt:     policy.CreatedAt,
		UpdatedAt:     policy.UpdatedAt,
	}
	if policy.Enabled {
		item.Status = "enabled"
	} else {
		item.Status = "disabled"
	}
	if snapshotCount > 0 {
		item.DeleteHint = "该规则组已经生成过执行规则，删除不会影响已有运行规则"
	}
	if includeRules {
		item.Rules = make([]*TrafficRuleGroupRule, 0, len(rules))
		for _, rule := range rules {
			item.Rules = append(item.Rules, toTrafficRuleGroupRule(rule))
		}
	}
	return item
}

func toTrafficRuleGroupRule(rule *n5model.TrafficPolicyRule) *TrafficRuleGroupRule {
	if rule == nil {
		return nil
	}
	return &TrafficRuleGroupRule{
		Id:           rule.Id,
		RuleType:     rule.RuleType,
		MatchMode:    rule.MatchMode,
		MatchValue:   rule.MatchValue,
		DisplayValue: formatTrafficRuleDisplayValue(rule.MatchMode, rule.MatchValue),
		Enabled:      rule.Enabled,
		SortOrder:    rule.SortOrder,
	}
}

func formatTrafficRuleDisplayValue(matchMode string, matchValue string) string {
	matchMode = strings.TrimSpace(strings.ToLower(matchMode))
	matchValue = strings.TrimSpace(matchValue)
	switch matchMode {
	case "exact":
		return "full:" + matchValue
	case "suffix":
		return "domain:" + matchValue
	case "keyword":
		return "keyword:" + matchValue
	case "regexp":
		return "regexp:" + matchValue
	default:
		return matchValue
	}
}

func normalizeSimpleGroupType(groupType string) string {
	return strings.TrimSpace(strings.ToLower(groupType))
}

func isSupportedSimpleGroupType(groupType string) bool {
	switch normalizeSimpleGroupType(groupType) {
	case simpleTrafficAI, simpleTrafficGame, simpleTrafficStreaming, simpleTrafficCustom:
		return true
	default:
		return false
	}
}

func defaultSimpleGroupName(groupType string) string {
	switch normalizeSimpleGroupType(groupType) {
	case simpleTrafficAI:
		return "AI分流"
	case simpleTrafficGame:
		return "游戏分流"
	case simpleTrafficStreaming:
		return "流媒体分流"
	case simpleTrafficCustom:
		return "自定义分流"
	default:
		return ""
	}
}

func simpleGroupKindLabel(groupType string) string {
	if isBuiltinSimpleGroupType(groupType) {
		return "内置"
	}
	return "自定义"
}

func isBuiltinSimpleGroupType(groupType string) bool {
	switch normalizeSimpleGroupType(groupType) {
	case simpleTrafficAI, simpleTrafficGame, simpleTrafficStreaming:
		return true
	default:
		return false
	}
}

func countSimpleRuleSnapshots(groupId int) int {
	if groupId <= 0 {
		return 0
	}
	policies := make([]*n5model.TrafficPolicy, 0)
	if err := database.GetDB().Model(&n5model.TrafficPolicy{}).Order("id asc").Find(&policies).Error; err != nil {
		return 0
	}
	count := 0
	for _, policy := range policies {
		if execRemark, ok := parseSimpleExecutionRemark(policy.Remark); ok {
			for _, item := range execRemark.Items {
				if item == nil {
					continue
				}
				if item.TrafficType != simpleTrafficGroup {
					continue
				}
				if item.GroupId == groupId {
					count++
					break
				}
				if item.GroupId <= 0 && normalizeSimpleGroupType(item.GroupType) == builtinSimpleGroupTypeByID(groupId) {
					count++
					break
				}
			}
			continue
		}
		meta, ok := parseSimpleRuleRemark(policy.Remark)
		if !ok {
			continue
		}
		if meta.TrafficType == simpleTrafficGroup && meta.GroupId == groupId {
			count++
		}
	}
	return count
}

func builtinSimpleGroupTypeByID(groupId int) string {
	switch groupId {
	case 1:
		return simpleTrafficAI
	case 2:
		return simpleTrafficGame
	case 3:
		return simpleTrafficStreaming
	default:
		return ""
	}
}

func buildSimpleGroupSnapshotRemark(group *TrafficRuleGroup) string {
	values := []string{
		simpleRuleRemarkPrefix + "type=" + simpleTrafficGroup,
		"groupId=" + strconv.Itoa(group.Id),
	}
	if strings.TrimSpace(group.Name) != "" {
		values = append(values, "groupName="+url.QueryEscape(strings.TrimSpace(group.Name)))
	}
	if strings.TrimSpace(group.GroupType) != "" {
		values = append(values, "groupType="+url.QueryEscape(strings.TrimSpace(group.GroupType)))
	}
	return strings.Join(values, "|")
}
