package simple

import (
	"bytes"
	"encoding/json"
	"github.com/torr9522/n6-ui/util/common"
	ssservice "github.com/torr9522/n6-ui/web/service/shadowsocks"
	"strings"
)

const simpleAdvancedConfigMessage = "该出口包含高级配置，请使用高级出口编辑"

type decodedSimpleEgress struct {
	Protocol          string
	Address           string
	Port              int
	Username          string
	Method            string
	Password          string
	UUID              string
	AlterID           int
	Network           string
	Security          string
	SNI               string
	Host              string
	Path              string
	ServiceName       string
	Flow              string
	Fingerprint       string
	PublicKey         string
	ShortID           string
	SpiderX           string
	Supported         bool
	UnsupportedReason string
}

type outboundEnvelope struct {
	Protocol       string          `json:"protocol"`
	Settings       json.RawMessage `json:"settings"`
	StreamSettings json.RawMessage `json:"streamSettings"`
}

type socksOutboundSettings struct {
	Servers []socksOutboundServer `json:"servers"`
}

type socksOutboundServer struct {
	Address string              `json:"address"`
	Port    int                 `json:"port"`
	Users   []socksOutboundUser `json:"users,omitempty"`
}

type socksOutboundUser struct {
	User string `json:"user"`
	Pass string `json:"pass"`
}

type shadowsocksOutboundSettings struct {
	Servers []shadowsocksOutboundServer `json:"servers"`
}

type shadowsocksOutboundServer struct {
	Address  string `json:"address"`
	Port     int    `json:"port"`
	Method   string `json:"method"`
	Password string `json:"password"`
}

type vmessOutboundSettings struct {
	Vnext []vmessOutboundTarget `json:"vnext"`
}

type vmessOutboundTarget struct {
	Address string              `json:"address"`
	Port    int                 `json:"port"`
	Users   []vmessOutboundUser `json:"users"`
}

type vmessOutboundUser struct {
	ID       string `json:"id"`
	AlterID  int    `json:"alterId,omitempty"`
	Security string `json:"security,omitempty"`
}

type vlessOutboundSettings struct {
	Vnext []vlessOutboundTarget `json:"vnext"`
}

type vlessOutboundTarget struct {
	Address string              `json:"address"`
	Port    int                 `json:"port"`
	Users   []vlessOutboundUser `json:"users"`
}

type vlessOutboundUser struct {
	ID         string `json:"id"`
	Encryption string `json:"encryption"`
	Flow       string `json:"flow,omitempty"`
}

type simpleStreamSettings struct {
	Network         string                 `json:"network,omitempty"`
	Security        string                 `json:"security,omitempty"`
	TLSSettings     *simpleTLSSettings     `json:"tlsSettings,omitempty"`
	RealitySettings *simpleRealitySettings `json:"realitySettings,omitempty"`
	TCPSettings     *simpleTCPSettings     `json:"tcpSettings,omitempty"`
	WSSettings      *simpleWSSettings      `json:"wsSettings,omitempty"`
	GRPCSettings    *simpleGRPCSettings    `json:"grpcSettings,omitempty"`
}

type simpleTLSSettings struct {
	ServerName string `json:"serverName,omitempty"`
}

type simpleRealitySettings struct {
	ServerName  string `json:"serverName"`
	Fingerprint string `json:"fingerprint,omitempty"`
	PublicKey   string `json:"publicKey"`
	ShortID     string `json:"shortId,omitempty"`
	SpiderX     string `json:"spiderX,omitempty"`
}

type simpleTCPSettings struct {
	Header *simpleTCPHeader `json:"header,omitempty"`
}

type simpleTCPHeader struct {
	Type string `json:"type,omitempty"`
}

type simpleWSSettings struct {
	Path    string                 `json:"path,omitempty"`
	Headers map[string]interface{} `json:"headers,omitempty"`
}

type simpleGRPCSettings struct {
	ServiceName string `json:"serviceName,omitempty"`
}

