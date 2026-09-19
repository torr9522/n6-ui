package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"x-ui/database/model"
	"x-ui/util/json_util"
	ssservice "x-ui/web/service/shadowsocks"
	"x-ui/xray"
)

func newShadowsocksInbound(port int, method string, password string) *model.Inbound {
	settings, _ := json.Marshal(map[string]interface{}{
		"method":   method,
		"password": password,
		"network":  "tcp,udp",
	})
	return &model.Inbound{
		UserId:         1,
		Remark:         "ss-test",
		Enable:         true,
		Port:           port,
		Protocol:       model.Shadowsocks,
		Settings:       string(settings),
		StreamSettings: `{}`,
		Tag:            fmt.Sprintf("inbound-%d", port),
		Sniffing:       `{}`,
	}
}

func TestInboundServiceShadowsocks2022CRUDAndConfig(t *testing.T) {
	initServiceTestDB(t)
	service := &InboundService{}
	methods := []string{
		ssservice.Method2022Blake3AES128GCM,
		ssservice.Method2022Blake3AES256GCM,
		ssservice.Method2022Blake3ChaCha20Poly1305,
	}
	for index, method := range methods {
		t.Run(method, func(t *testing.T) {
			key, err := ssservice.Generate2022Key(method)
			if err != nil {
				t.Fatal(err)
			}
			inbound := newShadowsocksInbound(36100+index, method, key)
			if err := service.AddInbound(inbound); err != nil {
				t.Fatalf("add inbound failed: %v", err)
			}
			stored, err := service.GetInbound(inbound.Id)
			if err != nil {
				t.Fatalf("get inbound failed: %v", err)
			}
			config := stored.GenXrayInboundConfig()
			if config.Protocol != "shadowsocks" || !strings.Contains(string(config.Settings), method) || !strings.Contains(string(config.Settings), key) {
				t.Fatalf("unexpected generated config: %#v", config)
			}

			stored.Enable = false
			stored.Remark = "ss-test-edited"
			if err := service.UpdateInbound(stored); err != nil {
				t.Fatalf("disable/edit inbound failed: %v", err)
			}
			stored.Enable = true
			if err := service.UpdateInbound(stored); err != nil {
				t.Fatalf("enable inbound failed: %v", err)
			}
			if err := service.DelInbound(stored.Id); err != nil {
				t.Fatalf("delete inbound failed: %v", err)
			}
			if _, err := service.GetInbound(stored.Id); err == nil {
				t.Fatal("deleted inbound still exists")
			}
		})
	}
}

func TestInboundServiceRejectsInvalidShadowsocksCredentials(t *testing.T) {
	initServiceTestDB(t)
	service := &InboundService{}
	tests := []struct {
		name     string
		method   string
		password string
		settings string
		want     string
	}{
		{"invalid base64", ssservice.Method2022Blake3AES128GCM, "invalid!", "", "Base64"},
		{"wrong length", ssservice.Method2022Blake3AES128GCM, base64.StdEncoding.EncodeToString(make([]byte, 15)), "", "16 字节"},
		{"unknown method", "2022-invalid", base64.StdEncoding.EncodeToString(make([]byte, 16)), "", "不支持"},
		{"multi user", ssservice.Method2022Blake3AES128GCM, base64.StdEncoding.EncodeToString(make([]byte, 16)), `{"method":"2022-blake3-aes-128-gcm","password":"AAAAAAAAAAAAAAAAAAAAAA==","network":"tcp,udp","clients":[{"password":"ignored"}]}`, "单用户"},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inbound := newShadowsocksInbound(36200+index, test.method, test.password)
			if test.settings != "" {
				inbound.Settings = test.settings
			}
			err := service.AddInbound(inbound)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}
}

func TestInboundServiceKeepsLegacyShadowsocksPasswordSemantics(t *testing.T) {
	initServiceTestDB(t)
	inbound := newShadowsocksInbound(36300, "aes-256-gcm", "ordinary legacy password")
	service := &InboundService{}
	if err := service.AddInbound(inbound); err != nil {
		t.Fatalf("legacy Shadowsocks add failed: %v", err)
	}
	stored, err := service.GetInbound(inbound.Id)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stored.Settings, "ordinary legacy password") {
		t.Fatalf("legacy password changed: %s", stored.Settings)
	}
}

func TestInboundServiceShadowsocks2022FormalXrayConfig(t *testing.T) {
	binary := strings.TrimSpace(os.Getenv("N5_FORMAL_XRAY_BIN"))
	if binary == "" {
		t.Skip("N5_FORMAL_XRAY_BIN is not set")
	}
	methods := []string{
		ssservice.Method2022Blake3AES128GCM,
		ssservice.Method2022Blake3AES256GCM,
		ssservice.Method2022Blake3ChaCha20Poly1305,
	}
	for index, method := range methods {
		t.Run(method, func(t *testing.T) {
			key, err := ssservice.Generate2022Key(method)
			if err != nil {
				t.Fatal(err)
			}
			inbound := newShadowsocksInbound(36400+index, method, key)
			config := &xray.Config{
				InboundConfigs:  []xray.InboundConfig{*inbound.GenXrayInboundConfig()},
				OutboundConfigs: json_util.RawMessage(`[{"protocol":"freedom","settings":{},"tag":"direct"}]`),
			}
			data, err := json.Marshal(config)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			output, err := exec.Command(binary, "run", "-test", "-format=json", "-c", path).CombinedOutput()
			if err != nil {
				t.Fatalf("formal Xray config test failed: %v: %s", err, strings.TrimSpace(string(output)))
			}
		})
	}
}
