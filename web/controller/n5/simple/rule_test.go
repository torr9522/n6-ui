package simple

import (
	"bytes"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"
	"x-ui/database/model"
	"x-ui/web/entity"
	simpleservice "x-ui/web/service/n5/simple"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

type fakeRuleService struct {
	listResult *simpleservice.SimpleRuleListResult
	created    *simpleservice.SimpleRule
	updated    *simpleservice.SimpleRule
	deletedID  string
	preview    *simpleservice.SimpleRuleConflictPreview
}

func (f *fakeRuleService) ListSimpleRules() (*simpleservice.SimpleRuleListResult, error) {
	return f.listResult, nil
}

func (f *fakeRuleService) CreateSimpleRule(req *simpleservice.CreateSimpleRuleRequest) (*simpleservice.SimpleRule, error) {
	return &simpleservice.SimpleRule{
		Id:           21,
		RuleId:       "simple-rule:21:YWk",
		PolicyId:     21,
		InboundId:    req.InboundId,
		TrafficType:  req.TrafficType,
		EgressId:     req.EgressId,
		CustomDomain: req.CustomDomain,
		Status:       "enabled",
		Enabled:      true,
	}, nil
}

func (f *fakeRuleService) UpdateSimpleRule(ruleId string, req *simpleservice.CreateSimpleRuleRequest) (*simpleservice.SimpleRule, error) {
	f.updated = &simpleservice.SimpleRule{
		Id:           21,
		RuleId:       ruleId,
		PolicyId:     21,
		InboundId:    req.InboundId,
		TrafficType:  req.TrafficType,
		EgressId:     req.EgressId,
		CustomDomain: req.CustomDomain,
		Status:       "enabled",
		Enabled:      true,
	}
	return f.updated, nil
}

func (f *fakeRuleService) DeleteSimpleRule(ruleId string) error {
	f.deletedID = ruleId
	return nil
}

func (f *fakeRuleService) CheckSimpleRuleConflicts(req *simpleservice.CreateSimpleRuleRequest, editingRuleId string) (*simpleservice.SimpleRuleConflictPreview, error) {
	if f.preview != nil {
		return f.preview, nil
	}
	return &simpleservice.SimpleRuleConflictPreview{}, nil
}

type fakeRuleRestart struct {
	calls int
}

func (f *fakeRuleRestart) SetToNeedRestart() {
	f.calls++
}

type fakeRuleSettingService struct {
	enabled bool
}

func (f *fakeRuleSettingService) GetAllSetting() (*entity.AllSetting, error) {
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

func (f *fakeRuleSettingService) GetN5XrayExtensionEnable() (bool, error) {
	return f.enabled, nil
}

func (f *fakeRuleSettingService) UpdateAllSetting(allSetting *entity.AllSetting) error {
	f.enabled = allSetting.N5XrayExtensionEnable
	return nil
}

func newRuleTestEngine(t *testing.T, svc ruleAPI, restart ruleRestartTrigger, setting ruleSettingAPI) *gin.Engine {
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
	engine.LoadHTMLFiles(testFiles()...)
	g := engine.Group("/")
	controller := &RuleController{service: svc, xrayService: restart, settingService: setting}
	controller.initRouter(g)
	return engine
}

func TestSimpleRulePageRouteRender(t *testing.T) {
	engine := newRuleTestEngine(t, &fakeRuleService{}, &fakeRuleRestart{}, &fakeRuleSettingService{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/n5/simple/rules", nil)
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}
}

func TestSimpleRuleAPIResponses(t *testing.T) {
	svc := &fakeRuleService{
		listResult: &simpleservice.SimpleRuleListResult{
			Rules: []*simpleservice.SimpleRule{
				{
					Id:          1,
					RuleId:      "simple-rule:1:YWxs",
					PolicyId:    1,
					InboundId:   7,
					InboundName: "simple-inbound",
					TrafficType: "ai",
					EgressId:    9,
					EgressName:  "simple-egress",
					Status:      "enabled",
					Enabled:     true,
				},
			},
			Inbounds: []*simpleservice.SimpleInboundOption{
				{Id: 7, Name: "simple-inbound", Tag: "simple-inbound-tag", Enabled: true},
			},
			Egresses: []*simpleservice.SimpleRuleEgressOption{
				{Id: 9, Name: "simple-egress", Tag: "n5-egress-0000000009", Enabled: true},
			},
			TrafficTypes: []*simpleservice.SimpleTrafficOption{
				{Value: "ai", Label: "AI 分流"},
			},
		},
	}
	restart := &fakeRuleRestart{}
	setting := &fakeRuleSettingService{}
	engine := newRuleTestEngine(t, svc, restart, setting)

	listResp := httptest.NewRecorder()
	listReq, _ := http.NewRequest(http.MethodGet, "/n5/api/simple/rule/list", nil)
	engine.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("unexpected list status: %d", listResp.Code)
	}
	if !bytes.Contains(listResp.Body.Bytes(), []byte(`"trafficType":"ai"`)) {
		t.Fatalf("unexpected list body: %s", listResp.Body.String())
	}

	statusResp := httptest.NewRecorder()
	statusReq, _ := http.NewRequest(http.MethodGet, "/n5/api/simple/rule/n5-status", nil)
	engine.ServeHTTP(statusResp, statusReq)
	if statusResp.Code != http.StatusOK {
		t.Fatalf("unexpected n5 status code: %d", statusResp.Code)
	}
	if !bytes.Contains(statusResp.Body.Bytes(), []byte(`"enabled":false`)) {
		t.Fatalf("unexpected n5 status body: %s", statusResp.Body.String())
	}

	addBody := bytes.NewBufferString(`{"inboundId":7,"trafficType":"ai","egressId":9}`)
	addReq, _ := http.NewRequest(http.MethodPost, "/n5/api/simple/rule/add", addBody)
	addReq.Header.Set("Content-Type", "application/json")
	addResp := httptest.NewRecorder()
	engine.ServeHTTP(addResp, addReq)
	if addResp.Code != http.StatusOK {
		t.Fatalf("unexpected add status: %d", addResp.Code)
	}
	if !bytes.Contains(addResp.Body.Bytes(), []byte(`"policyId":21`)) {
		t.Fatalf("unexpected add body: %s", addResp.Body.String())
	}
	if restart.calls != 1 {
		t.Fatalf("unexpected restart count after add: %d", restart.calls)
	}

	updateStatusBody := bytes.NewBufferString(`{"enabled":true}`)
	updateStatusReq, _ := http.NewRequest(http.MethodPost, "/n5/api/simple/rule/n5-status", updateStatusBody)
	updateStatusReq.Header.Set("Content-Type", "application/json")
	updateStatusResp := httptest.NewRecorder()
	engine.ServeHTTP(updateStatusResp, updateStatusReq)
	if updateStatusResp.Code != http.StatusOK {
		t.Fatalf("unexpected update n5 status code: %d", updateStatusResp.Code)
	}
	if !bytes.Contains(updateStatusResp.Body.Bytes(), []byte(`"enabled":true`)) {
		t.Fatalf("unexpected update n5 status body: %s", updateStatusResp.Body.String())
	}
	if restart.calls != 2 {
		t.Fatalf("unexpected restart count after status update: %d", restart.calls)
	}

	updateBody := bytes.NewBufferString(`{"ruleId":"simple-rule:21:YWk","inboundId":7,"trafficType":"ai","egressId":10}`)
	updateReq, _ := http.NewRequest(http.MethodPost, "/n5/api/simple/rule/update", updateBody)
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp := httptest.NewRecorder()
	engine.ServeHTTP(updateResp, updateReq)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("unexpected update status: %d", updateResp.Code)
	}
	if svc.updated == nil || svc.updated.RuleId != "simple-rule:21:YWk" {
		t.Fatalf("unexpected updated rule: %#v", svc.updated)
	}
	if restart.calls != 3 {
		t.Fatalf("unexpected restart count after update: %d", restart.calls)
	}

	previewBody := bytes.NewBufferString(`{"inboundId":7,"trafficType":"custom-domain","customDomain":"domain:openai.com","egressId":10}`)
	previewReq, _ := http.NewRequest(http.MethodPost, "/n5/api/simple/rule/conflicts", previewBody)
	previewReq.Header.Set("Content-Type", "application/json")
	previewResp := httptest.NewRecorder()
	engine.ServeHTTP(previewResp, previewReq)
	if previewResp.Code != http.StatusOK {
		t.Fatalf("unexpected preview status: %d", previewResp.Code)
	}
	if !bytes.Contains(previewResp.Body.Bytes(), []byte(`"hasConflict"`)) {
		t.Fatalf("unexpected preview body: %s", previewResp.Body.String())
	}
	if restart.calls != 3 {
		t.Fatalf("preview should not trigger restart: %d", restart.calls)
	}

	delBody := bytes.NewBufferString(`{"ruleId":"simple-rule:21:YWk"}`)
	delReq, _ := http.NewRequest(http.MethodPost, "/n5/api/simple/rule/delete", delBody)
	delReq.Header.Set("Content-Type", "application/json")
	delResp := httptest.NewRecorder()
	engine.ServeHTTP(delResp, delReq)
	if delResp.Code != http.StatusOK {
		t.Fatalf("unexpected delete status: %d", delResp.Code)
	}
	if svc.deletedID != "simple-rule:21:YWk" {
		t.Fatalf("unexpected deleted id: %s", svc.deletedID)
	}
	if restart.calls != 4 {
		t.Fatalf("unexpected restart count after delete: %d", restart.calls)
	}
}