func decodeSimpleOutbound(protocol string, outboundJSON string) *decodedSimpleEgress {
	decoded := &decodedSimpleEgress{
		Protocol:  toSimpleDisplayProtocol(protocol),
		Network:   "tcp",
		Security:  "none",
		Supported: true,
	}

	var topLevel map[string]json.RawMessage
	if err := json.Unmarshal([]byte(outboundJSON), &topLevel); err != nil {
		return markUnsupported(decoded, "出口配置无效")
	}
	if reason := unsupportedExtraKeys(topLevel, "protocol", "settings", "streamSettings", "tag"); reason != "" {
		return markUnsupported(decoded, reason)
	}

	var envelope outboundEnvelope
	if err := json.Unmarshal([]byte(outboundJSON), &envelope); err != nil {
		return markUnsupported(decoded, "出口配置无效")
	}

	rawProtocol := firstNonEmptyString(envelope.Protocol, protocol)
	if rawProtocol == "" {
		return markUnsupported(decoded, "出口协议缺失")
	}
	normalizedProtocol := normalizeSimpleProtocol(rawProtocol)
	decoded.Protocol = toSimpleDisplayProtocol(normalizedProtocol)

	switch normalizedProtocol {
	case "socks":
		return decodeSocksOutbound(decoded, envelope.Settings)
	case "shadowsocks":
		return decodeShadowsocksOutbound(decoded, envelope.Settings)
	case "vmess":
		return decodeVMessOutbound(decoded, envelope.Settings, envelope.StreamSettings)
	case "vless":
		return decodeVLESSOutbound(decoded, envelope.Settings, envelope.StreamSettings)
	default:
		decoded.Address, decoded.Port = extractGenericAddressPort(topLevel["settings"])
		return markUnsupported(decoded, "当前协议不支持 Simple 编辑")
	}
}

func buildSimpleOutbound(protocol string, req *CreateSimpleEgressRequest, address string) (map[string]interface{}, error) {
	switch protocol {
	case "socks":
		server := socksOutboundServer{
			Address: address,
			Port:    req.Port,
		}
		if strings.TrimSpace(req.Username) != "" || strings.TrimSpace(req.Password) != "" {
			server.Users = []socksOutboundUser{{
				User: strings.TrimSpace(req.Username),
				Pass: strings.TrimSpace(req.Password),
			}}
		}
		return marshalOutboundMap(outboundEnvelope{
			Protocol: "socks",
			Settings: mustMarshalJSON(socksOutboundSettings{
				Servers: []socksOutboundServer{server},
			}),
		})
	case "shadowsocks":
		method := strings.TrimSpace(strings.ToLower(req.Method))
		if method == "" {
			return nil, common.NewError("method is required")
		}
		password := strings.TrimSpace(req.Password)
		if err := ssservice.ValidateCredential(method, password); err != nil {
			return nil, err
		}
		return marshalOutboundMap(outboundEnvelope{
			Protocol: "shadowsocks",
			Settings: mustMarshalJSON(shadowsocksOutboundSettings{
				Servers: []shadowsocksOutboundServer{{
					Address:  address,
					Port:     req.Port,
					Method:   method,
					Password: password,
				}},
			}),
		})
	case "vmess":
		uuid := strings.TrimSpace(req.UUID)
		if uuid == "" {
			return nil, common.NewError("uuid is required")
		}
		stream, err := buildSimpleStreamSettings("vmess", req)
		if err != nil {
			return nil, err
		}
		return marshalOutboundMap(outboundEnvelope{
			Protocol: "vmess",
			Settings: mustMarshalJSON(vmessOutboundSettings{
				Vnext: []vmessOutboundTarget{{
					Address: address,
					Port:    req.Port,
					Users: []vmessOutboundUser{{
						ID:       uuid,
						AlterID:  req.AlterID,
						Security: "auto",
					}},
				}},
			}),
			StreamSettings: stream,
		})
	case "vless":
		uuid := strings.TrimSpace(req.UUID)
		if uuid == "" {
			return nil, common.NewError("uuid is required")
		}
		stream, err := buildSimpleStreamSettings("vless", req)
		if err != nil {
			return nil, err
		}
		flow := strings.TrimSpace(req.Flow)
		encryption := strings.TrimSpace(strings.ToLower(req.Encryption))
		if encryption == "" {
			encryption = "none"
		}
		if encryption != "none" {
			return nil, common.NewError("vless encryption must be none")
		}
		return marshalOutboundMap(outboundEnvelope{
			Protocol: "vless",
			Settings: mustMarshalJSON(vlessOutboundSettings{
				Vnext: []vlessOutboundTarget{{
					Address: address,
					Port:    req.Port,
					Users: []vlessOutboundUser{{
						ID:         uuid,
						Encryption: encryption,
						Flow:       flow,
					}},
				}},
			}),
			StreamSettings: stream,
		})
	default:
		return nil, common.NewError("simple mode only supports socks5, shadowsocks, vmess and vless")
	}
}

