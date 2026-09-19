package n5

import (
	"strconv"
	n5model "x-ui/database/model/n5"
	n5service "x-ui/web/service/n5"

	"github.com/gin-gonic/gin"
)

type EgressLabelController struct {
	service n5service.EgressLabelService
}

func NewEgressLabelController(g *gin.RouterGroup) *EgressLabelController {
	a := &EgressLabelController{}
	a.initRouter(g)
	return a
}

func (a *EgressLabelController) initRouter(g *gin.RouterGroup) {
	apiGroup := g.Group("/n5/api/egress-label")
	apiGroup.Use(checkLogin)
	apiGroup.GET("/list", a.list)
	apiGroup.POST("/add", a.add)
	apiGroup.POST("/update/:id", a.update)
	apiGroup.POST("/del/:id", a.del)
	apiGroup.POST("/bind", a.bind)
	apiGroup.POST("/unbind", a.unbind)
}

func (a *EgressLabelController) list(c *gin.Context) {
	records, err := a.service.List()
	if err != nil {
		jsonMsg(c, "list egress labels", err)
		return
	}
	jsonObj(c, records, nil)
}

func (a *EgressLabelController) add(c *gin.Context) {
	record := &n5model.EgressLabel{}
	if err := c.ShouldBind(record); err != nil {
		jsonMsg(c, "add egress label", err)
		return
	}
	created, err := a.service.Create(record)
	if err != nil {
		jsonMsg(c, "add egress label", err)
		return
	}
	jsonObj(c, created, nil)
}

func (a *EgressLabelController) update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "update egress label", err)
		return
	}
	record := &n5model.EgressLabel{Id: id}
	if err := c.ShouldBind(record); err != nil {
		jsonMsg(c, "update egress label", err)
		return
	}
	updated, err := a.service.Update(record)
	if err != nil {
		jsonMsg(c, "update egress label", err)
		return
	}
	jsonObj(c, updated, nil)
}

func (a *EgressLabelController) del(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "delete egress label", err)
		return
	}
	jsonMsg(c, "delete egress label", a.service.Delete(id))
}

func (a *EgressLabelController) bind(c *gin.Context) {
	payload := struct {
		EgressId int `json:"egressId" form:"egressId"`
		LabelId  int `json:"labelId" form:"labelId"`
	}{}
	if err := c.ShouldBind(&payload); err != nil {
		jsonMsg(c, "bind egress label", err)
		return
	}
	record, err := a.service.Bind(payload.EgressId, payload.LabelId)
	if err != nil {
		jsonMsg(c, "bind egress label", err)
		return
	}
	jsonObj(c, record, nil)
}

func (a *EgressLabelController) unbind(c *gin.Context) {
	payload := struct {
		EgressId int `json:"egressId" form:"egressId"`
		LabelId  int `json:"labelId" form:"labelId"`
	}{}
	if err := c.ShouldBind(&payload); err != nil {
		jsonMsg(c, "unbind egress label", err)
		return
	}
	jsonMsg(c, "unbind egress label", a.service.Unbind(payload.EgressId, payload.LabelId))
}
