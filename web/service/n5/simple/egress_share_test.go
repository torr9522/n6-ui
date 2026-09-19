package simple

import (
	"strings"
	"testing"

	n5service "x-ui/web/service/n5"
	ssservice "x-ui/web/service/shadowsocks"
)

func TestSimpleEgressShadowsocksShareRoundTrip(t *testing.T) {
	initSimpleTestDB(t)
	key, err := ssservice.Generate2022Key(ssservice.Method2022Blake3AES128GCM)
	if err != nil {
		t.Fatal(err)
	}
	svc := &EgressService{egressService: &n5service.EgressService{}, testService: &fakeTester{}}
	created, err := svc.CreateSimpleEgress(&CreateSimpleEgressRequest{
		Name:     "SS2022 测试 + 100%",
		Protocol: "ss",
		Address:  "2001:db8::10",
		Port:     8388,
		Method:   ssservice.Method2022Blake3AES128GCM,
		Password: key,
		Enabled:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	value, err := svc.ExportShadowsocksShareLink(created.Id)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := svc.ParseShadowsocksShareLink(value)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Name != "SS2022 测试 + 100%" || parsed.Method != ssservice.Method2022Blake3AES128GCM || parsed.Password != key || parsed.Address != "2001:db8::10" || parsed.Port != 8388 {
		t.Fatalf("share round trip mismatch: %#v", parsed)
	}
	if !parsed.Enabled || parsed.Protocol != "ss" {
		t.Fatalf("unexpected parsed defaults: %#v", parsed)
	}
}

func TestSimpleEgressShareImportFallbackAndValidation(t *testing.T) {
	key, err := ssservice.Generate2022Key(ssservice.Method2022Blake3AES256GCM)
	if err != nil {
		t.Fatal(err)
	}
	value, err := ssservice.BuildShareLink(&ssservice.ShareLink{
		Method: ssservice.Method2022Blake3AES256GCM, Password: key, Host: "ss.example.com", Port: 443,
	})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := (&EgressService{}).ParseShadowsocksShareLink(value)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Name != "ss.example.com" {
		t.Fatalf("unexpected fallback name: %#v", parsed)
	}
	_, err = (&EgressService{}).ParseShadowsocksShareLink("ss://invalid")
	if err == nil || !strings.Contains(err.Error(), "格式无效") {
		t.Fatalf("expected parse validation error, got %v", err)
	}
}

func TestSimpleEgressShareExportRejectsNonShadowsocks(t *testing.T) {
	initSimpleTestDB(t)
	svc := &EgressService{egressService: &n5service.EgressService{}, testService: &fakeTester{}}
	created, err := svc.CreateSimpleEgress(&CreateSimpleEgressRequest{
		Name: "socks", Protocol: "socks5", Address: "127.0.0.1", Port: 1080, Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.ExportShadowsocksShareLink(created.Id)
	if err == nil || !strings.Contains(err.Error(), "仅 Shadowsocks") {
		t.Fatalf("expected unsupported export error, got %v", err)
	}
}
