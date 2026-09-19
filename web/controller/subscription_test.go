package controller

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"x-ui/database"
	"x-ui/database/model"
	"x-ui/web/service"
)

func subscriptionTestFiles(t *testing.T) []string {
	t.Helper()
	files := make([]string, 0)
	root := filepath.Join("..", "html")
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".html" {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return files
}

func newSubscriptionTestEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.SetFuncMap(template.FuncMap{
		"i18n": func(key string, params ...string) (string, error) {
			return key, nil
		},
	})
	store := cookie.NewStore([]byte("test-secret"))
	engine.Use(sessions.Sessions("session", store))
	engine.Use(func(c *gin.Context) {
		c.Set("base_path", "/")
		s := sessions.Default(c)
		s.Set("LOGIN_USER", model.User{Id: 1, Username: "admin"})
		_ = s.Save()
		c.Next()
	})
	engine.LoadHTMLFiles(subscriptionTestFiles(t)...)

	root := engine.Group("/")
	xui := root.Group("/xui")
	base := &BaseController{}
	xui.Use(base.checkLogin)
	xui.GET("/subscriptions", (&XUIController{}).subscriptions)
	sub := &SubscriptionController{}
	sub.initAdminRouter(xui)
	public := &SubscriptionController{}
	public.initPublicRouter(root)
	return engine
}

func initSubscriptionControllerDB(t *testing.T) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "subscription-controller.db")
	if err := database.InitDB(dbPath); err != nil {
		t.Fatalf("init db failed: %v", err)
	}
}

