package n5

import (
	"github.com/gin-gonic/gin"
	"strconv"
	n5model "x-ui/database/model/n5"
	coreservice "x-ui/web/service"
	n5service "x-ui/web/service/n5"
)

type PoolController struct {
	poolService n5service.EgressPoolService
	xrayService poolRestartTrigger
}

type poolRestartTrigger interface {
	SetToNeedRestart()
}

func NewPoolController(g *gin.RouterGroup) *PoolController {
	a := &PoolController{
		xrayService: &coreservice.XrayService{},
	}
	a.initRouter(g)
	return a
}

func (a *PoolController) initRouter(g *gin.RouterGroup) {
	pageGroup := g.Group("/n5")
	pageGroup.Use(checkLogin)
	pageGroup.GET("/pools", a.page)

	apiGroup := g.Group("/n5/api/pool")
	apiGroup.Use(checkLogin)
	apiGroup.POST("/list", a.list)
	apiGroup.POST("/get/:id", a.get)
	apiGroup.POST("/add", a.add)
	apiGroup.POST("/member/list/:id", a.listMembers)
	apiGroup.POST("/member/add", a.addMember)
	apiGroup.POST("/member/del", a.delMember)
}

func (a *PoolController) page(c *gin.Context) {
	html(c, "pools.html", "出口线路池", nil)
}

func (a *PoolController) list(c *gin.Context) {
	records, err := a.poolService.List()
	if err != nil {
		jsonMsg(c, "list pool", err)
		return
	}
	jsonObj(c, records, nil)
}

func (a *PoolController) get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "get pool", err)
		return
	}
	record, err := a.poolService.Get(id)
	if err != nil {
		jsonMsg(c, "get pool", err)
		return
	}
	jsonObj(c, record, nil)
}

func (a *PoolController) add(c *gin.Context) {
	record := &n5model.EgressPool{}
	if err := c.ShouldBind(record); err != nil {
		jsonMsg(c, "add pool", err)
		return
	}
	created, err := a.poolService.Create(record)
	if err != nil {
		jsonMsg(c, "add pool", err)
		return
	}
	jsonObj(c, created, nil)
}

func (a *PoolController) listMembers(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "list pool members", err)
		return
	}
	records, err := a.poolService.ListMembers(id)
	if err != nil {
		jsonMsg(c, "list pool members", err)
		return
	}
	jsonObj(c, records, nil)
}

func (a *PoolController) addMember(c *gin.Context) {
	payload := struct {
		PoolId    int `json:"poolId" form:"poolId"`
		EgressId  int `json:"egressId" form:"egressId"`
		Weight    int `json:"weight" form:"weight"`
		SortOrder int `json:"sortOrder" form:"sortOrder"`
	}{}
	if err := c.ShouldBind(&payload); err != nil {
		jsonMsg(c, "add pool member", err)
		return
	}
	record, err := a.poolService.AddMember(payload.PoolId, payload.EgressId, payload.Weight, payload.SortOrder)
	if err != nil {
		jsonMsg(c, "add pool member", err)
		return
	}
	a.getXrayService().SetToNeedRestart()
	jsonObj(c, record, nil)
}

func (a *PoolController) delMember(c *gin.Context) {
	payload := struct {
		PoolId   int `json:"poolId" form:"poolId"`
		EgressId int `json:"egressId" form:"egressId"`
	}{}
	if err := c.ShouldBind(&payload); err != nil {
		jsonMsg(c, "delete pool member", err)
		return
	}
	err := a.poolService.RemoveMember(payload.PoolId, payload.EgressId)
	if err == nil {
		a.getXrayService().SetToNeedRestart()
	}
	jsonMsg(c, "delete pool member", err)
}

func (a *PoolController) getXrayService() poolRestartTrigger {
	if a.xrayService != nil {
		return a.xrayService
	}
	return &coreservice.XrayService{}
}
