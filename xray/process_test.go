package xray

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestProcessStartReturnsErrorWhenBinaryExitsImmediately(t *testing.T) {
	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("mkdir bin failed: %v", err)
	}
	binaryPath := filepath.Join(binDir, GetBinaryName())
	script := "#!/bin/sh\n" +
		"echo 'startup failure' >&2\n" +
		"exit 1\n"
	if err := os.WriteFile(binaryPath, []byte(script), 0755); err != nil {
		t.Fatalf("write fake binary failed: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	process := NewProcess(&Config{})
	if err := process.Start(); err == nil {
		t.Fatal("expected process start to fail")
	}
}

func TestProcessStartReturnsNilForLongRunningBinary(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("test expects a POSIX shell")
	}

	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("mkdir bin failed: %v", err)
	}
	binaryPath := filepath.Join(binDir, GetBinaryName())
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"-version\" ]; then\n" +
		"  echo 'Xray 0.0.0'\n" +
		"  exit 0\n" +
		"fi\n" +
		"sleep 5\n"
	if err := os.WriteFile(binaryPath, []byte(script), 0755); err != nil {
		t.Fatalf("write fake binary failed: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	process := NewProcess(&Config{})
	if err := process.Start(); err != nil {
		t.Fatalf("expected process start to succeed: %v", err)
	}
	defer func() {
		_ = process.Stop()
	}()

	time.Sleep(100 * time.Millisecond)
	if !process.IsRunning() {
		t.Fatal("expected process to still be running")
	}
}
