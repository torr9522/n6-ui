package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/torr9522/n6-ui/database"
	"github.com/torr9522/n6-ui/database/model"
	"github.com/torr9522/n6-ui/util/common"
)

type SubscriptionService struct {
	shareLinkService ShareLinkService
	settingService   SettingService
}

type SubscriptionForm struct {
	Remark     string `json:"remark" form:"remark"`
	Enable     bool   `json:"enable" form:"enable"`
	InboundIds []int  `json:"inboundIds" form:"inboundIds"`
}

type SubscriptionView struct {
	Id         int    `json:"id"`
	Token      string `json:"token"`
	Remark     string `json:"remark"`
	Enable     bool   `json:"enable"`
	InboundIds []int  `json:"inboundIds"`
	NodeCount  int    `json:"nodeCount"`
	CreatedAt  int64  `json:"createdAt"`
	UpdatedAt  int64  `json:"updatedAt"`
}

func (s *SubscriptionService) List() ([]*SubscriptionView, error) {
	db := database.GetDB()
	items := make([]*model.Subscription, 0)
	if err := db.Order("id asc").Find(&items).Error; err != nil {
		return nil, err
	}
	result := make([]*SubscriptionView, 0, len(items))
	for _, item := range items {
		view, err := s.toView(item)
		if err != nil {
			return nil, err
		}
		result = append(result, view)
	}
	return result, nil
}

func (s *SubscriptionService) Add(form *SubscriptionForm) (*SubscriptionView, error) {
	ids, err := s.normalizeInboundIDs(form.InboundIds)
	if err != nil {
		return nil, err
	}
	token, err := s.generateUniqueToken()
	if err != nil {
		return nil, err
	}
	rawIDs, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}
	item := &model.Subscription{
		Token:      token,
		Remark:     strings.TrimSpace(form.Remark),
		Enable:     form.Enable,
		InboundIds: string(rawIDs),
	}
	if err := database.GetDB().Create(item).Error; err != nil {
		return nil, err
	}
	return s.toView(item)
}

func (s *SubscriptionService) Update(id int, form *SubscriptionForm) (*SubscriptionView, error) {
	db := database.GetDB()
	item := &model.Subscription{}
	if err := db.Where("id = ?", id).First(item).Error; err != nil {
		return nil, err
	}
	ids, err := s.normalizeInboundIDs(form.InboundIds)
	if err != nil {
		return nil, err
	}
	rawIDs, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}
	item.Remark = strings.TrimSpace(form.Remark)
	item.Enable = form.Enable
	item.InboundIds = string(rawIDs)
	if err := db.Save(item).Error; err != nil {
		return nil, err
	}
	return s.toView(item)
}

func (s *SubscriptionService) Delete(id int) error {
	return database.GetDB().Delete(&model.Subscription{}, id).Error
}

func (s *SubscriptionService) RefreshToken(id int) (*SubscriptionView, error) {
	db := database.GetDB()
	item := &model.Subscription{}
	if err := db.Where("id = ?", id).First(item).Error; err != nil {
		return nil, err
	}
	token, err := s.generateUniqueToken()
	if err != nil {
		return nil, err
	}
	item.Token = token
	if err := db.Save(item).Error; err != nil {
		return nil, err
	}
	return s.toView(item)
}

