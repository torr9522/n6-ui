package n5

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/torr9522/n6-ui/database"
	legacyModel "github.com/torr9522/n6-ui/database/model"
	n5model "github.com/torr9522/n6-ui/database/model/n5"
	"github.com/torr9522/n6-ui/logger"

	logging "github.com/op/go-logging"
	"gorm.io/gorm"
)

func TestRestoredHistoricalDatabaseBindingMatrix(t *testing.T) {
	tests := []struct {
		name          string
		validBindings int
		danglingIDs   []int
	}{
		{name: "normal", validBindings: 1},
		{name: "single_orphan", danglingIDs: []int{999991}},
		{name: "multi_orphan", danglingIDs: []int{999991, 999992, 999999}},
		{name: "mixed", validBindings: 2, danglingIDs: []int{999999}},
		{name: "all_orphan", danglingIDs: []int{4, 6, 999999}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dbPath := filepath.Join(t.TempDir(), "restored.db")
			if err := database.InitDB(dbPath); err != nil {
				t.Fatalf("initialize fixture database failed: %v", err)
			}
			egress, err := (&EgressService{}).Create(&n5model.Egress{
				Name: "restore-egress", Protocol: "freedom", Enabled: true,
				OutboundJSON: freedomOutboundJSON(),
			})
			if err != nil {
				t.Fatalf("create restore egress failed: %v", err)
			}
			policy, err := (&TrafficPolicyService{}).Create(&n5model.TrafficPolicy{
				Name: "restore-policy", Enabled: true,
				DefaultTargetType: targetTypeEgress, DefaultTargetId: egress.Id,
			})
			if err != nil {
				t.Fatalf("create restore policy failed: %v", err)
			}

			validTags := make(map[string]bool, test.validBindings)
			for index := 0; index < test.validBindings; index++ {
				inbound := &legacyModel.Inbound{
					UserId: 1, Remark: fmt.Sprintf("restore-%d", index), Enable: true,
					Listen: "127.0.0.1", Port: 36100 + index, Protocol: legacyModel.Socks,
					Settings:       `{"auth":"noauth","udp":false,"ip":"127.0.0.1"}`,
					StreamSettings: `{}`, Tag: fmt.Sprintf("restore-inbound-%d", index), Sniffing: `{}`,
				}
				if err := database.GetDB().Create(inbound).Error; err != nil {
					t.Fatalf("create restore inbound failed: %v", err)
				}
				if err := database.GetDB().Create(&n5model.TrafficPolicyBinding{
					InboundId: inbound.Id, PolicyId: policy.Id, Enabled: true,
				}).Error; err != nil {
					t.Fatalf("create restore binding failed: %v", err)
				}
				validTags[inbound.Tag] = true
			}
			for _, inboundID := range test.danglingIDs {
				addDanglingBinding(t, inboundID, policy.Id)
			}

			sqlDB, err := database.GetDB().DB()
			if err != nil {
				t.Fatalf("get fixture sql database failed: %v", err)
			}
			if err := sqlDB.Close(); err != nil {
				t.Fatalf("close fixture database failed: %v", err)
			}
			if err := database.InitDB(dbPath); err != nil {
				t.Fatalf("open restored database failed: %v", err)
			}

			routing, err := (&XrayExtService{}).GenerateRoutingFragments()
			if err != nil {
				t.Fatalf("generate routing from restored database failed: %v", err)
			}
			tags := routingInboundTags(t, routing)
			if len(tags) != len(validTags) {
				t.Fatalf("restored routing tag count mismatch: got %#v want %#v", tags, validTags)
			}
			for tag := range validTags {
				if tags[tag] != 1 {
					t.Fatalf("restored routing lost valid tag %q: %#v", tag, routing)
				}
			}
		})
	}
}

func createIntegrityRoutingFixture(t *testing.T, inboundCount int) ([]*legacyModel.Inbound, *n5model.TrafficPolicy) {
	t.Helper()
	initTestDB(t)

	egress, err := (&EgressService{}).Create(&n5model.Egress{
		Name: "integrity-egress", Protocol: "freedom", Enabled: true,
		OutboundJSON: freedomOutboundJSON(),
	})
	if err != nil {
		t.Fatalf("create egress failed: %v", err)
	}
	policy, err := (&TrafficPolicyService{}).Create(&n5model.TrafficPolicy{
		Name: "integrity-policy", Enabled: true,
		DefaultTargetType: targetTypeEgress, DefaultTargetId: egress.Id,
	})
	if err != nil {
		t.Fatalf("create policy failed: %v", err)
	}

	inbounds := make([]*legacyModel.Inbound, 0, inboundCount)
	for index := 0; index < inboundCount; index++ {
		inbound := &legacyModel.Inbound{
			UserId: 1, Remark: fmt.Sprintf("integrity-%d", index), Enable: true,
			Listen: "127.0.0.1", Port: 36000 + index, Protocol: legacyModel.Socks,
			Settings:       `{"auth":"noauth","udp":false,"ip":"127.0.0.1"}`,
			StreamSettings: `{}`, Tag: fmt.Sprintf("integrity-inbound-%d", index), Sniffing: `{}`,
		}
		if err := database.GetDB().Create(inbound).Error; err != nil {
			t.Fatalf("create inbound failed: %v", err)
		}
		if err := database.GetDB().Create(&n5model.TrafficPolicyBinding{
			InboundId: inbound.Id, PolicyId: policy.Id, Enabled: true,
		}).Error; err != nil {
			t.Fatalf("create binding failed: %v", err)
		}
		inbounds = append(inbounds, inbound)
	}
	return inbounds, policy
}

