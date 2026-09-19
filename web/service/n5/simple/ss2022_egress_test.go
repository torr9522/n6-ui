package simple

import (
	"strings"
	"testing"

	n5service "github.com/torr9522/n6-ui/web/service/n5"
	ssservice "github.com/torr9522/n6-ui/web/service/shadowsocks"
)

func TestSimpleEgressServiceShadowsocks2022Methods(t *testing.T) {
	initSimpleTestDB(t)
	service := &EgressService{
		egressService: &n5service.EgressService{},
		testService:   &fakeTester{},
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
			created, err := service.CreateSimpleEgress(&CreateSimpleEgressRequest{
				Name:     "simple-" + method,
				Protocol: "ss",
				Address:  "ss.example.com",
				Port:     37300 + index,
				Method:   method,
				Password: key,
				Enabled:  true,
			})
			if err != nil {
				t.Fatalf("create Simple egress failed: %v", err)
			}
			if !created.Supported || created.Method != method || created.Password != key {
				t.Fatalf("unexpected Simple egress: %#v", created)
			}

			replacement, err := ssservice.Generate2022Key(method)
			if err != nil {
				t.Fatal(err)
			}
			updated, err := service.UpdateSimpleEgress(created.Id, &CreateSimpleEgressRequest{
				Name:     created.Name + "-edited",
				Protocol: "ss",
				Address:  "edited.example.com",
				Port:     37400 + index,
				Method:   method,
				Password: replacement,
				Enabled:  true,
			})
			if err != nil {
				t.Fatalf("update Simple egress failed: %v", err)
			}
			if updated.InternalTag != created.InternalTag || updated.Password != replacement {
				t.Fatalf("unexpected updated Simple egress: %#v", updated)
			}
		})
	}
}

func TestSimpleEgressServiceRejectsInvalidShadowsocks2022Key(t *testing.T) {
	initSimpleTestDB(t)
	service := &EgressService{
		egressService: &n5service.EgressService{},
		testService:   &fakeTester{},
	}
	_, err := service.CreateSimpleEgress(&CreateSimpleEgressRequest{
		Name:     "invalid-ss2022",
		Protocol: "ss",
		Address:  "ss.example.com",
		Port:     8388,
		Method:   ssservice.Method2022Blake3AES128GCM,
		Password: "not-base64",
		Enabled:  true,
	})
	if err == nil || !strings.Contains(err.Error(), "Base64") {
		t.Fatalf("expected Base64 validation error, got %v", err)
	}
}

func TestSimpleEgressServiceLegacyShadowsocksPasswordStillWorks(t *testing.T) {
	initSimpleTestDB(t)
	service := &EgressService{
		egressService: &n5service.EgressService{},
		testService:   &fakeTester{},
	}
	created, err := service.CreateSimpleEgress(&CreateSimpleEgressRequest{
		Name:     "legacy-ss",
		Protocol: "ss",
		Address:  "legacy.example.com",
		Port:     8388,
		Method:   "aes-256-gcm",
		Password: "ordinary legacy password",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("legacy Shadowsocks create failed: %v", err)
	}
	if created.Password != "ordinary legacy password" {
		t.Fatalf("legacy password changed: %#v", created)
	}
}
