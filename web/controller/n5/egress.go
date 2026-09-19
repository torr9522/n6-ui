package n5

import (
	"github.com/gin-gonic/gin"
	"github.com/torr9522/n6-ui/config"
	n5model "github.com/torr9522/n6-ui/database/model/n5"
	"github.com/torr9522/n6-ui/logger"
	"github.com/torr9522/n6-ui/util/common"
	"github.com/torr9522/n6-ui/web/entity"
	coreservice "github.com/torr9522/n6-ui/web/service"
	n5service "github.com/torr9522/n6-ui/web/service/n5"
	"github.com/torr9522/n6-ui/web/session"
	"net/http"
	"strconv"
	"strings"
)

type EgressController struct {
	egressService     n5service.EgressService
	detailService     n5service.EgressDetailService
	egressTestService n5service.EgressTestService
	xrayService       n5RestartTrigger
}

type n5RestartTrigger interface {
	SetToNeedRestart()
}

func NewEgressController(g *gin.RouterGroup) *EgressController {
	a := &EgressController{
		xrayService: &coreservice.XrayService{},
	}
	a.initRouter(g)
	return a
}

func (a *EgressController) initRouter(g *gin.RouterGroup) {
	pageGroup := g.Group("/n5")
	pageGroup.Use(checkLogin)
	pageGroup.GET("/egress", a.page)
	pageGroup.GET("/egress-detail", a.detailPage)

	apiGroup := g.Group("/n5/api/egress")
	apiGroup.Use(checkLogin)
	apiGroup.POST("/list", a.list)
	apiGroup.POST("/get/:id", a.get)
	apiGroup.GET("/detail/:id", a.detail)
	apiGroup.POST("/add", a.add)
	apiGroup.POST("/update/:id", a.update)
	apiGroup.POST("/del/:id", a.del)
	apiGroup.POST("/validate", a.validate)
	apiGroup.POST("/test", a.test)
}

func (a *EgressController) page(c *gin.Context) {
	html(c, "egress.html", "出口线路", nil)
}

func (a *EgressController) detailPage(c *gin.Context) {
	html(c, "egress_detail.html", "出口详情", nil)
}

func (a *EgressController) list(c *gin.Context) {
	records, err := a.egressService.List()
	if err != nil {
		jsonMsg(c, "list egress", err)
		return
	}
	jsonObj(c, records, nil)
}

func (a *EgressController) get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "get egress", err)
		return
	}
	record, err := a.egressService.Get(id)
	if err != nil {
		jsonMsg(c, "get egress", err)
		return
	}
	jsonObj(c, record, nil)
}

func (a *EgressController) detail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "get egress detail", err)
		return
	}
	record, err := a.detailService.Get(id)
	if err != nil {
		jsonMsg(c, "get egress detail", err)
		return
	}
	jsonObj(c, record, nil)
}

func (a *EgressController) add(c *gin.Context) {
	record := &n5model.Egress{}
	if err := c.ShouldBind(record); err != nil {
		jsonMsg(c, "add egress", err)
		return
	}
	created, err := a.egressService.Create(record)
	if err != nil {
		jsonMsg(c, "add egress", err)
		return
	}
	a.getXrayService().SetToNeedRestart()
	jsonObj(c, created, nil)
}

func (a *EgressController) update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "update egress", err)
		return
	}
	record := &n5model.Egress{Id: id}
	if err := c.ShouldBind(record); err != nil {
		jsonMsg(c, "update egress", err)
		return
	}
	updated, err := a.egressService.Update(record)
	if err != nil {
		jsonMsg(c, "update egress", err)
		return
	}
	a.getXrayService().SetToNeedRestart()
	jsonObj(c, updated, nil)
}

func (a *EgressController) del(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "delete egress", err)
		return
	}
	err = a.egressService.Delete(id)
	if err == nil {
		a.getXrayService().SetToNeedRestart()
	}
	jsonMsg(c, "delete egress", err)
}

func (a *EgressController) validate(c *gin.Context) {
	payload := struct {
		Protocol     string `json:"protocol" form:"protocol"`
		OutboundJSON string `json:"outboundJson" form:"outboundJson"`
		Tag          string `json:"tag" form:"tag"`
	}{}
	if err := c.ShouldBind(&payload); err != nil {
		jsonMsg(c, "validate egress", err)
		return
	}
	protocol, normalizedJSON, err := a.egressService.ValidateConfig(payload.Protocol, payload.OutboundJSON, strings.TrimSpace(payload.Tag))
	if err != nil {
		jsonMsg(c, "validate egress", err)
		return
	}
	jsonObj(c, gin.H{
		"protocol":     protocol,
		"outboundJson": normalizedJSON,
	}, nil)
}

func (a *EgressController) test(c *gin.Context) {
	payload := struct {
		Id int `json:"id" form:"id"`
	}{}
	if err := c.ShouldBind(&payload); err != nil {
		jsonMsg(c, "test egress", err)
		return
	}
	record, err := a.egressTestService.Test(payload.Id)
	if err != nil {
		jsonMsg(c, "test egress", err)
		return
	}
	jsonObj(c, record, nil)
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
		logger.Warning(msg+"失败: ", err)
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
	if len(c.Errors) > 0 {
		logger.Warning("render html failed: ", name, " errors: ", c.Errors.String())
	}
}

func (a *EgressController) getXrayService() n5RestartTrigger {
	if a.xrayService != nil {
		return a.xrayService
	}
	return &coreservice.XrayService{}
}
