package n5

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"x-ui/database"
	n5model "x-ui/database/model/n5"
	"x-ui/util/common"
	"x-ui/xray"

	"gorm.io/gorm"
)

const (
	egressTestStatusSuccess = "success"
	egressTestStatusFailed  = "failed"

	egressTestTCPAddress = "api.ipify.org:443"
	egressTestHTTPURL    = "https://api.ipify.org"
	egressTestTimeout    = 15 * time.Second
)

type egressTestRunner interface {
	Run(egress *n5model.Egress) (*egressTestExecution, error)
}

type egressTestExecution struct {
	Status  string
	Latency int
	ExitIP  string
	Message string
}

type EgressTestService struct {
	runner egressTestRunner
}

func (s *EgressTestService) Test(egressId int) (*n5model.EgressTest, error) {
	if egressId <= 0 {
		return nil, common.NewError("invalid egress id")
	}

	egress := &n5model.Egress{}
	if err := database.GetDB().Model(&n5model.Egress{}).Where("id = ?", egressId).First(egress).Error; err != nil {
		return nil, err
	}

	result, runErr := s.getRunner().Run(egress)
	record := buildEgressTestRecord(egressId, result, runErr)
	if err := persistEgressTestResult(egressId, record); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *EgressTestService) getRunner() egressTestRunner {
	if s.runner != nil {
		return s.runner
	}
	return &xrayEgressTestRunner{}
}

func buildEgressTestRecord(egressId int, result *egressTestExecution, runErr error) *n5model.EgressTest {
	record := &n5model.EgressTest{
		EgressId: egressId,
		Status:   egressTestStatusFailed,
		TestedAt: time.Now().UnixMilli(),
	}
	if result != nil {
		record.Status = normalizeEgressTestStatus(result.Status)
		record.Latency = result.Latency
		record.ExitIP = strings.TrimSpace(result.ExitIP)
		record.Message = strings.TrimSpace(result.Message)
	}
	if runErr != nil {
		record.Status = egressTestStatusFailed
		record.Message = strings.TrimSpace(runErr.Error())
	}
	if record.Status == egressTestStatusSuccess {
		record.Message = ""
	}
	return record
}

func normalizeEgressTestStatus(status string) string {
	if strings.EqualFold(strings.TrimSpace(status), egressTestStatusSuccess) {
		return egressTestStatusSuccess
	}
	return egressTestStatusFailed
}

func persistEgressTestResult(egressId int, record *n5model.EgressTest) error {
	if record == nil {
		return common.NewError("egress test result is nil")
	}

	return database.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(record).Error; err != nil {
			return err
		}

		updates := map[string]interface{}{
			"last_status":          record.Status,
			"last_exit_ip":         record.ExitIP,
			"last_test_time":       record.TestedAt,
			"last_test_status":     record.Status,
			"last_test_latency_ms": record.Latency,
			"last_test_error":      record.Message,
			"last_test_at":         record.TestedAt,
		}
		return tx.Model(&n5model.Egress{}).Where("id = ?", egressId).Updates(updates).Error
	})
}

type xrayEgressTestRunner struct {
}

func (r *xrayEgressTestRunner) Run(egress *n5model.Egress) (*egressTestExecution, error) {
	if egress == nil {
		return nil, common.NewError("egress is nil")
	}
	if _, err := os.Stat(xray.GetBinaryPath()); err != nil {
		return nil, common.NewErrorf("xray binary not found: %v", err)
	}

	outbound, err := parseJSONObject(egress.OutboundJSON)
	if err != nil {
		return nil, common.NewErrorf("invalid outbound json: %v", err)
	}
	outbound["tag"] = egress.Tag
	outbound["protocol"] = normalizeProtocol(egress.Protocol)

	outboundData, err := json.Marshal(outbound)
	if err != nil {
		return nil, err
	}

	port, err := allocateLocalTCPPort()
	if err != nil {
		return nil, err
	}

	cfg := &xray.Config{
		LogConfig: []byte(`{"loglevel":"warning"}`),
		InboundConfigs: []xray.InboundConfig{
			{
				Listen:         []byte(`"127.0.0.1"`),
				Port:           port,
				Protocol:       "socks",
				Settings:       []byte(`{"auth":"noauth","udp":false,"ip":"127.0.0.1"}`),
				StreamSettings: []byte(`{}`),
				Tag:            "n5-egress-test-in",
				Sniffing:       []byte(`{}`),
			},
		},
		OutboundConfigs: []byte("[" + string(outboundData) + "]"),
	}
	if err := xray.TestConfig(cfg); err != nil {
		return nil, err
	}

	configPath, err := writeTempXrayConfig(cfg)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = os.Remove(configPath)
	}()

	proxyAddr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	done, output, cleanup, err := startTempXrayProcess(configPath)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	time.Sleep(500 * time.Millisecond)
	select {
	case waitErr := <-done:
		return nil, common.NewErrorf("temp xray exited early: %v, output: %s", waitErr, strings.TrimSpace(output.String()))
	default:
	}

	latency, err := runSOCKSTCPProbe(proxyAddr, egressTestTCPAddress, egressTestTimeout)
	if err != nil {
		return &egressTestExecution{
			Status:  egressTestStatusFailed,
			Message: err.Error(),
		}, nil
	}

	exitIP, err := runSOCKSHTTPIPProbe(proxyAddr, egressTestHTTPURL, egressTestTimeout)
	if err != nil {
		return &egressTestExecution{
			Status:  egressTestStatusFailed,
			Latency: latency,
			Message: err.Error(),
		}, nil
	}

	return &egressTestExecution{
		Status:  egressTestStatusSuccess,
		Latency: latency,
		ExitIP:  exitIP,
	}, nil
}

func allocateLocalTCPPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, common.NewError("unexpected local tcp addr")
	}
	return addr.Port, nil
}

func writeTempXrayConfig(cfg *xray.Config) (string, error) {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", common.NewErrorf("marshal temp xray config failed: %v", err)
	}
	file, err := os.CreateTemp("", "n5-egress-test-*.json")
	if err != nil {
		return "", common.NewErrorf("create temp xray config failed: %v", err)
	}
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return "", common.NewErrorf("write temp xray config failed: %v", err)
	}
	return file.Name(), nil
}

func startTempXrayProcess(configPath string) (<-chan error, *bytes.Buffer, func(), error) {
	output := &bytes.Buffer{}
	cmd := exec.Command(xray.GetBinaryPath(), "-c", configPath)
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Start(); err != nil {
		return nil, nil, nil, common.NewErrorf("start temp xray failed: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	cleanup := func() {
		if cmd.Process != nil && cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
		}
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	}
	return done, output, cleanup, nil
}

func runSOCKSTCPProbe(proxyAddr string, targetAddr string, timeout time.Duration) (int, error) {
	start := time.Now()
	conn, err := dialSOCKS5(proxyAddr, targetAddr, timeout)
	if err != nil {
		return 0, common.NewErrorf("tcp probe failed: %v", err)
	}
	_ = conn.Close()
	return int(time.Since(start).Milliseconds()), nil
}

func runSOCKSHTTPIPProbe(proxyAddr string, url string, timeout time.Duration) (string, error) {
	transport := &http.Transport{
		DisableKeepAlives:     true,
		ResponseHeaderTimeout: timeout,
		TLSHandshakeTimeout:   timeout,
		DialContext: func(ctx context.Context, network string, addr string) (net.Conn, error) {
			_ = network
			dialTimeout := timeout
			if deadline, ok := ctx.Deadline(); ok {
				if remain := time.Until(deadline); remain > 0 && remain < dialTimeout {
					dialTimeout = remain
				}
			}
			return dialSOCKS5(proxyAddr, addr, dialTimeout)
		},
	}
	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "n5-ui-egress-test")

	resp, err := client.Do(req)
	if err != nil {
		return "", common.NewErrorf("http ip probe failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", common.NewErrorf("http ip probe status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 128))
	if err != nil {
		return "", common.NewErrorf("read http ip probe response failed: %v", err)
	}
	exitIP := strings.TrimSpace(string(body))
	if exitIP == "" {
		return "", common.NewError("http ip probe returned empty body")
	}
	return exitIP, nil
}

func dialSOCKS5(proxyAddr string, targetAddr string, timeout time.Duration) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", proxyAddr, timeout)
	if err != nil {
		return nil, err
	}

	deadline := time.Now().Add(timeout)
	_ = conn.SetDeadline(deadline)
	if _, err := conn.Write([]byte{0x05, 0x01, 0x00}); err != nil {
		_ = conn.Close()
		return nil, err
	}

	greeting := make([]byte, 2)
	if _, err := io.ReadFull(conn, greeting); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if greeting[0] != 0x05 || greeting[1] != 0x00 {
		_ = conn.Close()
		return nil, common.NewErrorf("unexpected socks greeting: %v", greeting)
	}

	request, err := buildSOCKS5ConnectRequest(targetAddr)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if _, err := conn.Write(request); err != nil {
		_ = conn.Close()
		return nil, err
	}

	response := make([]byte, 4)
	if _, err := io.ReadFull(conn, response); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if response[0] != 0x05 {
		_ = conn.Close()
		return nil, common.NewErrorf("unexpected socks version: %d", response[0])
	}
	if response[1] != 0x00 {
		_ = conn.Close()
		return nil, common.NewErrorf("socks connect failed, rep=%d", response[1])
	}

	if err := discardSOCKS5Address(conn, response[3]); err != nil {
		_ = conn.Close()
		return nil, err
	}
	_ = conn.SetDeadline(time.Time{})
	return conn, nil
}

func buildSOCKS5ConnectRequest(targetAddr string) ([]byte, error) {
	host, portText, err := net.SplitHostPort(targetAddr)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port <= 0 || port > 65535 {
		return nil, common.NewError("invalid target port")
	}

	buf := []byte{0x05, 0x01, 0x00}
	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			buf = append(buf, 0x01)
			buf = append(buf, ip4...)
		} else {
			buf = append(buf, 0x04)
			buf = append(buf, ip.To16()...)
		}
	} else {
		if len(host) > 255 {
			return nil, common.NewError("target host is too long")
		}
		buf = append(buf, 0x03, byte(len(host)))
		buf = append(buf, []byte(host)...)
	}
	buf = append(buf, byte(port>>8), byte(port))
	return buf, nil
}

func discardSOCKS5Address(conn net.Conn, atyp byte) error {
	var size int
	switch atyp {
	case 0x01:
		size = 4
	case 0x04:
		size = 16
	case 0x03:
		header := make([]byte, 1)
		if _, err := io.ReadFull(conn, header); err != nil {
			return err
		}
		size = int(header[0])
	default:
		return common.NewErrorf("unexpected socks atyp: %d", atyp)
	}

	if size > 0 {
		addrBytes := make([]byte, size)
		if _, err := io.ReadFull(conn, addrBytes); err != nil {
			return err
		}
	}
	portBytes := make([]byte, 2)
	_, err := io.ReadFull(conn, portBytes)
	return err
}
