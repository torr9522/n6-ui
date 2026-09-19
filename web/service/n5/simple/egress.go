package simple

import (
	"encoding/json"
	"strconv"
	"strings"
	n5model "x-ui/database/model/n5"
	"x-ui/util/common"
	n5service "x-ui/web/service/n5"
	ssservice "x-ui/web/service/shadowsocks"
)

type egressManager interface {
	List() ([]*n5model.Egress, error)
	Create(egress *n5model.Egress) (*n5model.Egress, error)
	Update(egress *n5model.Egress) (*n5model.Egress, error)
	Delete(id int) error
	Get(id int) (*n5model.Egress, error)
}

type egressTester interface {
	Test(egressId int) (*n5model.EgressTest, error)
}

type SimpleEgress struct {
	Id                int    `json:"id"`
	Name              string `json:"name"`
	Protocol          string `json:"protocol"`
	Address           string `json:"address"`
	Port              int    `json:"port"`
	Username          string `json:"username"`
	Method            string `json:"method"`
	Password          string `json:"password"`
	UUID              string `json:"uuid"`
	AlterID           int    `json:"alterId"`
	Network           string `json:"network"`
	Security          string `json:"security"`
	SNI               string `json:"sni"`
	Host              string `json:"host"`
	Path              string `json:"path"`
	ServiceName       string `json:"serviceName"`
	Flow              string `json:"flow"`
	Encryption        string `json:"encryption"`
	Fingerprint       string `json:"fingerprint"`
	PublicKey         string `json:"publicKey"`
	ShortID           string `json:"shortId"`
	SpiderX           string `json:"spiderX"`
	Status            string `json:"status"`
	ExitIP            string `json:"exitIp"`
	TestTime          int64  `json:"testTime"`
	LatencyMs         int    `json:"latencyMs"`
	LastError         string `json:"lastError"`
	Enabled           bool   `json:"enabled"`
	InternalTag       string `json:"internalTag"`
	DisplayLabel      string `json:"displayLabel"`
	Supported         bool   `json:"supported"`
	UnsupportedReason string `json:"unsupportedReason"`
}

type CreateSimpleEgressRequest struct {
	Name        string `json:"name" form:"name"`
	Protocol    string `json:"protocol" form:"protocol"`
	Address     string `json:"address" form:"address"`
	Port        int    `json:"port" form:"port"`
	Username    string `json:"username" form:"username"`
	Method      string `json:"method" form:"method"`
	Password    string `json:"password" form:"password"`
	UUID        string `json:"uuid" form:"uuid"`
	AlterID     int    `json:"alterId" form:"alterId"`
	Network     string `json:"network" form:"network"`
	Security    string `json:"security" form:"security"`
	SNI         string `json:"sni" form:"sni"`
	Host        string `json:"host" form:"host"`
	Path        string `json:"path" form:"path"`
	ServiceName string `json:"serviceName" form:"serviceName"`
	Flow        string `json:"flow" form:"flow"`
	Encryption  string `json:"encryption" form:"encryption"`
	Fingerprint string `json:"fingerprint" form:"fingerprint"`
	PublicKey   string `json:"publicKey" form:"publicKey"`
	ShortID     string `json:"shortId" form:"shortId"`
	SpiderX     string `json:"spiderX" form:"spiderX"`
	Enabled     bool   `json:"enabled" form:"enabled"`
}

type SimpleEgressTestResult struct {
	Id       int    `json:"id"`
	Status   string `json:"status"`
	Latency  int    `json:"latency"`
	ExitIP   string `json:"exitIp"`
	Message  string `json:"message"`
	TestedAt int64  `json:"testedAt"`
}

type EgressService struct {
	egressService egressManager
	testService   egressTester
}

func NewEgressService() *EgressService {
	return &EgressService{
		egressService: &n5service.EgressService{},
		testService:   &n5service.EgressTestService{},
	}
}

func (s *EgressService) getEgressService() egressManager {
	if s.egressService != nil {
		return s.egressService
	}
	return &n5service.EgressService{}
}

func (s *EgressService) getTestService() egressTester {
	if s.testService != nil {
		return s.testService
	}
	return &n5service.EgressTestService{}
}

func (s *EgressService) ListSimpleEgress() ([]*SimpleEgress, error) {
	records, err := s.getEgressService().List()
	if err != nil {
		return nil, err
	}

	items := make([]*SimpleEgress, 0, len(records))
	for _, record := range records {
		items = append(items, toSimpleEgress(record))
	}
	return items, nil
}

func (s *EgressService) GetSimpleEgress(id int) (*SimpleEgress, error) {
	record, err := s.getEgressService().Get(id)
	if err != nil {
		return nil, err
	}
	return toSimpleEgress(record), nil
}

func (s *EgressService) CreateSimpleEgress(req *CreateSimpleEgressRequest) (*SimpleEgress, error) {
	if req == nil {
		return nil, common.NewError("simple egress request is nil")
	}

	record, err := s.saveSimpleEgress(0, req)
	if err != nil {
		return nil, err
	}
	return toSimpleEgress(record), nil
}

func (s *EgressService) UpdateSimpleEgress(id int, req *CreateSimpleEgressRequest) (*SimpleEgress, error) {
	if id <= 0 {
		return nil, common.NewError("invalid simple egress id")
	}
	if req == nil {
		return nil, common.NewError("simple egress request is nil")
	}
	record, err := s.getEgressService().Get(id)
	if err != nil {
		return nil, err
	}
	decoded := decodeSimpleOutbound(record.Protocol, record.OutboundJSON)
	if !decoded.Supported {
		return nil, common.NewError(decoded.UnsupportedReason)
	}

	record, err = s.saveSimpleEgress(id, req)
	if err != nil {
		return nil, err
	}
	return toSimpleEgress(record), nil
}

