package shadowsocks

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/torr9522/n6-ui/util/common"
)

type ShareLink struct {
	Method   string
	Password string
	Host     string
	Port     int
	Name     string
}

func BuildShareLink(link *ShareLink) (string, error) {
	if err := ValidateShareLink(link); err != nil {
		return "", err
	}
	userinfo := base64.RawURLEncoding.EncodeToString([]byte(NormalizeMethod(link.Method) + ":" + link.Password))
	value := "ss://" + userinfo + "@" + net.JoinHostPort(strings.TrimSpace(link.Host), strconv.Itoa(link.Port))
	parsed, err := url.Parse(value)
	if err != nil {
		return "", common.NewError("Shadowsocks 分享链接生成失败")
	}
	parsed.Fragment = strings.TrimSpace(link.Name)
	return parsed.String(), nil
}

func ParseShareLink(value string) (*ShareLink, error) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(strings.ToLower(value), "ss://") {
		return nil, common.NewError("Shadowsocks 分享链接必须以 ss:// 开头")
	}
	raw := value[len("ss://"):]
	fragment := ""
	if index := strings.Index(raw, "#"); index >= 0 {
		encoded := raw[index+1:]
		raw = raw[:index]
		decoded, err := url.PathUnescape(encoded)
		if err != nil {
			return nil, common.NewError("Shadowsocks 分享链接名称编码无效")
		}
		fragment = decoded
	}
	if index := strings.Index(raw, "?"); index >= 0 {
		query, err := url.ParseQuery(raw[index+1:])
		if err != nil {
			return nil, common.NewError("Shadowsocks 分享链接参数无效")
		}
		if strings.TrimSpace(query.Get("plugin")) != "" {
			return nil, common.NewError("当前不支持 Shadowsocks plugin 分享链接")
		}
		raw = raw[:index]
	}

	userinfo := ""
	endpoint := ""
	if index := strings.LastIndex(raw, "@"); index >= 0 {
		userinfo = raw[:index]
		endpoint = raw[index+1:]
		decoded, err := url.PathUnescape(userinfo)
		if err != nil {
			return nil, common.NewError("Shadowsocks 分享链接认证信息编码无效")
		}
		if strings.Contains(decoded, ":") {
			userinfo = decoded
		} else {
			userinfo, err = decodeShareBase64(decoded)
			if err != nil {
				return nil, common.NewError("Shadowsocks 分享链接认证信息无效")
			}
		}
	} else {
		decoded, err := decodeShareBase64(raw)
		if err != nil {
			return nil, common.NewError("Shadowsocks 分享链接格式无效")
		}
		index := strings.LastIndex(decoded, "@")
		if index < 0 {
			return nil, common.NewError("Shadowsocks 分享链接格式无效")
		}
		userinfo = decoded[:index]
		endpoint = decoded[index+1:]
	}

	index := strings.Index(userinfo, ":")
	if index <= 0 {
		return nil, common.NewError("Shadowsocks 分享链接认证信息无效")
	}
	host, portText, err := net.SplitHostPort(endpoint)
	if err != nil || strings.TrimSpace(host) == "" {
		return nil, common.NewError("Shadowsocks 分享链接地址无效")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return nil, common.NewError("Shadowsocks 分享链接端口无效")
	}
	link := &ShareLink{
		Method:   NormalizeMethod(userinfo[:index]),
		Password: userinfo[index+1:],
		Host:     host,
		Port:     port,
		Name:     fragment,
	}
	if err := ValidateShareLink(link); err != nil {
		return nil, err
	}
	return link, nil
}

func ValidateShareLink(link *ShareLink) error {
	if link == nil {
		return common.NewError("Shadowsocks 分享链接不能为空")
	}
	host := strings.TrimSpace(link.Host)
	if host == "" {
		return common.NewError("Shadowsocks 分享链接地址不能为空")
	}
	if host != link.Host || strings.ContainsAny(host, "\r\n\t /?#@[]") {
		return common.NewError("Shadowsocks 分享链接地址无效")
	}
	if strings.Contains(host, ":") && net.ParseIP(host) == nil {
		return common.NewError("Shadowsocks 分享链接地址无效")
	}
	if link.Port < 1 || link.Port > 65535 {
		return common.NewError("Shadowsocks 分享链接端口无效")
	}
	return ValidateCredential(link.Method, link.Password)
}

func decodeShareBase64(value string) (string, error) {
	value = strings.TrimSpace(value)
	encodings := []*base64.Encoding{
		base64.RawURLEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.StdEncoding,
	}
	for _, encoding := range encodings {
		decoded, err := encoding.DecodeString(value)
		if err == nil {
			return string(decoded), nil
		}
	}
	return "", fmt.Errorf("invalid base64")
}
