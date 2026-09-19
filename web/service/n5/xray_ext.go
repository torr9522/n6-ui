package n5

import (
	"encoding/base64"
	"encoding/json"
	"github.com/torr9522/n6-ui/database"
	legacyModel "github.com/torr9522/n6-ui/database/model"
	n5model "github.com/torr9522/n6-ui/database/model/n5"
	"sort"
	"strings"
)

type XrayExtService struct {
}

func (s *XrayExtService) GenerateOutboundFragments() ([]map[string]interface{}, error) {
	db := database.GetDB()
	egresses := make([]*n5model.Egress, 0)
	if err := db.Model(&n5model.Egress{}).Where("enabled = ?", true).Order("id asc").Find(&egresses).Error; err != nil {
		return nil, err
	}

	fragments := make([]map[string]interface{}, 0, len(egresses))
	for _, egress := range egresses {
		obj, err := parseJSONObject(egress.OutboundJSON)
		if err != nil {
			return nil, err
		}
		fragments = append(fragments, obj)
	}
	return fragments, nil
}

func (s *XrayExtService) GenerateRoutingFragments() (map[string]interface{}, error) {
	db := database.GetDB()

	egresses := make([]*n5model.Egress, 0)
	if err := db.Model(&n5model.Egress{}).Where("enabled = ?", true).Find(&egresses).Error; err != nil {
		return nil, err
	}
	egressMap := make(map[int]*n5model.Egress, len(egresses))
	for _, item := range egresses {
		egressMap[item.Id] = item
	}

	pools := make([]*n5model.EgressPool, 0)
	if err := db.Model(&n5model.EgressPool{}).Where("enabled = ?", true).Find(&pools).Error; err != nil {
		return nil, err
	}
	poolMap := make(map[int]*n5model.EgressPool, len(pools))
	for _, item := range pools {
		poolMap[item.Id] = item
	}

	members := make([]*n5model.EgressPoolMember, 0)
	if err := db.Model(&n5model.EgressPoolMember{}).Where("enabled = ?", true).Order("sort_order asc, id asc").Find(&members).Error; err != nil {
		return nil, err
	}
	memberMap := make(map[int][]*n5model.EgressPoolMember)
	for _, member := range members {
		memberMap[member.PoolId] = append(memberMap[member.PoolId], member)
	}

	balancers := make([]interface{}, 0)
	for _, pool := range pools {
		selectors := make([]string, 0)
		for _, member := range memberMap[pool.Id] {
			egress, ok := egressMap[member.EgressId]
			if !ok {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(egress.LastStatus), egressTestStatusFailed) {
				continue
			}
			selectors = append(selectors, egress.Tag)
		}
		if len(selectors) == 0 {
			continue
		}
		balancer := map[string]interface{}{
			"tag":      pool.Tag,
			"selector": selectors,
		}
		if pool.Strategy != "" {
			balancer["strategy"] = map[string]interface{}{
				"type": pool.Strategy,
			}
		}
		if pool.FallbackType != "" && pool.FallbackTargetId > 0 {
			tag, _, err := resolveTargetTag(pool.FallbackType, pool.FallbackTargetId, egressMap, poolMap)
			if err != nil {
				return nil, err
			}
			balancer["fallbackTag"] = tag
		}
		balancers = append(balancers, balancer)
	}

	policies := make([]*n5model.TrafficPolicy, 0)
	if err := db.Model(&n5model.TrafficPolicy{}).Where("enabled = ?", true).Find(&policies).Error; err != nil {
		return nil, err
	}
	policyMap := make(map[int]*n5model.TrafficPolicy, len(policies))
	for _, policy := range policies {
		policyMap[policy.Id] = policy
	}

	rules := make([]*n5model.TrafficPolicyRule, 0)
	if err := db.Model(&n5model.TrafficPolicyRule{}).Where("enabled = ?", true).Order("policy_id asc, sort_order asc, id asc").Find(&rules).Error; err != nil {
		return nil, err
	}
	ruleMap := make(map[int][]*n5model.TrafficPolicyRule)
	for _, rule := range rules {
		ruleMap[rule.PolicyId] = append(ruleMap[rule.PolicyId], rule)
	}

	bindings := make([]*n5model.TrafficPolicyBinding, 0)
	if err := db.Model(&n5model.TrafficPolicyBinding{}).Where("enabled = ?", true).Order("id asc").Find(&bindings).Error; err != nil {
		return nil, err
	}

	routingRules := make([]interface{}, 0)
	for _, binding := range bindings {
		policy, ok := policyMap[binding.PolicyId]
		if !ok {
			continue
		}
		inbound := &legacyModel.Inbound{}
		if err := db.Model(&legacyModel.Inbound{}).Where("id = ?", binding.InboundId).First(inbound).Error; err != nil {
			return nil, err
		}
		if inbound.Tag == "" {
			continue
		}

		for _, rule := range sortedRoutingRulesForPolicy(policy, ruleMap[policy.Id]) {
			routingRule := map[string]interface{}{
				"type":       "field",
				"inboundTag": []string{inbound.Tag},
			}
			switch rule.RuleType {
			case ruleTypeDomain:
				routingRule["domain"] = []string{toDomainMatcher(rule.MatchMode, rule.MatchValue)}
			case ruleTypeIP:
				routingRule["ip"] = []string{rule.MatchValue}
			default:
				continue
			}

			tag, isBalancer, err := resolveTargetTag(rule.TargetType, rule.TargetId, egressMap, poolMap)
			if err != nil {
				return nil, err
			}
			if isBalancer {
				routingRule["balancerTag"] = tag
			} else {
				routingRule["outboundTag"] = tag
			}
			routingRules = append(routingRules, routingRule)
		}

		if policy.DefaultTargetType != "" && policy.DefaultTargetId > 0 {
			defaultRule := map[string]interface{}{
				"type":       "field",
				"inboundTag": []string{inbound.Tag},
			}
			tag, isBalancer, err := resolveTargetTag(policy.DefaultTargetType, policy.DefaultTargetId, egressMap, poolMap)
			if err != nil {
				return nil, err
			}
			if isBalancer {
				defaultRule["balancerTag"] = tag
			} else {
				defaultRule["outboundTag"] = tag
			}
			routingRules = append(routingRules, defaultRule)
		}
	}

	fragment := map[string]interface{}{
		"rules":     routingRules,
		"balancers": balancers,
	}

	data, err := json.Marshal(fragment)
	if err != nil {
		return nil, err
	}

	normalized := make(map[string]interface{})
	if err := json.Unmarshal(data, &normalized); err != nil {
		return nil, err
	}
	return normalized, nil
}

