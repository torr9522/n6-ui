package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/torr9522/n6-ui/database/model"
	"github.com/torr9522/n6-ui/util/common"
	ssservice "github.com/torr9522/n6-ui/web/service/shadowsocks"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

type ShareLinkService struct {
	shareAddressService ShareAddressService
	loadShareAddresses  func() ([]ShareAddress, error)
}

type shareContext struct {
	Scheme      string
	RequestHost string
}

type shareInbound struct {
	Inbound model.Inbound

	settings inboundSettings
	stream   inboundStreamSettings
}

type inboundSettings struct {
	VMess       *vmessSettings
	VLESS       *vlessSettings
	Trojan      *trojanSettings
	Shadowsocks *shadowsocksSettings
}

type vmessSettings struct {
	Clients []struct {
		ID      string `json:"id"`
		AlterID int    `json:"alterId"`
	} `json:"clients"`
}

type vlessSettings struct {
	Clients []struct {
		ID         string `json:"id"`
		Flow       string `json:"flow"`
		Encryption string `json:"encryption"`
	} `json:"clients"`
	Decryption string `json:"decryption"`
}

type trojanSettings struct {
	Clients []struct {
		Password string `json:"password"`
	} `json:"clients"`
}

type shadowsocksSettings struct {
	Method   string          `json:"method"`
	Password string          `json:"password"`
	Network  string          `json:"network"`
	Clients  json.RawMessage `json:"clients"`
}

type inboundStreamSettings struct {
	Network         string           `json:"network"`
	Security        string           `json:"security"`
	TLSSettings     *tlsSettings     `json:"tlsSettings"`
	XTLSSettings    *tlsSettings     `json:"xtlsSettings"`
	RealitySettings *realitySettings `json:"realitySettings"`
	TCPSettings     *tcpSettings     `json:"tcpSettings"`
	WSSettings      *wsSettings      `json:"wsSettings"`
	HTTPSettings    *httpSettings    `json:"httpSettings"`
	GRPCSettings    *grpcSettings    `json:"grpcSettings"`
}

type tlsSettings struct {
	ServerName string `json:"serverName"`
}

type realitySettings struct {
	ServerNames []string `json:"serverNames"`
	PrivateKey  string   `json:"privateKey"`
	ShortIds    []string `json:"shortIds"`
	Fingerprint string   `json:"fingerprint"`
	PublicKey   string   `json:"publicKey"`
	SpiderX     string   `json:"spiderX"`
}

type tcpSettings struct {
	Header struct {
		Type     string       `json:"type"`
		Request  *tcpRequest  `json:"request"`
		Response *tcpResponse `json:"response"`
	} `json:"header"`
}

type tcpRequest struct {
	Path    []string               `json:"path"`
	Headers map[string]interface{} `json:"headers"`
}

type tcpResponse struct{}

type wsSettings struct {
	Path    string                 `json:"path"`
	Headers map[string]interface{} `json:"headers"`
}

type httpSettings struct {
	Path string   `json:"path"`
	Host []string `json:"host"`
}

type grpcSettings struct {
	ServiceName string `json:"serviceName"`
}

type clashConfig struct {
	Proxies     []map[string]interface{} `yaml:"proxies"`
	ProxyGroups []map[string]interface{} `yaml:"proxy-groups"`
	Rules       []string                 `yaml:"rules"`
}

func (s *ShareLinkService) GenerateBase64(inbounds []*model.Inbound, ctx shareContext) (string, error) {
	links, err := s.GenerateLinks(inbounds, ctx)
	if err != nil {
		return "", err
	}
	if len(links) == 0 {
		return "", common.NewError("no supported inbounds in subscription")
	}
	return base64.StdEncoding.EncodeToString([]byte(strings.Join(links, "\n"))), nil
}

