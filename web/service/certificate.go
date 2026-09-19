package service

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"x-ui/util/common"
	"x-ui/web/entity"
)

const CertificateDir = "/etc/x-ui/certs"

type CertificateService struct {
}

var certDomainRe = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func (s *CertificateService) List() ([]entity.CertificateInfo, error) {
	return s.scanManaged()
}

func (s *CertificateService) Discover() ([]entity.CertificateInfo, error) {
	certs := make([]entity.CertificateInfo, 0)
	seen := map[string]bool{}
	add := func(items []entity.CertificateInfo) {
		for _, item := range items {
			key := item.CertFile + "|" + item.KeyFile
			if seen[key] {
				continue
			}
			seen[key] = true
			certs = append(certs, item)
		}
	}

	if items, err := s.scanManaged(); err == nil {
		add(items)
	}
	if items, err := s.scanAcme(); err == nil {
		add(items)
	}
	if items, err := s.scanLetsEncrypt(); err == nil {
		add(items)
	}
	return certs, nil
}

func (s *CertificateService) Import(form *entity.CertificateImportForm) (*entity.CertificateInfo, error) {
	if strings.TrimSpace(form.Domain) == "" {
		return nil, common.NewError("domain is empty")
	}
	certFile, err := s.safeExistingPath(form.CertFile)
	if err != nil {
		return nil, err
	}
	keyFile, err := s.safeExistingPath(form.KeyFile)
	if err != nil {
		return nil, err
	}
	if !s.isAllowedPath(certFile) || !s.isAllowedPath(keyFile) {
		return nil, common.NewError("certificate path is not allowed")
	}

	source := certificateFormSource(form)
	info, err := s.validatePair(form.Domain, form.Provider, source, certFile, keyFile, form.AutoRenew, false)
	if err != nil {
		return nil, err
	}
	if !info.Valid {
		return nil, common.NewError("certificate is expired")
	}

	domainDir := filepath.Join(CertificateDir, sanitizeCertDomain(form.Domain))
	if err := os.MkdirAll(domainDir, 0755); err != nil {
		return nil, err
	}
	targetCert := filepath.Join(domainDir, "fullchain.pem")
	targetKey := filepath.Join(domainDir, "privkey.pem")
	if err := copyFile(certFile, targetCert, 0644); err != nil {
		return nil, err
	}
	if err := copyFile(keyFile, targetKey, 0600); err != nil {
		return nil, err
	}

	meta := entity.CertificateInfo{
		Domain:    form.Domain,
		Provider:  form.Provider,
		Source:    source,
		CertFile:  targetCert,
		KeyFile:   targetKey,
		Created:   time.Now().Unix(),
		Expire:    info.Expire,
		AutoRenew: form.AutoRenew,
		Valid:     true,
		Issuer:    info.Issuer,
		Managed:   true,
	}
	if err := writeCertMeta(domainDir, &meta); err != nil {
		return nil, err
	}
	return s.validatePair(meta.Domain, meta.Provider, meta.Source, targetCert, targetKey, meta.AutoRenew, true)
}

func (s *CertificateService) Validate(certFile string, keyFile string) (*entity.CertificateInfo, error) {
	certFile, err := s.safeExistingPath(certFile)
	if err != nil {
		return nil, err
	}
	keyFile, err = s.safeExistingPath(keyFile)
	if err != nil {
		return nil, err
	}
	if !s.isAllowedPath(certFile) || !s.isAllowedPath(keyFile) {
		return nil, common.NewError("certificate path is not allowed")
	}
	return s.validatePair("", "", "custom", certFile, keyFile, false, strings.HasPrefix(certFile, CertificateDir))
}

func (s *CertificateService) scanManaged() ([]entity.CertificateInfo, error) {
	entries, err := os.ReadDir(CertificateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []entity.CertificateInfo{}, nil
		}
		return nil, err
	}
	certs := make([]entity.CertificateInfo, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(CertificateDir, entry.Name())
		certFile := filepath.Join(dir, "fullchain.pem")
		keyFile := filepath.Join(dir, "privkey.pem")
		meta := readCertMeta(dir)
		domain := meta.Domain
		if domain == "" {
			domain = entry.Name()
		}
		provider := meta.Provider
		if provider == "" {
			provider = "managed"
		}
		info, err := s.validatePair(domain, provider, meta.Source, certFile, keyFile, meta.AutoRenew, true)
		if err != nil {
			info = &entity.CertificateInfo{
				Domain:    domain,
				Provider:  provider,
				Source:    meta.Source,
				CertFile:  certFile,
				KeyFile:   keyFile,
				Created:   meta.Created,
				Expire:    meta.Expire,
				AutoRenew: meta.AutoRenew,
				Valid:     false,
				Error:     err.Error(),
				Managed:   true,
			}
		}
		if meta.Created > 0 {
			info.Created = meta.Created
		}
		certs = append(certs, *info)
	}
	return certs, nil
}

