package simple

import (
	"github.com/torr9522/n6-ui/config"
	"github.com/torr9522/n6-ui/util/common"
	"github.com/torr9522/n6-ui/web/entity"
	coreservice "github.com/torr9522/n6-ui/web/service"
	simpleservice "github.com/torr9522/n6-ui/web/service/n5/simple"
	ssservice "github.com/torr9522/n6-ui/web/service/shadowsocks"
	"github.com/torr9522/n6-ui/web/session"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
)

type egressAPI interface {
	ListSimpleEgress() ([]*simpleservice.SimpleEgress, error)
	GetSimpleEgress(id int) (*simpleservice.SimpleEgress, error)
	CreateSimpleEgress(req *simpleservice.CreateSimpleEgressRequest) (*simpleservice.SimpleEgress, error)
	UpdateSimpleEgress(id int, req *simpleservice.CreateSimpleEgressRequest) (*simpleservice.SimpleEgress, error)
	DeleteSimpleEgress(id int) error
	TestSimpleEgress(id int) (*simpleservice.SimpleEgressTestResult, error)
	ParseShadowsocksShareLink(value string) (*simpleservice.CreateSimpleEgressRequest, error)
	ExportShadowsocksShareLink(id int) (string, error)
}

type EgressController struct {
	service     egressAPI
	xrayService simpleEgressRestartTrigger
}

func NewEgressController(g *gin.RouterGroup) *EgressController {
	a := &EgressController{
		service:     simpleservice.NewEgressService(),
		xrayService: &coreservice.XrayService{},
	}
	a.initRouter(g)
	return a
}

type simpleEgressRestartTrigger interface {
	SetToNeedRestart()
}

func (a *EgressController) initRouter(g *gin.RouterGroup) {
	pageGroup := g.Group("/n5")
	pageGroup.Use(checkLogin)
	pageGroup.GET("/simple", a.page)
	pageGroup.GET("/simple/edit", a.editPage)

	apiGroup := g.Group("/n5/api/simple/egress")
	apiGroup.Use(checkLogin)
	apiGroup.GET("/list", a.list)
	apiGroup.GET("/get/:id", a.get)
	apiGroup.POST("/add", a.add)
	apiGroup.POST("/update/:id", a.update)
	apiGroup.POST("/test", a.test)
	apiGroup.POST("/delete", a.del)
	apiGroup.POST("/shadowsocks/key", a.generateShadowsocksKey)
	apiGroup.POST("/share/parse", a.parseShadowsocksShareLink)
	apiGroup.GET("/share/:id", a.exportShadowsocksShareLink)
}

func (a *EgressController) generateShadowsocksKey(c *gin.Context) {
	payload := struct {
		Method string `json:"method" form:"method"`
	}{}
	if err := c.ShouldBind(&payload); err != nil {
		jsonMsg(c, "生成 Shadowsocks 2022 密钥", err)
		return
	}
	key, err := ssservice.Generate2022Key(payload.Method)
	if err != nil {
		jsonMsg(c, "生成 Shadowsocks 2022 密钥", err)
		return
	}
	jsonObj(c, gin.H{
		"method":   ssservice.NormalizeMethod(payload.Method),
		"password": key,
	}, nil)
}

func (a *EgressController) parseShadowsocksShareLink(c *gin.Context) {
	payload := struct {
		Link string `json:"link" form:"link"`
	}{}
	if err := c.ShouldBind(&payload); err != nil {
		jsonMsg(c, "解析 Shadowsocks 分享链接", err)
		return
	}
	result, err := a.service.ParseShadowsocksShareLink(payload.Link)
	if err != nil {
		jsonMsg(c, "解析 Shadowsocks 分享链接", err)
		return
	}
	jsonObj(c, result, nil)
}

func (a *EgressController) exportShadowsocksShareLink(c *gin.Context) {
	id := parseID(c.Param("id"))
	if id <= 0 {
		jsonMsg(c, "导出 Shadowsocks 分享链接", common.NewError("invalid simple egress id"))
		return
	}
	value, err := a.service.ExportShadowsocksShareLink(id)
	if err != nil {
		jsonMsg(c, "导出 Shadowsocks 分享链接", err)
		return
	}
	jsonObj(c, gin.H{"url": value}, nil)
}

func (a *EgressController) page(c *gin.Context) {
	html(c, "simple.html", "n6-ui 出口", nil)
}

func (a *EgressController) editPage(c *gin.Context) {
	html(c, "simple_egress_edit.html", "编辑出口", nil)
}

func (a *EgressController) list(c *gin.Context) {
	records, err := a.service.ListSimpleEgress()
	if err != nil {
		jsonMsg(c, "list simple egress", err)
		return
	}
	jsonObj(c, records, nil)
}

