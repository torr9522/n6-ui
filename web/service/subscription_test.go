package service

import (
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"x-ui/database"
	"x-ui/database/model"
)

func createSubscriptionTestInbound(t *testing.T, inbound *model.Inbound) *model.Inbound {
	t.Helper()
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatalf("create inbound failed: %v", err)
	}
	return inbound
}

func createVMessTCPInbound(port int, remark string) *model.Inbound {
	return &model.Inbound{
		UserId:         1,
		Remark:         remark,
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           port,
		Protocol:       model.VMess,
		Settings:       `{"clients":[{"id":"11111111-1111-1111-1111-111111111111","alterId":0}]}`,
		StreamSettings: `{"network":"tcp","security":"none","tcpSettings":{"header":{"type":"none"}}}`,
		Tag:            remark + "-tag",
		Sniffing:       `{}`,
	}
}

func createVMessWSInbound(port int, remark string) *model.Inbound {
	return &model.Inbound{
		UserId:         1,
		Remark:         remark,
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           port,
		Protocol:       model.VMess,
		Settings:       `{"clients":[{"id":"22222222-2222-2222-2222-222222222222","alterId":0}]}`,
		StreamSettings: `{"network":"ws","security":"none","wsSettings":{"path":"/vmess-ws","headers":{"Host":"ws.example.com"}}}`,
		Tag:            remark + "-tag",
		Sniffing:       `{}`,
	}
}

func createVLESSTCPInbound(port int, remark string) *model.Inbound {
	return &model.Inbound{
		UserId:         1,
		Remark:         remark,
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           port,
		Protocol:       model.VLESS,
		Settings:       `{"clients":[{"id":"33333333-3333-3333-3333-333333333333","flow":"","encryption":"none"}],"decryption":"none"}`,
		StreamSettings: `{"network":"tcp","security":"none","tcpSettings":{"header":{"type":"none"}}}`,
		Tag:            remark + "-tag",
		Sniffing:       `{}`,
	}
}

func createVLESSTLSInbound(port int, remark string) *model.Inbound {
	return &model.Inbound{
		UserId:         1,
		Remark:         remark,
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           port,
		Protocol:       model.VLESS,
		Settings:       `{"clients":[{"id":"44444444-4444-4444-4444-444444444444","flow":"","encryption":"none"}],"decryption":"none"}`,
		StreamSettings: `{"network":"tcp","security":"tls","tlsSettings":{"serverName":"tls.example.com"},"tcpSettings":{"header":{"type":"none"}}}`,
		Tag:            remark + "-tag",
		Sniffing:       `{}`,
	}
}

func createVLESSRealityTCPInbound(port int, remark string) *model.Inbound {
	return &model.Inbound{
		UserId:         1,
		Remark:         remark,
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           port,
		Protocol:       model.VLESS,
		Settings:       `{"clients":[{"id":"55555555-5555-5555-5555-555555555555","flow":"xtls-rprx-vision","encryption":"none"}],"decryption":"none"}`,
		StreamSettings: `{"network":"tcp","security":"reality","realitySettings":{"serverNames":["reality.example.com"],"privateKey":"PRIVATE_KEY","shortIds":["012345"],"fingerprint":"chrome","publicKey":"PUBLIC_KEY","spiderX":"/spider"},"tcpSettings":{"header":{"type":"none"}}}`,
		Tag:            remark + "-tag",
		Sniffing:       `{}`,
	}
}

func createVLESSRealityGRPCInbound(port int, remark string) *model.Inbound {
	return &model.Inbound{
		UserId:         1,
		Remark:         remark,
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           port,
		Protocol:       model.VLESS,
		Settings:       `{"clients":[{"id":"66666666-6666-6666-6666-666666666666","flow":"xtls-rprx-vision","encryption":"none"}],"decryption":"none"}`,
		StreamSettings: `{"network":"grpc","security":"reality","realitySettings":{"serverNames":["grpc-reality.example.com"],"privateKey":"PRIVATE_KEY","shortIds":["012345"],"fingerprint":"chrome","publicKey":"PUBLIC_KEY","spiderX":"/grpc"},"grpcSettings":{"serviceName":"grpc-service"}}`,
		Tag:            remark + "-tag",
		Sniffing:       `{}`,
	}
}

