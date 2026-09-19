package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var shareAddressPath = "/etc/x-ui/share_addresses.json"

func SetShareAddressPathForTest(path string) string {
	old := shareAddressPath
	shareAddressPath = path
	return old
}

type ShareAddress struct {
	Id        string    `json:"id"`
	Type      string    `json:"type"`
	Address   string    `json:"address"`
	Remark    string    `json:"remark"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ShareAddressService struct {
}

func (s *ShareAddressService) GetAll() ([]ShareAddress, error) {
	return s.read()
}

func (s *ShareAddressService) Add(address string, remark string) (*ShareAddress, error) {
	address, err := normalizeShareAddress(address)
	if err != nil {
		return nil, err
	}

	addresses, err := s.read()
	if err != nil {
		return nil, err
	}
	if hasShareAddress(addresses, address, "") {
		return nil, errors.New("share address already exists")
	}

	now := time.Now().UTC()
	item := ShareAddress{
		Id:        newShareAddressId(),
		Type:      "domain",
		Address:   address,
		Remark:    strings.TrimSpace(remark),
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	addresses = append(addresses, item)
	if err := s.write(addresses); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *ShareAddressService) Update(id string, address string, remark string) (*ShareAddress, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("id is empty")
	}

	address, err := normalizeShareAddress(address)
	if err != nil {
		return nil, err
	}

	addresses, err := s.read()
	if err != nil {
		return nil, err
	}
	if hasShareAddress(addresses, address, id) {
		return nil, errors.New("share address already exists")
	}

	for i := range addresses {
		if addresses[i].Id == id {
			addresses[i].Type = "domain"
			addresses[i].Address = address
			addresses[i].Remark = strings.TrimSpace(remark)
			addresses[i].UpdatedAt = time.Now().UTC()
			if err := s.write(addresses); err != nil {
				return nil, err
			}
			return &addresses[i], nil
		}
	}

	return nil, errors.New("share address not found")
}

func (s *ShareAddressService) Delete(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("id is empty")
	}

	addresses, err := s.read()
	if err != nil {
		return err
	}

	for i := range addresses {
		if addresses[i].Id == id {
			addresses = append(addresses[:i], addresses[i+1:]...)
			return s.write(addresses)
		}
	}

	return errors.New("share address not found")
}

func (s *ShareAddressService) read() ([]ShareAddress, error) {
	data, err := ioutil.ReadFile(shareAddressPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []ShareAddress{}, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return []ShareAddress{}, nil
	}

	addresses := make([]ShareAddress, 0)
	if err := json.Unmarshal(data, &addresses); err != nil {
		return nil, err
	}
	return addresses, nil
}

func (s *ShareAddressService) write(addresses []ShareAddress) error {
	dir := filepath.Dir(shareAddressPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(addresses, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := ioutil.TempFile(dir, ".share_addresses_*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0644); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpName, shareAddressPath)
}

func normalizeShareAddress(input string) (string, error) {
	address := strings.TrimSpace(input)
	if address == "" {
		return "", errors.New("share address is empty")
	}

	lower := strings.ToLower(address)
	if strings.HasPrefix(lower, "http://") {
		address = address[len("http://"):]
	} else if strings.HasPrefix(lower, "https://") {
		address = address[len("https://"):]
	}

	if index := strings.IndexAny(address, "/?#"); index >= 0 {
		address = address[:index]
	}
	address = strings.TrimRight(strings.TrimSpace(address), "/")
	if address == "" {
		return "", errors.New("share address is empty")
	}
	if strings.ContainsAny(address, " \t\r\n") {
		return "", errors.New("share address contains spaces")
	}

	if strings.HasPrefix(address, "[") {
		index := strings.Index(address, "]")
		if index <= 0 {
			return "", errors.New("invalid ipv6 address")
		}
		address = address[1:index]
	} else if strings.Count(address, ":") == 1 {
		host, _, err := net.SplitHostPort(address)
		if err == nil && host != "" {
			address = host
		}
	}

	address = strings.TrimSpace(strings.Trim(address, "[]"))
	if address == "" {
		return "", errors.New("share address is empty")
	}
	if strings.ContainsAny(address, " \t\r\n") {
		return "", errors.New("share address contains spaces")
	}
	return address, nil
}

func hasShareAddress(addresses []ShareAddress, address string, ignoreId string) bool {
	for _, item := range addresses {
		if item.Id == ignoreId {
			continue
		}
		if strings.EqualFold(item.Address, address) {
			return true
		}
	}
	return false
}

func newShareAddressId() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return hex.EncodeToString(b[0:4]) + "-" +
		hex.EncodeToString(b[4:6]) + "-" +
		hex.EncodeToString(b[6:8]) + "-" +
		hex.EncodeToString(b[8:10]) + "-" +
		hex.EncodeToString(b[10:16])
}
