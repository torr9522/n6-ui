package n5

import (
	"github.com/gin-gonic/gin"
	n5service "x-ui/web/service/n5"
)

type XrayController struct {
	statusService  n5service.XrayStatusService
	historyService n5service.XrayHistoryService
}

func NewXrayController(g *gin.RouterGroup) *XrayController {
	a := &XrayController{}
	a.initRouter(g)
	return a
}

func (a *XrayController) initRouter(g *gin.RouterGroup) {
	pageGroup := g.Group("/n5")
	pageGroup.Use(checkLogin)
	pageGroup.GET("/xray-status", a.statusPage)
	pageGroup.GET("/config-history", a.historyPage)
	pageGroup.GET("/egress-test", a.egressTestPage)

	apiGroup := g.Group("/n5/api/xray")
	apiGroup.Use(checkLogin)
	apiGroup.GET("/status", a.status)
	apiGroup.POST("/status", a.status)
	apiGroup.POST("/history/list", a.historyList)
	apiGroup.GET("/history/list", a.historyList)
	apiGroup.POST("/egress-test/entry", a.egressTestEntry)
	apiGroup.GET("/egress-test/entry", a.egressTestEntry)
}

func (a *XrayController) statusPage(c *gin.Context) {
	html(c, "xray_status.html", "N5运行状态", nil)
}

func (a *XrayController) historyPage(c *gin.Context) {
	html(c, "config_history.html", "配置历史", nil)
}

func (a *XrayController) egressTestPage(c *gin.Context) {
	html(c, "egress_test.html", "出口测试入口", nil)
}

func (a *XrayController) status(c *gin.Context) {
	status, err := a.statusService.GetStatus()
	if err != nil {
		jsonMsg(c, "get n5 xray status", err)
		return
	}
	jsonObj(c, status, nil)
}

func (a *XrayController) historyList(c *gin.Context) {
	payload := struct {
		Limit int `json:"limit" form:"limit"`
	}{}
	_ = c.ShouldBind(&payload)
	records, err := a.historyService.List(payload.Limit)
	if err != nil {
		jsonMsg(c, "get n5 xray history", err)
		return
	}
	jsonObj(c, records, nil)
}

func (a *XrayController) egressTestEntry(c *gin.Context) {
	jsonObj(c, gin.H{
		"supported":   true,
		"planned":     false,
		"mode":        "manual-temporary-xray-test",
		"description": "当前版本支持手动触发临时 xray 子配置测试，不接入主运行链路，不启动后台常驻 worker。",
		"inputs": []string{
			"egressId",
		},
		"outputs": []string{
			"status",
			"latency",
			"exit_ip",
			"message",
			"tested_at",
		},
	}, nil)
}
