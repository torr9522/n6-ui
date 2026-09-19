package simple

import (
	"bytes"
	"github.com/torr9522/n6-ui/database/model"
	simpleservice "github.com/torr9522/n6-ui/web/service/n5/simple"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

type fakeTrafficRuleGroupService struct {
	listItems []*simpleservice.TrafficRuleGroup
	item      *simpleservice.TrafficRuleGroup
}

func (f *fakeTrafficRuleGroupService) ListGroups() ([]*simpleservice.TrafficRuleGroup, error) {
	return f.listItems, nil
}

func (f *fakeTrafficRuleGroupService) GetGroup(id int) (*simpleservice.TrafficRuleGroup, error) {
	if f.item != nil {
		return f.item, nil
	}
	return &simpleservice.TrafficRuleGroup{Id: id, Name: "AI分流", GroupType: "ai", GroupLabel: "AI分流", Enabled: true}, nil
}

func (f *fakeTrafficRuleGroupService) CreateGroup(req *simpleservice.CreateTrafficRuleGroupRequest) (*simpleservice.TrafficRuleGroup, error) {
	return &simpleservice.TrafficRuleGroup{Id: 7, Name: req.Name, GroupType: req.GroupType, GroupLabel: req.Name, Enabled: true}, nil
}

func (f *fakeTrafficRuleGroupService) UpdateGroup(id int, req *simpleservice.UpdateTrafficRuleGroupRequest) (*simpleservice.TrafficRuleGroup, error) {
	return &simpleservice.TrafficRuleGroup{Id: id, Name: req.Name, GroupType: "ai", GroupLabel: req.Name, Enabled: true}, nil
}

func (f *fakeTrafficRuleGroupService) DeleteGroup(id int) error {
	return nil
}

func (f *fakeTrafficRuleGroupService) AddDomainRule(req *simpleservice.AddTrafficRuleDomainRequest) (*simpleservice.TrafficRuleGroupRule, error) {
	return &simpleservice.TrafficRuleGroupRule{Id: 11, DisplayValue: req.Domain, Enabled: true}, nil
}

func (f *fakeTrafficRuleGroupService) UpdateDomainRule(ruleId int, req *simpleservice.UpdateTrafficRuleDomainRequest) (*simpleservice.TrafficRuleGroupRule, error) {
	return &simpleservice.TrafficRuleGroupRule{Id: ruleId, DisplayValue: req.Domain, Enabled: true}, nil
}

func (f *fakeTrafficRuleGroupService) DeleteDomainRule(groupId int, ruleId int) error {
	return nil
}

func (f *fakeTrafficRuleGroupService) EnableGroup(id int) (*simpleservice.TrafficRuleGroup, error) {
	return &simpleservice.TrafficRuleGroup{Id: id, Name: "AI分流", GroupType: "ai", GroupLabel: "AI分流", Enabled: true, Status: "enabled"}, nil
}

func (f *fakeTrafficRuleGroupService) DisableGroup(id int) (*simpleservice.TrafficRuleGroup, error) {
	return &simpleservice.TrafficRuleGroup{Id: id, Name: "AI分流", GroupType: "ai", GroupLabel: "AI分流", Enabled: false, Status: "disabled"}, nil
}

func newTrafficRuleGroupTestEngine(t *testing.T, svc trafficRuleGroupAPI) *gin.Engine {
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
	controller := &TrafficRuleGroupController{service: svc}
	controller.initRouter(g)
	return engine
}

func TestTrafficRuleGroupPageRouteRender(t *testing.T) {
	engine := newTrafficRuleGroupTestEngine(t, &fakeTrafficRuleGroupService{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/n5/simple/traffic-rules", nil)
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}

	detail := httptest.NewRecorder()
	detailReq, _ := http.NewRequest(http.MethodGet, "/n5/simple/traffic-rules/1", nil)
	engine.ServeHTTP(detail, detailReq)
	if detail.Code != http.StatusOK {
		t.Fatalf("unexpected detail status: %d", detail.Code)
	}
}

func TestTrafficRuleGroupAPIResponses(t *testing.T) {
	svc := &fakeTrafficRuleGroupService{
		listItems: []*simpleservice.TrafficRuleGroup{
			{Id: 1, Name: "AI分流", GroupType: "ai", GroupLabel: "AI分流", KindLabel: "内置", Builtin: true, Enabled: true, RuleCount: 5, SnapshotCount: 1, DeleteHint: "该规则组已经生成过执行规则，删除不会影响已有运行规则"},
		},
		item: &simpleservice.TrafficRuleGroup{
			Id:            1,
			Name:          "AI分流",
			GroupType:     "ai",
			GroupLabel:    "AI分流",
			KindLabel:     "内置",
			Builtin:       true,
			Enabled:       true,
			RuleCount:     5,
			SnapshotCount: 1,
			DeleteHint:    "该规则组已经生成过执行规则，删除不会影响已有运行规则",
			Rules: []*simpleservice.TrafficRuleGroupRule{
				{Id: 2, DisplayValue: "domain:openai.com", Enabled: true},
			},
		},
	}
	engine := newTrafficRuleGroupTestEngine(t, svc)

	listResp := httptest.NewRecorder()
	listReq, _ := http.NewRequest(http.MethodGet, "/n5/api/simple/traffic-rule-groups", nil)
	engine.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("unexpected list status: %d", listResp.Code)
	}
	if !bytes.Contains(listResp.Body.Bytes(), []byte(`"name":"AI分流"`)) {
		t.Fatalf("unexpected list body: %s", listResp.Body.String())
	}
	if !bytes.Contains(listResp.Body.Bytes(), []byte(`"kindLabel":"内置"`)) {
		t.Fatalf("unexpected list kind label: %s", listResp.Body.String())
	}

	getResp := httptest.NewRecorder()
	getReq, _ := http.NewRequest(http.MethodGet, "/n5/api/simple/traffic-rule-group/1", nil)
	engine.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("unexpected get status: %d", getResp.Code)
	}
	if !bytes.Contains(getResp.Body.Bytes(), []byte(`"displayValue":"domain:openai.com"`)) {
		t.Fatalf("unexpected get body: %s", getResp.Body.String())
	}
	if !bytes.Contains(getResp.Body.Bytes(), []byte(`"deleteHint":"该规则组已经生成过执行规则，删除不会影响已有运行规则"`)) {
		t.Fatalf("unexpected get delete hint: %s", getResp.Body.String())
	}

	addResp := httptest.NewRecorder()
	addReq, _ := http.NewRequest(http.MethodPost, "/n5/api/simple/traffic-rule-group/add", bytes.NewBufferString(`{"groupType":"ai","name":"AI分流"}`))
	addReq.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(addResp, addReq)
	if addResp.Code != http.StatusOK {
		t.Fatalf("unexpected add status: %d", addResp.Code)
	}

	ruleResp := httptest.NewRecorder()
	ruleReq, _ := http.NewRequest(http.MethodPost, "/n5/api/simple/traffic-rule/add", bytes.NewBufferString(`{"groupId":1,"domain":"full:api64.ipify.org"}`))
	ruleReq.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(ruleResp, ruleReq)
	if ruleResp.Code != http.StatusOK {
		t.Fatalf("unexpected add rule status: %d", ruleResp.Code)
	}
	if !bytes.Contains(ruleResp.Body.Bytes(), []byte(`"displayValue":"full:api64.ipify.org"`)) {
		t.Fatalf("unexpected add rule body: %s", ruleResp.Body.String())
	}

	updateRuleResp := httptest.NewRecorder()
	updateRuleReq, _ := http.NewRequest(http.MethodPost, "/n5/api/simple/traffic-rule/update/11", bytes.NewBufferString(`{"groupId":1,"domain":"domain:openai.com"}`))
	updateRuleReq.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(updateRuleResp, updateRuleReq)
	if updateRuleResp.Code != http.StatusOK {
		t.Fatalf("unexpected update rule status: %d", updateRuleResp.Code)
	}
	if !bytes.Contains(updateRuleResp.Body.Bytes(), []byte(`"displayValue":"domain:openai.com"`)) {
		t.Fatalf("unexpected update rule body: %s", updateRuleResp.Body.String())
	}
}