func (s *ShareLinkService) GenerateLinks(inbounds []*model.Inbound, ctx shareContext) ([]string, error) {
	links := make([]string, 0, len(inbounds))
	for _, inbound := range inbounds {
		parsed, err := s.parseInbound(inbound)
		if err != nil {
			return nil, err
		}
		link, err := s.buildLink(parsed, ctx)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, nil
}

func (s *ShareLinkService) GenerateClash(inbounds []*model.Inbound, ctx shareContext) (string, error) {
	return s.generateClashCompatible(inbounds, ctx, false)
}

func (s *ShareLinkService) GenerateMihomo(inbounds []*model.Inbound, ctx shareContext) (string, error) {
	return s.generateClashCompatible(inbounds, ctx, true)
}

func (s *ShareLinkService) generateClashCompatible(inbounds []*model.Inbound, ctx shareContext, allowShadowsocks2022 bool) (string, error) {
	proxies := make([]map[string]interface{}, 0, len(inbounds))
	nameCounts := map[string]int{}
	for _, inbound := range inbounds {
		parsed, err := s.parseInbound(inbound)
		if err != nil {
			return "", err
		}
		if parsed.settings.Shadowsocks != nil && ssservice.IsShadowsocks2022(parsed.settings.Shadowsocks.Method) && !allowShadowsocks2022 {
			return "", common.NewError("经典 Clash 格式不支持 Shadowsocks 2022，请使用 Mihomo 订阅")
		}
		proxy, err := s.buildClashProxy(parsed, ctx)
		if err != nil {
			return "", err
		}
		name := strings.TrimSpace(fmt.Sprint(proxy["name"]))
		if name == "" {
			name = fmt.Sprintf("%s-%d", parsed.Inbound.Protocol, parsed.Inbound.Id)
		}
		nameCounts[name]++
		if nameCounts[name] > 1 {
			name = fmt.Sprintf("%s #%d", name, nameCounts[name])
		}
		proxy["name"] = name
		proxies = append(proxies, proxy)
	}
	if len(proxies) == 0 {
		return "", common.NewError("no supported inbounds in subscription")
	}

	names := make([]string, 0, len(proxies))
	for _, proxy := range proxies {
		names = append(names, fmt.Sprint(proxy["name"]))
	}
	cfg := clashConfig{
		Proxies: proxies,
		ProxyGroups: []map[string]interface{}{
			{
				"name":    "PROXY",
				"type":    "select",
				"proxies": names,
			},
		},
		Rules: []string{"MATCH,PROXY"},
	}

	return renderClashConfig(cfg), nil
}

func (s *ShareLinkService) parseInbound(inbound *model.Inbound) (*shareInbound, error) {
	parsed := &shareInbound{Inbound: *inbound}
	switch inbound.Protocol {
	case model.VMess:
		parsed.settings.VMess = &vmessSettings{}
		if err := json.Unmarshal([]byte(inbound.Settings), parsed.settings.VMess); err != nil {
			return nil, common.NewError("invalid vmess settings")
		}
	case model.VLESS:
		parsed.settings.VLESS = &vlessSettings{}
		if err := json.Unmarshal([]byte(inbound.Settings), parsed.settings.VLESS); err != nil {
			return nil, common.NewError("invalid vless settings")
		}
	case model.Trojan:
		parsed.settings.Trojan = &trojanSettings{}
		if err := json.Unmarshal([]byte(inbound.Settings), parsed.settings.Trojan); err != nil {
			return nil, common.NewError("invalid trojan settings")
		}
	case model.Shadowsocks:
		parsed.settings.Shadowsocks = &shadowsocksSettings{}
		if err := json.Unmarshal([]byte(inbound.Settings), parsed.settings.Shadowsocks); err != nil {
			return nil, common.NewError("invalid shadowsocks settings")
		}
		if ssservice.IsShadowsocks2022(parsed.settings.Shadowsocks.Method) {
			if hasShadowsocksClients(parsed.settings.Shadowsocks.Clients) {
				return nil, common.NewError("本版本仅支持 Shadowsocks 2022 单用户模式")
			}
			if err := ssservice.ValidateCredential(parsed.settings.Shadowsocks.Method, parsed.settings.Shadowsocks.Password); err != nil {
				return nil, err
			}
		}
	default:
		return nil, common.NewError("unsupported inbound protocol:", inbound.Protocol)
	}

	streamJSON := strings.TrimSpace(inbound.StreamSettings)
	if streamJSON == "" {
		streamJSON = `{}`
	}
	if err := json.Unmarshal([]byte(streamJSON), &parsed.stream); err != nil {
		return nil, common.NewError("invalid stream settings")
	}
	if parsed.stream.Network == "" {
		parsed.stream.Network = "tcp"
	}
	if parsed.stream.Security == "" {
		parsed.stream.Security = "none"
	}
	return parsed, nil
}

func (s *ShareLinkService) buildLink(inbound *shareInbound, ctx shareContext) (string, error) {
	address, err := s.resolveAddress(inbound, ctx)
	if err != nil {
		return "", err
	}
	switch inbound.Inbound.Protocol {
	case model.VMess:
		return s.buildVMessLink(inbound, address)
	case model.VLESS:
		return s.buildVLESSLink(inbound, address)
	case model.Trojan:
		return s.buildTrojanLink(inbound, address)
	case model.Shadowsocks:
		return s.buildShadowsocksLink(inbound, address)
	default:
		return "", common.NewError("unsupported inbound protocol:", inbound.Inbound.Protocol)
	}
}

func (s *ShareLinkService) buildVMessLink(inbound *shareInbound, address string) (string, error) {
	if inbound.settings.VMess == nil || len(inbound.settings.VMess.Clients) == 0 {
		return "", common.NewError("vmess client is empty")
	}
	client := inbound.settings.VMess.Clients[0]
	host := ""
	path := ""
	network := inbound.stream.Network
	typ := "none"
	switch network {
	case "tcp":
		if inbound.stream.TCPSettings != nil && inbound.stream.TCPSettings.Header.Type == "http" && inbound.stream.TCPSettings.Header.Request != nil {
			typ = "http"
			path = strings.Join(inbound.stream.TCPSettings.Header.Request.Path, ",")
			host = firstHeaderValue(inbound.stream.TCPSettings.Header.Request.Headers, "Host")
		}
	case "ws":
		if inbound.stream.WSSettings != nil {
			path = inbound.stream.WSSettings.Path
			host = firstHeaderValue(inbound.stream.WSSettings.Headers, "Host")
		}
	case "http":
		network = "h2"
		if inbound.stream.HTTPSettings != nil {
			path = inbound.stream.HTTPSettings.Path
			host = strings.Join(inbound.stream.HTTPSettings.Host, ",")
		}
	case "grpc":
		if inbound.stream.GRPCSettings != nil {
			path = inbound.stream.GRPCSettings.ServiceName
		}
	}
	if inbound.stream.Security == "tls" && inbound.tlsServerName() != "" {
		address = inbound.tlsServerName()
	}
	obj := map[string]interface{}{
		"v":    "2",
		"ps":   inbound.Inbound.Remark,
		"add":  address,
		"port": strconv.Itoa(inbound.Inbound.Port),
		"id":   client.ID,
		"aid":  strconv.Itoa(client.AlterID),
		"scy":  "auto",
		"net":  network,
		"type": typ,
		"host": host,
		"path": path,
		"tls":  inbound.stream.Security,
		"sni":  "",
	}
	if inbound.stream.Security == "tls" {
		obj["sni"] = inbound.tlsServerName()
	}
	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "", err
	}
	return "vmess://" + base64.StdEncoding.EncodeToString(data), nil
}

