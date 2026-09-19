package simple

import (
	"github.com/torr9522/n6-ui/util/common"
	simpleservice "github.com/torr9522/n6-ui/web/service/n5/simple"

	"github.com/gin-gonic/gin"
)

type trafficRuleGroupAPI interface {
	ListGroups() ([]*simpleservice.TrafficRuleGroup, error)
	GetGroup(id int) (*simpleservice.TrafficRuleGroup, error)
	CreateGroup(req *simpleservice.CreateTrafficRuleGroupRequest) (*simpleservice.TrafficRuleGroup, error)
	UpdateGroup(id int, req *simpleservice.UpdateTrafficRuleGroupRequest) (*simpleservice.TrafficRuleGroup, error)
	DeleteGroup(id int) error
	AddDomainRule(req *simpleservice.AddTrafficRuleDomainRequest) (*simpleservice.TrafficRuleGroupRule, error)
	UpdateDomainRule(ruleId int, req *simpleservice.UpdateTrafficRuleDomainRequest) (*simpleservice.TrafficRuleGroupRule, error)
	DeleteDomainRule(groupId int, ruleId int) error
	EnableGroup(id int) (*simpleservice.TrafficRuleGroup, error)
	DisableGroup(id int) (*simpleservice.TrafficRuleGroup, error)
}

type TrafficRuleGroupController struct {
	service trafficRuleGroupAPI
}

func NewTrafficRuleGroupController(g *gin.RouterGroup) *TrafficRuleGroupController {
	a := &TrafficRuleGroupController{
		service: simpleservice.NewTrafficRuleGroupService(),
	}
	a.initRouter(g)
	return a
}

func (a *TrafficRuleGroupController) initRouter(g *gin.RouterGroup) {
	pageGroup := g.Group("/n5")
	pageGroup.Use(checkLogin)
	pageGroup.GET("/simple/traffic-rules", a.page)
	pageGroup.GET("/simple/traffic-rules/:id", a.detailPage)

	apiGroup := g.Group("/n5/api/simple")
	apiGroup.Use(checkLogin)
	apiGroup.GET("/traffic-rule-groups", a.list)
	apiGroup.GET("/traffic-rule-group/:id", a.get)
	apiGroup.POST("/traffic-rule-group/add", a.add)
	apiGroup.POST("/traffic-rule-group/update/:id", a.update)
	apiGroup.POST("/traffic-rule-group/delete/:id", a.del)
	apiGroup.POST("/traffic-rule-group/enable/:id", a.enable)
	apiGroup.POST("/traffic-rule-group/disable/:id", a.disable)
	apiGroup.POST("/traffic-rule/add", a.addRule)
	apiGroup.POST("/traffic-rule/update/:id", a.updateRule)
	apiGroup.POST("/traffic-rule/delete/:id", a.delRule)
}

func (a *TrafficRuleGroupController) page(c *gin.Context) {
	html(c, "simple_traffic_rules.html", "分流规则", nil)
}

func (a *TrafficRuleGroupController) detailPage(c *gin.Context) {
	html(c, "simple_traffic_rule_group.html", "管理规则", nil)
}

func (a *TrafficRuleGroupController) list(c *gin.Context) {
	items, err := a.service.ListGroups()
	if err != nil {
		jsonMsg(c, "list traffic rule groups", err)
		return
	}
	jsonObj(c, items, nil)
}

func (a *TrafficRuleGroupController) get(c *gin.Context) {
	id := parseID(c.Param("id"))
	if id <= 0 {
		jsonMsg(c, "get traffic rule group", common.NewError("invalid traffic rule group id"))
		return
	}
	item, err := a.service.GetGroup(id)
	if err != nil {
		jsonMsg(c, "get traffic rule group", err)
		return
	}
	jsonObj(c, item, nil)
}

