package common

import (
	"errors"
	"fmt"
	"github.com/torr9522/n6-ui/logger"
	"strings"
)

var CtxDone = errors.New("context done")

func NewErrorf(format string, a ...interface{}) error {
	msg := fmt.Sprintf(format, a...)
	return errors.New(msg)
}

func NewError(a ...interface{}) error {
	msg := fmt.Sprintln(a...)
	return errors.New(msg)
}

func SafeErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "database is locked") ||
		strings.Contains(lower, "database table is locked") ||
		strings.Contains(lower, "sqlite_busy") ||
		strings.Contains(lower, "sqlite_locked") {
		return "数据库当前正忙，请稍后重试。"
	}
	return msg
}

func Recover(msg string) interface{} {
	panicErr := recover()
	if panicErr != nil {
		if msg != "" {
			logger.Error(msg, "panic:", panicErr)
		}
	}
	return panicErr
}