func createTrojanTLSInbound(port int, remark string) *model.Inbound {
	return &model.Inbound{
		UserId:         1,
		Remark:         remark,
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           port,
		Protocol:       model.Trojan,
		Settings:       `{"clients":[{"password":"trojan-password"}]}`,
		StreamSettings: `{"network":"tcp","security":"tls","tlsSettings":{"serverName":"trojan.example.com"},"tcpSettings":{"header":{"type":"none"}}}`,
		Tag:            remark + "-tag",
		Sniffing:       `{}`,
	}
}

func createShadowsocksInbound(port int, remark string) *model.Inbound {
	return &model.Inbound{
		UserId:         1,
		Remark:         remark,
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           port,
		Protocol:       model.Shadowsocks,
		Settings:       `{"method":"aes-256-gcm","password":"ss-password"}`,
		StreamSettings: `{}`,
		Tag:            remark + "-tag",
		Sniffing:       `{}`,
	}
}

func TestSubscriptionServiceAddUpdateRefreshPreservesEnableAndOrder(t *testing.T) {
	initServiceTestDB(t)

	i1 := createSubscriptionTestInbound(t, createVMessTCPInbound(30101, "vmess-a"))
	i2 := createSubscriptionTestInbound(t, createVLESSTCPInbound(30102, "vless-b"))
	i3 := createSubscriptionTestInbound(t, createTrojanTLSInbound(30103, "trojan-c"))

	svc := &SubscriptionService{}
	added, err := svc.Add(&SubscriptionForm{
		Remark:     "test",
		Enable:     false,
		InboundIds: []int{i3.Id, i1.Id, i2.Id},
	})
	if err != nil {
		t.Fatalf("add subscription failed: %v", err)
	}
	if added.Enable {
		t.Fatal("expected enable=false to be preserved")
	}
	if !reflect.DeepEqual(added.InboundIds, []int{i3.Id, i1.Id, i2.Id}) {
		t.Fatalf("unexpected add inbound order: %#v", added.InboundIds)
	}

	stored := &model.Subscription{}
	if err := database.GetDB().Where("id = ?", added.Id).First(stored).Error; err != nil {
		t.Fatalf("query subscription failed: %v", err)
	}
	if stored.Enable {
		t.Fatal("expected stored enable=false")
	}
	var storedIDs []int
	if err := json.Unmarshal([]byte(stored.InboundIds), &storedIDs); err != nil {
		t.Fatalf("unmarshal stored inbound ids failed: %v", err)
	}
	if !reflect.DeepEqual(storedIDs, []int{i3.Id, i1.Id, i2.Id}) {
		t.Fatalf("unexpected stored inbound ids: %#v", storedIDs)
	}

	updated, err := svc.Update(added.Id, &SubscriptionForm{
		Remark:     "test-updated",
		Enable:     true,
		InboundIds: []int{i2.Id, i1.Id},
	})
	if err != nil {
		t.Fatalf("update subscription failed: %v", err)
	}
	if !updated.Enable {
		t.Fatal("expected enable=true after update")
	}
	if !reflect.DeepEqual(updated.InboundIds, []int{i2.Id, i1.Id}) {
		t.Fatalf("unexpected updated inbound order: %#v", updated.InboundIds)
	}

	oldToken := updated.Token
	refreshed, err := svc.RefreshToken(updated.Id)
	if err != nil {
		t.Fatalf("refresh token failed: %v", err)
	}
	if refreshed.Token == oldToken {
		t.Fatal("expected refreshed token to change")
	}
	if _, err := svc.GetByToken(oldToken); !database.IsNotFound(err) {
		t.Fatalf("expected old token not found, got: %v", err)
	}
	if _, err := svc.GetByToken(refreshed.Token); err != nil {
		t.Fatalf("expected new token available, got: %v", err)
	}

	if err := (&SettingService{}).setString("webBasePath", "/panel/"); err != nil {
		t.Fatalf("set base path failed: %v", err)
	}
	publicURL, err := svc.BuildPublicURL(refreshed.Token, "clash", "panel.example.com:443")
	if err != nil {
		t.Fatalf("build public url failed: %v", err)
	}
	if publicURL != "https://panel.example.com:443/panel/sub/"+refreshed.Token+"?format=clash" {
		t.Fatalf("unexpected public url: %s", publicURL)
	}
}

