package n5

import (
	"github.com/torr9522/n6-ui/database"
	n5model "github.com/torr9522/n6-ui/database/model/n5"
	"github.com/torr9522/n6-ui/util/common"
	"strings"

	"gorm.io/gorm"
)

type TrafficPolicyService struct {
	db *gorm.DB
}

const simpleManagedTrafficPolicyMutationMessage = "该策略由 n6-ui 简易出口规则管理，请在“出口规则”页面修改"

func (s *TrafficPolicyService) WithDB(db *gorm.DB) *TrafficPolicyService {
	if s == nil {
		return &TrafficPolicyService{db: db}
	}
	clone := *s
	clone.db = db
	return &clone
}

func (s *TrafficPolicyService) getDB() *gorm.DB {
	if s != nil && s.db != nil {
		return s.db
	}
	return database.GetDB()
}

func (s *TrafficPolicyService) Create(policy *n5model.TrafficPolicy) (*n5model.TrafficPolicy, error) {
	if policy == nil {
		return nil, common.NewError("traffic policy is nil")
	}

	record := &n5model.TrafficPolicy{
		Name:              normalizeName(policy.Name),
		Remark:            strings.TrimSpace(policy.Remark),
		Enabled:           policy.Enabled,
		DefaultTargetType: normalizeTargetType(policy.DefaultTargetType),
		DefaultTargetId:   policy.DefaultTargetId,
	}
	if !policy.Enabled {
		record.Enabled = false
	} else {
		record.Enabled = true
	}
	if record.Name == "" {
		return nil, common.NewError("policy name is required")
	}
	if record.DefaultTargetType != "" || record.DefaultTargetId > 0 {
		if err := validateTarget(record.DefaultTargetType, record.DefaultTargetId); err != nil {
			return nil, err
		}
	}

	if err := s.getDB().Create(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (s *TrafficPolicyService) CreatePolicyFromAdvanced(policy *n5model.TrafficPolicy) (*n5model.TrafficPolicy, error) {
	if policy != nil && isSimpleManagedTrafficPolicyRemark(policy.Remark) {
		return nil, simpleManagedTrafficPolicyMutationError()
	}
	return s.Create(policy)
}

func (s *TrafficPolicyService) Get(id int) (*n5model.TrafficPolicy, error) {
	return s.GetPolicy(id)
}

func (s *TrafficPolicyService) GetPolicy(id int) (*n5model.TrafficPolicy, error) {
	if id <= 0 {
		return nil, common.NewError("invalid policy id")
	}
	record := &n5model.TrafficPolicy{}
	if err := s.getDB().Model(&n5model.TrafficPolicy{}).Where("id = ?", id).First(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (s *TrafficPolicyService) List() ([]*n5model.TrafficPolicy, error) {
	records := make([]*n5model.TrafficPolicy, 0)
	err := s.getDB().Model(&n5model.TrafficPolicy{}).Order("id asc").Find(&records).Error
	return records, err
}

func (s *TrafficPolicyService) UpdatePolicy(policy *n5model.TrafficPolicy) (*n5model.TrafficPolicy, error) {
	if policy == nil || policy.Id <= 0 {
		return nil, common.NewError("invalid traffic policy")
	}

	record, err := s.GetPolicy(policy.Id)
	if err != nil {
		return nil, err
	}

	name := normalizeName(policy.Name)
	if name == "" {
		return nil, common.NewError("policy name is required")
	}
	targetType := normalizeTargetType(policy.DefaultTargetType)
	if targetType != "" || policy.DefaultTargetId > 0 {
		if err := validateTarget(targetType, policy.DefaultTargetId); err != nil {
			return nil, err
		}
	}

	record.Name = name
	nextRemark := strings.TrimSpace(policy.Remark)
	currentManaged := isSimpleManagedTrafficPolicyRemark(record.Remark)
	nextManaged := isSimpleManagedTrafficPolicyRemark(nextRemark)
	if currentManaged != nextManaged {
		return nil, common.NewError("该策略由 Simple 出口规则管理，请在出口规则页面修改")
	}
	record.Remark = nextRemark
	record.DefaultTargetType = targetType
	record.DefaultTargetId = policy.DefaultTargetId
	record.Enabled = policy.Enabled
	if err := s.getDB().Save(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (s *TrafficPolicyService) UpdatePolicyFromAdvanced(policy *n5model.TrafficPolicy) (*n5model.TrafficPolicy, error) {
	if policy == nil || policy.Id <= 0 {
		return nil, common.NewError("invalid traffic policy")
	}
	current, err := s.GetPolicy(policy.Id)
	if err != nil {
		return nil, err
	}
	if isSimpleManagedTrafficPolicyRemark(current.Remark) || isSimpleManagedTrafficPolicyRemark(policy.Remark) {
		return nil, simpleManagedTrafficPolicyMutationError()
	}
	return s.UpdatePolicy(policy)
}

func (s *TrafficPolicyService) DeletePolicy(id int) error {
	if id <= 0 {
		return common.NewError("invalid policy id")
	}

	return s.getDB().Transaction(func(tx *gorm.DB) error {
		record := &n5model.TrafficPolicy{}
		if err := tx.Model(&n5model.TrafficPolicy{}).Where("id = ?", id).First(record).Error; err != nil {
			return err
		}
		if err := tx.Where("policy_id = ?", id).Delete(&n5model.TrafficPolicyRule{}).Error; err != nil {
			return err
		}
		if err := tx.Where("policy_id = ?", id).Delete(&n5model.TrafficPolicyBinding{}).Error; err != nil {
			return err
		}
		return tx.Delete(&n5model.TrafficPolicy{}, id).Error
	})
}

func (s *TrafficPolicyService) DeletePolicyFromAdvanced(id int) error {
	if err := s.ensurePolicyOrdinary(id); err != nil {
		return err
	}
	return s.DeletePolicy(id)
}

func (s *TrafficPolicyService) EnablePolicy(id int) (*n5model.TrafficPolicy, error) {
	return s.updatePolicyEnabled(id, true)
}

func (s *TrafficPolicyService) DisablePolicy(id int) (*n5model.TrafficPolicy, error) {
	return s.updatePolicyEnabled(id, false)
}

func (s *TrafficPolicyService) EnablePolicyFromAdvanced(id int) (*n5model.TrafficPolicy, error) {
	if err := s.ensurePolicyOrdinary(id); err != nil {
		return nil, err
	}
	return s.EnablePolicy(id)
}

func (s *TrafficPolicyService) DisablePolicyFromAdvanced(id int) (*n5model.TrafficPolicy, error) {
	if err := s.ensurePolicyOrdinary(id); err != nil {
		return nil, err
	}
	return s.DisablePolicy(id)
}

func (s *TrafficPolicyService) updatePolicyEnabled(id int, enabled bool) (*n5model.TrafficPolicy, error) {
	record, err := s.GetPolicy(id)
	if err != nil {
		return nil, err
	}
	record.Enabled = enabled
	if err := s.getDB().Save(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (s *TrafficPolicyService) AddRule(rule *n5model.TrafficPolicyRule) (*n5model.TrafficPolicyRule, error) {
	if rule == nil || rule.PolicyId <= 0 {
		return nil, common.NewError("invalid traffic policy rule")
	}

	db := s.getDB()
	var policyCount int64
	if err := db.Model(&n5model.TrafficPolicy{}).Where("id = ?", rule.PolicyId).Count(&policyCount).Error; err != nil {
		return nil, err
	}
	if policyCount == 0 {
		return nil, common.NewError("traffic policy not found")
	}

	ruleType := normalizeRuleType(rule.RuleType)
	matchMode := normalizeMatchMode(rule.MatchMode)
	matchValue := strings.TrimSpace(rule.MatchValue)
	targetType := normalizeTargetType(rule.TargetType)
	if err := validateTarget(targetType, rule.TargetId); err != nil {
		return nil, err
	}

	switch ruleType {
	case ruleTypeDomain:
		if err := validateDomainRule(matchMode, matchValue); err != nil {
			return nil, err
		}
	case ruleTypeIP:
		if err := validateIPRule(matchMode, matchValue); err != nil {
			return nil, err
		}
	default:
		return nil, common.NewError("invalid rule type")
	}

	record := &n5model.TrafficPolicyRule{
		PolicyId:   rule.PolicyId,
		RuleType:   ruleType,
		MatchMode:  matchMode,
		MatchValue: matchValue,
		TargetType: targetType,
		TargetId:   rule.TargetId,
		SortOrder:  rule.SortOrder,
		Enabled:    rule.Enabled,
	}
	if !rule.Enabled {
		record.Enabled = false
	} else {
		record.Enabled = true
	}
	if record.SortOrder <= 0 {
		var maxSort int
		db.Model(&n5model.TrafficPolicyRule{}).Where("policy_id = ?", rule.PolicyId).Select("coalesce(max(sort_order), 0)").Scan(&maxSort)
		record.SortOrder = maxSort + 1
	}

	if err := db.Create(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (s *TrafficPolicyService) AddRuleFromAdvanced(rule *n5model.TrafficPolicyRule) (*n5model.TrafficPolicyRule, error) {
	if rule == nil || rule.PolicyId <= 0 {
		return nil, common.NewError("invalid traffic policy rule")
	}
	if err := s.ensurePolicyOrdinary(rule.PolicyId); err != nil {
		return nil, err
	}
	return s.AddRule(rule)
}

func (s *TrafficPolicyService) DeleteRule(ruleId int) error {
	if ruleId <= 0 {
		return common.NewError("invalid rule id")
	}
	return s.getDB().Delete(&n5model.TrafficPolicyRule{}, ruleId).Error
}

func (s *TrafficPolicyService) DeleteRuleFromAdvanced(ruleId int) error {
	if _, err := s.ensureRulePolicyOrdinary(ruleId, 0); err != nil {
		return err
	}
	return s.DeleteRule(ruleId)
}

func (s *TrafficPolicyService) ListRules(policyId int) ([]*n5model.TrafficPolicyRule, error) {
	if policyId <= 0 {
		return nil, common.NewError("invalid policy id")
	}
	records := make([]*n5model.TrafficPolicyRule, 0)
	err := s.getDB().Model(&n5model.TrafficPolicyRule{}).
		Where("policy_id = ?", policyId).
		Order("sort_order asc, id asc").
		Find(&records).Error
	return records, err
}

const (
	simpleManagedExecRemarkPrefix   = "n5-simple-exec|"
	simpleManagedLegacyRemarkPrefix = "n5-simple|"
)

func isSimpleExecutionTrafficPolicyRemark(remark string) bool {
	return strings.HasPrefix(strings.TrimSpace(remark), simpleManagedExecRemarkPrefix)
}

func isLegacySimpleTrafficPolicyRemark(remark string) bool {
	return strings.HasPrefix(strings.TrimSpace(remark), simpleManagedLegacyRemarkPrefix)
}

func isSimpleManagedTrafficPolicyRemark(remark string) bool {
	return isSimpleExecutionTrafficPolicyRemark(remark) || isLegacySimpleTrafficPolicyRemark(remark)
}

func isOrdinaryTrafficPolicyRemark(remark string) bool {
	return !isSimpleManagedTrafficPolicyRemark(remark)
}

func simpleManagedTrafficPolicyMutationError() error {
	return common.NewError(simpleManagedTrafficPolicyMutationMessage)
}

func (s *TrafficPolicyService) ensurePolicyOrdinary(policyId int) error {
	if policyId <= 0 {
		return common.NewError("invalid policy id")
	}
	policy, err := s.GetPolicy(policyId)
	if err != nil {
		return err
	}
	if isSimpleManagedTrafficPolicyRemark(policy.Remark) {
		return simpleManagedTrafficPolicyMutationError()
	}
	return nil
}

func (s *TrafficPolicyService) ensureRulePolicyOrdinary(ruleId int, expectedPolicyId int) (*n5model.TrafficPolicyRule, error) {
	if ruleId <= 0 {
		return nil, common.NewError("invalid rule id")
	}
	rule := &n5model.TrafficPolicyRule{}
	if err := s.getDB().Model(&n5model.TrafficPolicyRule{}).Where("id = ?", ruleId).First(rule).Error; err != nil {
		return nil, err
	}
	if expectedPolicyId > 0 && rule.PolicyId != expectedPolicyId {
		return nil, common.NewError("rule not found in policy")
	}
	if err := s.ensurePolicyOrdinary(rule.PolicyId); err != nil {
		return nil, err
	}
	return rule, nil
}

func (s *TrafficPolicyService) UpdateRule(rule *n5model.TrafficPolicyRule) (*n5model.TrafficPolicyRule, error) {
	if rule == nil || rule.Id <= 0 {
		return nil, common.NewError("invalid traffic policy rule")
	}

	record := &n5model.TrafficPolicyRule{}
	db := s.getDB()
	if err := db.Model(&n5model.TrafficPolicyRule{}).Where("id = ?", rule.Id).First(record).Error; err != nil {
		return nil, err
	}

	ruleType := normalizeRuleType(rule.RuleType)
	matchMode := normalizeMatchMode(rule.MatchMode)
	matchValue := strings.TrimSpace(rule.MatchValue)
	targetType := normalizeTargetType(rule.TargetType)
	if err := validateTarget(targetType, rule.TargetId); err != nil {
		return nil, err
	}
	switch ruleType {
	case ruleTypeDomain:
		if err := validateDomainRule(matchMode, matchValue); err != nil {
			return nil, err
		}
	case ruleTypeIP:
		if err := validateIPRule(matchMode, matchValue); err != nil {
			return nil, err
		}
	default:
		return nil, common.NewError("invalid rule type")
	}

	record.RuleType = ruleType
	record.MatchMode = matchMode
	record.MatchValue = matchValue
	record.TargetType = targetType
	record.TargetId = rule.TargetId
	if rule.SortOrder > 0 {
		record.SortOrder = rule.SortOrder
	}
	if err := db.Save(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (s *TrafficPolicyService) UpdateRuleFromAdvanced(rule *n5model.TrafficPolicyRule) (*n5model.TrafficPolicyRule, error) {
	if rule == nil || rule.Id <= 0 {
		return nil, common.NewError("invalid traffic policy rule")
	}
	if _, err := s.ensureRulePolicyOrdinary(rule.Id, rule.PolicyId); err != nil {
		return nil, err
	}
	return s.UpdateRule(rule)
}

func (s *TrafficPolicyService) EnableRule(id int) (*n5model.TrafficPolicyRule, error) {
	return s.updateRuleEnabled(id, true)
}

func (s *TrafficPolicyService) DisableRule(id int) (*n5model.TrafficPolicyRule, error) {
	return s.updateRuleEnabled(id, false)
}

func (s *TrafficPolicyService) EnableRuleFromAdvanced(id int) (*n5model.TrafficPolicyRule, error) {
	if _, err := s.ensureRulePolicyOrdinary(id, 0); err != nil {
		return nil, err
	}
	return s.EnableRule(id)
}

func (s *TrafficPolicyService) DisableRuleFromAdvanced(id int) (*n5model.TrafficPolicyRule, error) {
	if _, err := s.ensureRulePolicyOrdinary(id, 0); err != nil {
		return nil, err
	}
	return s.DisableRule(id)
}

func (s *TrafficPolicyService) updateRuleEnabled(id int, enabled bool) (*n5model.TrafficPolicyRule, error) {
	if id <= 0 {
		return nil, common.NewError("invalid rule id")
	}
	record := &n5model.TrafficPolicyRule{}
	db := s.getDB()
	if err := db.Model(&n5model.TrafficPolicyRule{}).Where("id = ?", id).First(record).Error; err != nil {
		return nil, err
	}
	record.Enabled = enabled
	if err := db.Save(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (s *TrafficPolicyService) ReorderRules(policyId int, ruleIds []int) error {
	if policyId <= 0 {
		return common.NewError("invalid policy id")
	}
	if len(ruleIds) == 0 {
		return common.NewError("rule ids are required")
	}

	return s.getDB().Transaction(func(tx *gorm.DB) error {
		records := make([]*n5model.TrafficPolicyRule, 0)
		if err := tx.Model(&n5model.TrafficPolicyRule{}).
			Where("policy_id = ?", policyId).
			Order("sort_order asc, id asc").
			Find(&records).Error; err != nil {
			return err
		}
		if len(records) != len(ruleIds) {
			return common.NewError("rule reorder count does not match")
		}

		recordMap := make(map[int]*n5model.TrafficPolicyRule, len(records))
		for _, record := range records {
			recordMap[record.Id] = record
		}
		for index, ruleId := range ruleIds {
			record, ok := recordMap[ruleId]
			if !ok {
				return common.NewError("rule not found in policy")
			}
			record.SortOrder = index + 1
			if err := tx.Save(record).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *TrafficPolicyService) ReorderRulesFromAdvanced(policyId int, ruleIds []int) error {
	if err := s.ensurePolicyOrdinary(policyId); err != nil {
		return err
	}
	return s.ReorderRules(policyId, ruleIds)
}

func (s *TrafficPolicyService) BindInboundPolicy(inboundId int, policyId int) (*n5model.TrafficPolicyBinding, error) {
	return s.RebindInboundPolicy(inboundId, policyId)
}

func (s *TrafficPolicyService) BindInboundPolicyFromAdvanced(inboundId int, policyId int) (*n5model.TrafficPolicyBinding, error) {
	return s.RebindInboundPolicyFromAdvanced(inboundId, policyId)
}

func (s *TrafficPolicyService) RebindInboundPolicy(inboundId int, policyId int) (*n5model.TrafficPolicyBinding, error) {
	if inboundId <= 0 || policyId <= 0 {
		return nil, common.NewError("invalid policy binding")
	}
	if _, err := getInboundByID(inboundId); err != nil {
		return nil, err
	}
	db := s.getDB()
	var policyCount int64
	if err := db.Model(&n5model.TrafficPolicy{}).Where("id = ?", policyId).Count(&policyCount).Error; err != nil {
		return nil, err
	}
	if policyCount == 0 {
		return nil, common.NewError("traffic policy not found")
	}

	record := &n5model.TrafficPolicyBinding{}
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("inbound_id = ?", inboundId).Delete(&n5model.TrafficPolicyBinding{}).Error; err != nil {
			return err
		}
		record.InboundId = inboundId
		record.PolicyId = policyId
		record.Enabled = true
		return tx.Create(record).Error
	})
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (s *TrafficPolicyService) RebindInboundPolicyFromAdvanced(inboundId int, policyId int) (*n5model.TrafficPolicyBinding, error) {
	if err := s.ensurePolicyOrdinary(policyId); err != nil {
		return nil, err
	}
	if err := s.ensureInboundBindingOrdinary(inboundId); err != nil {
		return nil, err
	}
	return s.RebindInboundPolicy(inboundId, policyId)
}

func (s *TrafficPolicyService) UnbindInboundPolicy(inboundId int) error {
	if inboundId <= 0 {
		return common.NewError("invalid inbound id")
	}
	if _, err := getInboundByID(inboundId); err != nil {
		return err
	}
	return s.getDB().Where("inbound_id = ?", inboundId).Delete(&n5model.TrafficPolicyBinding{}).Error
}

func (s *TrafficPolicyService) UnbindInboundPolicyFromAdvanced(inboundId int) error {
	if err := s.ensureInboundBindingOrdinary(inboundId); err != nil {
		return err
	}
	return s.UnbindInboundPolicy(inboundId)
}

func (s *TrafficPolicyService) ensureInboundBindingOrdinary(inboundId int) error {
	if inboundId <= 0 {
		return common.NewError("invalid inbound id")
	}
	binding := &n5model.TrafficPolicyBinding{}
	err := s.getDB().Model(&n5model.TrafficPolicyBinding{}).Where("inbound_id = ?", inboundId).First(binding).Error
	if database.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.ensurePolicyOrdinary(binding.PolicyId)
}

func (s *TrafficPolicyService) ListBindings() ([]*n5model.TrafficPolicyBinding, error) {
	records := make([]*n5model.TrafficPolicyBinding, 0)
	err := s.getDB().Model(&n5model.TrafficPolicyBinding{}).Order("inbound_id asc, id asc").Find(&records).Error
	return records, err
}

func (s *TrafficPolicyService) ListBindingsByPolicy(policyId int) ([]*n5model.TrafficPolicyBinding, error) {
	if policyId <= 0 {
		return nil, common.NewError("invalid policy id")
	}
	records := make([]*n5model.TrafficPolicyBinding, 0)
	err := s.getDB().
		Model(&n5model.TrafficPolicyBinding{}).
		Where("policy_id = ?", policyId).
		Order("inbound_id asc, id asc").
		Find(&records).Error
	return records, err
}
