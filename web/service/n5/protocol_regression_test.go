package n5

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"x-ui/database"
	n5model "x-ui/database/model/n5"
	ssservice "x-ui/web/service/shadowsocks"
)

func startRegressionXray(t *testing.T, binaryPath string, config string) *bytes.Buffer {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	output := &bytes.Buffer{}
	command := exec.Command(binaryPath, "run", "-format=json", "-c", configPath)
	command.Stdout = output
	command.Stderr = output
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if command.Process != nil {
			_ = command.Process.Kill()
		}
		_ = command.Wait()
	})
	return output
}

func allocateRegressionTCPPorts(t *testing.T, count int) []int {
	t.Helper()
	ports := make([]int, 0, count)
	seen := make(map[int]bool, count)
	for len(ports) < count {
		port, err := allocateLocalTCPPort()
		if err != nil {
			t.Fatal(err)
		}
		if seen[port] {
			continue
		}
		seen[port] = true
		ports = append(ports, port)
	}
	return ports
}

func TestShadowsocksProtocolFamilyFormalTCPUDPRegression(t *testing.T) {
	binaryPath := strings.TrimSpace(os.Getenv("N5_FORMAL_XRAY_BIN"))
	if binaryPath == "" {
		t.Skip("N5_FORMAL_XRAY_BIN is not set")
	}
	tests := []struct {
		name     string
		method   string
		password func(t *testing.T) string
	}{
		{name: "legacy-aes-256-gcm", method: "aes-256-gcm", password: func(*testing.T) string { return "legacy regression password" }},
	}
	for _, method := range ss2022Methods() {
		method := method
		tests = append(tests, struct {
			name     string
			method   string
			password func(t *testing.T) string
		}{
			name:   method,
			method: method,
			password: func(t *testing.T) string {
				key, err := ssservice.Generate2022Key(method)
				if err != nil {
					t.Fatal(err)
				}
				return key
			},
		})
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			password := test.password(t)
			ports := allocateRegressionTCPPorts(t, 3)
			serverPort, socksPort, udpInboundPort := ports[0], ports[1], ports[2]

			serverConfig := fmt.Sprintf(`{"log":{"loglevel":"warning"},"inbounds":[{"listen":"127.0.0.1","port":%d,"protocol":"shadowsocks","settings":{"method":%q,"password":%q,"network":"tcp,udp"},"tag":"ss-server"}],"outbounds":[{"protocol":"freedom","settings":{},"tag":"direct"}]}`, serverPort, test.method, password)
			serverOutput := startRegressionXray(t, binaryPath, serverConfig)
			if err := waitForTCP(net.JoinHostPort("127.0.0.1", strconv.Itoa(serverPort)), 5*time.Second); err != nil {
				t.Fatalf("server did not start: %v: %s", err, serverOutput.String())
			}

			clientConfig := fmt.Sprintf(`{"log":{"loglevel":"warning"},"inbounds":[{"listen":"127.0.0.1","port":%d,"protocol":"socks","settings":{"auth":"noauth","udp":true,"ip":"127.0.0.1"},"tag":"tcp-client"},{"listen":"127.0.0.1","port":%d,"protocol":"dokodemo-door","settings":{"address":"1.1.1.1","port":53,"network":"udp"},"tag":"udp-client"}],"outbounds":[{"protocol":"shadowsocks","settings":{"servers":[{"address":"127.0.0.1","port":%d,"method":%q,"password":%q}]},"tag":"ss-out"}]}`, socksPort, udpInboundPort, serverPort, test.method, password)
			clientOutput := startRegressionXray(t, binaryPath, clientConfig)
			socksAddress := net.JoinHostPort("127.0.0.1", strconv.Itoa(socksPort))
			if err := waitForTCP(socksAddress, 5*time.Second); err != nil {
				t.Fatalf("client did not start: %v: %s", err, clientOutput.String())
			}

			exitIP, err := runSOCKSHTTPIPProbe(socksAddress, egressTestHTTPURL, 15*time.Second)
			if err != nil {
				t.Fatalf("TCP HTTPS regression failed for %s: %v: server=%s client=%s", test.name, err, serverOutput.String(), clientOutput.String())
			}
			if strings.TrimSpace(exitIP) == "" {
				t.Fatal("TCP HTTPS regression returned an empty exit IP")
			}

			udpConnection, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: udpInboundPort})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = udpConnection.Close() })
			if err := udpConnection.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
				t.Fatal(err)
			}
			udpPayload := []byte{
				0x4e, 0x35, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x07, 'e', 'x', 'a', 'm', 'p', 'l', 'e',
				0x03, 'c', 'o', 'm', 0x00, 0x00, 0x01, 0x00, 0x01,
			}
			if _, err := udpConnection.Write(udpPayload); err != nil {
				t.Fatal(err)
			}
			udpReply := make([]byte, 2048)
			count, err := udpConnection.Read(udpReply)
			if err != nil {
				t.Fatalf("UDP regression failed: %v: server=%s client=%s", err, serverOutput.String(), clientOutput.String())
			}
			if count < 12 || binary.BigEndian.Uint16(udpReply[:2]) != 0x4e35 || udpReply[2]&0x80 == 0 {
				prefixLength := count
				if prefixLength > 12 {
					prefixLength = 12
				}
				t.Fatalf("invalid UDP DNS response: count=%d prefix=%x", count, udpReply[:prefixLength])
			}
		})
	}
}

