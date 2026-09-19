package database

import (
	"encoding/json"
	"path/filepath"
	"testing"

	n5model "x-ui/database/model/n5"
)

func initN5TestDB(t *testing.T) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "n5_phase2.db")
	if err := InitDB(dbPath); err != nil {
		t.Fatalf("init db failed: %v", err)
	}
}

func TestMigrateN5StableTagsMigratesLegacyRows(t *testing.T) {
	initN5TestDB(t)

	legacyEgress := &n5model.Egress{
		Id:           1,
		Name:         "legacy-egress",
		Protocol:     "freedom",
		Enabled:      true,
		Tag:          "n5-egress-1",
		OutboundJSON: `{"protocol":"freedom","settings":{},"tag":"n5-egress-1"}`,
	}
	if err := db.Create(legacyEgress).Error; err != nil {
		t.Fatalf("create legacy egress failed: %v", err)
	}

	newEgress := &n5model.Egress{
		Id:           2,
		Name:         "new-egress",
		Protocol:     "freedom",
		Enabled:      true,
		Tag:          "n5-egress-0000000002",
		OutboundJSON: `{"protocol":"freedom","settings":{},"tag":"n5-egress-0000000002"}`,
	}
	if err := db.Create(newEgress).Error; err != nil {
		t.Fatalf("create new egress failed: %v", err)
	}

	legacyPool := &n5model.EgressPool{
		Id:       1,
		Name:     "legacy-pool",
		Strategy: "random",
		Enabled:  true,
		Tag:      "n5-pool-1",
	}
	if err := db.Create(legacyPool).Error; err != nil {
		t.Fatalf("create legacy pool failed: %v", err)
	}

	newPool := &n5model.EgressPool{
		Id:       2,
		Name:     "new-pool",
		Strategy: "random",
		Enabled:  true,
		Tag:      "n5-pool-0000000002",
	}
	if err := db.Create(newPool).Error; err != nil {
		t.Fatalf("create new pool failed: %v", err)
	}

	if err := migrateN5StableTags(); err != nil {
		t.Fatalf("migrate stable tags failed: %v", err)
	}

	migratedEgress := &n5model.Egress{}
	if err := db.First(migratedEgress, legacyEgress.Id).Error; err != nil {
		t.Fatalf("load migrated egress failed: %v", err)
	}
	if migratedEgress.Tag != "n5-egress-0000000001" {
		t.Fatalf("unexpected migrated egress tag: %s", migratedEgress.Tag)
	}

	obj := make(map[string]interface{})
	if err := json.Unmarshal([]byte(migratedEgress.OutboundJSON), &obj); err != nil {
		t.Fatalf("unmarshal migrated outbound json failed: %v", err)
	}
	if obj["tag"] != "n5-egress-0000000001" {
		t.Fatalf("unexpected migrated outbound json tag: %#v", obj["tag"])
	}

	unchangedEgress := &n5model.Egress{}
	if err := db.First(unchangedEgress, newEgress.Id).Error; err != nil {
		t.Fatalf("load unchanged egress failed: %v", err)
	}
	if unchangedEgress.Tag != "n5-egress-0000000002" {
		t.Fatalf("unexpected unchanged egress tag: %s", unchangedEgress.Tag)
	}

	migratedPool := &n5model.EgressPool{}
	if err := db.First(migratedPool, legacyPool.Id).Error; err != nil {
		t.Fatalf("load migrated pool failed: %v", err)
	}
	if migratedPool.Tag != "n5-pool-0000000001" {
		t.Fatalf("unexpected migrated pool tag: %s", migratedPool.Tag)
	}

	unchangedPool := &n5model.EgressPool{}
	if err := db.First(unchangedPool, newPool.Id).Error; err != nil {
		t.Fatalf("load unchanged pool failed: %v", err)
	}
	if unchangedPool.Tag != "n5-pool-0000000002" {
		t.Fatalf("unexpected unchanged pool tag: %s", unchangedPool.Tag)
	}
}