func decodeSocksOutbound(decoded *decodedSimpleEgress, settingsRaw json.RawMessage) *decodedSimpleEgress {
	settingsObj, err := parseJSONRawObject(settingsRaw)
	if err != nil {
		return markUnsupported(decoded, "SOCKS 配置无效")
	}
	if reason := unsupportedExtraKeys(settingsObj, "servers"); reason != "" {
		return markUnsupported(decoded, reason)
	}
	settings := socksOutboundSettings{}
	if err := json.Unmarshal(settingsRaw, &settings); err != nil {
		return markUnsupported(decoded, "SOCKS 配置无效")
	}
	if len(settings.Servers) != 1 {
		return markUnsupported(decoded, "SOCKS Simple 仅支持单个 server")
	}
	server := settings.Servers[0]
	decoded.Address = strings.TrimSpace(server.Address)
	decoded.Port = server.Port
	if len(server.Users) > 1 {
		return markUnsupported(decoded, "SOCKS Simple 仅支持单个认证用户")
	}
	if len(server.Users) == 1 {
		decoded.Username = strings.TrimSpace(server.Users[0].User)
		decoded.Password = strings.TrimSpace(server.Users[0].Pass)
	}
	return decoded
}

func decodeShadowsocksOutbound(decoded *decodedSimpleEgress, settingsRaw json.RawMessage) *decodedSimpleEgress {
	settingsObj, err := parseJSONRawObject(settingsRaw)
	if err != nil {
		return markUnsupported(decoded, "Shadowsocks 配置无效")
	}
	if reason := unsupportedExtraKeys(settingsObj, "servers"); reason != "" {
		return markUnsupported(decoded, reason)
	}
	settings := shadowsocksOutboundSettings{}
	if err := json.Unmarshal(settingsRaw, &settings); err != nil {
		return markUnsupported(decoded, "Shadowsocks 配置无效")
	}
	if len(settings.Servers) != 1 {
		return markUnsupported(decoded, "Shadowsocks Simple 仅支持单个 server")
	}
	server := settings.Servers[0]
	decoded.Address = strings.TrimSpace(server.Address)
	decoded.Port = server.Port
	decoded.Method = strings.TrimSpace(strings.ToLower(server.Method))
	decoded.Password = strings.TrimSpace(server.Password)
	if err := ssservice.ValidateCredential(decoded.Method, decoded.Password); err != nil {
		return markUnsupported(decoded, err.Error())
	}
	return decoded
}

func decodeVMessOutbound(decoded *decodedSimpleEgress, settingsRaw json.RawMessage, streamRaw json.RawMessage) *decodedSimpleEgress {
	settingsObj, err := parseJSONRawObject(settingsRaw)
	if err != nil {
		return markUnsupported(decoded, "VMess 配置无效")
	}
	if reason := unsupportedExtraKeys(settingsObj, "vnext"); reason != "" {
		return markUnsupported(decoded, reason)
	}
	settings := vmessOutboundSettings{}
	if err := json.Unmarshal(settingsRaw, &settings); err != nil {
		return markUnsupported(decoded, "VMess 配置无效")
	}
	targetObj, userObj, reason := extractTargetAndUserObjects(settingsObj["vnext"])
	if reason != "" {
		return markUnsupported(decoded, reason)
	}
	if reason := unsupportedExtraKeys(targetObj, "address", "port", "users"); reason != "" {
		return markUnsupported(decoded, reason)
	}
	if reason := unsupportedExtraKeys(userObj, "id", "alterId", "security"); reason != "" {
		return markUnsupported(decoded, reason)
	}
	if len(settings.Vnext) != 1 {
		return markUnsupported(decoded, "VMess Simple 仅支持单个 vnext")
	}
	target := settings.Vnext[0]
	decoded.Address = strings.TrimSpace(target.Address)
	decoded.Port = target.Port
	if len(target.Users) != 1 {
		return markUnsupported(decoded, "VMess Simple 仅支持单个用户")
	}
	user := target.Users[0]
	decoded.UUID = strings.TrimSpace(user.ID)
	decoded.AlterID = user.AlterID
	security := strings.TrimSpace(strings.ToLower(user.Security))
	if security != "" && security != "auto" {
		return markUnsupported(decoded, "VMess Simple 仅支持 auto security")
	}
	return decodeStreamSettings(decoded, "vmess", streamRaw)
}

