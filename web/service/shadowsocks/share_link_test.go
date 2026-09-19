package shadowsocks

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestShareLinkSS2022RoundTrip(t *testing.T) {
	methods := []string{
		Method2022Blake3AES128GCM,
		Method2022Blake3AES256GCM,
		Method2022Blake3ChaCha20Poly1305,
	}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			key, err := Generate2022Key(method)
			if err != nil {
				t.Fatal(err)
			}
			original := &ShareLink{Method: method, Password: key, Host: "2001:db8::1", Port: 8388, Name: "N5/测试? # + 100%"}
			value, err := BuildShareLink(original)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(value, "ss://") || !strings.Contains(value, "@[2001:db8::1]:8388#") {
				t.Fatalf("unexpected share link: %s", value)
			}
			parsed, err := ParseShareLink(value)
			if err != nil {
				t.Fatal(err)
			}
			if *parsed != *original {
				t.Fatalf("round trip mismatch: got %#v want %#v", parsed, original)
			}
		})
	}
}

func TestParseShareLinkCompatibilityForms(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(make([]byte, 16))
	userinfo := Method2022Blake3AES128GCM + ":" + key
	modernRaw := base64.RawURLEncoding.EncodeToString([]byte(userinfo))
	modernPadded := base64.URLEncoding.EncodeToString([]byte(userinfo))
	legacyRaw := base64.RawURLEncoding.EncodeToString([]byte(userinfo + "@example.com:8388"))
	legacyPadded := base64.URLEncoding.EncodeToString([]byte(userinfo + "@example.com:8388"))
	tests := []string{
		"ss://" + modernRaw + "@example.com:8388#raw",
		"ss://" + modernPadded + "@example.com:8388#padded",
		"ss://" + legacyRaw + "#legacy-raw",
		"ss://" + legacyPadded + "#legacy-padded",
	}
	for _, value := range tests {
		parsed, err := ParseShareLink(value)
		if err != nil {
			t.Fatalf("parse %q failed: %v", value, err)
		}
		if parsed.Method != Method2022Blake3AES128GCM || parsed.Password != key || parsed.Host != "example.com" || parsed.Port != 8388 {
			t.Fatalf("unexpected parsed link: %#v", parsed)
		}
	}
}

func TestParseShareLinkRejectsInvalidInput(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(make([]byte, 16))
	validUser := base64.RawURLEncoding.EncodeToString([]byte(Method2022Blake3AES128GCM + ":" + key))
	urlSafeKey := base64.URLEncoding.EncodeToString([]byte{0xfb, 0xff, 0xff, 0xfb, 0xff, 0xff, 0xfb, 0xff, 0xff, 0xfb, 0xff, 0xff, 0xfb, 0xff, 0xff, 0xfb})
	tests := []struct {
		value string
		want  string
	}{
		{"http://example.com", "ss://"},
		{"ss://invalid", "格式无效"},
		{"ss://" + validUser + "@example.com:0", "端口"},
		{"ss://" + validUser + "@:8388", "地址"},
		{"ss://" + validUser + "@bad/path:8388", "地址"},
		{"ss://" + validUser + "@example.com:8388?plugin=v2ray-plugin", "plugin"},
		{"ss://" + base64.RawURLEncoding.EncodeToString([]byte(Method2022Blake3AES128GCM+":bad!")) + "@example.com:8388", "Base64"},
		{"ss://" + base64.RawURLEncoding.EncodeToString([]byte(Method2022Blake3AES128GCM+":"+strings.TrimRight(key, "="))) + "@example.com:8388", "带填充"},
		{"ss://" + base64.RawURLEncoding.EncodeToString([]byte(Method2022Blake3AES128GCM+":"+urlSafeKey)) + "@example.com:8388", "Base64"},
	}
	for _, test := range tests {
		_, err := ParseShareLink(test.value)
		if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Fatalf("expected %q for %q, got %v", test.want, test.value, err)
		}
	}
}

func TestShareLinkLegacyPasswordRoundTrip(t *testing.T) {
	original := &ShareLink{Method: "aes-256-gcm", Password: "ordinary:legacy/password", Host: "legacy.example.com", Port: 443, Name: "legacy"}
	value, err := BuildShareLink(original)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseShareLink(value)
	if err != nil {
		t.Fatal(err)
	}
	if *parsed != *original {
		t.Fatalf("legacy round trip mismatch: %#v", parsed)
	}
}