func (s *CertificateService) scanAcme() ([]entity.CertificateInfo, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	root := filepath.Join(home, ".acme.sh")
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return []entity.CertificateInfo{}, nil
		}
		return nil, err
	}

	certs := make([]entity.CertificateInfo, 0)
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() || path == root {
			return nil
		}
		name := filepath.Base(path)
		if strings.HasSuffix(name, "_ecc") {
			name = strings.TrimSuffix(name, "_ecc")
		}
		candidates := [][2]string{
			{filepath.Join(path, "fullchain.cer"), filepath.Join(path, name+".key")},
			{filepath.Join(path, "fullchain.pem"), filepath.Join(path, "privkey.pem")},
		}
		for _, pair := range candidates {
			if fileExists(pair[0]) && fileExists(pair[1]) {
				info, err := s.validatePair(name, "acme.sh", "acme.sh", pair[0], pair[1], true, false)
				if err == nil {
					certs = append(certs, *info)
				}
				break
			}
		}
		return nil
	})
	return certs, nil
}

func (s *CertificateService) scanLetsEncrypt() ([]entity.CertificateInfo, error) {
	root := "/etc/letsencrypt/live"
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []entity.CertificateInfo{}, nil
		}
		return nil, err
	}
	certs := make([]entity.CertificateInfo, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(root, entry.Name())
		certFile := filepath.Join(dir, "fullchain.pem")
		keyFile := filepath.Join(dir, "privkey.pem")
		info, err := s.validatePair(entry.Name(), "letsencrypt", "letsencrypt", certFile, keyFile, true, false)
		if err == nil {
			certs = append(certs, *info)
		}
	}
	return certs, nil
}

func (s *CertificateService) validatePair(domain string, provider string, source string, certFile string, keyFile string, autoRenew bool, managed bool) (*entity.CertificateInfo, error) {
	pair, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	if len(pair.Certificate) == 0 {
		return nil, common.NewError("certificate is empty")
	}
	cert, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return nil, err
	}
	if domain == "" {
		domain = cert.Subject.CommonName
		if domain == "" && len(cert.DNSNames) > 0 {
			domain = cert.DNSNames[0]
		}
	}
	if provider == "" {
		provider = "unknown"
	}
	return &entity.CertificateInfo{
		Domain:    domain,
		Provider:  provider,
		Source:    source,
		CertFile:  certFile,
		KeyFile:   keyFile,
		Expire:    cert.NotAfter.Unix(),
		AutoRenew: autoRenew,
		Valid:     time.Now().Before(cert.NotAfter),
		Issuer:    cert.Issuer.CommonName,
		Managed:   managed,
	}, nil
}

func (s *CertificateService) safeExistingPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", common.NewError("certificate path is empty")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	realPath, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(realPath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", common.NewError("certificate path is a directory")
	}
	return realPath, nil
}

func (s *CertificateService) isAllowedPath(path string) bool {
	home, _ := os.UserHomeDir()
	roots := []string{
		CertificateDir,
		filepath.Join(home, ".acme.sh"),
		"/etc/letsencrypt",
	}
	for _, root := range roots {
		if root == "" {
			continue
		}
		absRoot, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		if realRoot, err := filepath.EvalSymlinks(absRoot); err == nil {
			absRoot = realRoot
		}
		rel, err := filepath.Rel(absRoot, path)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, "../") {
			return true
		}
	}
	return false
}

func sanitizeCertDomain(domain string) string {
	domain = strings.TrimSpace(domain)
	domain = strings.TrimPrefix(domain, "*.")
	domain = certDomainRe.ReplaceAllString(domain, "_")
	domain = strings.Trim(domain, "._-")
	if domain == "" {
		return "unknown"
	}
	return domain
}

func copyFile(src string, dst string, perm os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, perm)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func readCertMeta(dir string) entity.CertificateInfo {
	meta := entity.CertificateInfo{}
	data, err := os.ReadFile(filepath.Join(dir, "meta.json"))
	if err != nil {
		return meta
	}
	_ = json.Unmarshal(data, &meta)
	return meta
}

func writeCertMeta(dir string, meta *entity.CertificateInfo) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "meta.json"), data, 0644)
}

func certificateFormSource(form *entity.CertificateImportForm) string {
	if strings.TrimSpace(form.Provider) != "" {
		return form.Provider
	}
	return "import"
}