func (a *EgressController) get(c *gin.Context) {
	id := parseID(c.Param("id"))
	if id <= 0 {
		jsonMsg(c, "get simple egress", common.NewError("invalid simple egress id"))
		return
	}
	record, err := a.service.GetSimpleEgress(id)
	if err != nil {
		jsonMsg(c, "get simple egress", err)
		return
	}
	jsonObj(c, record, nil)
}

func (a *EgressController) add(c *gin.Context) {
	record := &simpleservice.CreateSimpleEgressRequest{}
	if err := c.ShouldBind(record); err != nil {
		jsonMsg(c, "add simple egress", err)
		return
	}
	if !record.Enabled {
		record.Enabled = true
	}
	created, err := a.service.CreateSimpleEgress(record)
	if err != nil {
		jsonMsg(c, "add simple egress", err)
		return
	}
	a.getXrayService().SetToNeedRestart()
	jsonObj(c, created, nil)
}

func (a *EgressController) update(c *gin.Context) {
	id := parseID(c.Param("id"))
	if id <= 0 {
		jsonMsg(c, "update simple egress", common.NewError("invalid simple egress id"))
		return
	}
	record := &simpleservice.CreateSimpleEgressRequest{}
	if err := c.ShouldBind(record); err != nil {
		jsonMsg(c, "update simple egress", err)
		return
	}
	if !record.Enabled {
		record.Enabled = true
	}
	updated, err := a.service.UpdateSimpleEgress(id, record)
	if err != nil {
		jsonMsg(c, "update simple egress", err)
		return
	}
	a.getXrayService().SetToNeedRestart()
	jsonObj(c, updated, nil)
}

func (a *EgressController) test(c *gin.Context) {
	payload := struct {
		Id int `json:"id" form:"id"`
	}{}
	if err := c.ShouldBind(&payload); err != nil {
		jsonMsg(c, "test simple egress", err)
		return
	}
	record, err := a.service.TestSimpleEgress(payload.Id)
	if err != nil {
		jsonMsg(c, "test simple egress", err)
		return
	}
	jsonObj(c, record, nil)
}

func (a *EgressController) del(c *gin.Context) {
	payload := struct {
		Id int `json:"id" form:"id"`
	}{}
	if err := c.ShouldBind(&payload); err != nil {
		jsonMsg(c, "delete simple egress", err)
		return
	}
	err := a.service.DeleteSimpleEgress(payload.Id)
	if err == nil {
		a.getXrayService().SetToNeedRestart()
	}
	jsonMsg(c, "delete simple egress", err)
}

func (a *EgressController) getXrayService() simpleEgressRestartTrigger {
	if a.xrayService != nil {
		return a.xrayService
	}
	return &coreservice.XrayService{}
}

func checkLogin(c *gin.Context) {
	if !session.IsLogin(c) {
		if isAjax(c) {
			pureJsonMsg(c, false, "登录时效已过，请重新登录")
		} else {
			c.Redirect(http.StatusTemporaryRedirect, c.GetString("base_path"))
		}
		c.Abort()
		return
	}
	c.Next()
}

func isAjax(c *gin.Context) bool {
	return c.GetHeader("X-Requested-With") == "XMLHttpRequest"
}

func jsonMsg(c *gin.Context, msg string, err error) {
	jsonMsgObj(c, msg, nil, err)
}

func jsonObj(c *gin.Context, obj interface{}, err error) {
	jsonMsgObj(c, "", obj, err)
}

func jsonMsgObj(c *gin.Context, msg string, obj interface{}, err error) {
	m := entity.Msg{Obj: obj}
	if err == nil {
		m.Success = true
		if msg != "" {
			m.Msg = msg + "成功"
		}
	} else {
		m.Success = false
		m.Msg = msg + "失败: " + common.SafeErrorMessage(err)
	}
	c.JSON(http.StatusOK, m)
}

func pureJsonMsg(c *gin.Context, success bool, msg string) {
	c.JSON(http.StatusOK, entity.Msg{
		Success: success,
		Msg:     msg,
	})
}

func html(c *gin.Context, name string, title string, data gin.H) {
	if data == nil {
		data = gin.H{}
	}
	data["title"] = title
	data["request_uri"] = c.Request.RequestURI
	data["base_path"] = c.GetString("base_path")
	data["cur_ver"] = config.GetVersion()
	c.HTML(http.StatusOK, name, data)
}

func testFiles() []string {
	files := make([]string, 0)
	root := filepath.Join("..", "..", "..", "html")
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
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
	})
	return files
}

func parseID(value string) int {
	id, err := strconv.Atoi(value)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}
