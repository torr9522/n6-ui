package common

import (
	"errors"
	"testing"
)

func TestSafeErrorMessageSanitizesSQLiteBusy(t *testing.T) {
	tests := []string{
		"database is locked",
		"database table is locked: n5_traffic_policies",
		"SQLITE_BUSY: database is locked",
		"SQLITE_LOCKED",
	}
	for _, input := range tests {
		if got := SafeErrorMessage(errors.New(input)); got != "数据库当前正忙，请稍后重试。" {
			t.Fatalf("SafeErrorMessage(%q) = %q", input, got)
		}
	}
}

func TestSafeErrorMessagePreservesBusinessError(t *testing.T) {
	const input = "该入站已存在 AI 分流规则，请编辑已有规则"
	if got := SafeErrorMessage(errors.New(input)); got != input {
		t.Fatalf("SafeErrorMessage preserved = %q", got)
	}
}
