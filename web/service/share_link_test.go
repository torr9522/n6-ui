package service

import (
	"encoding/base64"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
	"x-ui/database/model"
)

func TestShareLinkServiceProtocolMatrixAndRealitySafety(t *testing.T) {
	inbounds := []*model.Inbound{
		createVMessTCPInbound(31101, "vmess-tcp"),
		createVMessWSInbound(31102, "vmess-ws"),
		createVLESSTCPInbound(31103, "vless-tcp"),
		createVLESSTLSInbound(31104, "vless-tls"),
		createVLESSRealityTCPInbound(31105, "vless-reality-tcp"),
		createVLESSRealityGRPCInbound(31106, "vless-reality-grpc"),
		createTrojanTLSInbound(31107, "trojan-tls"),
		createShadowsocksInbound(31108, "ss"),
	}
	svc := &ShareLinkService{
		loadShareAddresses: func() ([]ShareAddress, error) {
			return []ShareAddress{{Id: "share-1", Type: "domain", Address: "share.example.com", Enabled: true}}, nil
		},
	}

	links, err := svc.GenerateLinks(inbounds, shareContext{RequestHost: "127.0.0.1:54321"})
	if err != nil {
		t.Fatalf("generate links failed: %v", err)
	}
	if len(links) != 8 {
		t.Fatalf("unexpected link count: %d", len(links))
	}
	expectedPrefixes := []string{"vmess://", "vmess://", "vless://", "vless://", "vless://", "vless://", "trojan://", "ss://"}
	for i, prefix := range expectedPrefixes {
		if !strings.HasPrefix(links[i], prefix) {
			t.Fatalf("link %d does not start with %s: %s", i, prefix, links[i])
		}
	}
	if strings.Contains(strings.Join(links, "\n"), "PRIVATE_KEY") || strings.Contains(strings.Join(links, "\n"), "privateKey") {
		t.Fatal("share links leaked reality private key")
	}
	if !strings.Contains(links[4], "pbk=PUBLIC_KEY") || !strings.Contains(links[4], "sid=012345") || !strings.Contains(links[4], "flow=xtls-rprx-vision") {
		t.Fatalf("unexpected reality tcp link: %s", links[4])
	}
	if !strings.Contains(links[5], "serviceName=grpc-service") {
		t.Fatalf("unexpected reality grpc link: %s", links[5])
	}

	subscriptionBase64, err := svc.GenerateBase64(inbounds, shareContext{RequestHost: "127.0.0.1:54321"})
	if err != nil {
		t.Fatalf("generate subscription base64 failed: %v", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(subscriptionBase64)
	if err != nil {
		t.Fatalf("decode subscription base64 failed: %v", err)
	}
	text := string(decoded)
	for _, prefix := range []string{"vmess://", "vless://", "trojan://", "ss://"} {
		if !strings.Contains(text, prefix) {
			t.Fatalf("decoded output missing %s: %s", prefix, text)
		}
	}
	if strings.Contains(text, "PRIVATE_KEY") || strings.Contains(text, "privateKey") {
		t.Fatal("decoded base64 leaked reality private key")
	}

	clashYAML, err := svc.GenerateClash(inbounds, shareContext{RequestHost: "127.0.0.1:54321"})
	if err != nil {
		t.Fatalf("generate clash failed: %v", err)
	}
	if !strings.Contains(clashYAML, "proxies:") || !strings.Contains(clashYAML, "proxy-groups:") || !strings.Contains(clashYAML, "rules:") {
		t.Fatalf("unexpected clash yaml: %s", clashYAML)
	}
	if strings.Contains(clashYAML, "PRIVATE_KEY") || strings.Contains(clashYAML, "privateKey") {
		t.Fatal("clash yaml leaked reality private key")
	}

	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(clashYAML), &parsed); err != nil {
		t.Fatalf("parse clash yaml failed: %v", err)
	}
	proxies, ok := parsed["proxies"].([]interface{})
	if !ok || len(proxies) != 8 {
		t.Fatalf("unexpected proxies: %#v", parsed["proxies"])
	}
	foundReality := false
	for _, item := range proxies {
		proxy := item.(map[interface{}]interface{})
		if proxy["type"] == "vless" && proxy["flow"] == "xtls-rprx-vision" {
			foundReality = true
			if proxy["client-fingerprint"] != "chrome" {
				t.Fatalf("unexpected client fingerprint: %#v", proxy)
			}
			realityOpts, ok := proxy["reality-opts"].(map[interface{}]interface{})
			if !ok {
				t.Fatalf("missing reality opts: %#v", proxy)
			}
			if realityOpts["public-key"] != "PUBLIC_KEY" {
				t.Fatalf("unexpected public key: %#v", realityOpts)
			}
			if shortID, ok := realityOpts["short-id"].(string); !ok || shortID != "012345" {
				t.Fatalf("unexpected short id type/value: %#v", realityOpts["short-id"])
			}
		}
	}
	if !foundReality {
		t.Fatal("expected reality proxy in clash output")
	}
}

func TestShareLinkServiceClashQuotesDuplicateNames(t *testing.T) {
	inbounds := []*model.Inbound{
		createVLESSRealityTCPInbound(31201, "Node's"),
		createTrojanTLSInbound(31202, "Node's"),
	}
	svc := &ShareLinkService{
		loadShareAddresses: func() ([]ShareAddress, error) {
			return []ShareAddress{{Id: "share-1", Type: "domain", Address: "share.example.com", Enabled: true}}, nil
		},
	}
	clashYAML, err := svc.GenerateClash(inbounds, shareContext{RequestHost: "127.0.0.1:54321"})
	if err != nil {
		t.Fatalf("generate clash failed: %v", err)
	}
	if !strings.Contains(clashYAML, "name: 'Node''s'") {
		t.Fatalf("expected quoted name in yaml: %s", clashYAML)
	}
	if !strings.Contains(clashYAML, "name: 'Node''s #2'") {
		t.Fatalf("expected duplicate name suffix in yaml: %s", clashYAML)
	}
}

func TestShareLinkServiceRejectsNonPublicShareAddresses(t *testing.T) {
	inbound := createVMessTCPInbound(31301, "vmess-private")
	svc := &ShareLinkService{
		loadShareAddresses: func() ([]ShareAddress, error) {
			return []ShareAddress{
				{Id: "share-1", Type: "domain", Address: "localhost", Enabled: true},
				{Id: "share-2", Type: "domain", Address: "192.168.1.1", Enabled: true},
			}, nil
		},
	}
	_, err := svc.GenerateLinks([]*model.Inbound{inbound}, shareContext{RequestHost: "127.0.0.1:54321"})
	if err == nil || !strings.Contains(err.Error(), "no public share address configured") {
		t.Fatalf("expected no public share address error, got: %v", err)
	}
}