const (
	simpleExecRemarkPrefixForRouting = "n5-simple-exec|"

	simpleRoutingTrafficGroup        = "group"
	simpleRoutingTrafficCustomDomain = "custom-domain"
	simpleRoutingGroupAI             = "ai"
	simpleRoutingGroupGame           = "game"
	simpleRoutingGroupStreaming      = "streaming"
	simpleRoutingGroupCustom         = "custom"
)

type simpleRoutingExecutionRemark struct {
	Version int                           `json:"version"`
	Items   []*simpleRoutingExecutionItem `json:"items"`
}

type simpleRoutingExecutionItem struct {
	TrafficType  string `json:"trafficType"`
	GroupId      int    `json:"groupId,omitempty"`
	GroupName    string `json:"groupName,omitempty"`
	GroupType    string `json:"groupType,omitempty"`
	CustomDomain string `json:"customDomain,omitempty"`
	RuleIDs      []int  `json:"ruleIds,omitempty"`
}

type simpleRoutingRuleRank struct {
	Priority int
	ItemPos  int
	RulePos  int
}

func sortedRoutingRulesForPolicy(policy *n5model.TrafficPolicy, rules []*n5model.TrafficPolicyRule) []*n5model.TrafficPolicyRule {
	if policy == nil || len(rules) <= 1 {
		return rules
	}
	remark, ok := parseSimpleRoutingExecutionRemark(policy.Remark)
	if !ok {
		return rules
	}
	rankByRuleID := make(map[int]simpleRoutingRuleRank)
	for itemPos, item := range remark.Items {
		if item == nil {
			continue
		}
		for rulePos, ruleID := range item.RuleIDs {
			rankByRuleID[ruleID] = simpleRoutingRuleRank{
				Priority: simpleRoutingItemPriority(item, ruleID, rules),
				ItemPos:  itemPos,
				RulePos:  rulePos,
			}
		}
	}
	if len(rankByRuleID) == 0 {
		return rules
	}
	sorted := append([]*n5model.TrafficPolicyRule(nil), rules...)
	originalPos := make(map[int]int, len(sorted))
	for idx, rule := range sorted {
		if rule != nil {
			originalPos[rule.Id] = idx
		}
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		left := sorted[i]
		right := sorted[j]
		if left == nil || right == nil {
			return false
		}
		leftRank, leftOK := rankByRuleID[left.Id]
		rightRank, rightOK := rankByRuleID[right.Id]
		if leftOK != rightOK {
			return leftOK
		}
		if leftOK && rightOK {
			if leftRank.Priority != rightRank.Priority {
				return leftRank.Priority < rightRank.Priority
			}
			if leftRank.ItemPos != rightRank.ItemPos {
				return leftRank.ItemPos < rightRank.ItemPos
			}
			if leftRank.RulePos != rightRank.RulePos {
				return leftRank.RulePos < rightRank.RulePos
			}
		}
		return originalPos[left.Id] < originalPos[right.Id]
	})
	return sorted
}