func TestSubscriptionServiceGenerateBase64PreservesInboundOrder(t *testing.T) {
	initServiceTestDB(t)

	vmess := createSubscriptionTestInbound(t, createVMessTCPInbound(30201, "vmess-first"))
	vless := createSubscriptionTestInbound(t, createVLESSTCPInbound(30202, "vless-second"))

	svc := &SubscriptionService{}
	sub, err := svc.Add(&SubscriptionForm{
		Remark:     "ordered",
		Enable:     true,
		InboundIds: []int{vless.Id, vmess.Id},
	})
	if err != nil {
		t.Fatalf("add subscription failed: %v", err)
	}

	if err := (&SettingService{}).setString("webBasePath", "/"); err != nil {
		t.Fatalf("reset base path failed: %v", err)
	}

	if err := (&ShareLinkService{}).shareAddressService.write([]ShareAddress{{
		Id:      "share-1",
		Type:    "domain",
		Address: "share.example.com",
		Enabled: true,
	}}); err != nil {
		t.Fatalf("write share addresses failed: %v", err)
	}

	encoded, err := svc.GenerateBase64(sub.Token, "127.0.0.1:54321")
	if err != nil {
		t.Fatalf("generate base64 failed: %v", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode base64 failed: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(decoded)), "\n")
	if len(lines) != 2 {
		t.Fatalf("unexpected base64 line count: %d", len(lines))
	}
	if !strings.HasPrefix(lines[0], "vless://") || !strings.HasPrefix(lines[1], "vmess://") {
		t.Fatalf("unexpected link order: %#v", lines)
	}
}

func TestSubscriptionServiceGenerateBase64Returns404ForDisabledOrDeletedTokens(t *testing.T) {
	initServiceTestDB(t)
	inbound := createSubscriptionTestInbound(t, createVMessTCPInbound(30301, "vmess-disabled"))
	svc := &SubscriptionService{}

	sub, err := svc.Add(&SubscriptionForm{
		Remark:     "disabled",
		Enable:     false,
		InboundIds: []int{inbound.Id},
	})
	if err != nil {
		t.Fatalf("add subscription failed: %v", err)
	}
	if _, err := svc.GenerateBase64(sub.Token, "panel.example.com"); err == nil || !strings.Contains(strings.ToLower(err.Error()), "not found") {
		t.Fatalf("expected disabled token to behave like not found, got: %v", err)
	}

	enabled, err := svc.Update(sub.Id, &SubscriptionForm{
		Remark:     sub.Remark,
		Enable:     true,
		InboundIds: []int{inbound.Id},
	})
	if err != nil {
		t.Fatalf("enable subscription failed: %v", err)
	}
	if err := svc.Delete(enabled.Id); err != nil {
		t.Fatalf("delete subscription failed: %v", err)
	}
	if _, err := svc.GenerateBase64(enabled.Token, "panel.example.com"); !database.IsNotFound(err) {
		t.Fatalf("expected deleted token not found, got: %v", err)
	}
}

func TestSubscriptionURLsUseRuntimeShareAddresses(t *testing.T) {
	initServiceTestDB(t)
	shareFile := filepath.Join(t.TempDir(), "share_addresses.json")
	original := shareAddressPath
	shareAddressPath = shareFile
	defer func() {
		shareAddressPath = original
	}()

	inbound := createSubscriptionTestInbound(t, createVMessTCPInbound(30401, "vmess-runtime"))
	svc := &SubscriptionService{}
	sub, err := svc.Add(&SubscriptionForm{
		Remark:     "runtime-share",
		Enable:     true,
		InboundIds: []int{inbound.Id},
	})
	if err != nil {
		t.Fatalf("add subscription failed: %v", err)
	}
	if _, err := (&ShareAddressService{}).Add("runtime.example.com", "runtime"); err != nil {
		t.Fatalf("add share address failed: %v", err)
	}
	encoded, err := svc.GenerateBase64(sub.Token, "127.0.0.1:54321")
	if err != nil {
		t.Fatalf("generate base64 failed: %v", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode base64 failed: %v", err)
	}
	link := strings.TrimSpace(string(decoded))
	if !strings.HasPrefix(link, "vmess://") {
		t.Fatalf("unexpected decoded subscription payload: %s", link)
	}
	vmessPayload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(link, "vmess://"))
	if err != nil {
		t.Fatalf("decode vmess payload failed: %v", err)
	}
	if !strings.Contains(string(vmessPayload), "runtime.example.com") {
		t.Fatalf("expected runtime share address in vmess payload: %s", string(vmessPayload))
	}
}
