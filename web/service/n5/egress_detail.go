package n5

import (
	"strings"
	"x-ui/database"
	n5model "x-ui/database/model/n5"
	"x-ui/util/common"
)

type EgressDetailPool struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

type EgressDetailPolicy struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type EgressDetail struct {
	Id           int                    `json:"id"`
	Name         string                 `json:"name"`
	Protocol     string                 `json:"protocol"`
	Address      string                 `json:"address"`
	Tag          string                 `json:"tag"`
	LastStatus   string                 `json:"lastStatus"`
	LastExitIP   string                 `json:"lastExitIP"`
	LastTestTime int64                  `json:"lastTestTime"`
	Labels       []*n5model.EgressLabel `json:"labels"`
	Pools        []*EgressDetailPool    `json:"pools"`
	Policies     []*EgressDetailPolicy  `json:"policies"`
}

type EgressDetailService struct {
	labelService EgressLabelService
}

func (s *EgressDetailService) Get(id int) (*EgressDetail, error) {
	if id <= 0 {
		return nil, common.NewError("invalid egress id")
	}

	db := database.GetDB()
	egress := &n5model.Egress{}
	if err := db.Model(&n5model.Egress{}).Where("id = ?", id).First(egress).Error; err != nil {
		return nil, err
	}

	labels, err := s.labelService.ListByEgress(id)
	if err != nil {
		return nil, err
	}
	pools, poolIDs, err := listEgressDetailPools(id)
	if err != nil {
		return nil, err
	}
	policies, err := listEgressDetailPolicies(id, poolIDs)
	if err != nil {
		return nil, err
	}

	return &EgressDetail{
		Id:           egress.Id,
		Name:         egress.Name,
		Protocol:     egress.Protocol,
		Address:      extractOutboundAddress(egress.OutboundJSON),
		Tag:          egress.Tag,
		LastStatus:   egress.LastStatus,
		LastExitIP:   egress.LastExitIP,
		LastTestTime: egress.LastTestTime,
		Labels:       labels,
		Pools:        pools,
		Policies:     policies,
	}, nil
}

func listEgressDetailPools(egressId int) ([]*EgressDetailPool, []int, error) {
	type poolRecord struct {
		Id   int
		Name string
		Tag  string
	}

	rows := make([]*poolRecord, 0)
	err := database.GetDB().
		Table("n5_egress_pools").
		Select("n5_egress_pools.id, n5_egress_pools.name, n5_egress_pools.tag").
		Joins("join n5_egress_pool_members on n5_egress_pool_members.pool_id = n5_egress_pools.id").
		Where("n5_egress_pool_members.egress_id = ? and n5_egress_pool_members.enabled = ?", egressId, true).
		Order("n5_egress_pools.id asc").
		Scan(&rows).Error
	if err != nil {
		return nil, nil, err
	}

	pools := make([]*EgressDetailPool, 0, len(rows))
	poolIDs := make([]int, 0, len(rows))
	for _, row := range rows {
		pools = append(pools, &EgressDetailPool{
			Id:   row.Id,
			Name: row.Name,
			Tag:  row.Tag,
		})
		poolIDs = append(poolIDs, row.Id)
	}
	return pools, poolIDs, nil
}

func listEgressDetailPolicies(egressId int, poolIDs []int) ([]*EgressDetailPolicy, error) {
	policies := make([]*n5model.TrafficPolicy, 0)
	db := database.GetDB().Model(&n5model.TrafficPolicy{}).Where("enabled = ?", true)
	condition := db.Where("default_target_type = ? and default_target_id = ?", targetTypeEgress, egressId)
	if len(poolIDs) > 0 {
		condition = condition.Or("default_target_type = ? and default_target_id in ?", targetTypePool, poolIDs)
	}
	if err := condition.Find(&policies).Error; err != nil {
		return nil, err
	}

	rules := make([]*n5model.TrafficPolicyRule, 0)
	ruleQuery := database.GetDB().Model(&n5model.TrafficPolicyRule{}).Where("enabled = ?", true)
	ruleCondition := ruleQuery.Where("target_type = ? and target_id = ?", targetTypeEgress, egressId)
	if len(poolIDs) > 0 {
		ruleCondition = ruleCondition.Or("target_type = ? and target_id in ?", targetTypePool, poolIDs)
	}
	if err := ruleCondition.Find(&rules).Error; err != nil {
		return nil, err
	}

	policyMap := make(map[int]*EgressDetailPolicy)
	for _, policy := range policies {
		policyMap[policy.Id] = &EgressDetailPolicy{
			Id:   policy.Id,
			Name: policy.Name,
		}
	}

	if len(rules) > 0 {
		policyIDs := make([]int, 0, len(rules))
		for _, rule := range rules {
			policyIDs = append(policyIDs, rule.PolicyId)
		}
		relatedPolicies := make([]*n5model.TrafficPolicy, 0)
		if err := database.GetDB().
			Model(&n5model.TrafficPolicy{}).
			Where("id in ?", policyIDs).
			Order("id asc").
			Find(&relatedPolicies).Error; err != nil {
			return nil, err
		}
		for _, policy := range relatedPolicies {
			policyMap[policy.Id] = &EgressDetailPolicy{
				Id:   policy.Id,
				Name: policy.Name,
			}
		}
	}

	result := make([]*EgressDetailPolicy, 0, len(policyMap))
	for _, policy := range policyMap {
		result = append(result, policy)
	}
	sortEgressDetailPolicies(result)
	return result, nil
}

func sortEgressDetailPolicies(items []*EgressDetailPolicy) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].Id < items[i].Id {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

func extractOutboundAddress(outboundJSON string) string {
	obj, err := parseJSONObject(outboundJSON)
	if err != nil {
		return ""
	}
	settings, _ := obj["settings"].(map[string]interface{})
	if len(settings) == 0 {
		return ""
	}

	if value := firstStringFromValue(settings["address"]); value != "" {
		return value
	}
	if value := firstStringFromValue(settings["server"]); value != "" {
		return value
	}
	if value := firstAddressFromSlice(settings["servers"]); value != "" {
		return value
	}
	if value := firstAddressFromSlice(settings["vnext"]); value != "" {
		return value
	}
	if value := firstAddressFromSlice(settings["peers"]); value != "" {
		return value
	}
	return ""
}

func firstAddressFromSlice(value interface{}) string {
	items, ok := value.([]interface{})
	if !ok || len(items) == 0 {
		return ""
	}
	first, ok := items[0].(map[string]interface{})
	if !ok {
		return firstStringFromValue(items[0])
	}
	if address := firstStringFromValue(first["address"]); address != "" {
		return address
	}
	if server := firstStringFromValue(first["server"]); server != "" {
		return server
	}
	if endpoint := firstStringFromValue(first["endpoint"]); endpoint != "" {
		return endpoint
	}
	return ""
}

func firstStringFromValue(value interface{}) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}
