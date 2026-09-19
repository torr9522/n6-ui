package shadowsocks

import (
	"crypto/rand"
	"encoding/base64"
	"strings"

	"github.com/torr9522/n6-ui/util/common"
)

const (
	Method2022Blake3AES128GCM        = "2022-blake3-aes-128-gcm"
	Method2022Blake3AES256GCM        = "2022-blake3-aes-256-gcm"
	Method2022Blake3ChaCha20Poly1305 = "2022-blake3-chacha20-poly1305"
)

var shadowsocks2022KeyLengths = map[string]int{
	Method2022Blake3AES128GCM:        16,
	Method2022Blake3AES256GCM:        32,
	Method2022Blake3ChaCha20Poly1305: 32,
}

var legacyMethods = map[string]bool{
	"aes-128-gcm":             true,
	"aead_aes_128_gcm":        true,
	"aes-256-gcm":             true,
	"aead_aes_256_gcm":        true,
	"chacha20-poly1305":       true,
	"chacha20-ietf-poly1305":  true,
	"aead_chacha20_poly1305":  true,
	"xchacha20-poly1305":      true,
	"xchacha20-ietf-poly1305": true,
	"aead_xchacha20_poly1305": true,
	"none":                    true,
	"plain":                   true,
}

func NormalizeMethod(method string) string {
	return strings.ToLower(strings.TrimSpace(method))
}

func IsShadowsocks2022(method string) bool {
	_, ok := shadowsocks2022KeyLengths[NormalizeMethod(method)]
	return ok
}

func IsLegacyShadowsocks(method string) bool {
	return legacyMethods[NormalizeMethod(method)]
}

func IsSupportedMethod(method string) bool {
	return IsShadowsocks2022(method) || IsLegacyShadowsocks(method)
}

func KeyLength(method string) (int, bool) {
	length, ok := shadowsocks2022KeyLengths[NormalizeMethod(method)]
	return length, ok
}

func Generate2022Key(method string) (string, error) {
	length, ok := KeyLength(method)
	if !ok {
		return "", common.NewError("该加密方式不是受支持的 Shadowsocks 2022 单用户方式")
	}
	key := make([]byte, length)
	if _, err := rand.Read(key); err != nil {
		return "", common.NewErrorf("生成 Shadowsocks 2022 密钥失败: %v", err)
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

func ValidateCredential(method string, password string) error {
	method = NormalizeMethod(method)
	if method == "" {
		return common.NewError("Shadowsocks 加密方式不能为空")
	}
	if !IsSupportedMethod(method) {
		return common.NewError("不支持的 Shadowsocks 加密方式")
	}
	if password == "" {
		if IsShadowsocks2022(method) {
			return common.NewError("Shadowsocks 2022 密钥不能为空")
		}
		return common.NewError("Shadowsocks 密码不能为空")
	}
	if !IsShadowsocks2022(method) {
		return nil
	}

	decoded, err := base64.StdEncoding.DecodeString(password)
	if err != nil || base64.StdEncoding.EncodeToString(decoded) != password {
		return common.NewError("Shadowsocks 2022 密钥必须是标准且带填充的 Base64")
	}
	expected, _ := KeyLength(method)
	if len(decoded) != expected {
		return common.NewErrorf("Shadowsocks 2022 密钥解码后必须为 %d 字节", expected)
	}
	return nil
}
