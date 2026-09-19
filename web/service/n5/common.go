package n5

import (
	"encoding/json"
	"fmt"
	"github.com/torr9522/n6-ui/database"
	legacyModel "github.com/torr9522/n6-ui/database/model"
	n5model "github.com/torr9522/n6-ui/database/model/n5"
	"github.com/torr9522/n6-ui/util/common"
	ssservice "github.com/torr9522/n6-ui/web/service/shadowsocks"
	"net"
	"regexp"
	"strings"
)

const (
	n5TagIDWidth = 10

	targetTypeEgress = "egress"
	targetTypePool   = "pool"

	labelTypeRegion  = "region"
	labelTypeUsage   = "usage"
	labelTypeQuality = "quality"

	ruleTypeDomain = "domain"
	ruleTypeIP     = "ip"

	domainModeExact   = "exact"
	domainModeSuffix  = "suffix"
	domainModeKeyword = "keyword"
	domainModeRegexp  = "regexp"

	ipModeIP   = "ip"
	ipModeCIDR = "cidr"
)

func formatN5Tag(prefix string, id int) string {
	return fmt.Sprintf("%s-%0*d", prefix, n5TagIDWidth, id)
}

var allowedOutboundProtocols = map[string]bool{
	"blackhole":   true,
	"dns":         true,
	"freedom":     true,
	"http":        true,
	"shadowsocks": true,
	"socks":       true,
	"trojan":      true,
	"vless":       true,
	"vmess":       true,
	"wireguard":   true,
}

func normalizeName(name string) string {
	return strings.TrimSpace(name)
}

func normalizeProtocol(protocol string) string {
	return strings.TrimSpace(strings.ToLower(protocol))
}

func normalizeTargetType(targetType string) string {
	return strings.TrimSpace(strings.ToLower(targetType))
}

func normalizeRuleType(ruleType string) string {
	return strings.TrimSpace(strings.ToLower(ruleType))
}

func normalizeMatchMode(matchMode string) string {
	return strings.TrimSpace(strings.ToLower(matchMode))
}

func normalizeLabelType(labelType string) string {
	return strings.TrimSpace(strings.ToLower(labelType))
}

func allowedPoolStrategy(strategy string) bool {
	switch normalizeProtocol(strategy) {
	case "", "random", "roundrobin", "leastping", "leastload":
		return true
	default:
		return false
	}
}

func validateLabelType(labelType string) error {
	switch normalizeLabelType(labelType) {
	case labelTypeRegion, labelTypeUsage, labelTypeQuality:
		return nil
	default:
		return common.NewError("invalid label type")
	}
}

func validateTarget(targetType string, targetId int) error {
	targetType = normalizeTargetType(targetType)
	if targetType == "" {
		return common.NewError("target type is required")
	}
	if targetId <= 0 {
		return common.NewError("target id is required")
	}

	db := database.GetDB()
	switch targetType {
	case targetTypeEgress:
		var count int64
		if err := db.Model(&n5model.Egress{}).Where("id = ?", targetId).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return common.NewError("egress not found")
		}
	case targetTypePool:
		var count int64
		if err := db.Model(&n5model.EgressPool{}).Where("id = ?", targetId).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return common.NewError("egress pool not found")
		}
	default:
		return common.NewError("unknown target type")
	}

	return nil
}

func validateDomainRule(mode, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return common.NewError("match value is required")
	}

	switch normalizeMatchMode(mode) {
	case domainModeExact, domainModeSuffix, domainModeKeyword:
		return nil
	case domainModeRegexp:
		if _, err := regexp.Compile(value); err != nil {
			return common.NewErrorf("invalid regexp: %v", err)
		}
		return nil
	default:
		return common.NewError("invalid domain match mode")
	}
}

func validateIPRule(mode, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return common.NewError("match value is required")
	}

	switch normalizeMatchMode(mode) {
	case ipModeIP:
		if ip := net.ParseIP(value); ip == nil {
			return common.NewError("invalid ip address")
		}
		return nil
	case ipModeCIDR:
		if _, _, err := net.ParseCIDR(value); err != nil {
			return common.NewErrorf("invalid cidr: %v", err)
		}
		if !strings.Contains(value, "/") {
			return common.NewError("cidr match value must contain /")
		}
		return nil
	default:
		return common.NewError("invalid ip match mode")
	}
}

func toDomainMatcher(mode, value string) string {
	value = strings.TrimSpace(value)
	switch normalizeMatchMode(mode) {
	case domainModeExact:
		return "full:" + value
	case domainModeSuffix:
		return "domain:" + value
	case domainModeKeyword:
		return "keyword:" + value
	case domainModeRegexp:
		return "regexp:" + value
	default:
		return value
	}
}

func parseJSONObject(raw string) (map[string]interface{}, error) {
	obj := make(map[string]interface{})
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func validateOutboundShape(protocol string, obj map[string]interface{}) error {
	if !allowedOutboundProtocols[protocol] {
		return common.NewError("unsupported outbound protocol")
	}
	settings, exists := obj["settings"]
	if !exists || settings == nil {
		return common.NewError("outbound settings is required")
	}
	if _, ok := settings.(map[string]interface{}); !ok {
		return common.NewError("outbound settings must be an object")
	}
	if protocol == "shadowsocks" {
		return validateShadowsocksOutboundSettings(settings.(map[string]interface{}))
	}
	return nil
}

func validateShadowsocksOutboundSettings(settings map[string]interface{}) error {
	servers, ok := settings["servers"].([]interface{})
	if !ok || len(servers) != 1 {
		return common.NewError("Shadowsocks 出口必须且只能包含一个 server")
	}
	server, ok := servers[0].(map[string]interface{})
	if !ok {
		return common.NewError("Shadowsocks server 配置格式无效")
	}
	address, _ := server["address"].(string)
	if strings.TrimSpace(address) == "" {
		return common.NewError("Shadowsocks 出口地址不能为空")
	}
	port, ok := server["port"].(float64)
	if !ok || port < 1 || port > 65535 || port != float64(int(port)) {
		return common.NewError("Shadowsocks 出口端口必须是 1-65535 的整数")
	}
	method, _ := server["method"].(string)
	password, _ := server["password"].(string)
	return ssservice.ValidateCredential(method, password)
}

func resolveTargetTag(targetType string, targetId int, egresses map[int]*n5model.Egress, pools map[int]*n5model.EgressPool) (string, bool, error) {
	switch normalizeTargetType(targetType) {
	case targetTypeEgress:
		egress, ok := egresses[targetId]
		if !ok {
			return "", false, common.NewError("target egress not found")
		}
		return egress.Tag, false, nil
	case targetTypePool:
		pool, ok := pools[targetId]
		if !ok {
			return "", true, common.NewError("target pool not found")
		}
		return pool.Tag, true, nil
	default:
		return "", false, common.NewError("unknown target type")
	}
}

func getInboundByID(inboundId int) (*legacyModel.Inbound, error) {
	db := database.GetDB()
	inbound := &legacyModel.Inbound{}
	if err := db.Model(&legacyModel.Inbound{}).Where("id = ?", inboundId).First(inbound).Error; err != nil {
		return nil, err
	}
	return inbound, nil
}
