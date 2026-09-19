package controller

import (
	"github.com/gin-gonic/gin"
	"x-ui/web/service"
)

type ShareAddressController struct {
	shareAddressService service.ShareAddressService
}

type shareAddressForm struct {
	Address string `json:"address" form:"address"`
	Remark  string `json:"remark" form:"remark"`
}

func NewShareAddressController(g *gin.RouterGroup) *ShareAddressController {
	a := &ShareAddressController{}
	a.initRouter(g)
	return a
}

func (a *ShareAddressController) initRouter(g *gin.RouterGroup) {
	g = g.Group("/share-domains")

	g.GET("", a.list)
	g.POST("", a.add)
	g.PUT("/:id", a.update)
	g.DELETE("/:id", a.delete)
}

func (a *ShareAddressController) list(c *gin.Context) {
	addresses, err := a.shareAddressService.GetAll()
	if err != nil {
		jsonMsg(c, "get share addresses", err)
		return
	}
	jsonObj(c, addresses, nil)
}

func (a *ShareAddressController) add(c *gin.Context) {
	form := &shareAddressForm{}
	if err := c.ShouldBind(form); err != nil {
		jsonMsg(c, "add share address", err)
		return
	}
	address, err := a.shareAddressService.Add(form.Address, form.Remark)
	jsonObj(c, address, err)
}

func (a *ShareAddressController) update(c *gin.Context) {
	form := &shareAddressForm{}
	if err := c.ShouldBind(form); err != nil {
		jsonMsg(c, "update share address", err)
		return
	}
	address, err := a.shareAddressService.Update(c.Param("id"), form.Address, form.Remark)
	jsonObj(c, address, err)
}

func (a *ShareAddressController) delete(c *gin.Context) {
	err := a.shareAddressService.Delete(c.Param("id"))
	jsonMsg(c, "delete share address", err)
}
