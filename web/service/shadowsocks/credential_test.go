package shadowsocks

import (
	"encoding/base64"
	"strings"
	"testing"
)

func fixedKey(length int) string {
	return base64.StdEncoding.EncodeToString(make([]byte, length))
}

func TestShadowsocks2022MethodClassification(t *testing.T) {
	methods := []string{
		Method2022Blake3AES128GCM,
		Method2022Blake3AES256GCM,
		Method2022Blake3ChaCha20Poly1305,
	}
	for _, method := range methods {
		if !IsShadowsocks2022(method) || IsLegacyShadowsocks(method) || !IsSupportedMethod(method) {
			t.Fatalf("unexpected classification for %s", method)
		}
	}
	if !IsShadowsocks2022(" 2022-BLAKE3-AES-128-GCM ") {
		t.Fatal("method classification should normalize case and whitespace")
	}
	if !IsLegacyShadowsocks("aes-256-gcm") || IsShadowsocks2022("aes-256-gcm") {
		t.Fatal("legacy method classification changed")
	}
	if IsSupportedMethod("2022-not-a-method") {
		t.Fatal("unknown method must not be supported")
	}
}

func TestGenerate2022KeyUsesRequiredLength(t *testing.T) {
	tests := []struct {
		method string
		length int
	}{
		{Method2022Blake3AES128GCM, 16},
		{Method2022Blake3AES256GCM, 32},
		{Method2022Blake3ChaCha20Poly1305, 32},
	}
	for _, test := range tests {
		t.Run(test.method, func(t *testing.T) {
			first, err := Generate2022Key(test.method)
			if err != nil {
				t.Fatalf("generate key failed: %v", err)
			}
			second, err := Generate2022Key(test.method)
			if err != nil {
				t.Fatalf("generate second key failed: %v", err)
			}
			if first == second {
				t.Fatal("independent generated keys unexpectedly match")
			}
			decoded, err := base64.StdEncoding.DecodeString(first)
			if err != nil || len(decoded) != test.length {
				t.Fatalf("unexpected generated key: decoded length=%d err=%v", len(decoded), err)
			}
			if err := ValidateCredential(test.method, first); err != nil {
				t.Fatalf("generated key did not validate: %v", err)
			}
		})
	}
}

func TestValidateCredential(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		password string
		want     string
	}{
		{"aes128 valid", Method2022Blake3AES128GCM, fixedKey(16), ""},
		{"aes256 valid", Method2022Blake3AES256GCM, fixedKey(32), ""},
		{"chacha valid", Method2022Blake3ChaCha20Poly1305, fixedKey(32), ""},
		{"invalid base64", Method2022Blake3AES128GCM, "not base64!", "Base64"},
		{"raw base64", Method2022Blake3AES128GCM, strings.TrimRight(fixedKey(16), "="), "Base64"},
		{"wrong aes128 length", Method2022Blake3AES128GCM, fixedKey(15), "16 字节"},
		{"wrong aes256 length", Method2022Blake3AES256GCM, fixedKey(16), "32 字节"},
		{"empty 2022 key", Method2022Blake3AES128GCM, "", "密钥不能为空"},
		{"legacy ordinary password", "aes-256-gcm", "ordinary password / 123", ""},
		{"legacy empty password", "aes-256-gcm", "", "密码不能为空"},
		{"unknown method", "2022-not-a-method", fixedKey(32), "不支持"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateCredential(test.method, test.password)
			if test.want == "" {
				if err != nil {
					t.Fatalf("unexpected validation error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}
}

func TestMethodSwitchRequiresMatchingKeyLength(t *testing.T) {
	aes128Key, err := Generate2022Key(Method2022Blake3AES128GCM)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateCredential(Method2022Blake3AES256GCM, aes128Key); err == nil || !strings.Contains(err.Error(), "32 字节") {
		t.Fatalf("AES-128 key must be rejected after switching to AES-256: %v", err)
	}
	aes256Key, err := Generate2022Key(Method2022Blake3AES256GCM)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateCredential(Method2022Blake3AES256GCM, aes256Key); err != nil {
		t.Fatalf("replacement AES-256 key must validate: %v", err)
	}
}
