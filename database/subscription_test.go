package database

import (
	"path/filepath"
	"testing"

	"x-ui/database/model"
	n5model "x-ui/database/model/n5"
)

func TestInitDBMigratesSubscriptionTableWithoutDroppingExistingData(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "subscription-migrate.db")
	if err := InitDB(dbPath); err != nil {
		t.Fatalf("init db failed: %v", err)
	}

	db := GetDB()
	user := &model.User{Username: "legacy-user", Password: "legacy-pass"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	inbound := &model.Inbound{
		UserId:         user.Id,
		Remark:         "legacy-inbound",
		Enable:         true,
		Port:           23001,
		Protocol:       model.VMess,
		Settings:       `{"clients":[{"id":"11111111-1111-1111-1111-111111111111","alterId":0}]}`,
		StreamSettings: `{"network":"tcp","security":"none","tcpSettings":{"header":{"type":"none"}}}`,
		Tag:            "legacy-inbound-tag",
		Sniffing:       `{}`,
	}
	if err := db.Create(inbound).Error; err != nil {
		t.Fatalf("create inbound failed: %v", err)
	}
	setting := &model.Setting{Key: "legacy-key", Value: "legacy-value"}
	if err := db.Create(setting).Error; err != nil {
		t.Fatalf("create setting failed: %v", err)
	}
	egress := &n5model.Egress{
		Name:         "legacy-egress",
		Protocol:     "freedom",
		Enabled:      true,
		Tag:          "n5-egress-legacy",
		OutboundJSON: `{"protocol":"freedom","settings":{},"tag":"n5-egress-legacy"}`,
	}
	if err := db.Create(egress).Error; err != nil {
		t.Fatalf("create n5 egress failed: %v", err)
	}

	if err := InitDB(dbPath); err != nil {
		t.Fatalf("re-init db failed: %v", err)
	}

	db = GetDB()
	if !db.Migrator().HasTable(&model.Subscription{}) {
		t.Fatal("subscriptions table not created")
	}

	var inboundCount int64
	if err := db.Model(&model.Inbound{}).Where("id = ?", inbound.Id).Count(&inboundCount).Error; err != nil {
		t.Fatalf("count inbound failed: %v", err)
	}
	if inboundCount != 1 {
		t.Fatalf("expected inbound retained, got %d", inboundCount)
	}

	var settingCount int64
	if err := db.Model(&model.Setting{}).Where("key = ?", setting.Key).Count(&settingCount).Error; err != nil {
		t.Fatalf("count setting failed: %v", err)
	}
	if settingCount != 1 {
		t.Fatalf("expected setting retained, got %d", settingCount)
	}

	var egressCount int64
	if err := db.Model(&n5model.Egress{}).Where("id = ?", egress.Id).Count(&egressCount).Error; err != nil {
		t.Fatalf("count n5 egress failed: %v", err)
	}
	if egressCount != 1 {
		t.Fatalf("expected n5 egress retained, got %d", egressCount)
	}
}