func (a *TrafficRuleGroupController) add(c *gin.Context) {
	record := &simpleservice.CreateTrafficRuleGroupRequest{}
	if err := c.ShouldBind(record); err != nil {
		jsonMsg(c, "add traffic rule group", err)
		return
	}
	item, err := a.service.CreateGroup(record)
	if err != nil {
		jsonMsg(c, "add traffic rule group", err)
		return
	}
	jsonObj(c, item, nil)
}

func (a *TrafficRuleGroupController) update(c *gin.Context) {
	id := parseID(c.Param("id"))
	if id <= 0 {
		jsonMsg(c, "update traffic rule group", common.NewError("invalid traffic rule group id"))
		return
	}
	record := &simpleservice.UpdateTrafficRuleGroupRequest{}
	if err := c.ShouldBind(record); err != nil {
		jsonMsg(c, "update traffic rule group", err)
		return
	}
	item, err := a.service.UpdateGroup(id, record)
	if err != nil {
		jsonMsg(c, "update traffic rule group", err)
		return
	}
	jsonObj(c, item, nil)
}

func (a *TrafficRuleGroupController) del(c *gin.Context) {
	id := parseID(c.Param("id"))
	if id <= 0 {
		jsonMsg(c, "delete traffic rule group", common.NewError("invalid traffic rule group id"))
		return
	}
	jsonMsg(c, "delete traffic rule group", a.service.DeleteGroup(id))
}

func (a *TrafficRuleGroupController) enable(c *gin.Context) {
	id := parseID(c.Param("id"))
	if id <= 0 {
		jsonMsg(c, "enable traffic rule group", common.NewError("invalid traffic rule group id"))
		return
	}
	item, err := a.service.EnableGroup(id)
	if err != nil {
		jsonMsg(c, "enable traffic rule group", err)
		return
	}
	jsonObj(c, item, nil)
}

func (a *TrafficRuleGroupController) disable(c *gin.Context) {
	id := parseID(c.Param("id"))
	if id <= 0 {
		jsonMsg(c, "disable traffic rule group", common.NewError("invalid traffic rule group id"))
		return
	}
	item, err := a.service.DisableGroup(id)
	if err != nil {
		jsonMsg(c, "disable traffic rule group", err)
		return
	}
	jsonObj(c, item, nil)
}

func (a *TrafficRuleGroupController) addRule(c *gin.Context) {
	record := &simpleservice.AddTrafficRuleDomainRequest{}
	if err := c.ShouldBind(record); err != nil {
		jsonMsg(c, "add traffic rule", err)
		return
	}
	item, err := a.service.AddDomainRule(record)
	if err != nil {
		jsonMsg(c, "add traffic rule", err)
		return
	}
	jsonObj(c, item, nil)
}

func (a *TrafficRuleGroupController) updateRule(c *gin.Context) {
	ruleID := parseID(c.Param("id"))
	if ruleID <= 0 {
		jsonMsg(c, "update traffic rule", common.NewError("invalid traffic rule id"))
		return
	}
	record := &simpleservice.UpdateTrafficRuleDomainRequest{}
	if err := c.ShouldBind(record); err != nil {
		jsonMsg(c, "update traffic rule", err)
		return
	}
	item, err := a.service.UpdateDomainRule(ruleID, record)
	if err != nil {
		jsonMsg(c, "update traffic rule", err)
		return
	}
	jsonObj(c, item, nil)
}

func (a *TrafficRuleGroupController) delRule(c *gin.Context) {
	ruleID := parseID(c.Param("id"))
	payload := struct {
		GroupId int `json:"groupId" form:"groupId"`
	}{}
	if err := c.ShouldBind(&payload); err != nil {
		jsonMsg(c, "delete traffic rule", err)
		return
	}
	if ruleID <= 0 || payload.GroupId <= 0 {
		jsonMsg(c, "delete traffic rule", common.NewError("invalid traffic rule id"))
		return
	}
	jsonMsg(c, "delete traffic rule", a.service.DeleteDomainRule(payload.GroupId, ruleID))
}
