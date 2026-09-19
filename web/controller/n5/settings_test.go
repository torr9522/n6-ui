package n5

import (
	"bytes"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"
	"x-ui/database/model"
	"x-ui/web/entity"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

type fakeN5SettingsService struct {
	allSetting   *entity.AllSetting
	enabled      bool
	updateCalled int
}

type fakeN5SettingsRestart struct {
	calls int
}

func (f *fakeN5SettingsRestart) SetToNeedRestart() {
	f.calls++
}

func (f *fakeN5SettingsService) GetAllSetting() (*entity.AllSetting, error) {
	if f.allSetting != nil {
		copyValue := *f.allSetting
		copyValue.N5XrayExtensionEnable = f.enabled
		return &copyValue, nil
	}
	return &entity.AllSetting{
		WebListen:             "",
		WebPort:               54321,
		WebCertFile:           "",
		WebKeyFile:            "",
		WebBasePath:           "/",
		XrayTemplateConfig:    `{"log":{},"inbounds":[],"outbounds":[]}`,
		N5XrayExtensionEnable: f.enabled,
		TimeLocation:          "Asia/Shanghai",
	}, nil
}

func (f *fakeN5SettingsService) GetN5XrayExtensionEnable() (bool, error) {
	return f.enabled, nil
}

func (f *fakeN5SettingsService) UpdateAllSetting(allSetting *entity.AllSetting) error {
	f.updateCalled++
	f.enabled = allSetting.N5XrayExtensionEnable
	f.allSetting = allSetting
	return nil
}

func newN5SettingsTestEngine(t *testing.T, setting n5SettingAPI, restart n5RestartTrigger) *gin.Engine {
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
	engine.LoadHTMLFiles(testFiles(t)...)
	g := engine.Group("/")
	controller := &N5SettingsController{settingService: setting, xrayService: restart}
	controller.initRouter(g)
	return engine
}

func TestN5SettingsPageRouteRender(t *testing.T) {
	engine := newN5SettingsTestEngine(t, &fakeN5SettingsService{}, &fakeN5SettingsRestart{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/n5/settings", nil)
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}
}

func TestN5SettingsAPIResponses(t *testing.T) {
	setting := &fakeN5SettingsService{
		enabled: false,
		allSetting: &entity.AllSetting{
			WebListen:             "",
			WebPort:               54321,
			WebCertFile:           "",
			WebKeyFile:            "",
			WebBasePath:           "/",
			XrayTemplateConfig:    `{"log":{},"inbounds":[],"outbounds":[]}`,
			N5XrayExtensionEnable: false,
			TimeLocation:          "Asia/Shanghai",
		},
	}
	restart := &fakeN5SettingsRestart{}
	engine := newN5SettingsTestEngine(t, setting, restart)

	getResp := httptest.NewRecorder()
	getReq, _ := http.NewRequest(http.MethodGet, "/n5/api/settings", nil)
	engine.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("unexpected get status: %d", getResp.Code)
	}
	if !bytes.Contains(getResp.Body.Bytes(), []byte(`"enabled":false`)) {
		t.Fatalf("unexpected get body: %s", getResp.Body.String())
	}

	enableBody := bytes.NewBufferString(`{"enabled":true}`)
	enableReq, _ := http.NewRequest(http.MethodPost, "/n5/api/settings", enableBody)
	enableReq.Header.Set("Content-Type", "application/json")
	enableResp := httptest.NewRecorder()
	engine.ServeHTTP(enableResp, enableReq)
	if enableResp.Code != http.StatusOK {
		t.Fatalf("unexpected enable status: %d", enableResp.Code)
	}
	if !bytes.Contains(enableResp.Body.Bytes(), []byte(`"enabled":true`)) {
		t.Fatalf("unexpected enable body: %s", enableResp.Body.String())
	}
	if !setting.enabled {
		t.Fatal("expected setting enabled after update")
	}
	if setting.allSetting == nil || setting.allSetting.WebPort != 54321 || setting.allSetting.TimeLocation != "Asia/Shanghai" {
		t.Fatalf("unexpected setting payload preserved state: %#v", setting.allSetting)
	}
	if setting.updateCalled != 1 {
		t.Fatalf("unexpected update count after enable: %d", setting.updateCalled)
	}
	if restart.calls != 1 {
		t.Fatalf("unexpected restart count after enable: %d", restart.calls)
	}

	disableBody := bytes.NewBufferString(`{"enabled":false}`)
	disableReq, _ := http.NewRequest(http.MethodPost, "/n5/api/settings", disableBody)
	disableReq.Header.Set("Content-Type", "application/json")
	disableResp := httptest.NewRecorder()
	engine.ServeHTTP(disableResp, disableReq)
	if disableResp.Code != http.StatusOK {
		t.Fatalf("unexpected disable status: %d", disableResp.Code)
	}
	if !bytes.Contains(disableResp.Body.Bytes(), []byte(`"enabled":false`)) {
		t.Fatalf("unexpected disable body: %s", disableResp.Body.String())
	}
	if setting.enabled {
		t.Fatal("expected setting disabled after update")
	}
	if setting.updateCalled != 2 {
		t.Fatalf("unexpected update count after disable: %d", setting.updateCalled)
	}
	if restart.calls != 2 {
		t.Fatalf("unexpected restart count after disable: %d", restart.calls)
	}
}
