package service

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
	"x-ui/database/model"
	ssservice "x-ui/web/service/shadowsocks"
)

func createShadowsocks2022SubscriptionInbound(t *testing.T, port int, remark string, method string, network string) (*model.Inbound, string) {
	t.Helper()
	key, err := ssservice.Generate2022Key(method)
	if err != nil {
		t.Fatal(err)
	}
	settings, err := json.Marshal(map[string]interface{}{
		"method": method, "password": key, "network": network,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &model.Inbound{
		UserId:         1,
		Remark:         remark,
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           port,
		Protocol:       model.Shadowsocks,
		Settings:       string(settings),
		StreamSettings: `{}`,
		Tag:            remark + "-tag",
		Sniffing:       `{}`,
	}, key
}

func TestShareLinkServiceShadowsocks2022Subscriptions(t *testing.T) {
	methods := []string{
		ssservice.Method2022Blake3AES128GCM,
		ssservice.Method2022Blake3AES256GCM,
		ssservice.Method2022Blake3ChaCha20Poly1305,
	}
	networks := []string{"tcp", "udp", "tcp,udp"}
	inbounds := make([]*model.Inbound, 0, len(methods))
	keys := make([]string, 0, len(methods))
	for index, method := range methods {
		inbound, key := createShadowsocks2022SubscriptionInbound(t, 32101+index, "ss2022-"+method, method, networks[index])
		inbounds = append(inbounds, inbound)
		keys = append(keys, key)
	}
	svc := &ShareLinkService{loadShareAddresses: func() ([]ShareAddress, error) {
		return []ShareAddress{{Id: "share-1", Type: "domain", Address: "share.example.com", Enabled: true}}, nil
	}}
	ctx := shareContext{RequestHost: "127.0.0.1:54321"}

	links, err := svc.GenerateLinks(inbounds, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != len(methods) {
		t.Fatalf("unexpected link count: %d", len(links))
	}
	for index, value := range links {
		parsed, err := ssservice.ParseShareLink(value)
		if err != nil {
			t.Fatalf("parse generated link %d: %v", index, err)
		}
		if parsed.Method != methods[index] || parsed.Password != keys[index] || parsed.Host != "share.example.com" || parsed.Port != 32101+index || parsed.Name != "ss2022-"+methods[index] {
			t.Fatalf("unexpected generated link %d: %#v", index, parsed)
		}
	}

	encoded, err := svc.GenerateBase64(inbounds, ctx)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(links, "\n") != string(decoded) {
		t.Fatalf("unexpected Base64 subscription payload: %s", decoded)
	}

	mihomo, err := svc.GenerateMihomo(inbounds, ctx)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]interface{}
	if err := yaml.Unmarshal([]byte(mihomo), &config); err != nil {
		t.Fatal(err)
	}
	proxies, ok := config["proxies"].([]interface{})
	if !ok || len(proxies) != len(methods) {
		t.Fatalf("unexpected Mihomo proxies: %#v", config["proxies"])
	}
	for index, item := range proxies {
		proxy := item.(map[interface{}]interface{})
		if proxy["type"] != "ss" || proxy["cipher"] != methods[index] || proxy["password"] != keys[index] {
			t.Fatalf("unexpected Mihomo proxy %d: %#v", index, proxy)
		}
		wantUDP := networks[index] != "tcp"
		if proxy["udp"] != wantUDP {
			t.Fatalf("unexpected Mihomo UDP flag %d: got %#v want %v", index, proxy["udp"], wantUDP)
		}
	}

	_, err = svc.GenerateClash(inbounds, ctx)
	if err == nil || !strings.Contains(err.Error(), "使用 Mihomo") {
		t.Fatalf("classic Clash should reject SS2022 with guidance, got %v", err)
	}
}

func TestShareLinkServiceKeepsLegacyShadowsocksSubscriptionFormat(t *testing.T) {
	inbound := createShadowsocksInbound(32110, "legacy ss +")
	svc := &ShareLinkService{loadShareAddresses: func() ([]ShareAddress, error) {
		return []ShareAddress{{Id: "share-1", Type: "domain", Address: "share.example.com", Enabled: true}}, nil
	}}
	links, err := svc.GenerateLinks([]*model.Inbound{inbound}, shareContext{RequestHost: "127.0.0.1:54321"})
	if err != nil {
		t.Fatal(err)
	}
	wantPayload := base64.RawURLEncoding.EncodeToString([]byte("aes-256-gcm:ss-password@share.example.com:32110"))
	want := "ss://" + wantPayload + "#legacy+ss+%2B"
	if len(links) != 1 || links[0] != want {
		t.Fatalf("legacy Shadowsocks share format changed: got %#v want %q", links, want)
	}
	clash, err := svc.GenerateClash([]*model.Inbound{inbound}, shareContext{RequestHost: "127.0.0.1:54321"})
	if err != nil || !strings.Contains(clash, "cipher: 'aes-256-gcm'") || !strings.Contains(clash, "password: 'ss-password'") {
		t.Fatalf("legacy Shadowsocks Clash output changed: %v\n%s", err, clash)
	}
}

func TestSubscriptionServiceShadowsocks2022Formats(t *testing.T) {
	initServiceTestDB(t)
	inbound, key := createShadowsocks2022SubscriptionInbound(t, 32201, "ss2022-sub", ssservice.Method2022Blake3AES256GCM, "tcp,udp")
	inbound = createSubscriptionTestInbound(t, inbound)
	svc := &SubscriptionService{shareLinkService: ShareLinkService{loadShareAddresses: func() ([]ShareAddress, error) {
		return []ShareAddress{{Id: "share-1", Type: "domain", Address: "share.example.com", Enabled: true}}, nil
	}}}
	sub, err := svc.Add(&SubscriptionForm{Remark: "ss2022", Enable: true, InboundIds: []int{inbound.Id}})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := svc.GenerateBase64(sub.Token, "panel.example.com")
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatal(err)
	}
	link, err := ssservice.ParseShareLink(strings.TrimSpace(string(decoded)))
	if err != nil || link.Password != key {
		t.Fatalf("unexpected default subscription link: %#v %v", link, err)
	}
	mihomo, err := svc.GenerateMihomo(sub.Token, "panel.example.com")
	if err != nil || !strings.Contains(mihomo, ssservice.Method2022Blake3AES256GCM) || !strings.Contains(mihomo, key) {
		t.Fatalf("unexpected Mihomo subscription: %v\n%s", err, mihomo)
	}
	if _, err := svc.GenerateClash(sub.Token, "panel.example.com"); err == nil || !strings.Contains(err.Error(), "使用 Mihomo") {
		t.Fatalf("unexpected classic Clash result: %v", err)
	}
}

func TestMihomoValidatesShadowsocks2022Subscription(t *testing.T) {
	bin := strings.TrimSpace(os.Getenv("N5_MIHOMO_BIN"))
	if bin == "" {
		t.Skip("N5_MIHOMO_BIN is not set")
	}
	methods := []string{
		ssservice.Method2022Blake3AES128GCM,
		ssservice.Method2022Blake3AES256GCM,
		ssservice.Method2022Blake3ChaCha20Poly1305,
	}
	inbounds := make([]*model.Inbound, 0, len(methods))
	for index, method := range methods {
		inbound, _ := createShadowsocks2022SubscriptionInbound(t, 32301+index, "mihomo-"+method, method, "tcp,udp")
		inbounds = append(inbounds, inbound)
	}
	svc := &ShareLinkService{loadShareAddresses: func() ([]ShareAddress, error) {
		return []ShareAddress{{Id: "share-1", Type: "domain", Address: "share.example.com", Enabled: true}}, nil
	}}
	content, err := svc.GenerateMihomo(inbounds, shareContext{RequestHost: "127.0.0.1:54321"})
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(t.TempDir(), "mihomo.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "-t", "-f", configPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Mihomo rejected generated SS2022 subscription: %v\n%s\n%s", err, output, content)
	}
}

func TestShareLinkServiceRejectsInvalidShadowsocks2022Subscription(t *testing.T) {
	inbound, _ := createShadowsocks2022SubscriptionInbound(t, 32401, "invalid", ssservice.Method2022Blake3AES128GCM, "tcp,udp")
	inbound.Settings = `{"method":"2022-blake3-aes-128-gcm","password":"invalid!","network":"tcp,udp"}`
	_, err := (&ShareLinkService{}).GenerateLinks([]*model.Inbound{inbound}, shareContext{RequestHost: "example.com"})
	if err == nil || !strings.Contains(err.Error(), "Base64") {
		t.Fatalf("expected invalid SS2022 key rejection, got %v", err)
	}
	inbound.Settings = `{"method":"2022-blake3-aes-128-gcm","password":"AAAAAAAAAAAAAAAAAAAAAA==","network":"tcp,udp","clients":[{"password":"multi"}]}`
	_, err = (&ShareLinkService{}).GenerateMihomo([]*model.Inbound{inbound}, shareContext{RequestHost: "example.com"})
	if err == nil || !strings.Contains(err.Error(), "单用户") {
		t.Fatalf("expected multi-user rejection, got %v", err)
	}
}