func (s *SubscriptionService) GetByToken(token string) (*model.Subscription, error) {
	item := &model.Subscription{}
	if err := database.GetDB().Where("token = ?", strings.TrimSpace(token)).First(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func (s *SubscriptionService) GenerateBase64(token string, reqHost string) (string, error) {
	_, inbounds, ctx, err := s.loadEnabledSubscription(token, reqHost)
	if err != nil {
		return "", err
	}
	return s.shareLinkService.GenerateBase64(inbounds, ctx)
}

func (s *SubscriptionService) GenerateClash(token string, reqHost string) (string, error) {
	_, inbounds, ctx, err := s.loadEnabledSubscription(token, reqHost)
	if err != nil {
		return "", err
	}
	return s.shareLinkService.GenerateClash(inbounds, ctx)
}

func (s *SubscriptionService) GenerateMihomo(token string, reqHost string) (string, error) {
	_, inbounds, ctx, err := s.loadEnabledSubscription(token, reqHost)
	if err != nil {
		return "", err
	}
	return s.shareLinkService.GenerateMihomo(inbounds, ctx)
}

func (s *SubscriptionService) BuildPublicURL(token string, format string, reqHost string) (string, error) {
	basePath, err := s.settingService.GetBasePath()
	if err != nil {
		return "", err
	}
	host := extractHost(reqHost)
	if host == "" {
		return "", common.NewError("request host is empty")
	}
	base := fmt.Sprintf("https://%s%ssub/%s", reqHost, basePath, token)
	if format == "" || format == "base64" {
		return base, nil
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("format", format)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (s *SubscriptionService) loadSubscriptionInbounds(sub *model.Subscription) ([]*model.Inbound, error) {
	ids, err := s.decodeInboundIDs(sub.InboundIds)
	if err != nil {
		return nil, err
	}
	db := database.GetDB()
	all := make([]*model.Inbound, 0)
	if len(ids) == 0 {
		return all, nil
	}
	if err := db.Where("id in ?", ids).Find(&all).Error; err != nil {
		return nil, err
	}
	index := map[int]*model.Inbound{}
	for _, inbound := range all {
		index[inbound.Id] = inbound
	}
	result := make([]*model.Inbound, 0, len(ids))
	for _, id := range ids {
		inbound, ok := index[id]
		if !ok {
			continue
		}
		if !IsSubscriptionProtocol(inbound.Protocol) {
			continue
		}
		result = append(result, inbound)
	}
	return result, nil
}

func (s *SubscriptionService) loadEnabledSubscription(token string, reqHost string) (*model.Subscription, []*model.Inbound, shareContext, error) {
	sub, err := s.GetByToken(token)
	if err != nil {
		return nil, nil, shareContext{}, err
	}
	if !sub.Enable {
		return nil, nil, shareContext{}, common.NewError("subscription not found")
	}
	inbounds, err := s.loadSubscriptionInbounds(sub)
	if err != nil {
		return nil, nil, shareContext{}, err
	}
	ctx := shareContext{
		Scheme:      "https",
		RequestHost: reqHost,
	}
	return sub, inbounds, ctx, nil
}

func (s *SubscriptionService) toView(item *model.Subscription) (*SubscriptionView, error) {
	ids, err := s.decodeInboundIDs(item.InboundIds)
	if err != nil {
		return nil, err
	}
	return &SubscriptionView{
		Id:         item.Id,
		Token:      item.Token,
		Remark:     item.Remark,
		Enable:     item.Enable,
		InboundIds: ids,
		NodeCount:  len(ids),
		CreatedAt:  item.CreatedAt,
		UpdatedAt:  item.UpdatedAt,
	}, nil
}

func (s *SubscriptionService) decodeInboundIDs(raw string) ([]int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []int{}, nil
	}
	ids := make([]int, 0)
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil, common.NewError("invalid inbound ids")
	}
	return ids, nil
}

func (s *SubscriptionService) normalizeInboundIDs(ids []int) ([]int, error) {
	seen := map[int]bool{}
	normalized := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		normalized = append(normalized, id)
	}
	if len(normalized) == 0 {
		return nil, common.NewError("inboundIds is empty")
	}
	all := make([]*model.Inbound, 0)
	if err := database.GetDB().Where("id in ?", normalized).Find(&all).Error; err != nil {
		return nil, err
	}
	index := map[int]*model.Inbound{}
	for _, inbound := range all {
		index[inbound.Id] = inbound
	}
	for _, id := range normalized {
		inbound, ok := index[id]
		if !ok {
			return nil, common.NewError("inbound not found:", id)
		}
		if !IsSubscriptionProtocol(inbound.Protocol) {
			return nil, common.NewError("unsupported inbound protocol:", inbound.Protocol)
		}
	}
	return normalized, nil
}

func (s *SubscriptionService) generateUniqueToken() (string, error) {
	for i := 0; i < 5; i++ {
		buf := make([]byte, 24)
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		token := hex.EncodeToString(buf)
		var count int64
		if err := database.GetDB().Model(&model.Subscription{}).Where("token = ?", token).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return token, nil
		}
	}
	return "", common.NewError("generate unique token failed")
}

func IsSubscriptionProtocol(protocol model.Protocol) bool {
	switch protocol {
	case model.VMess, model.VLESS, model.Trojan, model.Shadowsocks:
		return true
	default:
		return false
	}
}