func TestSubscriptionPageRenders(t *testing.T) {
	initSubscriptionControllerDB(t)
	engine := newSubscriptionTestEngine(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/xui/subscriptions", nil)
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "订阅管理") {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestSubscriptionAdminAPIAndPublicRoute(t *testing.T) {
	initSubscriptionControllerDB(t)
	shareFile := filepath.Join(t.TempDir(), "share_addresses.json")
	original := service.SetShareAddressPathForTest(shareFile)
	defer service.SetShareAddressPathForTest(original)
	if _, err := (&service.ShareAddressService{}).Add("panel.example.com", "panel"); err != nil {
		t.Fatalf("add share address failed: %v", err)
	}

	inbound := &model.Inbound{
		UserId:         1,
		Remark:         "controller-vmess",
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           32101,
		Protocol:       model.VMess,
		Settings:       `{"clients":[{"id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","alterId":0}]}`,
		StreamSettings: `{"network":"tcp","security":"none","tcpSettings":{"header":{"type":"none"}}}`,
		Tag:            "controller-vmess-tag",
		Sniffing:       `{}`,
	}
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatalf("create inbound failed: %v", err)
	}

	engine := newSubscriptionTestEngine(t)

	inboundsResp := httptest.NewRecorder()
	inboundsReq, _ := http.NewRequest(http.MethodPost, "/xui/subscription/inbounds", nil)
	engine.ServeHTTP(inboundsResp, inboundsReq)
	if inboundsResp.Code != http.StatusOK || !bytes.Contains(inboundsResp.Body.Bytes(), []byte(`"protocol":"vmess"`)) {
		t.Fatalf("unexpected inbounds response: %d %s", inboundsResp.Code, inboundsResp.Body.String())
	}

	addResp := httptest.NewRecorder()
	addReq, _ := http.NewRequest(http.MethodPost, "/xui/subscription/add", strings.NewReader("remark=test&enable=false&inboundIds="+strconv.Itoa(inbound.Id)))
	addReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	engine.ServeHTTP(addResp, addReq)
	if addResp.Code != http.StatusOK {
		t.Fatalf("unexpected add status: %d", addResp.Code)
	}
	if !bytes.Contains(addResp.Body.Bytes(), []byte(`"enable":false`)) {
		t.Fatalf("expected enable=false in add response: %s", addResp.Body.String())
	}

	var addMsg struct {
		Success bool                      `json:"success"`
		Obj     *service.SubscriptionView `json:"obj"`
	}
	if err := json.Unmarshal(addResp.Body.Bytes(), &addMsg); err != nil {
		t.Fatalf("unmarshal add response failed: %v", err)
	}
	if addMsg.Obj == nil {
		t.Fatalf("missing add response object: %s", addResp.Body.String())
	}

	listResp := httptest.NewRecorder()
	listReq, _ := http.NewRequest(http.MethodPost, "/xui/subscription/list", nil)
	engine.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK || !bytes.Contains(listResp.Body.Bytes(), []byte(`"remark":"test"`)) {
		t.Fatalf("unexpected list response: %d %s", listResp.Code, listResp.Body.String())
	}

	publicDisabledResp := httptest.NewRecorder()
	publicDisabledReq, _ := http.NewRequest(http.MethodGet, "/sub/"+addMsg.Obj.Token, nil)
	engine.ServeHTTP(publicDisabledResp, publicDisabledReq)
	if publicDisabledResp.Code != http.StatusNotFound {
		t.Fatalf("expected disabled token 404, got %d", publicDisabledResp.Code)
	}

	updateResp := httptest.NewRecorder()
	updateReq, _ := http.NewRequest(http.MethodPost, "/xui/subscription/update/"+strconv.Itoa(addMsg.Obj.Id), strings.NewReader("remark=test-updated&enable=true&inboundIds="+strconv.Itoa(inbound.Id)))
	updateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	engine.ServeHTTP(updateResp, updateReq)
	if updateResp.Code != http.StatusOK || !bytes.Contains(updateResp.Body.Bytes(), []byte(`"enable":true`)) {
		t.Fatalf("unexpected update response: %d %s", updateResp.Code, updateResp.Body.String())
	}

	publicResp := httptest.NewRecorder()
	publicReq, _ := http.NewRequest(http.MethodGet, "/sub/"+addMsg.Obj.Token, nil)
	publicReq.Host = "panel.example.com"
	engine.ServeHTTP(publicResp, publicReq)
	if publicResp.Code != http.StatusOK {
		t.Fatalf("unexpected public status: %d", publicResp.Code)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(publicResp.Body.String()))
	if err != nil {
		t.Fatalf("decode base64 failed: %v", err)
	}
	if !strings.Contains(string(decoded), "vmess://") {
		t.Fatalf("unexpected public body: %s", string(decoded))
	}

	clashResp := httptest.NewRecorder()
	clashReq, _ := http.NewRequest(http.MethodGet, "/sub/"+addMsg.Obj.Token+"?format=clash", nil)
	clashReq.Host = "panel.example.com"
	engine.ServeHTTP(clashResp, clashReq)
	if clashResp.Code != http.StatusOK || !strings.Contains(clashResp.Body.String(), "proxies:") {
		t.Fatalf("unexpected clash response: %d %s", clashResp.Code, clashResp.Body.String())
	}

	mihomoResp := httptest.NewRecorder()
	mihomoReq, _ := http.NewRequest(http.MethodGet, "/sub/"+addMsg.Obj.Token+"?format=mihomo", nil)
	mihomoReq.Host = "panel.example.com"
	engine.ServeHTTP(mihomoResp, mihomoReq)
	if mihomoResp.Code != http.StatusOK || !strings.Contains(mihomoResp.Body.String(), "proxies:") {
		t.Fatalf("unexpected mihomo response: %d %s", mihomoResp.Code, mihomoResp.Body.String())
	}

	unknownFormatResp := httptest.NewRecorder()
	unknownFormatReq, _ := http.NewRequest(http.MethodGet, "/sub/"+addMsg.Obj.Token+"?format=unknown", nil)
	engine.ServeHTTP(unknownFormatResp, unknownFormatReq)
	if unknownFormatResp.Code != http.StatusNotFound {
		t.Fatalf("expected unknown format 404, got %d", unknownFormatResp.Code)
	}

	invalidTokenResp := httptest.NewRecorder()
	invalidTokenReq, _ := http.NewRequest(http.MethodGet, "/sub/does-not-exist", nil)
	engine.ServeHTTP(invalidTokenResp, invalidTokenReq)
	if invalidTokenResp.Code != http.StatusNotFound {
		t.Fatalf("expected invalid token 404, got %d", invalidTokenResp.Code)
	}

	refreshResp := httptest.NewRecorder()
	refreshReq, _ := http.NewRequest(http.MethodPost, "/xui/subscription/refresh-token/"+strconv.Itoa(addMsg.Obj.Id), nil)
	engine.ServeHTTP(refreshResp, refreshReq)
	if refreshResp.Code != http.StatusOK {
		t.Fatalf("unexpected refresh status: %d", refreshResp.Code)
	}
	var refreshMsg struct {
		Success bool                      `json:"success"`
		Obj     *service.SubscriptionView `json:"obj"`
	}
	if err := json.Unmarshal(refreshResp.Body.Bytes(), &refreshMsg); err != nil {
		t.Fatalf("unmarshal refresh response failed: %v", err)
	}
	if refreshMsg.Obj == nil || refreshMsg.Obj.Token == addMsg.Obj.Token {
		t.Fatalf("expected refreshed token change: %s", refreshResp.Body.String())
	}

	oldTokenResp := httptest.NewRecorder()
	oldTokenReq, _ := http.NewRequest(http.MethodGet, "/sub/"+addMsg.Obj.Token, nil)
	engine.ServeHTTP(oldTokenResp, oldTokenReq)
	if oldTokenResp.Code != http.StatusNotFound {
		t.Fatalf("expected old token 404 after refresh, got %d", oldTokenResp.Code)
	}

	newTokenResp := httptest.NewRecorder()
	newTokenReq, _ := http.NewRequest(http.MethodGet, "/sub/"+refreshMsg.Obj.Token, nil)
	newTokenReq.Host = "panel.example.com"
	engine.ServeHTTP(newTokenResp, newTokenReq)
	if newTokenResp.Code != http.StatusOK {
		t.Fatalf("expected new token 200, got %d", newTokenResp.Code)
	}

	delResp := httptest.NewRecorder()
	delReq, _ := http.NewRequest(http.MethodPost, "/xui/subscription/del/"+strconv.Itoa(addMsg.Obj.Id), nil)
	engine.ServeHTTP(delResp, delReq)
	if delResp.Code != http.StatusOK {
		t.Fatalf("unexpected delete status: %d", delResp.Code)
	}

	deletedResp := httptest.NewRecorder()
	deletedReq, _ := http.NewRequest(http.MethodGet, "/sub/"+refreshMsg.Obj.Token, nil)
	engine.ServeHTTP(deletedResp, deletedReq)
	if deletedResp.Code != http.StatusNotFound {
		t.Fatalf("expected deleted token 404, got %d", deletedResp.Code)
	}
}

func TestShadowsocks2022PublicSubscriptionFormats(t *testing.T) {
	initSubscriptionControllerDB(t)
	shareFile := filepath.Join(t.TempDir(), "share_addresses.json")
	original := service.SetShareAddressPathForTest(shareFile)
	defer service.SetShareAddressPathForTest(original)
	if _, err := (&service.ShareAddressService{}).Add("panel.example.com", "panel"); err != nil {
		t.Fatal(err)
	}
	inbound := &model.Inbound{
		UserId: 1, Remark: "controller-ss2022", Enable: true, Listen: "0.0.0.0", Port: 32102,
		Protocol:       model.Shadowsocks,
		Settings:       `{"method":"2022-blake3-aes-128-gcm","password":"AAAAAAAAAAAAAAAAAAAAAA==","network":"tcp,udp"}`,
		StreamSettings: `{}`, Tag: "controller-ss2022-tag", Sniffing: `{}`,
	}
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatal(err)
	}
	sub, err := (&service.SubscriptionService{}).Add(&service.SubscriptionForm{
		Remark: "ss2022", Enable: true, InboundIds: []int{inbound.Id},
	})
	if err != nil {
		t.Fatal(err)
	}
	engine := newSubscriptionTestEngine(t)

	base64Resp := httptest.NewRecorder()
	base64Req, _ := http.NewRequest(http.MethodGet, "/sub/"+sub.Token, nil)
	base64Req.Host = "panel.example.com"
	engine.ServeHTTP(base64Resp, base64Req)
	if base64Resp.Code != http.StatusOK {
		t.Fatalf("unexpected SS2022 Base64 response: %d %s", base64Resp.Code, base64Resp.Body.String())
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(base64Resp.Body.String()))
	if err != nil || !strings.Contains(string(decoded), "ss://") {
		t.Fatalf("unexpected SS2022 Base64 payload: %v %s", err, decoded)
	}

	mihomoResp := httptest.NewRecorder()
	mihomoReq, _ := http.NewRequest(http.MethodGet, "/sub/"+sub.Token+"?format=mihomo", nil)
	mihomoReq.Host = "panel.example.com"
	engine.ServeHTTP(mihomoResp, mihomoReq)
	if mihomoResp.Code != http.StatusOK || !strings.Contains(mihomoResp.Body.String(), "2022-blake3-aes-128-gcm") || !strings.Contains(mihomoResp.Body.String(), "udp: true") {
		t.Fatalf("unexpected SS2022 Mihomo response: %d %s", mihomoResp.Code, mihomoResp.Body.String())
	}

	clashResp := httptest.NewRecorder()
	clashReq, _ := http.NewRequest(http.MethodGet, "/sub/"+sub.Token+"?format=clash", nil)
	clashReq.Host = "panel.example.com"
	engine.ServeHTTP(clashResp, clashReq)
	if clashResp.Code != http.StatusInternalServerError || !strings.Contains(clashResp.Body.String(), "请使用 Mihomo") {
		t.Fatalf("unexpected SS2022 classic Clash response: %d %s", clashResp.Code, clashResp.Body.String())
	}
}