func countRegressionRows(t *testing.T, query string, values ...interface{}) int64 {
	t.Helper()
	var count int64
	if err := database.GetDB().Raw(query, values...).Scan(&count).Error; err != nil {
		t.Fatal(err)
	}
	return count
}

func TestSS2022N5DatabaseConsistency(t *testing.T) {
	initTestDB(t)
	egress := createSS2022AdvancedRoutingEgress(t, "ss2022-consistency", ssservice.Method2022Blake3AES128GCM, 37950)
	pool, err := (&EgressPoolService{}).Create(&n5model.EgressPool{Name: "ss2022-consistency-pool", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (&EgressPoolService{}).AddMember(pool.Id, egress.Id, 1, 1); err != nil {
		t.Fatal(err)
	}
	inbound := createTrafficTestInbound(t, 37951, "ss2022-consistency-inbound")
	policySvc := &TrafficPolicyService{}
	policy, err := policySvc.Create(&n5model.TrafficPolicy{
		Name:              "ss2022-consistency-policy",
		Enabled:           true,
		DefaultTargetType: targetTypeEgress,
		DefaultTargetId:   egress.Id,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := policySvc.AddRule(&n5model.TrafficPolicyRule{
		PolicyId:   policy.Id,
		RuleType:   ruleTypeDomain,
		MatchMode:  domainModeSuffix,
		MatchValue: "consistency.example",
		TargetType: targetTypePool,
		TargetId:   pool.Id,
		Enabled:    true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := policySvc.BindInboundPolicy(inbound.Id, policy.Id); err != nil {
		t.Fatal(err)
	}

	checks := map[string]string{
		"orphan policy":                `SELECT count(*) FROM n5_traffic_policies p LEFT JOIN n5_traffic_policy_bindings b ON b.policy_id = p.id WHERE b.id IS NULL`,
		"orphan rule":                  `SELECT count(*) FROM n5_traffic_policy_rules r LEFT JOIN n5_traffic_policies p ON p.id = r.policy_id WHERE p.id IS NULL`,
		"orphan binding":               `SELECT count(*) FROM n5_traffic_policy_bindings b LEFT JOIN n5_traffic_policies p ON p.id = b.policy_id WHERE p.id IS NULL`,
		"missing inbound binding":      `SELECT count(*) FROM n5_traffic_policy_bindings b LEFT JOIN inbounds i ON i.id = b.inbound_id WHERE i.id IS NULL`,
		"duplicate inbound binding":    `SELECT count(*) FROM (SELECT inbound_id FROM n5_traffic_policy_bindings GROUP BY inbound_id HAVING count(*) > 1)`,
		"pool member missing pool":     `SELECT count(*) FROM n5_egress_pool_members m LEFT JOIN n5_egress_pools p ON p.id = m.pool_id WHERE p.id IS NULL`,
		"pool member missing egress":   `SELECT count(*) FROM n5_egress_pool_members m LEFT JOIN n5_egresses e ON e.id = m.egress_id WHERE e.id IS NULL`,
		"policy missing egress target": `SELECT count(*) FROM n5_traffic_policies p LEFT JOIN n5_egresses e ON e.id = p.default_target_id WHERE p.default_target_type = 'egress' AND e.id IS NULL`,
		"policy missing pool target":   `SELECT count(*) FROM n5_traffic_policies p LEFT JOIN n5_egress_pools x ON x.id = p.default_target_id WHERE p.default_target_type = 'pool' AND x.id IS NULL`,
		"rule missing egress target":   `SELECT count(*) FROM n5_traffic_policy_rules r LEFT JOIN n5_egresses e ON e.id = r.target_id WHERE r.target_type = 'egress' AND e.id IS NULL`,
		"rule missing pool target":     `SELECT count(*) FROM n5_traffic_policy_rules r LEFT JOIN n5_egress_pools p ON p.id = r.target_id WHERE r.target_type = 'pool' AND p.id IS NULL`,
	}
	for name, query := range checks {
		if count := countRegressionRows(t, query); count != 0 {
			t.Errorf("%s count = %d, want 0", name, count)
		}
	}
}
