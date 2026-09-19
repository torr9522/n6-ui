package simple

import (
	"github.com/torr9522/n6-ui/web/entity"
	coreservice "github.com/torr9522/n6-ui/web/service"
	simpleservice "github.com/torr9522/n6-ui/web/service/n5/simple"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ruleAPI interface {
	ListSimpleRules() (*simpleservice.SimpleRuleListResult, error)
	CreateSimpleRule(req *simpleservice.CreateSimpleRuleRequest) (*simpleservice.SimpleRule, error)
	UpdateSimpleRule(ruleId string, req *simpleservice.CreateSimpleRuleRequest) (*simpleservice.SimpleRule, error)
	DeleteSimpleRule(ruleId string) error
	CheckSimpleRuleConflicts(req *simpleservice.CreateSimpleRuleRequest, editingRuleId string) (*simpleservice.SimpleRuleConflictPreview, error)
}

type ruleRestartTrigger interface {
	SetToNeedRestart()
}

type ruleSettingAPI interface {
	GetAllSetting() (*entity.AllSetting, error)
	GetN5XrayExtensionEnable() (bool, error)
	UpdateAllSetting(allSetting *entity.AllSetting) error
}

type RuleController struct {
	service        ruleAPI
	xrayService    ruleRestartTrigger
	settingService ruleSettingAPI
}

func NewRuleController(g *gin.RouterGroup) *RuleController {
	a := &RuleController{
		service:        simpleservice.NewRuleService(),
		xrayService:    &coreservice.XrayService{},
		settingService: &coreservice.SettingService{},
	}
	a.initRouter(g)
	return a
}

func (a *RuleController) initRouter(g *gin.RouterGroup) {
	pageGroup := g.Group("/n5")
	pageGroup.Use(checkLogin)
	pageGroup.GET("/simple/rules", a.page)

	apiGroup := g.Group("/n5/api/simple/rule")
	apiGroup.Use(checkLogin)
	apiGroup.GET("/list", a.list)
	apiGroup.POST("/add", a.add)
	apiGroup.POST("/update", a.update)
	apiGroup.POST("/conflicts", a.conflicts)
	apiGroup.POST("/delete", a.del)
	apiGroup.GET("/n5-status", a.n5Status)
	apiGroup.POST("/n5-status", a.updateN5Status)
}

func (a *RuleController) page(c *gin.Context) {
	html(c, "simple_rules.html", "出口规则", nil)
}

func (a *RuleController) list(c *gin.Context) {
	record, err := a.service.ListSimpleRules()
	if err != nil {
		jsonMsg(c, "list simple rule", err)
		return
	}
	jsonObj(c, record, nil)
}

func (a *RuleController) add(c *gin.Context) {
	record := &simpleservice.CreateSimpleRuleRequest{}
	if err := c.ShouldBind(record); err != nil {
		jsonMsg(c, "add simple rule", err)
		return
	}
	created, err := a.service.CreateSimpleRule(record)
	if err != nil {
		jsonMsg(c, "add simple rule", err)
		return
	}
	a.getXrayService().SetToNeedRestart()
	jsonObj(c, created, nil)
}

func (a *RuleController) update(c *gin.Context) {
	payload := struct {
		RuleId string `json:"ruleId" form:"ruleId"`
		simpleservice.CreateSimpleRuleRequest
	}{}
	if err := c.ShouldBind(&payload); err != nil {
		jsonMsg(c, "update simple rule", err)
		return
	}
	if payload.RuleId == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "update simple rule失败: invalid simple rule id",
		})
		return
	}
	updated, err := a.service.UpdateSimpleRule(payload.RuleId, &payload.CreateSimpleRuleRequest)
	if err != nil {
		jsonMsg(c, "update simple rule", err)
		return
	}
	a.getXrayService().SetToNeedRestart()
	jsonObj(c, updated, nil)
}

func (a *RuleController) conflicts(c *gin.Context) {
	payload := struct {
		RuleId string `json:"ruleId" form:"ruleId"`
		simpleservice.CreateSimpleRuleRequest
	}{}
	if err := c.ShouldBind(&payload); err != nil {
		jsonMsg(c, "check simple rule conflicts", err)
		return
	}
	preview, err := a.service.CheckSimpleRuleConflicts(&payload.CreateSimpleRuleRequest, payload.RuleId)
	if err != nil {
		jsonMsg(c, "check simple rule conflicts", err)
		return
	}
	jsonObj(c, preview, nil)
}

func (a *RuleController) del(c *gin.Context) {
	payload := struct {
		Id     int    `json:"id" form:"id"`
		RuleId string `json:"ruleId" form:"ruleId"`
	}{}
	if err := c.ShouldBind(&payload); err != nil {
		jsonMsg(c, "delete simple rule", err)
		return
	}
	ruleId := payload.RuleId
	if ruleId == "" && payload.Id > 0 {
		ruleId = strconv.Itoa(payload.Id)
	}
	if ruleId == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "delete simple rule失败: invalid simple rule id",
		})
		return
	}
	err := a.service.DeleteSimpleRule(ruleId)
	if err == nil {
		a.getXrayService().SetToNeedRestart()
	}
	jsonMsg(c, "delete simple rule", err)
}

func (a *RuleController) n5Status(c *gin.Context) {
	enabled, err := a.getSettingService().GetN5XrayExtensionEnable()
	if err != nil {
		jsonMsg(c, "get n5 simple status", err)
		return
	}
	jsonObj(c, gin.H{"enabled": enabled}, nil)
}

func (a *RuleController) updateN5Status(c *gin.Context) {
	payload := struct {
		Enabled bool `json:"enabled" form:"enabled"`
	}{}
	if err := c.ShouldBind(&payload); err != nil {
		jsonMsg(c, "update n5 simple status", err)
		return
	}
	allSetting, err := a.getSettingService().GetAllSetting()
	if err != nil {
		jsonMsg(c, "update n5 simple status", err)
		return
	}
	allSetting.N5XrayExtensionEnable = payload.Enabled
	if err := a.getSettingService().UpdateAllSetting(allSetting); err != nil {
		jsonMsg(c, "update n5 simple status", err)
		return
	}
	a.getXrayService().SetToNeedRestart()
	jsonObj(c, gin.H{"enabled": payload.Enabled}, nil)
}

func (a *RuleController) getXrayService() ruleRestartTrigger {
	if a.xrayService != nil {
		return a.xrayService
	}
	return &coreservice.XrayService{}
}

func (a *RuleController) getSettingService() ruleSettingAPI {
	if a.settingService != nil {
		return a.settingService
	}
	return &coreservice.SettingService{}
}
