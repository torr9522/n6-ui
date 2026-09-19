package n5

import (
	"x-ui/web/entity"
	coreservice "x-ui/web/service"

	"github.com/gin-gonic/gin"
)

type n5SettingAPI interface {
	GetAllSetting() (*entity.AllSetting, error)
	GetN5XrayExtensionEnable() (bool, error)
	UpdateAllSetting(allSetting *entity.AllSetting) error
}

type N5SettingsController struct {
	settingService n5SettingAPI
	xrayService    n5RestartTrigger
}

func NewSettingsController(g *gin.RouterGroup) *N5SettingsController {
	a := &N5SettingsController{
		settingService: &coreservice.SettingService{},
		xrayService:    &coreservice.XrayService{},
	}
	a.initRouter(g)
	return a
}

func (a *N5SettingsController) initRouter(g *gin.RouterGroup) {
	pageGroup := g.Group("/n5")
	pageGroup.Use(checkLogin)
	pageGroup.GET("/settings", a.page)

	apiGroup := g.Group("/n5/api/settings")
	apiGroup.Use(checkLogin)
	apiGroup.GET("", a.get)
	apiGroup.POST("", a.update)
}

func (a *N5SettingsController) page(c *gin.Context) {
	html(c, "settings.html", "N5设置", nil)
}

func (a *N5SettingsController) get(c *gin.Context) {
	enabled, err := a.getSettingService().GetN5XrayExtensionEnable()
	if err != nil {
		jsonMsg(c, "get n5 settings", err)
		return
	}
	jsonObj(c, gin.H{"enabled": enabled}, nil)
}

func (a *N5SettingsController) update(c *gin.Context) {
	payload := struct {
		Enabled bool `json:"enabled" form:"enabled"`
	}{}
	if err := c.ShouldBind(&payload); err != nil {
		jsonMsg(c, "update n5 settings", err)
		return
	}

	allSetting, err := a.getSettingService().GetAllSetting()
	if err != nil {
		jsonMsg(c, "update n5 settings", err)
		return
	}
	allSetting.N5XrayExtensionEnable = payload.Enabled
	if err := a.getSettingService().UpdateAllSetting(allSetting); err != nil {
		jsonMsg(c, "update n5 settings", err)
		return
	}
	a.getXrayService().SetToNeedRestart()
	jsonObj(c, gin.H{"enabled": payload.Enabled}, nil)
}

func (a *N5SettingsController) getSettingService() n5SettingAPI {
	if a.settingService != nil {
		return a.settingService
	}
	return &coreservice.SettingService{}
}

func (a *N5SettingsController) getXrayService() n5RestartTrigger {
	if a.xrayService != nil {
		return a.xrayService
	}
	return &coreservice.XrayService{}
}
