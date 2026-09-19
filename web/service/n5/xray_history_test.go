package n5

import (
	"testing"
	"x-ui/database"
	n5model "x-ui/database/model/n5"
)

func TestXrayHistoryServiceListReturnsNewestFirstAndHonorsLimit(t *testing.T) {
	initTestDB(t)

	items := []*n5model.XrayConfigHistory{
		{Source: "n5-merge", ConfigHash: "hash-1", ApplyStatus: "generated"},
		{Source: "n5-merge", ConfigHash: "hash-2", ApplyStatus: "validated"},
		{Source: "n5-merge", ConfigHash: "hash-3", ApplyStatus: "applied"},
	}
	for _, item := range items {
		if err := database.GetDB().Create(item).Error; err != nil {
			t.Fatalf("create history failed: %v", err)
		}
	}

	svc := &XrayHistoryService{}
	records, err := svc.List(2)
	if err != nil {
		t.Fatalf("list history failed: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("unexpected history count: %d", len(records))
	}
	if records[0].ConfigHash != "hash-3" || records[1].ConfigHash != "hash-2" {
		t.Fatalf("unexpected history order: %#v", records)
	}
}
