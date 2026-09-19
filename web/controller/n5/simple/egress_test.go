package simple

import (
	"bytes"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"
	"x-ui/database/model"
	simpleservice "x-ui/web/service/n5/simple"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

type fakeService struct {
	listRecords []*simpleservice.SimpleEgress
	gotRecord   *simpleservice.SimpleEgress
	created     *simpleservice.SimpleEgress
	updated     *simpleservice.SimpleEgress
	tested      *simpleservice.SimpleEgressTestResult
	parsedShare *simpleservice.CreateSimpleEgressRequest
	exportedURL string
	deletedID   int
}

func (f *fakeService) ListSimpleEgress() ([]*simpleservice.SimpleEgress, error) {
	return f.listRecords, nil
}

func (f *fakeService) GetSimpleEgress(id int) (*simpleservice.SimpleEgress, error) {
	if f.gotRecord != nil {
		return f.gotRecord, nil
	}
	return &simpleservice.SimpleEgress{
		Id:       id,
		Name:     "simple-egress",
		Protocol: "socks5",
		Address:  "example.com",
		Port:     1080,
	}, nil
}

func (f *fakeService) CreateSimpleEgress(req *simpleservice.CreateSimpleEgressRequest) (*simpleservice.SimpleEgress, error) {
	return &simpleservice.SimpleEgress{
		Id:       11,
		Name:     req.Name,
		Protocol: req.Protocol,
		Address:  req.Address,
		Port:     req.Port,
	}, nil
}

func (f *fakeService) UpdateSimpleEgress(id int, req *simpleservice.CreateSimpleEgressRequest) (*simpleservice.SimpleEgress, error) {
	return &simpleservice.SimpleEgress{
		Id:       id,
		Name:     req.Name,
		Protocol: req.Protocol,
		Address:  req.Address,
		Port:     req.Port,
	}, nil
}

func (f *fakeService) DeleteSimpleEgress(id int) error {
	f.deletedID = id
	return nil
}

func (f *fakeService) TestSimpleEgress(id int) (*simpleservice.SimpleEgressTestResult, error) {
	return &simpleservice.SimpleEgressTestResult{
		Id:       id,
		Status:   "success",
		Latency:  18,
		ExitIP:   "203.0.113.8",
		Message:  "",
		TestedAt: 1234567890,
	}, nil
}

func (f *fakeService) ParseShadowsocksShareLink(value string) (*simpleservice.CreateSimpleEgressRequest, error) {
	if f.parsedShare != nil {
		return f.parsedShare, nil
	}
	return &simpleservice.CreateSimpleEgressRequest{
		Name: "parsed-ss", Protocol: "ss", Address: "ss.example.com", Port: 8388,
	}, nil
}

func (f *fakeService) ExportShadowsocksShareLink(id int) (string, error) {
	if f.exportedURL != "" {
		return f.exportedURL, nil
	}
	return "ss://exported", nil
}

type fakeEgressRestart struct {
	calls int
}

func (f *fakeEgressRestart) SetToNeedRestart() {
	f.calls++
}

func newTestEngine(t *testing.T, svc egressAPI, restart simpleEgressRestartTrigger) *gin.Engine {
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
	controller := &EgressController{service: svc, xrayService: restart}
	controller.initRouter(g)
	return engine
}

func TestSimplePageRouteRender(t *testing.T) {
	engine := newTestEngine(t, &fakeService{}, &fakeEgressRestart{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/n5/simple", nil)
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	for _, expected := range []string{
		"/n5/api/simple/egress/share/parse",
		"/n5/api/simple/egress/share/${row.id}",
		"qrModal.show('Shadowsocks 分享二维码', value)",
	} {
		if !bytes.Contains(w.Body.Bytes(), []byte(expected)) {
			t.Fatalf("simple page missing %q", expected)
		}
	}

	editResp := httptest.NewRecorder()
	editReq, _ := http.NewRequest(http.MethodGet, "/n5/simple/edit?id=1", nil)
	engine.ServeHTTP(editResp, editReq)
	if editResp.Code != http.StatusOK {
		t.Fatalf("unexpected edit page status: %d", editResp.Code)
	}
	if !bytes.Contains(editResp.Body.Bytes(), []byte("/n5/api/simple/egress/share/parse")) {
		t.Fatal("simple edit page does not use the backend Shadowsocks share parser")
	}
}

func TestSimpleEgressAPIResponses(t *testing.T) {
	svc := &fakeService{
		listRecords: []*simpleservice.SimpleEgress{
			{
				Id:       1,
				Name:     "simple-egress",
				Protocol: "socks5",
				Address:  "example.com",
				Port:     1080,
			},
		},
	}
	restart := &fakeEgressRestart{}
	engine := newTestEngine(t, svc, restart)

	listResp := httptest.NewRecorder()
	listReq, _ := http.NewRequest(http.MethodGet, "/n5/api/simple/egress/list", nil)
	engine.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("unexpected list status: %d", listResp.Code)
	}
	if !bytes.Contains(listResp.Body.Bytes(), []byte(`"name":"simple-egress"`)) {
		t.Fatalf("unexpected list body: %s", listResp.Body.String())
	}

	getResp := httptest.NewRecorder()
	getReq, _ := http.NewRequest(http.MethodGet, "/n5/api/simple/egress/get/1", nil)
	engine.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("unexpected get status: %d", getResp.Code)
	}
	if !bytes.Contains(getResp.Body.Bytes(), []byte(`"id":1`)) {
		t.Fatalf("unexpected get body: %s", getResp.Body.String())
	}

	addBody := bytes.NewBufferString(`{"name":"simple-added","protocol":"socks5","address":"example.org","port":1080,"enabled":true}`)
	addReq, _ := http.NewRequest(http.MethodPost, "/n5/api/simple/egress/add", addBody)
	addReq.Header.Set("Content-Type", "application/json")
	addResp := httptest.NewRecorder()
	engine.ServeHTTP(addResp, addReq)
	if addResp.Code != http.StatusOK {
		t.Fatalf("unexpected add status: %d", addResp.Code)
	}
	if !bytes.Contains(addResp.Body.Bytes(), []byte(`"name":"simple-added"`)) {
		t.Fatalf("unexpected add body: %s", addResp.Body.String())
	}
	if restart.calls != 1 {
		t.Fatalf("unexpected restart count after add: %d", restart.calls)
	}

	updateBody := bytes.NewBufferString(`{"name":"simple-updated","protocol":"ss","address":"ss.example.org","port":8388,"method":"aes-256-gcm","password":"demo","enabled":true}`)
	updateReq, _ := http.NewRequest(http.MethodPost, "/n5/api/simple/egress/update/5", updateBody)
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp := httptest.NewRecorder()
	engine.ServeHTTP(updateResp, updateReq)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("unexpected update status: %d", updateResp.Code)
	}
	if !bytes.Contains(updateResp.Body.Bytes(), []byte(`"name":"simple-updated"`)) {
		t.Fatalf("unexpected update body: %s", updateResp.Body.String())
	}
	if restart.calls != 2 {
		t.Fatalf("unexpected restart count after update: %d", restart.calls)
	}

	testBody := bytes.NewBufferString(`{"id":1}`)
	testReq, _ := http.NewRequest(http.MethodPost, "/n5/api/simple/egress/test", testBody)
	testReq.Header.Set("Content-Type", "application/json")
	testResp := httptest.NewRecorder()
	engine.ServeHTTP(testResp, testReq)
	if testResp.Code != http.StatusOK {
		t.Fatalf("unexpected test status: %d", testResp.Code)
	}
	if !bytes.Contains(testResp.Body.Bytes(), []byte(`"status":"success"`)) {
		t.Fatalf("unexpected test body: %s", testResp.Body.String())
	}

	delBody := bytes.NewBufferString(`{"id":9}`)
	delReq, _ := http.NewRequest(http.MethodPost, "/n5/api/simple/egress/delete", delBody)
	delReq.Header.Set("Content-Type", "application/json")
	delResp := httptest.NewRecorder()
	engine.ServeHTTP(delResp, delReq)
	if delResp.Code != http.StatusOK {
		t.Fatalf("unexpected delete status: %d", delResp.Code)
	}
	if svc.deletedID != 9 {
		t.Fatalf("unexpected deleted id: %d", svc.deletedID)
	}
	if restart.calls != 3 {
		t.Fatalf("unexpected restart count after delete: %d", restart.calls)
	}
}

