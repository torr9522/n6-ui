package n5

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	n5model "github.com/torr9522/n6-ui/database/model/n5"
	ssservice "github.com/torr9522/n6-ui/web/service/shadowsocks"
)

func ss2022OutboundJSON(method string, key string, address string, port int) string {
	data, _ := json.Marshal(map[string]interface{}{
		"protocol": "shadowsocks",
		"settings": map[string]interface{}{
			"servers": []map[string]interface{}{{
				"address":  address,
				"port":     port,
				"method":   method,
				"password": key,
			}},
		},
	})
	return string(data)
}

func ss2022Methods() []string {
	return []string{
		ssservice.Method2022Blake3AES128GCM,
		ssservice.Method2022Blake3AES256GCM,
		ssservice.Method2022Blake3ChaCha20Poly1305,
	}
}

func TestEgressServiceShadowsocks2022CRUD(t *testing.T) {
	initTestDB(t)
	service := &EgressService{}
	for index, method := range ss2022Methods() {
		t.Run(method, func(t *testing.T) {
			key, err := ssservice.Generate2022Key(method)
			if err != nil {
				t.Fatal(err)
			}
			created, err := service.Create(&n5model.Egress{
				Name:         "ss2022-" + method,
				Protocol:     "shadowsocks",
				Enabled:      true,
				OutboundJSON: ss2022OutboundJSON(method, key, "127.0.0.1", 37000+index),
			})
			if err != nil {
				t.Fatalf("create egress failed: %v", err)
			}
			if created.Tag == "" || created.Protocol != "shadowsocks" {
				t.Fatalf("unexpected created egress: %#v", created)
			}

			replacement, err := ssservice.Generate2022Key(method)
			if err != nil {
				t.Fatal(err)
			}
			created.Name += "-edited"
			created.OutboundJSON = ss2022OutboundJSON(method, replacement, "127.0.0.1", 37100+index)
			updated, err := service.Update(created)
			if err != nil {
				t.Fatalf("update egress failed: %v", err)
			}
			if updated.Tag != created.Tag || !strings.Contains(updated.OutboundJSON, replacement) {
				t.Fatalf("unexpected updated egress: %#v", updated)
			}
			if _, err := service.Get(updated.Id); err != nil {
				t.Fatalf("get egress failed: %v", err)
			}
			if err := service.Delete(updated.Id); err != nil {
				t.Fatalf("delete egress failed: %v", err)
			}
		})
	}
}

func TestEgressServiceRejectsInvalidShadowsocks2022(t *testing.T) {
	validKey, err := ssservice.Generate2022Key(ssservice.Method2022Blake3AES128GCM)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"invalid base64", ss2022OutboundJSON(ssservice.Method2022Blake3AES128GCM, "bad!", "127.0.0.1", 8388), "Base64"},
		{"wrong key length", ss2022OutboundJSON(ssservice.Method2022Blake3AES256GCM, validKey, "127.0.0.1", 8388), "32 字节"},
		{"unknown method", ss2022OutboundJSON("2022-invalid", validKey, "127.0.0.1", 8388), "不支持"},
		{"empty address", ss2022OutboundJSON(ssservice.Method2022Blake3AES128GCM, validKey, "", 8388), "地址不能为空"},
		{"invalid port", ss2022OutboundJSON(ssservice.Method2022Blake3AES128GCM, validKey, "127.0.0.1", 0), "端口"},
	}
	service := &EgressService{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := service.ValidateConfig("shadowsocks", test.raw, "test-tag")
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}
}

func TestEgressServiceShadowsocks2022FormalXrayConfig(t *testing.T) {
	binary := strings.TrimSpace(os.Getenv("N5_FORMAL_XRAY_BIN"))
	if binary == "" {
		t.Skip("N5_FORMAL_XRAY_BIN is not set")
	}
	for index, method := range ss2022Methods() {
		t.Run(method, func(t *testing.T) {
			key, err := ssservice.Generate2022Key(method)
			if err != nil {
				t.Fatal(err)
			}
			outbound := ss2022OutboundJSON(method, key, "127.0.0.1", 37200+index)
			config := fmt.Sprintf(`{"inbounds":[],"outbounds":[%s]}`, outbound)
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(config), 0600); err != nil {
				t.Fatal(err)
			}
			output, err := exec.Command(binary, "run", "-test", "-format=json", "-c", path).CombinedOutput()
			if err != nil {
				t.Fatalf("formal Xray config test failed: %v: %s", err, strings.TrimSpace(string(output)))
			}
		})
	}
}

func TestEgressTestServiceShadowsocks2022FormalRuntime(t *testing.T) {
	binary := strings.TrimSpace(os.Getenv("N5_FORMAL_XRAY_BIN"))
	if binary == "" {
		t.Skip("N5_FORMAL_XRAY_BIN is not set")
	}
	oldWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	runtimeDirectory := t.TempDir()
	if err := os.Mkdir(filepath.Join(runtimeDirectory, "bin"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(binary, filepath.Join(runtimeDirectory, "bin", "xray-linux-amd64")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(runtimeDirectory); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWorkingDirectory) }()

	initTestDB(t)
	for _, method := range ss2022Methods() {
		t.Run(method, func(t *testing.T) {
			key, err := ssservice.Generate2022Key(method)
			if err != nil {
				t.Fatal(err)
			}
			port, err := allocateLocalTCPPort()
			if err != nil {
				t.Fatal(err)
			}
			serverConfig := fmt.Sprintf(`{"log":{"loglevel":"warning"},"inbounds":[{"listen":"127.0.0.1","port":%d,"protocol":"shadowsocks","settings":{"method":%q,"password":%q,"network":"tcp,udp"},"tag":"ss2022-server"}],"outbounds":[{"protocol":"freedom","settings":{},"tag":"direct"}]}`, port, method, key)
			serverConfigPath := filepath.Join(t.TempDir(), "server.json")
			if err := os.WriteFile(serverConfigPath, []byte(serverConfig), 0600); err != nil {
				t.Fatal(err)
			}
			serverOutput := &bytes.Buffer{}
			server := exec.Command(binary, "run", "-format=json", "-c", serverConfigPath)
			server.Stdout = serverOutput
			server.Stderr = serverOutput
			if err := server.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if server.Process != nil {
					_ = server.Process.Kill()
				}
				_ = server.Wait()
			}()
			if err := waitForTCP(net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 5*time.Second); err != nil {
				t.Fatalf("formal Xray server did not start: %v: %s", err, strings.TrimSpace(serverOutput.String()))
			}

			egress, err := (&EgressService{}).Create(&n5model.Egress{
				Name:         "runtime-" + method,
				Protocol:     "shadowsocks",
				Enabled:      true,
				OutboundJSON: ss2022OutboundJSON(method, key, "127.0.0.1", port),
			})
			if err != nil {
				t.Fatalf("create runtime egress failed: %v", err)
			}
			result, err := (&EgressTestService{}).Test(egress.Id)
			if err != nil {
				t.Fatalf("N5 egress test failed: %v: %s", err, strings.TrimSpace(serverOutput.String()))
			}
			if result.Status != egressTestStatusSuccess || strings.TrimSpace(result.ExitIP) == "" {
				t.Fatalf("unexpected N5 egress test result: %#v: %s", result, strings.TrimSpace(serverOutput.String()))
			}
		})
	}
}

func waitForTCP(address string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		connection, err := net.DialTimeout("tcp", address, 200*time.Millisecond)
		if err == nil {
			_ = connection.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for %s", address)
}