func (s *ShareLinkService) buildVLESSLink(inbound *shareInbound, address string) (string, error) {
	if inbound.settings.VLESS == nil || len(inbound.settings.VLESS.Clients) == 0 {
		return "", common.NewError("vless client is empty")
	}
	client := inbound.settings.VLESS.Clients[0]
	hostForURL := bracketIPv6(address)
	link := fmt.Sprintf("vless://%s@%s:%d", client.ID, hostForURL, inbound.Inbound.Port)
	u, err := url.Parse(link)
	if err != nil {
		return "", err
	}
	params := url.Values{}
	decryption := inbound.settings.VLESS.Decryption
	if decryption == "" {
		decryption = "none"
	}
	params.Set("encryption", decryption)
	params.Set("type", inbound.stream.Network)
	security := inbound.stream.Security
	params.Set("security", security)
	switch inbound.stream.Network {
	case "tcp":
		if inbound.stream.TCPSettings != nil && inbound.stream.TCPSettings.Header.Type == "http" && inbound.stream.TCPSettings.Header.Request != nil {
			params.Set("path", strings.Join(inbound.stream.TCPSettings.Header.Request.Path, ","))
			if host := firstHeaderValue(inbound.stream.TCPSettings.Header.Request.Headers, "Host"); host != "" {
				params.Set("host", host)
			}
		}
	case "ws":
		if inbound.stream.WSSettings != nil {
			params.Set("path", inbound.stream.WSSettings.Path)
			if host := firstHeaderValue(inbound.stream.WSSettings.Headers, "Host"); host != "" {
				params.Set("host", host)
			}
		}
	case "http":
		if inbound.stream.HTTPSettings != nil {
			params.Set("path", inbound.stream.HTTPSettings.Path)
			if len(inbound.stream.HTTPSettings.Host) > 0 {
				params.Set("host", strings.Join(inbound.stream.HTTPSettings.Host, ","))
			}
		}
	case "grpc":
		if inbound.stream.GRPCSettings != nil {
			params.Set("serviceName", inbound.stream.GRPCSettings.ServiceName)
		}
	}
	if security == "reality" {
		if inbound.stream.Network != "tcp" && inbound.stream.Network != "grpc" {
			return "", common.NewError("Reality share only supports tcp or grpc")
		}
		reality := inbound.stream.RealitySettings
		if reality == nil || strings.TrimSpace(reality.PublicKey) == "" {
			return "", common.NewError("Reality publicKey is empty")
		}
		params.Set("security", "reality")
		params.Set("sni", firstNonEmpty(reality.ServerNames, address))
		params.Set("fp", defaultString(reality.Fingerprint, "chrome"))
		params.Set("pbk", reality.PublicKey)
		params.Set("sid", firstNonEmpty(reality.ShortIds, ""))
		if strings.TrimSpace(reality.SpiderX) != "" {
			params.Set("spx", reality.SpiderX)
		}
	} else if security == "tls" {
		if sni := inbound.tlsServerName(); sni != "" {
			address = sni
			u.Host = net.JoinHostPort(bracketIPv6(address), strconv.Itoa(inbound.Inbound.Port))
			params.Set("sni", sni)
		}
	}
	if strings.TrimSpace(client.Flow) != "" {
		params.Set("flow", client.Flow)
	}
	u.RawQuery = params.Encode()
	u.Fragment = inbound.Inbound.Remark
	return u.String(), nil
}

