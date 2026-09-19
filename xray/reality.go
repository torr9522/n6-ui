package xray

import (
	"os/exec"
	"strings"
	"x-ui/util/common"
)

type X25519KeyPair struct {
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

func GenerateX25519KeyPair() (*X25519KeyPair, error) {
	cmd := exec.Command(GetBinaryPath(), "x25519")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, common.NewErrorf("x25519 command failed")
	}

	keyPair := &X25519KeyPair{}
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "PrivateKey:") {
			keyPair.PrivateKey = strings.TrimSpace(strings.TrimPrefix(line, "PrivateKey:"))
		}
		if strings.HasPrefix(line, "Password (PublicKey):") {
			keyPair.PublicKey = strings.TrimSpace(strings.TrimPrefix(line, "Password (PublicKey):"))
		}
	}

	if keyPair.PrivateKey == "" || keyPair.PublicKey == "" {
		return nil, common.NewErrorf("x25519 output parse failed")
	}
	return keyPair, nil
}