func addDanglingBinding(t *testing.T, inboundID, policyID int) *n5model.TrafficPolicyBinding {
	t.Helper()
	binding := &n5model.TrafficPolicyBinding{InboundId: inboundID, PolicyId: policyID, Enabled: true}
	if err := database.GetDB().Create(binding).Error; err != nil {
		t.Fatalf("create dangling binding failed: %v", err)
	}
	return binding
}

func routingInboundTags(t *testing.T, routing map[string]interface{}) map[string]int {
	t.Helper()
	tags := make(map[string]int)
	for _, rawRule := range routing["rules"].([]interface{}) {
		rule := rawRule.(map[string]interface{})
		for _, rawTag := range rule["inboundTag"].([]interface{}) {
			tags[rawTag.(string)]++
		}
	}
	return tags
}

func TestGenerateConfigPreservesNormalTrafficPolicyBinding(t *testing.T) {
	inbounds, _ := createIntegrityRoutingFixture(t, 1)
	routing, err := (&XrayExtService{}).GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("normal binding generation failed: %v", err)
	}
	if tags := routingInboundTags(t, routing); tags[inbounds[0].Tag] != 1 {
		t.Fatalf("normal inbound routing was not preserved: %#v", routing)
	}
}

func TestGenerateConfigSkipsDanglingTrafficPolicyBinding(t *testing.T) {
	_, policy := createIntegrityRoutingFixture(t, 0)
	addDanglingBinding(t, 999999, policy.Id)
	routing, err := (&XrayExtService{}).GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("dangling binding must not fail generation: %v", err)
	}
	if len(routing["rules"].([]interface{})) != 0 {
		t.Fatalf("dangling binding generated routing rules: %#v", routing)
	}
}

func TestGenerateConfigPreservesValidBindingsWhenOneIsDangling(t *testing.T) {
	inbounds, policy := createIntegrityRoutingFixture(t, 2)
	addDanglingBinding(t, 999999, policy.Id)
	routing, err := (&XrayExtService{}).GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("mixed binding generation failed: %v", err)
	}
	tags := routingInboundTags(t, routing)
	for _, inbound := range inbounds {
		if tags[inbound.Tag] != 1 {
			t.Fatalf("valid inbound routing was not preserved for %s: %#v", inbound.Tag, routing)
		}
	}
	if len(tags) != 2 {
		t.Fatalf("dangling binding leaked into routing: %#v", routing)
	}
}

func TestGenerateConfigWithAllDanglingBindingsProducesBaseFragment(t *testing.T) {
	_, policy := createIntegrityRoutingFixture(t, 0)
	for _, inboundID := range []int{4, 6, 7, 999999} {
		addDanglingBinding(t, inboundID, policy.Id)
	}
	routing, err := (&XrayExtService{}).GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("all-dangling generation failed: %v", err)
	}
	if len(routing["rules"].([]interface{})) != 0 {
		t.Fatalf("all-dangling bindings generated rules: %#v", routing)
	}
}

func TestGenerateConfigReturnsNonNotFoundDatabaseErrors(t *testing.T) {
	createIntegrityRoutingFixture(t, 1)
	db := database.GetDB()
	wantErr := errors.New("injected inbound query failure")
	callbackName := "n6:test_inbound_query_error"
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == "inbounds" {
			tx.AddError(wantErr)
		}
	}); err != nil {
		t.Fatalf("register query callback failed: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Query().Remove(callbackName) })

	_, err := (&XrayExtService{}).GenerateRoutingFragments()
	if !errors.Is(err, wantErr) {
		t.Fatalf("non-not-found error was not preserved: got %v want %v", err, wantErr)
	}
}

func TestDanglingBindingWarningIncludesBindingInboundAndPolicyIDs(t *testing.T) {
	_, policy := createIntegrityRoutingFixture(t, 0)
	binding := addDanglingBinding(t, 999999, policy.Id)
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("create log pipe failed: %v", err)
	}
	originalStderr := os.Stderr
	os.Stderr = writePipe
	logger.InitLogger(logging.INFO)
	t.Cleanup(func() {
		os.Stderr = originalStderr
		logger.InitLogger(logging.INFO)
	})

	if _, err := (&XrayExtService{}).GenerateRoutingFragments(); err != nil {
		t.Fatalf("dangling binding generation failed: %v", err)
	}
	_ = writePipe.Close()
	logData, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatalf("read warning log failed: %v", err)
	}
	_ = readPipe.Close()
	for _, field := range []string{
		fmt.Sprintf("binding_id=%d", binding.Id),
		fmt.Sprintf("inbound_id=%d", binding.InboundId),
		fmt.Sprintf("policy_id=%d", binding.PolicyId),
	} {
		if !strings.Contains(string(logData), field) {
			t.Fatalf("warning missing %q: %s", field, logData)
		}
	}
}