func decodeVLESSOutbound(decoded *decodedSimpleEgress, settingsRaw json.RawMessage, streamRaw json.RawMessage) *decodedSimpleEgress {
	settingsObj, err := parseJSONRawObject(settingsRaw)
	if err != nil {
		return markUnsupported(decoded, "VLESS 配置无效")
	}
	if reason := unsupportedExtraKeys(settingsObj, "vnext"); reason != "" {
		return markUnsupported(decoded, reason)
	}
	settings := vlessOutboundSettings{}
	if err := json.Unmarshal(settingsRaw, &settings); err != nil {
		return markUnsupported(decoded, "VLESS 配置无效")
	}
	targetObj, userObj, reason := extractTargetAndUserObjects(settingsObj["vnext"])
	if reason != "" {
		return markUnsupported(decoded, reason)
	}
	if reason := unsupportedExtraKeys(targetObj, "address", "port", "users"); reason != "" {
		return markUnsupported(decoded, reason)
	}
	if reason := unsupportedExtraKeys(userObj, "id", "encryption", "flow"); reason != "" {
		return markUnsupported(decoded, reason)
	}
	if len(settings.Vnext) != 1 {
		return markUnsupported(decoded, "VLESS Simple 仅支持单个 vnext")
	}
	target := settings.Vnext[0]
	decoded.Address = strings.TrimSpace(target.Address)
	decoded.Port = target.Port
	if len(target.Users) != 1 {
		return markUnsupported(decoded, "VLESS Simple 仅支持单个用户")
	}
	user := target.Users[0]
	decoded.UUID = strings.TrimSpace(user.ID)
	decoded.Flow = strings.TrimSpace(user.Flow)
	encryption := strings.TrimSpace(strings.ToLower(user.Encryption))
	if encryption != "" && encryption != "none" {
		return markUnsupported(decoded, "VLESS Simple 仅支持 encryption=none")
	}
	return decodeStreamSettings(decoded, "vless", streamRaw)
}

func decodeStreamSettings(decoded *decodedSimpleEgress, protocol string, streamRaw json.RawMessage) *decodedSimpleEgress {
	if len(trimJSONRaw(streamRaw)) == 0 || bytes.Equal(trimJSONRaw(streamRaw), []byte("null")) {
		decoded.Network = "tcp"
		decoded.Security = "none"
		return decoded
	}

	streamObj, err := parseJSONRawObject(streamRaw)
	if err != nil {
		return markUnsupported(decoded, protocolDisplayName(protocol)+" streamSettings 无效")
	}
	if reason := unsupportedExtraKeys(streamObj, "network", "security", "tlsSettings", "realitySettings", "tcpSettings", "wsSettings", "grpcSettings"); reason != "" {
		return markUnsupported(decoded, reason)
	}

	stream := simpleStreamSettings{}
	if err := json.Unmarshal(streamRaw, &stream); err != nil {
		return markUnsupported(decoded, protocolDisplayName(protocol)+" streamSettings 无效")
	}
	network := strings.TrimSpace(strings.ToLower(stream.Network))
	if network == "" {
		network = "tcp"
	}
	switch network {
	case "tcp", "ws", "grpc":
	default:
		return markUnsupported(decoded, "当前 n6-ui Simple 出口暂不支持该传输方式")
	}

	security := strings.TrimSpace(strings.ToLower(stream.Security))
	if security == "" {
		security = "none"
	}
	switch protocol {
	case "vmess":
		if security != "none" && security != "tls" {
			return markUnsupported(decoded, "当前 n6-ui Simple VMess 仅支持 none 或 tls")
		}
	case "vless":
		if security != "none" && security != "tls" && security != "reality" {
			return markUnsupported(decoded, "当前 n6-ui Simple VLESS 仅支持 none、tls 或 reality")
		}
	}

	decoded.Network = network
	decoded.Security = security

	if network == "tcp" {
		reason := decodeTCPSettings(decoded, streamObj["tcpSettings"])
		if reason != "" {
			return markUnsupported(decoded, reason)
		}
	}
	if network == "ws" {
		reason := decodeWSSettings(decoded, streamObj["wsSettings"])
		if reason != "" {
			return markUnsupported(decoded, reason)
		}
	}
	if network == "grpc" {
		reason := decodeGRPCSettings(decoded, streamObj["grpcSettings"])
		if reason != "" {
			return markUnsupported(decoded, reason)
		}
	}

	if security == "tls" {
		reason := decodeTLSSettings(decoded, streamObj["tlsSettings"])
		if reason != "" {
			return markUnsupported(decoded, reason)
		}
	}
	if security == "reality" {
		if protocol != "vless" {
			return markUnsupported(decoded, "Reality 仅支持 VLESS")
		}
		if network != "tcp" && network != "grpc" {
			return markUnsupported(decoded, "Reality 仅支持 tcp 或 grpc")
		}
		reason := decodeRealitySettings(decoded, streamObj["realitySettings"])
		if reason != "" {
			return markUnsupported(decoded, reason)
		}
	}

	return decoded
}