func (s *ShareLinkService) buildTrojanLink(inbound *shareInbound, address string) (string, error) {
	if inbound.settings.Trojan == nil || len(inbound.settings.Trojan.Clients) == 0 {
		return "", common.NewError("trojan client is empty")
	}
	if inbound.stream.Security == "tls" || inbound.stream.Security == "xtls" {
		if sni := inbound.tlsServerName(); sni != "" {
			address = sni
		}
	}
	link := fmt.Sprintf("trojan://%s@%s:%d", url.QueryEscape(inbound.settings.Trojan.Clients[0].Password), bracketIPv6(address), inbound.Inbound.Port)
	u, err := url.Parse(link)
	if err != nil {
		return "", err
	}
	params := url.Values{}
	params.Set("security", inbound.stream.Security)
	params.Set("type", inbound.stream.Network)
	switch inbound.stream.Network {
	case "tcp":
		if inbound.stream.TCPSettings != nil && inbound.stream.TCPSettings.Header.Type == "http" && inbound.stream.TCPSettings.Header.Request != nil {
			params.Set("path", strings.Join(inbound.stream.TCPSettings.Header.Request.Path, ","))
			if host := firstHeaderValue(inbound.stream.TCPSettings.Header.Request.Headers, "Host"); host != "" {
				params.Set("host", host)
			}
		}
	case "ws":
		if inbound.stream.WSSettings != nil {
			params.Set("path", inbound.stream.WSSettings.Path)
			if host := firstHeaderValue(inbound.stream.WSSettings.Headers, "Host"); host != "" {
				params.Set("host", host)
			}
		}
	case "http":
		if inbound.stream.HTTPSettings != nil {
			params.Set("path", inbound.stream.HTTPSettings.Path)
			if len(inbound.stream.HTTPSettings.Host) > 0 {
				params.Set("host", strings.Join(inbound.stream.HTTPSettings.Host, ","))
			}
		}
	case "grpc":
		if inbound.stream.GRPCSettings != nil {
			params.Set("serviceName", inbound.stream.GRPCSettings.ServiceName)
		}
	}
	if sni := inbound.tlsServerName(); sni != "" && (inbound.stream.Security == "tls" || inbound.stream.Security == "xtls") {
		params.Set("sni", sni)
	}
	u.RawQuery = params.Encode()
	u.Fragment = inbound.Inbound.Remark
	return u.String(), nil
}

