package service

import (
	"crypto/rand"
	"encoding/hex"
	"x-ui/util/common"
	"x-ui/xray"
)

type RealityService struct {
}

type RealityDefaultConfig struct {
	Template    string   `json:"template"`
	Dest        string   `json:"dest"`
	ServerNames []string `json:"serverNames"`
	PrivateKey  string   `json:"privateKey"`
	ShortIds    []string `json:"shortIds"`
	Fingerprint string   `json:"fingerprint"`
	PublicKey   string   `json:"publicKey"`
	SpiderX     string   `json:"spiderX"`
}

type realityTemplate struct {
	Dest        string
	ServerName  string
	Fingerprint string
	SpiderX     string
}

var realityTemplates = map[string]realityTemplate{
	"cloudflare": {
		Dest:        "www.cloudflare.com:443",
		ServerName:  "www.cloudflare.com",
		Fingerprint: "chrome",
		SpiderX:     "/",
	},
	"apple": {
		Dest:        "itunes.apple.com:443",
		ServerName:  "itunes.apple.com",
		Fingerprint: "chrome",
		SpiderX:     "/",
	},
	"google": {
		Dest:        "www.google.com:443",
		ServerName:  "www.google.com",
		Fingerprint: "chrome",
		SpiderX:     "/",
	},
	"microsoft": {
		Dest:        "www.microsoft.com:443",
		ServerName:  "www.microsoft.com",
		Fingerprint: "chrome",
		SpiderX:     "/",
	},
	"custom": {},
}

func (s *RealityService) GenerateX25519KeyPair() (*xray.X25519KeyPair, error) {
	return xray.GenerateX25519KeyPair()
}

func (s *RealityService) GenerateDefaultConfig(template string) (*RealityDefaultConfig, error) {
	if template == "" {
		template = "cloudflare"
	}
	templateConfig, ok := realityTemplates[template]
	if !ok {
		return nil, common.NewErrorf("unknown reality template")
	}

	keyPair, err := xray.GenerateX25519KeyPair()
	if err != nil {
		return nil, err
	}

	shortId, err := randomShortId()
	if err != nil {
		return nil, err
	}

	config := &RealityDefaultConfig{
		Template:    template,
		Dest:        templateConfig.Dest,
		ServerNames: []string{},
		PrivateKey:  keyPair.PrivateKey,
		ShortIds:    []string{shortId},
		Fingerprint: templateConfig.Fingerprint,
		PublicKey:   keyPair.PublicKey,
		SpiderX:     templateConfig.SpiderX,
	}
	if templateConfig.ServerName != "" {
		config.ServerNames = []string{templateConfig.ServerName}
	}
	return config, nil
}

func randomShortId() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", common.NewErrorf("shortId generate failed")
	}
	return hex.EncodeToString(b), nil
}