func parseSimpleRoutingExecutionRemark(remark string) (*simpleRoutingExecutionRemark, bool) {
	remark = strings.TrimSpace(remark)
	if !strings.HasPrefix(remark, simpleExecRemarkPrefixForRouting) {
		return nil, false
	}
	payload := strings.TrimPrefix(remark, simpleExecRemarkPrefixForRouting)
	if len(payload) == 0 {
		return nil, false
	}
	data, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, false
	}
	parsed := &simpleRoutingExecutionRemark{}
	if err := json.Unmarshal(data, parsed); err != nil {
		return nil, false
	}
	if parsed.Version != 1 || parsed.Items == nil {
		return nil, false
	}
	return parsed, true
}

func simpleRoutingItemPriority(item *simpleRoutingExecutionItem, ruleID int, rules []*n5model.TrafficPolicyRule) int {
	if item == nil {
		return 100
	}
	if strings.EqualFold(strings.TrimSpace(item.TrafficType), simpleRoutingTrafficCustomDomain) {
		for _, rule := range rules {
			if rule == nil || rule.Id != ruleID {
				continue
			}
			switch normalizeMatchMode(rule.MatchMode) {
			case domainModeExact:
				return 10
			case domainModeSuffix:
				return 20
			case domainModeKeyword:
				return 30
			case domainModeRegexp:
				return 40
			default:
				return 49
			}
		}
		return 49
	}
	if strings.EqualFold(strings.TrimSpace(item.TrafficType), simpleRoutingTrafficGroup) {
		switch normalizeMatchMode(item.GroupType) {
		case simpleRoutingGroupCustom, "":
			return 50
		case simpleRoutingGroupAI:
			return 60
		case simpleRoutingGroupGame:
			return 70
		case simpleRoutingGroupStreaming:
			return 80
		default:
			return 50
		}
	}
	switch normalizeMatchMode(item.TrafficType) {
	case simpleRoutingGroupAI:
		return 60
	case simpleRoutingGroupGame:
		return 70
	case simpleRoutingGroupStreaming:
		return 80
	default:
		return 100
	}
}