func decodeTCPSettings(decoded *decodedSimpleEgress, raw json.RawMessage) string {
	if len(trimJSONRaw(raw)) == 0 || bytes.Equal(trimJSONRaw(raw), []byte("null")) {
		return ""
	}
	obj, err := parseJSONRawObject(raw)
	if err != nil {
		return "tcpSettings 无效"
	}
	if reason := unsupportedExtraKeys(obj, "header"); reason != "" {
		return reason
	}
	headerRaw := obj["header"]
	if len(trimJSONRaw(headerRaw)) == 0 || bytes.Equal(trimJSONRaw(headerRaw), []byte("null")) {
		return ""
	}
	headerObj, err := parseJSONRawObject(headerRaw)
	if err != nil {
		return "tcp header 无效"
	}
	if reason := unsupportedExtraKeys(headerObj, "type"); reason != "" {
		return reason
	}
	header := simpleTCPHeader{}
	if err := json.Unmarshal(headerRaw, &header); err != nil {
		return "tcp header 无效"
	}
	headerType := strings.TrimSpace(strings.ToLower(header.Type))
	if headerType == "" {
		headerType = "none"
	}
	if headerType != "none" {
		return "当前 n6-ui Simple 出口暂不支持该传输方式"
	}
	return ""
}

func decodeWSSettings(decoded *decodedSimpleEgress, raw json.RawMessage) string {
	if len(trimJSONRaw(raw)) == 0 || bytes.Equal(trimJSONRaw(raw), []byte("null")) {
		decoded.Path = "/"
		return ""
	}
	obj, err := parseJSONRawObject(raw)
	if err != nil {
		return "wsSettings 无效"
	}
	if reason := unsupportedExtraKeys(obj, "path", "headers"); reason != "" {
		return reason
	}
	settings := simpleWSSettings{}
	if err := json.Unmarshal(raw, &settings); err != nil {
		return "wsSettings 无效"
	}
	decoded.Path = strings.TrimSpace(settings.Path)
	if decoded.Path == "" {
		decoded.Path = "/"
	}
	for key, value := range settings.Headers {
		if !strings.EqualFold(strings.TrimSpace(key), "host") {
			return "当前配置包含额外 WS Header，请使用高级出口编辑"
		}
		decoded.Host = strings.TrimSpace(toStringValue(value))
	}
	return ""
}

func decodeGRPCSettings(decoded *decodedSimpleEgress, raw json.RawMessage) string {
	if len(trimJSONRaw(raw)) == 0 || bytes.Equal(trimJSONRaw(raw), []byte("null")) {
		return ""
	}
	obj, err := parseJSONRawObject(raw)
	if err != nil {
		return "grpcSettings 无效"
	}
	if reason := unsupportedExtraKeys(obj, "serviceName"); reason != "" {
		return reason
	}
	settings := simpleGRPCSettings{}
	if err := json.Unmarshal(raw, &settings); err != nil {
		return "grpcSettings 无效"
	}
	decoded.ServiceName = strings.TrimSpace(settings.ServiceName)
	return ""
}

func decodeTLSSettings(decoded *decodedSimpleEgress, raw json.RawMessage) string {
	if len(trimJSONRaw(raw)) == 0 || bytes.Equal(trimJSONRaw(raw), []byte("null")) {
		return ""
	}
	obj, err := parseJSONRawObject(raw)
	if err != nil {
		return "tlsSettings 无效"
	}
	if reason := unsupportedExtraKeys(obj, "serverName"); reason != "" {
		return reason
	}
	settings := simpleTLSSettings{}
	if err := json.Unmarshal(raw, &settings); err != nil {
		return "tlsSettings 无效"
	}
	decoded.SNI = strings.TrimSpace(settings.ServerName)
	return ""
}

