package n5

import (
	"strconv"
	coreservice "x-ui/web/service"
	n5service "x-ui/web/service/n5"

	"github.com/gin-gonic/gin"
)

type TrafficTemplateController struct {
	service     n5service.TrafficTemplateService
	xrayService templateRestartTrigger
}

type templateRestartTrigger interface {
	SetToNeedRestart()
}

func NewTrafficTemplateController(g *gin.RouterGroup) *TrafficTemplateController {
	a := &TrafficTemplateController{
		xrayService: &coreservice.XrayService{},
	}
	a.initRouter(g)
	return a
}

func (a *TrafficTemplateController) initRouter(g *gin.RouterGroup) {
	apiGroup := g.Group("/n5/api/traffic-template")
	apiGroup.Use(checkLogin)
	apiGroup.GET("/list", a.list)
	apiGroup.GET("/preview/:name", a.preview)
	apiGroup.POST("/create", a.create)
}

func (a *TrafficTemplateController) list(c *gin.Context) {
	records, err := a.service.List()
	if err != nil {
		jsonMsg(c, "list traffic templates", err)
		return
	}
	jsonObj(c, records, nil)
}

func (a *TrafficTemplateController) preview(c *gin.Context) {
	name := c.Param("name")
	record, err := a.service.Preview(name)
	if err != nil {
		jsonMsg(c, "preview traffic template", err)
		return
	}
	jsonObj(c, record, nil)
}

func (a *TrafficTemplateController) create(c *gin.Context) {
	payload := &n5service.TrafficTemplateCreateRequest{}
	if err := c.ShouldBind(payload); err != nil {
		jsonMsg(c, "create traffic template", err)
		return
	}
	record, err := a.service.Create(payload)
	if err != nil {
		jsonMsg(c, "create traffic template", err)
		return
	}
	a.getXrayService().SetToNeedRestart()
	jsonObj(c, record, nil)
}

func parsePositiveInt(value string) int {
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func (a *TrafficTemplateController) getXrayService() templateRestartTrigger {
	if a.xrayService != nil {
		return a.xrayService
	}
	return &coreservice.XrayService{}
}