func TestSimpleEgressShareAPIResponses(t *testing.T) {
	svc := &fakeService{
		parsedShare: &simpleservice.CreateSimpleEgressRequest{
			Name: "SS2022 节点", Protocol: "ss", Address: "ss.example.com", Port: 8388,
			Method: "2022-blake3-aes-128-gcm", Password: "AAAAAAAAAAAAAAAAAAAAAA==", Enabled: true,
		},
		exportedURL: "ss://canonical-export",
	}
	engine := newTestEngine(t, svc, &fakeEgressRestart{})

	parseBody := bytes.NewBufferString(`{"link":"ss://input"}`)
	parseReq, _ := http.NewRequest(http.MethodPost, "/n5/api/simple/egress/share/parse", parseBody)
	parseReq.Header.Set("Content-Type", "application/json")
	parseResp := httptest.NewRecorder()
	engine.ServeHTTP(parseResp, parseReq)
	if parseResp.Code != http.StatusOK || !bytes.Contains(parseResp.Body.Bytes(), []byte(`"method":"2022-blake3-aes-128-gcm"`)) {
		t.Fatalf("unexpected parse response: %d %s", parseResp.Code, parseResp.Body.String())
	}

	exportReq, _ := http.NewRequest(http.MethodGet, "/n5/api/simple/egress/share/7", nil)
	exportResp := httptest.NewRecorder()
	engine.ServeHTTP(exportResp, exportReq)
	if exportResp.Code != http.StatusOK || !bytes.Contains(exportResp.Body.Bytes(), []byte(`"url":"ss://canonical-export"`)) {
		t.Fatalf("unexpected export response: %d %s", exportResp.Code, exportResp.Body.String())
	}
}