func decodeRealitySettings(decoded *decodedSimpleEgress, raw json.RawMessage) string {
	obj, err := parseJSONRawObject(raw)
	if err != nil {
		return "realitySettings 无效"
	}
	if reason := unsupportedExtraKeys(obj, "serverName", "fingerprint", "publicKey", "shortId", "spiderX"); reason != "" {
		return reason
	}
	settings := simpleRealitySettings{}
	if err := json.Unmarshal(raw, &settings); err != nil {
		return "realitySettings 无效"
	}
	decoded.SNI = strings.TrimSpace(settings.ServerName)
	decoded.Fingerprint = strings.TrimSpace(settings.Fingerprint)
	decoded.PublicKey = strings.TrimSpace(settings.PublicKey)
	decoded.ShortID = strings.TrimSpace(settings.ShortID)
	decoded.SpiderX = strings.TrimSpace(settings.SpiderX)
	if decoded.SNI == "" || decoded.PublicKey == "" {
		return "Reality 配置缺少必要字段"
	}
	return ""
}

func buildSimpleStreamSettings(protocol string, req *CreateSimpleEgressRequest) (json.RawMessage, error) {
	network := strings.TrimSpace(strings.ToLower(req.Network))
	if network == "" {
		network = "tcp"
	}
	switch network {
	case "tcp", "ws", "grpc":
	default:
		return nil, common.NewError("当前 n6-ui Simple 出口暂不支持该传输方式")
	}

	security := strings.TrimSpace(strings.ToLower(req.Security))
	if security == "" {
		security = "none"
	}
	switch protocol {
	case "vmess":
		if security != "none" && security != "tls" {
			return nil, common.NewError("vmess only supports none or tls")
		}
	case "vless":
		if security != "none" && security != "tls" && security != "reality" {
			return nil, common.NewError("vless only supports none, tls or reality")
		}
		if security == "reality" && network != "tcp" && network != "grpc" {
			return nil, common.NewError("Reality only supports tcp or grpc")
		}
	}

	stream := simpleStreamSettings{
		Network:  network,
		Security: security,
	}

	if network == "tcp" {
		stream.TCPSettings = &simpleTCPSettings{
			Header: &simpleTCPHeader{Type: "none"},
		}
	}
	if network == "ws" {
		path := strings.TrimSpace(req.Path)
		if path == "" {
			path = "/"
		}
		ws := &simpleWSSettings{Path: path}
		if host := strings.TrimSpace(req.Host); host != "" {
			ws.Headers = map[string]interface{}{"Host": host}
		}
		stream.WSSettings = ws
	}
	if network == "grpc" {
		stream.GRPCSettings = &simpleGRPCSettings{
			ServiceName: strings.TrimSpace(req.ServiceName),
		}
	}
	if security == "tls" {
		stream.TLSSettings = &simpleTLSSettings{
			ServerName: strings.TrimSpace(req.SNI),
		}
	}
	if security == "reality" {
		fingerprint := strings.TrimSpace(req.Fingerprint)
		if fingerprint == "" {
			fingerprint = "chrome"
		}
		spiderX := strings.TrimSpace(req.SpiderX)
		if spiderX == "" {
			spiderX = "/"
		}
		if !strings.HasPrefix(spiderX, "/") {
			return nil, common.NewError("spiderX must start with /")
		}
		sni := strings.TrimSpace(req.SNI)
		publicKey := strings.TrimSpace(req.PublicKey)
		if sni == "" || publicKey == "" {
			return nil, common.NewError("reality requires sni and public key")
		}
		stream.RealitySettings = &simpleRealitySettings{
			ServerName:  sni,
			Fingerprint: fingerprint,
			PublicKey:   publicKey,
			ShortID:     strings.TrimSpace(req.ShortID),
			SpiderX:     spiderX,
		}
	}

	data, err := json.Marshal(stream)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func marshalOutboundMap(envelope outboundEnvelope) (map[string]interface{}, error) {
	data, err := json.Marshal(envelope)
	if err != nil {
		return nil, err
	}
	result := make(map[string]interface{})
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func mustMarshalJSON(value interface{}) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

func parseJSONRawObject(raw json.RawMessage) (map[string]json.RawMessage, error) {
	trimmed := trimJSONRaw(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return map[string]json.RawMessage{}, nil
	}
	obj := make(map[string]json.RawMessage)
	if err := json.Unmarshal(trimmed, &obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func trimJSONRaw(raw json.RawMessage) []byte {
	return bytes.TrimSpace(raw)
}

func unsupportedExtraKeys(obj map[string]json.RawMessage, allowed ...string) string {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, key := range allowed {
		allowedSet[key] = struct{}{}
	}
	for key := range obj {
		if _, ok := allowedSet[key]; !ok {
			return simpleAdvancedConfigMessage
		}
	}
	return ""
}

func markUnsupported(decoded *decodedSimpleEgress, detail string) *decodedSimpleEgress {
	decoded.Supported = false
	if strings.TrimSpace(detail) == "" || detail == simpleAdvancedConfigMessage {
		decoded.UnsupportedReason = simpleAdvancedConfigMessage
		return decoded
	}
	decoded.UnsupportedReason = simpleAdvancedConfigMessage + "：" + detail
	return decoded
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func toStringValue(value interface{}) string {
	text, _ := value.(string)
	return text
}

func extractGenericAddressPort(settingsRaw json.RawMessage) (string, int) {
	settings := make(map[string]interface{})
	if err := json.Unmarshal(settingsRaw, &settings); err != nil {
		return "", 0
	}
	if address, port := extractAddressPortFromValue(settings["servers"]); address != "" || port > 0 {
		return address, port
	}
	if address, port := extractAddressPortFromValue(settings["vnext"]); address != "" || port > 0 {
		return address, port
	}
	if address, port := extractAddressPortFromValue(settings["peers"]); address != "" || port > 0 {
		return address, port
	}
	return strings.TrimSpace(toStringValue(settings["address"])), toIntValue(settings["port"])
}

func extractAddressPortFromValue(value interface{}) (string, int) {
	items, ok := value.([]interface{})
	if !ok || len(items) == 0 {
		return "", 0
	}
	first, ok := items[0].(map[string]interface{})
	if !ok {
		return "", 0
	}
	address := strings.TrimSpace(firstNonEmptyString(
		toStringValue(first["address"]),
		toStringValue(first["server"]),
		toStringValue(first["endpoint"]),
	))
	return address, toIntValue(first["port"])
}

func toIntValue(value interface{}) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func extractTargetAndUserObjects(raw json.RawMessage) (map[string]json.RawMessage, map[string]json.RawMessage, string) {
	targets, err := parseJSONRawArray(raw)
	if err != nil || len(targets) != 1 {
		return nil, nil, simpleAdvancedConfigMessage
	}
	targetObj, ok := targets[0].(map[string]interface{})
	if !ok {
		return nil, nil, simpleAdvancedConfigMessage
	}
	targetRaw, err := json.Marshal(targetObj)
	if err != nil {
		return nil, nil, simpleAdvancedConfigMessage
	}
	targetMap, err := parseJSONRawObject(targetRaw)
	if err != nil {
		return nil, nil, simpleAdvancedConfigMessage
	}
	users, err := parseJSONRawArray(targetMap["users"])
	if err != nil || len(users) != 1 {
		return nil, nil, simpleAdvancedConfigMessage
	}
	userObj, ok := users[0].(map[string]interface{})
	if !ok {
		return nil, nil, simpleAdvancedConfigMessage
	}
	userRaw, err := json.Marshal(userObj)
	if err != nil {
		return nil, nil, simpleAdvancedConfigMessage
	}
	userMap, err := parseJSONRawObject(userRaw)
	if err != nil {
		return nil, nil, simpleAdvancedConfigMessage
	}
	return targetMap, userMap, ""
}

func parseJSONRawArray(raw json.RawMessage) ([]interface{}, error) {
	trimmed := trimJSONRaw(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return []interface{}{}, nil
	}
	items := make([]interface{}, 0)
	if err := json.Unmarshal(trimmed, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func protocolDisplayName(protocol string) string {
	switch normalizeSimpleProtocol(protocol) {
	case "vmess":
		return "VMess"
	case "vless":
		return "VLESS"
	case "shadowsocks":
		return "Shadowsocks"
	case "socks":
		return "SOCKS5"
	default:
		return strings.TrimSpace(protocol)
	}
}
