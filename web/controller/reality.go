package controller

import (
	"github.com/gin-gonic/gin"
	"x-ui/web/service"
)

type RealityController struct {
	realityService service.RealityService
}

func NewRealityController(g *gin.RouterGroup) *RealityController {
	a := &RealityController{}
	a.initRouter(g)
	return a
}

func (a *RealityController) initRouter(g *gin.RouterGroup) {
	g = g.Group("/reality")

	g.POST("/x25519", a.generateX25519KeyPair)
	g.POST("/default", a.generateDefaultConfig)
}

func (a *RealityController) generateX25519KeyPair(c *gin.Context) {
	keyPair, err := a.realityService.GenerateX25519KeyPair()
	jsonObj(c, keyPair, err)
}

func (a *RealityController) generateDefaultConfig(c *gin.Context) {
	template := c.Query("template")
	if template == "" {
		template = c.PostForm("template")
	}
	config, err := a.realityService.GenerateDefaultConfig(template)
	jsonObj(c, config, err)
}
