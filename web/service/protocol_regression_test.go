package service

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/torr9522/n6-ui/util/json_util"
	ssservice "github.com/torr9522/n6-ui/web/service/shadowsocks"
	"github.com/torr9522/n6-ui/xray"
)

func TestProtocolRegressionFormalXrayConfigMatrix(t *testing.T) {
	binary := strings.TrimSpace(os.Getenv("N5_FORMAL_XRAY_BIN"))
	if binary == "" {
		t.Skip("N5_FORMAL_XRAY_BIN is not set")
	}

	type configCase struct {
		name   string
		config *xray.Config
	}
	cases := make([]configCase, 0, 10)
	methods := []string{
		ssservice.Method2022Blake3AES128GCM,
		ssservice.Method2022Blake3AES256GCM,
		ssservice.Method2022Blake3ChaCha20Poly1305,
	}
	for index, method := range methods {
		key, err := ssservice.Generate2022Key(method)
		if err != nil {
			t.Fatal(err)
		}
		inbound := newShadowsocksInbound(36500+index, method, key)
		cases = append(cases, configCase{
			name: "ss2022-inbound-" + method,
			config: &xray.Config{
				InboundConfigs:  []xray.InboundConfig{*inbound.GenXrayInboundConfig()},
				OutboundConfigs: json_util.RawMessage(`[{"protocol":"freedom","settings":{},"tag":"direct"}]`),
			},
		})
		outbound, err := json.Marshal([]map[string]interface{}{{
			"protocol": "shadowsocks",
			"settings": map[string]interface{}{
				"servers": []map[string]interface{}{{
					"address":  "127.0.0.1",
					"port":     37500 + index,
					"method":   method,
					"password": key,
				}},
			},
			"tag": "ss2022-outbound",
		}})
		if err != nil {
			t.Fatal(err)
		}
		cases = append(cases, configCase{
			name: "ss2022-outbound-" + method,
			config: &xray.Config{
				InboundConfigs:  []xray.InboundConfig{},
				OutboundConfigs: json_util.RawMessage(outbound),
			},
		})
	}

	legacy := newShadowsocksInbound(36510, "aes-256-gcm", "legacy regression password")
	vless := createVLESSTCPInbound(36512, "regression-vless")
	vless.Settings = `{"clients":[{"id":"33333333-3333-3333-3333-333333333333","flow":""}],"decryption":"none"}`
	cases = append(cases,
		configCase{
			name: "legacy-shadowsocks",
			config: &xray.Config{
				InboundConfigs:  []xray.InboundConfig{*legacy.GenXrayInboundConfig()},
				OutboundConfigs: json_util.RawMessage(`[{"protocol":"shadowsocks","settings":{"servers":[{"address":"127.0.0.1","port":37510,"method":"aes-256-gcm","password":"legacy regression password"}]},"tag":"legacy-ss"}]`),
			},
		},
		configCase{
			name: "vmess",
			config: &xray.Config{
				InboundConfigs:  []xray.InboundConfig{*createVMessTCPInbound(36511, "regression-vmess").GenXrayInboundConfig()},
				OutboundConfigs: json_util.RawMessage(`[{"protocol":"freedom","settings":{},"tag":"direct"}]`),
			},
		},
		configCase{
			name: "vless",
			config: &xray.Config{
				InboundConfigs:  []xray.InboundConfig{*vless.GenXrayInboundConfig()},
				OutboundConfigs: json_util.RawMessage(`[{"protocol":"freedom","settings":{},"tag":"direct"}]`),
			},
		},
	)

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			data, err := json.Marshal(test.config)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			output, err := exec.Command(binary, "run", "-test", "-format=json", "-c", path).CombinedOutput()
			if err != nil {
				t.Fatalf("formal Xray rejected %s: %v: %s", test.name, err, strings.TrimSpace(string(output)))
			}
		})
	}
}