func (s *ShareLinkService) buildShadowsocksLink(inbound *shareInbound, address string) (string, error) {
	if inbound.settings.Shadowsocks == nil {
		return "", common.NewError("shadowsocks settings are empty")
	}
	if sni := inbound.tlsServerName(); sni != "" {
		address = sni
	}
	if !ssservice.IsShadowsocks2022(inbound.settings.Shadowsocks.Method) {
		raw := fmt.Sprintf("%s:%s@%s:%d", inbound.settings.Shadowsocks.Method, inbound.settings.Shadowsocks.Password, address, inbound.Inbound.Port)
		return "ss://" + base64.RawURLEncoding.EncodeToString([]byte(raw)) + "#" + url.QueryEscape(inbound.Inbound.Remark), nil
	}
	return ssservice.BuildShareLink(&ssservice.ShareLink{
		Method:   inbound.settings.Shadowsocks.Method,
		Password: inbound.settings.Shadowsocks.Password,
		Host:     address,
		Port:     inbound.Inbound.Port,
		Name:     inbound.Inbound.Remark,
	})
}

func (s *ShareLinkService) buildClashProxy(inbound *shareInbound, ctx shareContext) (map[string]interface{}, error) {
	address, err := s.resolveAddress(inbound, ctx)
	if err != nil {
		return nil, err
	}
	proxy := map[string]interface{}{
		"name":   inbound.Inbound.Remark,
		"server": address,
		"port":   inbound.Inbound.Port,
	}
	switch inbound.Inbound.Protocol {
	case model.VMess:
		if inbound.settings.VMess == nil || len(inbound.settings.VMess.Clients) == 0 {
			return nil, common.NewError("vmess client is empty")
		}
		client := inbound.settings.VMess.Clients[0]
		proxy["type"] = "vmess"
		proxy["uuid"] = client.ID
		proxy["alterId"] = client.AlterID
		proxy["cipher"] = "auto"
	case model.VLESS:
		if inbound.settings.VLESS == nil || len(inbound.settings.VLESS.Clients) == 0 {
			return nil, common.NewError("vless client is empty")
		}
		client := inbound.settings.VLESS.Clients[0]
		proxy["type"] = "vless"
		proxy["uuid"] = client.ID
		proxy["cipher"] = defaultString(inbound.settings.VLESS.Decryption, "none")
		if strings.TrimSpace(client.Flow) != "" {
			proxy["flow"] = client.Flow
		}
	case model.Trojan:
		if inbound.settings.Trojan == nil || len(inbound.settings.Trojan.Clients) == 0 {
			return nil, common.NewError("trojan client is empty")
		}
		proxy["type"] = "trojan"
		proxy["password"] = inbound.settings.Trojan.Clients[0].Password
	case model.Shadowsocks:
		if inbound.settings.Shadowsocks == nil {
			return nil, common.NewError("shadowsocks settings are empty")
		}
		proxy["type"] = "ss"
		proxy["cipher"] = inbound.settings.Shadowsocks.Method
		proxy["password"] = inbound.settings.Shadowsocks.Password
		if ssservice.IsShadowsocks2022(inbound.settings.Shadowsocks.Method) {
			network := normalizeShadowsocksNetwork(inbound.settings.Shadowsocks.Network)
			proxy["udp"] = network == "udp" || network == "tcp,udp"
		}
	default:
		return nil, common.NewError("unsupported inbound protocol:", inbound.Inbound.Protocol)
	}

	s.applyNetworkToClashProxy(proxy, inbound)
	return proxy, nil
}

