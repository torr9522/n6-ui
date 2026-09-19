package simple

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"x-ui/database"
	n5model "x-ui/database/model/n5"
	n5service "x-ui/web/service/n5"
	"x-ui/xray"
)

type fakeTester struct {
	record *n5model.EgressTest
	err    error
}

func (f *fakeTester) Test(egressId int) (*n5model.EgressTest, error) {
	if f.record != nil {
		f.record.EgressId = egressId
	}
	return f.record, f.err
}

func initSimpleTestDB(t *testing.T) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "simple.db")
	if err := database.InitDB(dbPath); err != nil {
		t.Fatalf("init db failed: %v", err)
	}
}

func assertOutboundTestConfig(t *testing.T, outboundJSON string) {
	t.Helper()
	if _, err := os.Stat(xray.GetBinaryPath()); err != nil {
		t.Skipf("xray binary not found: %v", err)
	}
	cfg := &xray.Config{
		InboundConfigs:  []xray.InboundConfig{},
		OutboundConfigs: []byte("[" + outboundJSON + "]"),
	}
	if err := xray.TestConfig(cfg); err != nil {
		t.Fatalf("xray test config failed: %v", err)
	}
}

func TestSimpleEgressServiceCreateSocksAndList(t *testing.T) {
	initSimpleTestDB(t)

	svc := &EgressService{
		egressService: &n5service.EgressService{},
		testService:   &fakeTester{},
	}

	created, err := svc.CreateSimpleEgress(&CreateSimpleEgressRequest{
		Name:     "simple-socks",
		Protocol: "socks5",
		Address:  "example.com",
		Port:     1080,
		Username: "demo-user",
		Password: "demo-pass",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("create simple egress failed: %v", err)
	}
	if created.Address != "example.com" || created.Port != 1080 {
		t.Fatalf("unexpected created egress: %#v", created)
	}

	items, err := svc.ListSimpleEgress()
	if err != nil {
		t.Fatalf("list simple egress failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected list count: %d", len(items))
	}
	if items[0].InternalTag != "n5-egress-0000000001" {
		t.Fatalf("unexpected tag: %#v", items[0])
	}
	if items[0].Protocol != "socks5" {
		t.Fatalf("unexpected protocol: %#v", items[0])
	}

	record, err := (&n5service.EgressService{}).Get(created.Id)
	if err != nil {
		t.Fatalf("get egress failed: %v", err)
	}
	if record.Protocol != "socks" {
		t.Fatalf("unexpected stored protocol: %#v", record)
	}

	outbound := make(map[string]interface{})
	if err := json.Unmarshal([]byte(record.OutboundJSON), &outbound); err != nil {
		t.Fatalf("unmarshal outbound failed: %v", err)
	}
	settings, ok := outbound["settings"].(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected outbound settings: %#v", outbound)
	}
	servers, ok := settings["servers"].([]interface{})
	if !ok || len(servers) != 1 {
		t.Fatalf("unexpected socks servers: %#v", settings)
	}
	server, ok := servers[0].(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected socks server: %#v", servers[0])
	}
	users, ok := server["users"].([]interface{})
	if !ok || len(users) != 1 {
		t.Fatalf("unexpected socks users: %#v", server)
	}
}

func TestSimpleEgressServiceDelete(t *testing.T) {
	initSimpleTestDB(t)

	svc := &EgressService{
		egressService: &n5service.EgressService{},
		testService:   &fakeTester{},
	}

	created, err := svc.CreateSimpleEgress(&CreateSimpleEgressRequest{
		Name:     "delete-simple-egress",
		Protocol: "socks5",
		Address:  "example.org",
		Port:     2080,
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("create simple egress failed: %v", err)
	}

	if err := svc.DeleteSimpleEgress(created.Id); err != nil {
		t.Fatalf("delete simple egress failed: %v", err)
	}

	items, err := svc.ListSimpleEgress()
	if err != nil {
		t.Fatalf("list simple egress failed: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("unexpected list count after delete: %d", len(items))
	}
}

func TestSimpleEgressServiceTest(t *testing.T) {
	initSimpleTestDB(t)

	svc := &EgressService{
		egressService: &n5service.EgressService{},
		testService: &fakeTester{
			record: &n5model.EgressTest{
				Status:   "success",
				Latency:  25,
				ExitIP:   "203.0.113.10",
				Message:  "",
				TestedAt: 1234567890,
			},
		},
	}

	created, err := svc.CreateSimpleEgress(&CreateSimpleEgressRequest{
		Name:     "test-simple-egress",
		Protocol: "socks",
		Address:  "example.net",
		Port:     3080,
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("create simple egress failed: %v", err)
	}

	record, err := svc.TestSimpleEgress(created.Id)
	if err != nil {
		t.Fatalf("test simple egress failed: %v", err)
	}
	if record.Status != "success" || record.ExitIP != "203.0.113.10" {
		t.Fatalf("unexpected test result: %#v", record)
	}
}

func TestSimpleEgressServiceCreateShadowsocksConfig(t *testing.T) {
	initSimpleTestDB(t)

	svc := &EgressService{
		egressService: &n5service.EgressService{},
		testService:   &fakeTester{},
	}

	created, err := svc.CreateSimpleEgress(&CreateSimpleEgressRequest{
		Name:     "simple-ss",
		Protocol: "ss",
		Address:  "ss.example.com",
		Port:     8388,
		Method:   "aes-256-gcm",
		Password: "ss-password",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("create simple shadowsocks egress failed: %v", err)
	}
	if created.Protocol != "ss" {
		t.Fatalf("unexpected created protocol: %#v", created)
	}

	record, err := (&n5service.EgressService{}).Get(created.Id)
	if err != nil {
		t.Fatalf("get egress failed: %v", err)
	}
	if record.Protocol != "shadowsocks" {
		t.Fatalf("unexpected stored protocol: %#v", record)
	}

	outbound := make(map[string]interface{})
	if err := json.Unmarshal([]byte(record.OutboundJSON), &outbound); err != nil {
		t.Fatalf("unmarshal outbound failed: %v", err)
	}
	if outbound["protocol"] != "shadowsocks" {
		t.Fatalf("unexpected outbound protocol: %#v", outbound)
	}
	settings, ok := outbound["settings"].(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected outbound settings: %#v", outbound)
	}
	servers, ok := settings["servers"].([]interface{})
	if !ok || len(servers) != 1 {
		t.Fatalf("unexpected shadowsocks servers: %#v", settings)
	}
	server, ok := servers[0].(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected shadowsocks server: %#v", servers[0])
	}
	if server["address"] != "ss.example.com" || int(server["port"].(float64)) != 8388 {
		t.Fatalf("unexpected shadowsocks address/port: %#v", server)
	}
	if server["method"] != "aes-256-gcm" || server["password"] != "ss-password" {
		t.Fatalf("unexpected shadowsocks config: %#v", server)
	}
}

func TestSimpleEgressServiceUpdateSocksKeepsTag(t *testing.T) {
	initSimpleTestDB(t)

	svc := &EgressService{
		egressService: &n5service.EgressService{},
		testService:   &fakeTester{},
	}

	created, err := svc.CreateSimpleEgress(&CreateSimpleEgressRequest{
		Name:     "editable-socks",
		Protocol: "socks5",
		Address:  "old.example.com",
		Port:     1080,
		Username: "old-user",
		Password: "old-pass",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("create simple egress failed: %v", err)
	}
	oldTag := created.InternalTag

	updated, err := svc.UpdateSimpleEgress(created.Id, &CreateSimpleEgressRequest{
		Name:     "editable-socks-new",
		Protocol: "socks5",
		Address:  "new.example.com",
		Port:     2080,
		Username: "new-user",
		Password: "new-pass",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("update simple egress failed: %v", err)
	}
	if updated.InternalTag != oldTag {
		t.Fatalf("tag changed after update: old=%s new=%s", oldTag, updated.InternalTag)
	}
	if updated.Address != "new.example.com" || updated.Port != 2080 || updated.Username != "new-user" {
		t.Fatalf("unexpected updated socks egress: %#v", updated)
	}

	record, err := (&n5service.EgressService{}).Get(created.Id)
	if err != nil {
		t.Fatalf("get egress failed: %v", err)
	}
	if record.Tag != oldTag {
		t.Fatalf("stored tag changed after update: %#v", record)
	}

	outbound := make(map[string]interface{})
	if err := json.Unmarshal([]byte(record.OutboundJSON), &outbound); err != nil {
		t.Fatalf("unmarshal outbound failed: %v", err)
	}
	settings := outbound["settings"].(map[string]interface{})
	server := settings["servers"].([]interface{})[0].(map[string]interface{})
	if server["address"] != "new.example.com" || int(server["port"].(float64)) != 2080 {
		t.Fatalf("unexpected updated socks server: %#v", server)
	}
	users := server["users"].([]interface{})
	user := users[0].(map[string]interface{})
	if user["user"] != "new-user" || user["pass"] != "new-pass" {
		t.Fatalf("unexpected updated socks auth: %#v", user)
	}
}

func TestSimpleEgressServiceUpdateShadowsocksKeepsTag(t *testing.T) {
	initSimpleTestDB(t)

	svc := &EgressService{
		egressService: &n5service.EgressService{},
		testService:   &fakeTester{},
	}

	created, err := svc.CreateSimpleEgress(&CreateSimpleEgressRequest{
		Name:     "editable-ss",
		Protocol: "ss",
		Address:  "old-ss.example.com",
		Port:     8388,
		Method:   "aes-128-gcm",
		Password: "old-ss-pass",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("create shadowsocks egress failed: %v", err)
	}
	oldTag := created.InternalTag

	updated, err := svc.UpdateSimpleEgress(created.Id, &CreateSimpleEgressRequest{
		Name:     "editable-ss-new",
		Protocol: "ss",
		Address:  "new-ss.example.com",
		Port:     8488,
		Method:   "chacha20-poly1305",
		Password: "new-ss-pass",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("update shadowsocks egress failed: %v", err)
	}
	if updated.InternalTag != oldTag {
		t.Fatalf("tag changed after update: old=%s new=%s", oldTag, updated.InternalTag)
	}
	if updated.Method != "chacha20-poly1305" || updated.Password != "new-ss-pass" {
		t.Fatalf("unexpected updated shadowsocks egress: %#v", updated)
	}

	record, err := (&n5service.EgressService{}).Get(created.Id)
	if err != nil {
		t.Fatalf("get egress failed: %v", err)
	}
	if record.Tag != oldTag {
		t.Fatalf("stored tag changed after update: %#v", record)
	}

	outbound := make(map[string]interface{})
	if err := json.Unmarshal([]byte(record.OutboundJSON), &outbound); err != nil {
		t.Fatalf("unmarshal outbound failed: %v", err)
	}
	settings := outbound["settings"].(map[string]interface{})
	server := settings["servers"].([]interface{})[0].(map[string]interface{})
	if server["address"] != "new-ss.example.com" || int(server["port"].(float64)) != 8488 {
		t.Fatalf("unexpected updated shadowsocks server: %#v", server)
	}
	if server["method"] != "chacha20-poly1305" || server["password"] != "new-ss-pass" {
		t.Fatalf("unexpected updated shadowsocks config: %#v", server)
	}
}

func TestSimpleEgressServiceCreateVMessRoundTrip(t *testing.T) {
	initSimpleTestDB(t)

	svc := &EgressService{
		egressService: &n5service.EgressService{},
		testService:   &fakeTester{},
	}

	created, err := svc.CreateSimpleEgress(&CreateSimpleEgressRequest{
		Name:     "vmess-ws-tls",
		Protocol: "vmess",
		Address:  "vmess.example.com",
		Port:     443,
		UUID:     "11111111-1111-1111-1111-111111111111",
		AlterID:  0,
		Network:  "ws",
		Security: "tls",
		SNI:      "vmess.example.com",
		Host:     "edge.example.com",
		Path:     "/vmess",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("create vmess egress failed: %v", err)
	}
	if created.Protocol != "vmess" || created.Network != "ws" || created.Security != "tls" {
		t.Fatalf("unexpected created vmess egress: %#v", created)
	}

	record, err := (&n5service.EgressService{}).Get(created.Id)
	if err != nil {
		t.Fatalf("get vmess egress failed: %v", err)
	}
	assertOutboundTestConfig(t, record.OutboundJSON)

	outbound := make(map[string]interface{})
	if err := json.Unmarshal([]byte(record.OutboundJSON), &outbound); err != nil {
		t.Fatalf("unmarshal vmess outbound failed: %v", err)
	}
	if outbound["protocol"] != "vmess" {
		t.Fatalf("unexpected outbound protocol: %#v", outbound)
	}

	loaded, err := svc.GetSimpleEgress(created.Id)
	if err != nil {
		t.Fatalf("get simple vmess failed: %v", err)
	}
	if !loaded.Supported || loaded.Address != "vmess.example.com" || loaded.UUID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("unexpected loaded vmess egress: %#v", loaded)
	}
	if loaded.Host != "edge.example.com" || loaded.Path != "/vmess" || loaded.SNI != "vmess.example.com" {
		t.Fatalf("unexpected vmess stream settings: %#v", loaded)
	}
}

func TestSimpleEgressServiceCreateVLESSRealityRoundTrip(t *testing.T) {
	initSimpleTestDB(t)

	svc := &EgressService{
		egressService: &n5service.EgressService{},
		testService:   &fakeTester{},
	}

	created, err := svc.CreateSimpleEgress(&CreateSimpleEgressRequest{
		Name:        "vless-reality",
		Protocol:    "vless",
		Address:     "reality.example.com",
		Port:        443,
		UUID:        "22222222-2222-2222-2222-222222222222",
		Network:     "tcp",
		Security:    "reality",
		SNI:         "www.cloudflare.com",
		PublicKey:   "PUBLIC_KEY_TEST",
		ShortID:     "01234567",
		Fingerprint: "chrome",
		SpiderX:     "/probe",
		Flow:        "xtls-rprx-vision",
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("create vless reality egress failed: %v", err)
	}
	if created.Protocol != "vless" || created.Security != "reality" {
		t.Fatalf("unexpected created vless egress: %#v", created)
	}

	record, err := (&n5service.EgressService{}).Get(created.Id)
	if err != nil {
		t.Fatalf("get vless reality egress failed: %v", err)
	}
	if strings.Contains(record.OutboundJSON, "privateKey") {
		t.Fatalf("vless reality outbound leaked privateKey: %s", record.OutboundJSON)
	}
	assertOutboundTestConfig(t, record.OutboundJSON)

	loaded, err := svc.GetSimpleEgress(created.Id)
	if err != nil {
		t.Fatalf("get simple vless reality failed: %v", err)
	}
	if !loaded.Supported || loaded.PublicKey != "PUBLIC_KEY_TEST" || loaded.ShortID != "01234567" {
		t.Fatalf("unexpected loaded vless reality egress: %#v", loaded)
	}
	if loaded.SNI != "www.cloudflare.com" || loaded.Flow != "xtls-rprx-vision" || loaded.SpiderX != "/probe" {
		t.Fatalf("unexpected vless reality stream settings: %#v", loaded)
	}
}

func TestSimpleEgressServiceCreateVLESSGRPCTLSRoundTrip(t *testing.T) {
	initSimpleTestDB(t)

	svc := &EgressService{
		egressService: &n5service.EgressService{},
		testService:   &fakeTester{},
	}

	created, err := svc.CreateSimpleEgress(&CreateSimpleEgressRequest{
		Name:        "vless-grpc-tls",
		Protocol:    "vless",
		Address:     "grpc.example.com",
		Port:        443,
		UUID:        "33333333-3333-3333-3333-333333333333",
		Network:     "grpc",
		Security:    "tls",
		SNI:         "grpc.example.com",
		ServiceName: "grpc-service",
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("create vless grpc egress failed: %v", err)
	}

	record, err := (&n5service.EgressService{}).Get(created.Id)
	if err != nil {
		t.Fatalf("get vless grpc egress failed: %v", err)
	}
	assertOutboundTestConfig(t, record.OutboundJSON)

	loaded, err := svc.GetSimpleEgress(created.Id)
	if err != nil {
		t.Fatalf("get simple vless grpc failed: %v", err)
	}
	if !loaded.Supported || loaded.Network != "grpc" || loaded.ServiceName != "grpc-service" || loaded.SNI != "grpc.example.com" {
		t.Fatalf("unexpected loaded vless grpc egress: %#v", loaded)
	}
}

func TestSimpleEgressServiceAdvancedVMessCanRoundTripInSimple(t *testing.T) {
	initSimpleTestDB(t)

	raw, err := (&n5service.EgressService{}).Create(&n5model.Egress{
		Name:         "advanced-vmess",
		Protocol:     "vmess",
		Enabled:      true,
		OutboundJSON: `{"protocol":"vmess","settings":{"vnext":[{"address":"advanced.example.com","port":443,"users":[{"id":"44444444-4444-4444-4444-444444444444","alterId":0,"security":"auto"}]}]},"streamSettings":{"network":"ws","security":"tls","wsSettings":{"path":"/advanced","headers":{"Host":"cdn.example.com"}},"tlsSettings":{"serverName":"advanced.example.com"}}}`,
	})
	if err != nil {
		t.Fatalf("create advanced vmess failed: %v", err)
	}

	svc := &EgressService{
		egressService: &n5service.EgressService{},
		testService:   &fakeTester{},
	}

	loaded, err := svc.GetSimpleEgress(raw.Id)
	if err != nil {
		t.Fatalf("get advanced vmess in simple failed: %v", err)
	}
	if !loaded.Supported || loaded.Host != "cdn.example.com" || loaded.Path != "/advanced" {
		t.Fatalf("unexpected advanced vmess simple view: %#v", loaded)
	}

	updated, err := svc.UpdateSimpleEgress(raw.Id, &CreateSimpleEgressRequest{
		Name:     loaded.Name,
		Protocol: loaded.Protocol,
		Address:  loaded.Address,
		Port:     loaded.Port,
		UUID:     loaded.UUID,
		AlterID:  loaded.AlterID,
		Network:  loaded.Network,
		Security: loaded.Security,
		SNI:      loaded.SNI,
		Host:     loaded.Host,
		Path:     loaded.Path,
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("update advanced vmess in simple failed: %v", err)
	}
	if updated.Host != "cdn.example.com" || updated.Path != "/advanced" {
		t.Fatalf("unexpected updated advanced vmess: %#v", updated)
	}
}

func TestSimpleEgressServiceRejectUnsupportedAdvanced(t *testing.T) {
	initSimpleTestDB(t)

	raw, err := (&n5service.EgressService{}).Create(&n5model.Egress{
		Name:         "unsupported-vmess",
		Protocol:     "vmess",
		Enabled:      true,
		OutboundJSON: `{"protocol":"vmess","settings":{"vnext":[{"address":"unsupported.example.com","port":443,"users":[{"id":"55555555-5555-5555-5555-555555555555","alterId":0,"security":"auto"}]}]},"streamSettings":{"network":"tcp","security":"none","tcpSettings":{"header":{"type":"http","request":{"path":["/"]}}}}}`,
	})
	if err != nil {
		t.Fatalf("create unsupported vmess failed: %v", err)
	}

	svc := &EgressService{
		egressService: &n5service.EgressService{},
		testService:   &fakeTester{},
	}

	loaded, err := svc.GetSimpleEgress(raw.Id)
	if err != nil {
		t.Fatalf("get unsupported vmess failed: %v", err)
	}
	if loaded.Supported {
		t.Fatalf("unsupported vmess should not be simple-supported: %#v", loaded)
	}
	if !strings.Contains(loaded.UnsupportedReason, "高级配置") {
		t.Fatalf("unexpected unsupported reason: %#v", loaded)
	}

	_, err = svc.UpdateSimpleEgress(raw.Id, &CreateSimpleEgressRequest{
		Name:     "should-fail",
		Protocol: "vmess",
		Address:  "unsupported.example.com",
		Port:     443,
		UUID:     "55555555-5555-5555-5555-555555555555",
		Network:  "tcp",
		Security: "none",
		Enabled:  true,
	})
	if err == nil {
		t.Fatal("expected update to reject unsupported advanced config")
	}
}