func (s *EgressService) saveSimpleEgress(id int, req *CreateSimpleEgressRequest) (*n5model.Egress, error) {
	protocol := normalizeSimpleProtocol(req.Protocol)
	address := strings.TrimSpace(req.Address)
	if address == "" {
		return nil, common.NewError("address is required")
	}
	if req.Port <= 0 || req.Port > 65535 {
		return nil, common.NewError("port is invalid")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, common.NewError("name is required")
	}

	outbound, err := buildSimpleOutbound(protocol, req, address)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(outbound)
	if err != nil {
		return nil, err
	}

	if id > 0 {
		return s.getEgressService().Update(&n5model.Egress{
			Id:           id,
			Name:         name,
			Protocol:     protocol,
			Enabled:      req.Enabled,
			OutboundJSON: string(data),
		})
	}

	return s.getEgressService().Create(&n5model.Egress{
		Name:         name,
		Protocol:     protocol,
		Enabled:      req.Enabled,
		OutboundJSON: string(data),
	})
}

func (s *EgressService) DeleteSimpleEgress(id int) error {
	return s.getEgressService().Delete(id)
}

func (s *EgressService) TestSimpleEgress(id int) (*SimpleEgressTestResult, error) {
	record, err := s.getTestService().Test(id)
	if err != nil {
		return nil, err
	}
	return &SimpleEgressTestResult{
		Id:       record.Id,
		Status:   record.Status,
		Latency:  record.Latency,
		ExitIP:   record.ExitIP,
		Message:  record.Message,
		TestedAt: record.TestedAt,
	}, nil
}

func (s *EgressService) ParseShadowsocksShareLink(value string) (*CreateSimpleEgressRequest, error) {
	link, err := ssservice.ParseShareLink(value)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(link.Name)
	if name == "" {
		name = link.Host
	}
	return &CreateSimpleEgressRequest{
		Name:     name,
		Protocol: "ss",
		Address:  link.Host,
		Port:     link.Port,
		Method:   link.Method,
		Password: link.Password,
		Network:  "tcp",
		Security: "none",
		Enabled:  true,
	}, nil
}

func (s *EgressService) ExportShadowsocksShareLink(id int) (string, error) {
	if id <= 0 {
		return "", common.NewError("invalid simple egress id")
	}
	record, err := s.getEgressService().Get(id)
	if err != nil {
		return "", err
	}
	item := toSimpleEgress(record)
	if item == nil || item.Protocol != "ss" {
		return "", common.NewError("仅 Shadowsocks 出口支持 ss:// 导出")
	}
	if !item.Supported {
		return "", common.NewError(item.UnsupportedReason)
	}
	return ssservice.BuildShareLink(&ssservice.ShareLink{
		Method:   item.Method,
		Password: item.Password,
		Host:     item.Address,
		Port:     item.Port,
		Name:     item.Name,
	})
}

func normalizeSimpleProtocol(protocol string) string {
	protocol = strings.TrimSpace(strings.ToLower(protocol))
	switch protocol {
	case "", "socks", "socks5":
		return "socks"
	case "ss", "shadowsocks":
		return "shadowsocks"
	case "vmess":
		return "vmess"
	case "vless":
		return "vless"
	default:
		return protocol
	}
}

func toSimpleEgress(record *n5model.Egress) *SimpleEgress {
	if record == nil {
		return nil
	}
	decoded := decodeSimpleOutbound(record.Protocol, record.OutboundJSON)
	item := &SimpleEgress{
		Id:                record.Id,
		Name:              record.Name,
		Protocol:          decoded.Protocol,
		Address:           decoded.Address,
		Port:              decoded.Port,
		Username:          decoded.Username,
		Method:            decoded.Method,
		Password:          decoded.Password,
		UUID:              decoded.UUID,
		AlterID:           decoded.AlterID,
		Network:           decoded.Network,
		Security:          decoded.Security,
		SNI:               decoded.SNI,
		Host:              decoded.Host,
		Path:              decoded.Path,
		ServiceName:       decoded.ServiceName,
		Flow:              decoded.Flow,
		Encryption:        "none",
		Fingerprint:       decoded.Fingerprint,
		PublicKey:         decoded.PublicKey,
		ShortID:           decoded.ShortID,
		SpiderX:           decoded.SpiderX,
		Status:            record.LastStatus,
		ExitIP:            record.LastExitIP,
		TestTime:          record.LastTestTime,
		LatencyMs:         record.LastTestLatencyMs,
		LastError:         record.LastTestError,
		Enabled:           record.Enabled,
		InternalTag:       record.Tag,
		Supported:         decoded.Supported,
		UnsupportedReason: decoded.UnsupportedReason,
	}
	if item.Address != "" && item.Port > 0 {
		item.DisplayLabel = item.Address + ":" + strconv.Itoa(item.Port)
	} else if item.Address != "" {
		item.DisplayLabel = item.Address
	} else {
		item.DisplayLabel = "-"
	}
	return item
}

func extractAddressPort(outboundJSON string) (string, int) {
	decoded := decodeSimpleOutbound("", outboundJSON)
	return decoded.Address, decoded.Port
}

func toSimpleDisplayProtocol(protocol string) string {
	if strings.TrimSpace(protocol) == "" {
		return ""
	}
	switch normalizeSimpleProtocol(protocol) {
	case "socks":
		return "socks5"
	case "shadowsocks":
		return "ss"
	case "vmess":
		return "vmess"
	case "vless":
		return "vless"
	default:
		return strings.TrimSpace(strings.ToLower(protocol))
	}
}