func (s *ShareLinkService) applyNetworkToClashProxy(proxy map[string]interface{}, inbound *shareInbound) {
	security := inbound.stream.Security
	network := inbound.stream.Network
	if security == "tls" {
		proxy["tls"] = true
		if sni := inbound.tlsServerName(); sni != "" {
			proxy["servername"] = sni
			proxy["sni"] = sni
			proxy["server"] = sni
		}
	}
	if security == "reality" && inbound.stream.RealitySettings != nil {
		reality := inbound.stream.RealitySettings
		proxy["tls"] = true
		proxy["servername"] = firstNonEmpty(reality.ServerNames, fmt.Sprint(proxy["server"]))
		proxy["client-fingerprint"] = defaultString(reality.Fingerprint, "chrome")
		proxy["reality-opts"] = map[string]interface{}{
			"public-key": reality.PublicKey,
			"short-id":   firstNonEmpty(reality.ShortIds, ""),
		}
	}
	switch network {
	case "ws":
		proxy["network"] = "ws"
		if inbound.stream.WSSettings != nil {
			if inbound.stream.WSSettings.Path != "" {
				proxy["ws-opts"] = map[string]interface{}{
					"path":    inbound.stream.WSSettings.Path,
					"headers": inbound.stream.WSSettings.Headers,
				}
			}
		}
	case "grpc":
		proxy["network"] = "grpc"
		if inbound.stream.GRPCSettings != nil {
			proxy["grpc-opts"] = map[string]interface{}{
				"grpc-service-name": inbound.stream.GRPCSettings.ServiceName,
			}
		}
	case "http":
		proxy["network"] = "h2"
		if inbound.stream.HTTPSettings != nil {
			proxy["h2-opts"] = map[string]interface{}{
				"host": inbound.stream.HTTPSettings.Host,
				"path": inbound.stream.HTTPSettings.Path,
			}
		}
	case "tcp":
		if inbound.stream.TCPSettings != nil && inbound.stream.TCPSettings.Header.Type == "http" && inbound.stream.TCPSettings.Header.Request != nil {
			proxy["network"] = "http"
			proxy["http-opts"] = map[string]interface{}{
				"path": inbound.stream.TCPSettings.Header.Request.Path,
				"headers": map[string]interface{}{
					"Host": firstHeaderValue(inbound.stream.TCPSettings.Header.Request.Headers, "Host"),
				},
			}
		}
	}
}

func (s *ShareLinkService) resolveAddress(inbound *shareInbound, ctx shareContext) (string, error) {
	candidates := make([]string, 0, 4)
	if listen := strings.TrimSpace(inbound.Inbound.Listen); listen != "" {
		candidates = append(candidates, listen)
	}
	addresses, err := s.getShareAddresses()
	if err != nil {
		return "", err
	}
	sort.SliceStable(addresses, func(i, j int) bool {
		return addresses[i].CreatedAt.Before(addresses[j].CreatedAt)
	})
	for _, item := range addresses {
		if !item.Enabled {
			continue
		}
		candidates = append(candidates, item.Address)
	}
	if host := extractHost(ctx.RequestHost); host != "" {
		candidates = append(candidates, host)
	}

	for _, candidate := range candidates {
		normalized, err := normalizeShareAddress(candidate)
		if err != nil {
			continue
		}
		if isPublicShareAddress(normalized) {
			return normalized, nil
		}
	}
	return "", common.NewError("no public share address configured")
}

func (inbound *shareInbound) tlsServerName() string {
	switch inbound.stream.Security {
	case "tls":
		if inbound.stream.TLSSettings != nil {
			return strings.TrimSpace(inbound.stream.TLSSettings.ServerName)
		}
	case "xtls":
		if inbound.stream.XTLSSettings != nil {
			return strings.TrimSpace(inbound.stream.XTLSSettings.ServerName)
		}
	}
	return ""
}

func extractHost(hostport string) string {
	hostport = strings.TrimSpace(hostport)
	if hostport == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(hostport); err == nil {
		return strings.Trim(host, "[]")
	}
	return strings.Trim(hostport, "[]")
}

func isPublicShareAddress(address string) bool {
	lower := strings.ToLower(strings.TrimSpace(address))
	switch lower {
	case "", "localhost", "0.0.0.0", "::", "::1":
		return false
	}
	ip := net.ParseIP(address)
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsUnspecified() || isPrivateIP(ip) {
		return false
	}
	if ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() {
		return false
	}
	return true
}

