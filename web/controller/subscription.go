package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"x-ui/database"
	"x-ui/web/service"
)

type SubscriptionController struct {
	BaseController
	subscriptionService service.SubscriptionService
	inboundService      service.InboundService
}

func NewSubscriptionController(g *gin.RouterGroup) *SubscriptionController {
	a := &SubscriptionController{}
	a.initAdminRouter(g)
	return a
}

func NewPublicSubscriptionController(g *gin.RouterGroup) *SubscriptionController {
	a := &SubscriptionController{}
	a.initPublicRouter(g)
	return a
}

func (a *SubscriptionController) initAdminRouter(g *gin.RouterGroup) {
	g = g.Group("/subscription")
	g.POST("/list", a.list)
	g.POST("/add", a.add)
	g.POST("/update/:id", a.update)
	g.POST("/del/:id", a.del)
	g.POST("/refresh-token/:id", a.refreshToken)
	g.POST("/inbounds", a.inbounds)
}

func (a *SubscriptionController) initPublicRouter(g *gin.RouterGroup) {
	g.GET("/sub/:token", a.publicGet)
}

func (a *SubscriptionController) list(c *gin.Context) {
	items, err := a.subscriptionService.List()
	jsonObj(c, items, err)
}

func (a *SubscriptionController) add(c *gin.Context) {
	form := &service.SubscriptionForm{}
	if err := c.ShouldBind(form); err != nil {
		jsonMsg(c, "add subscription", err)
		return
	}
	item, err := a.subscriptionService.Add(form)
	jsonObj(c, item, err)
}

func (a *SubscriptionController) update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "update subscription", err)
		return
	}
	form := &service.SubscriptionForm{}
	if err := c.ShouldBind(form); err != nil {
		jsonMsg(c, "update subscription", err)
		return
	}
	item, err := a.subscriptionService.Update(id, form)
	jsonObj(c, item, err)
}

func (a *SubscriptionController) del(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "delete subscription", err)
		return
	}
	err = a.subscriptionService.Delete(id)
	jsonMsg(c, "delete subscription", err)
}

func (a *SubscriptionController) refreshToken(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "refresh subscription token", err)
		return
	}
	item, err := a.subscriptionService.RefreshToken(id)
	jsonObj(c, item, err)
}

func (a *SubscriptionController) inbounds(c *gin.Context) {
	items, err := a.inboundService.GetAllInbounds()
	if err != nil {
		jsonMsg(c, "get subscription inbounds", err)
		return
	}
	filtered := make([]interface{}, 0, len(items))
	for _, item := range items {
		if !service.IsSubscriptionProtocol(item.Protocol) {
			continue
		}
		filtered = append(filtered, item)
	}
	jsonObj(c, filtered, nil)
}

func (a *SubscriptionController) publicGet(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	format := strings.TrimSpace(strings.ToLower(c.Query("format")))
	if format == "" {
		format = "base64"
	}
	var (
		content string
		err     error
	)
	switch format {
	case "base64":
		content, err = a.subscriptionService.GenerateBase64(token, c.Request.Host)
	case "clash":
		content, err = a.subscriptionService.GenerateClash(token, c.Request.Host)
	case "mihomo":
		content, err = a.subscriptionService.GenerateMihomo(token, c.Request.Host)
	default:
		c.Status(http.StatusNotFound)
		return
	}
	if err != nil {
		if database.IsNotFound(err) || errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(strings.ToLower(err.Error()), "not found") {
			c.Status(http.StatusNotFound)
			return
		}
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, content)
}
