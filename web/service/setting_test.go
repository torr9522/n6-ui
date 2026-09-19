package service

import (
	"testing"
	"x-ui/database"
	"x-ui/database/model"
)

func TestUpdateAllSettingN5ExtensionPreservesOtherSettingsAndDoesNotDuplicateKey(t *testing.T) {
	initServiceTestDB(t)

	settingService := &SettingService{}

	original, err := settingService.GetAllSetting()
	if err != nil {
		t.Fatalf("get all setting failed: %v", err)
	}
	if original.WebPort != 54321 {
		t.Fatalf("unexpected default web port: %d", original.WebPort)
	}
	if original.TimeLocation != "Asia/Shanghai" {
		t.Fatalf("unexpected default time location: %s", original.TimeLocation)
	}

	original.N5XrayExtensionEnable = true
	if err := settingService.UpdateAllSetting(original); err != nil {
		t.Fatalf("enable n5 xray extension failed: %v", err)
	}

	var trueCount int64
	if err := database.GetDB().Model(&model.Setting{}).Where("key = ?", "n5XrayExtensionEnable").Count(&trueCount).Error; err != nil {
		t.Fatalf("count enabled n5 setting failed: %v", err)
	}
	if trueCount != 1 {
		t.Fatalf("unexpected n5 setting key count after enable: %d", trueCount)
	}

	current, err := settingService.GetAllSetting()
	if err != nil {
		t.Fatalf("get updated setting failed: %v", err)
	}
	if !current.N5XrayExtensionEnable {
		t.Fatal("expected n5 setting enabled")
	}
	if current.WebPort != 54321 {
		t.Fatalf("unexpected web port after enable: %d", current.WebPort)
	}
	if current.TimeLocation != "Asia/Shanghai" {
		t.Fatalf("unexpected time location after enable: %s", current.TimeLocation)
	}
	if current.XrayTemplateConfig == "" {
		t.Fatal("expected xray template config to remain populated")
	}

	current.N5XrayExtensionEnable = false
	if err := settingService.UpdateAllSetting(current); err != nil {
		t.Fatalf("disable n5 xray extension failed: %v", err)
	}

	var falseCount int64
	if err := database.GetDB().Model(&model.Setting{}).Where("key = ?", "n5XrayExtensionEnable").Count(&falseCount).Error; err != nil {
		t.Fatalf("count disabled n5 setting failed: %v", err)
	}
	if falseCount != 1 {
		t.Fatalf("unexpected n5 setting key count after disable: %d", falseCount)
	}

	finalSetting, err := settingService.GetAllSetting()
	if err != nil {
		t.Fatalf("get final setting failed: %v", err)
	}
	if finalSetting.N5XrayExtensionEnable {
		t.Fatal("expected n5 setting disabled")
	}
	if finalSetting.WebPort != 54321 {
		t.Fatalf("unexpected web port after disable: %d", finalSetting.WebPort)
	}
	if finalSetting.TimeLocation != "Asia/Shanghai" {
		t.Fatalf("unexpected time location after disable: %s", finalSetting.TimeLocation)
	}
}
