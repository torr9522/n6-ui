package n5

import (
	"strconv"
	"x-ui/database"
	"x-ui/database/model"
	n5model "x-ui/database/model/n5"
)

type XrayStatusLastApply struct {
	Source              string `json:"source"`
	Status              string `json:"status"`
	Hash                string `json:"hash"`
	BaseConfigHash      string `json:"baseConfigHash"`
	ExtensionConfigHash string `json:"extensionConfigHash"`
	ApplyError          string `json:"applyError"`
	CreatedAt           int64  `json:"createdAt"`
}

type XrayStatus struct {
	Enabled       bool                 `json:"enabled"`
	LastApply     *XrayStatusLastApply `json:"lastApply"`
	OutboundCount int                  `json:"outboundCount"`
	RoutingCount  int                  `json:"routingCount"`
	Hash          string               `json:"hash"`
}

type XrayStatusService struct {
	xrayExt XrayExtService
}

func (s *XrayStatusService) GetStatus() (*XrayStatus, error) {
	enabled, err := getN5XrayExtensionEnable()
	if err != nil {
		return nil, err
	}

	outbounds, err := s.xrayExt.GenerateOutboundFragments()
	if err != nil {
		return nil, err
	}
	routing, err := s.xrayExt.GenerateRoutingFragments()
	if err != nil {
		return nil, err
	}
	routingCount, err := countRoutingRules(routing)
	if err != nil {
		return nil, err
	}

	status := &XrayStatus{
		Enabled:       enabled,
		OutboundCount: len(outbounds),
		RoutingCount:  routingCount,
	}

	history := &n5model.XrayConfigHistory{}
	err = database.GetDB().
		Model(&n5model.XrayConfigHistory{}).
		Order("id desc").
		First(history).Error
	if database.IsNotFound(err) {
		return status, nil
	}
	if err != nil {
		return nil, err
	}

	status.Hash = history.ConfigHash
	status.LastApply = &XrayStatusLastApply{
		Source:              history.Source,
		Status:              history.ApplyStatus,
		Hash:                history.ConfigHash,
		BaseConfigHash:      history.BaseConfigHash,
		ExtensionConfigHash: history.ExtensionConfigHash,
		ApplyError:          history.ApplyError,
		CreatedAt:           history.CreatedAt,
	}
	return status, nil
}

func getN5XrayExtensionEnable() (bool, error) {
	setting := &model.Setting{}
	err := database.GetDB().Model(&model.Setting{}).Where("key = ?", "n5XrayExtensionEnable").First(setting).Error
	if database.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(setting.Value)
}