func firstNonEmpty(values []string, fallback string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return fallback
}

func defaultString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func bracketIPv6(host string) string {
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		return "[" + host + "]"
	}
	return host
}

func renderClashConfig(cfg clashConfig) string {
	builder := &strings.Builder{}
	builder.WriteString("proxies:\n")
	for _, proxy := range cfg.Proxies {
		builder.WriteString("  -")
		writeYAMLMap(builder, proxy, 4, true)
	}
	builder.WriteString("proxy-groups:\n")
	for _, group := range cfg.ProxyGroups {
		builder.WriteString("  -")
		writeYAMLMap(builder, group, 4, true)
	}
	builder.WriteString("rules:\n")
	for _, rule := range cfg.Rules {
		builder.WriteString("  - ")
		builder.WriteString(quoteYAMLString(rule))
		builder.WriteString("\n")
	}
	return builder.String()
}

func writeYAMLMap(builder *strings.Builder, value map[string]interface{}, indent int, inlineFirst bool) {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for i, key := range keys {
		val := value[key]
		if i == 0 && inlineFirst {
			builder.WriteString(" ")
		} else {
			builder.WriteString(strings.Repeat(" ", indent))
		}
		builder.WriteString(key)
		switch typed := val.(type) {
		case map[string]interface{}:
			builder.WriteString(":\n")
			writeYAMLMap(builder, typed, indent+2, false)
		case []string:
			builder.WriteString(":\n")
			for _, item := range typed {
				builder.WriteString(strings.Repeat(" ", indent+2))
				builder.WriteString("- ")
				builder.WriteString(quoteYAMLString(item))
				builder.WriteString("\n")
			}
		case []interface{}:
			builder.WriteString(":\n")
			for _, item := range typed {
				builder.WriteString(strings.Repeat(" ", indent+2))
				builder.WriteString("- ")
				writeYAMLScalarOrInlineMap(builder, item, indent+4)
			}
		default:
			builder.WriteString(": ")
			writeYAMLScalar(builder, typed)
			builder.WriteString("\n")
		}
	}
}

func writeYAMLScalarOrInlineMap(builder *strings.Builder, value interface{}, indent int) {
	switch typed := value.(type) {
	case map[string]interface{}:
		builder.WriteString("\n")
		writeYAMLMap(builder, typed, indent, false)
	default:
		writeYAMLScalar(builder, typed)
		builder.WriteString("\n")
	}
}

func writeYAMLScalar(builder *strings.Builder, value interface{}) {
	switch typed := value.(type) {
	case string:
		builder.WriteString(quoteYAMLString(typed))
	case bool:
		if typed {
			builder.WriteString("true")
		} else {
			builder.WriteString("false")
		}
	case int:
		builder.WriteString(strconv.Itoa(typed))
	case int64:
		builder.WriteString(strconv.FormatInt(typed, 10))
	case nil:
		builder.WriteString("''")
	default:
		builder.WriteString(quoteYAMLString(fmt.Sprint(typed)))
	}
}

func quoteYAMLString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func firstHeaderValue(headers map[string]interface{}, key string) string {
	if headers == nil {
		return ""
	}
	for name, value := range headers {
		if !strings.EqualFold(name, key) {
			continue
		}
		switch v := value.(type) {
		case string:
			return strings.TrimSpace(v)
		case []interface{}:
			for _, item := range v {
				if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
					return strings.TrimSpace(text)
				}
			}
		case []string:
			for _, item := range v {
				if strings.TrimSpace(item) != "" {
					return strings.TrimSpace(item)
				}
			}
		}
	}
	return ""
}

func isPrivateIP(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		switch {
		case ip4[0] == 10:
			return true
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return true
		case ip4[0] == 192 && ip4[1] == 168:
			return true
		case ip4[0] == 169 && ip4[1] == 254:
			return true
		default:
			return false
		}
	}
	if len(ip) == net.IPv6len {
		return (ip[0] & 0xfe) == 0xfc
	}
	return false
}

func (s *ShareLinkService) getShareAddresses() ([]ShareAddress, error) {
	if s.loadShareAddresses != nil {
		return s.loadShareAddresses()
	}
	return s.shareAddressService.GetAll()
}
