package n5

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"x-ui/database"
	n5model "x-ui/database/model/n5"
	"x-ui/xray"
)

const (
	xrayHistorySourceN5Merge   = "n5-merge"
	xrayHistoryStatusGenerated = "generated"
	xrayHistoryStatusValidated = "validated"
	xrayHistoryStatusApplied   = "applied"
	xrayHistoryStatusFailed    = "failed"
)

func XrayHistoryStatusValidated() string {
	return xrayHistoryStatusValidated
}

func XrayHistoryStatusApplied() string {
	return xrayHistoryStatusApplied
}

func XrayHistoryStatusFailed() string {
	return xrayHistoryStatusFailed
}

type XrayMergeService struct {
	xrayExt XrayExtService
}

type XrayMergeResult struct {
	Config              *xray.Config
	HistoryID           int
	BaseConfigHash      string
	ExtensionConfigHash string
	ConfigHash          string
	OutboundCount       int
	RoutingRuleCount    int
	BindingCount        int
}

func (s *XrayMergeService) Merge(base *xray.Config) (*xray.Config, error) {
	result, err := s.MergeWithMeta(base)
	if err != nil {
		return nil, err
	}
	return result.Config, nil
}

func (s *XrayMergeService) MergeWithMeta(base *xray.Config) (*XrayMergeResult, error) {
	if base == nil {
		base = &xray.Config{}
	}

	merged := cloneXrayConfig(base)
	baseHash, err := hashXrayConfig(base)
	if err != nil {
		return nil, err
	}

	n5Outbounds, err := s.xrayExt.GenerateOutboundFragments()
	if err != nil {
		return nil, err
	}
	if err := mergeOutboundConfigs(merged, n5Outbounds); err != nil {
		return nil, err
	}

	n5Routing, err := s.xrayExt.GenerateRoutingFragments()
	if err != nil {
		return nil, err
	}
	if err := mergeRoutingConfig(merged, n5Routing); err != nil {
		return nil, err
	}

	extensionHash, err := hashJSONValue(map[string]interface{}{
		"outbounds": n5Outbounds,
		"routing":   n5Routing,
	})
	if err != nil {
		return nil, err
	}
	configHash, configJSON, err := serializeXrayConfig(merged)
	if err != nil {
		return nil, err
	}
	routingRuleCount, err := countRoutingRules(n5Routing)
	if err != nil {
		return nil, err
	}
	bindingCount, err := countEnabledBindings()
	if err != nil {
		return nil, err
	}

	history := &n5model.XrayConfigHistory{
		Source:              xrayHistorySourceN5Merge,
		BaseConfigHash:      baseHash,
		ExtensionConfigHash: extensionHash,
		ConfigHash:          configHash,
		ConfigJSON:          configJSON,
		ApplyStatus:         xrayHistoryStatusGenerated,
	}
	if err := database.GetDB().Create(history).Error; err != nil {
		return nil, err
	}

	return &XrayMergeResult{
		Config:              merged,
		HistoryID:           history.Id,
		BaseConfigHash:      baseHash,
		ExtensionConfigHash: extensionHash,
		ConfigHash:          configHash,
		OutboundCount:       len(n5Outbounds),
		RoutingRuleCount:    routingRuleCount,
		BindingCount:        bindingCount,
	}, nil
}

func (s *XrayMergeService) UpdateHistoryStatus(historyID int, status string, applyErr error) error {
	if historyID <= 0 {
		return nil
	}

	updates := map[string]interface{}{
		"apply_status": status,
		"apply_error":  "",
	}
	if applyErr != nil {
		updates["apply_error"] = strings.TrimSpace(applyErr.Error())
	}

	return database.GetDB().Model(&n5model.XrayConfigHistory{}).
		Where("id = ?", historyID).
		Updates(updates).Error
}

func cloneXrayConfig(base *xray.Config) *xray.Config {
	clone := &xray.Config{
		LogConfig:       append([]byte(nil), base.LogConfig...),
		RouterConfig:    append([]byte(nil), base.RouterConfig...),
		DNSConfig:       append([]byte(nil), base.DNSConfig...),
		OutboundConfigs: append([]byte(nil), base.OutboundConfigs...),
		Transport:       append([]byte(nil), base.Transport...),
		Policy:          append([]byte(nil), base.Policy...),
		API:             append([]byte(nil), base.API...),
		Stats:           append([]byte(nil), base.Stats...),
		Reverse:         append([]byte(nil), base.Reverse...),
		FakeDNS:         append([]byte(nil), base.FakeDNS...),
	}
	if len(base.InboundConfigs) > 0 {
		clone.InboundConfigs = append([]xray.InboundConfig(nil), base.InboundConfigs...)
	}
	return clone
}

func mergeOutboundConfigs(cfg *xray.Config, appended []map[string]interface{}) error {
	baseOutbounds := make([]map[string]interface{}, 0)
	if len(cfg.OutboundConfigs) > 0 && string(cfg.OutboundConfigs) != "null" {
		if err := json.Unmarshal(cfg.OutboundConfigs, &baseOutbounds); err != nil {
			return err
		}
	}

	for _, outbound := range appended {
		baseOutbounds = append(baseOutbounds, outbound)
	}

	data, err := json.Marshal(baseOutbounds)
	if err != nil {
		return err
	}
	cfg.OutboundConfigs = data
	return nil
}

func mergeRoutingConfig(cfg *xray.Config, extension map[string]interface{}) error {
	baseRouting := make(map[string]interface{})
	if len(cfg.RouterConfig) > 0 && string(cfg.RouterConfig) != "null" {
		if err := json.Unmarshal(cfg.RouterConfig, &baseRouting); err != nil {
			return err
		}
	}

	baseRules, err := normalizeInterfaceSlice(baseRouting["rules"])
	if err != nil {
		return err
	}
	extRules, err := normalizeInterfaceSlice(extension["rules"])
	if err != nil {
		return err
	}
	baseRouting["rules"] = append(baseRules, extRules...)

	baseBalancers, err := normalizeInterfaceSlice(baseRouting["balancers"])
	if err != nil {
		return err
	}
	extBalancers, err := normalizeInterfaceSlice(extension["balancers"])
	if err != nil {
		return err
	}
	if len(baseBalancers) > 0 || len(extBalancers) > 0 {
		baseRouting["balancers"] = append(baseBalancers, extBalancers...)
	}

	data, err := json.Marshal(baseRouting)
	if err != nil {
		return err
	}
	cfg.RouterConfig = data
	return nil
}

func normalizeInterfaceSlice(value interface{}) ([]interface{}, error) {
	if value == nil {
		return []interface{}{}, nil
	}
	items, ok := value.([]interface{})
	if ok {
		return items, nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	items = make([]interface{}, 0)
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func countRoutingRules(routing map[string]interface{}) (int, error) {
	items, err := normalizeInterfaceSlice(routing["rules"])
	if err != nil {
		return 0, err
	}
	return len(items), nil
}

func countEnabledBindings() (int, error) {
	var count int64
	err := database.GetDB().
		Model(&n5model.TrafficPolicyBinding{}).
		Where("enabled = ?", true).
		Count(&count).Error
	return int(count), err
}

func serializeXrayConfig(cfg *xray.Config) (string, string, error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), string(data), nil
}

func hashXrayConfig(cfg *xray.Config) (string, error) {
	hash, _, err := serializeXrayConfig(cfg)
	return hash, err
}

func hashJSONValue(value interface{}) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
