package n5

import (
	"fmt"
	"strings"
	"time"
	"x-ui/database"
	n5model "x-ui/database/model/n5"
	"x-ui/util/common"

	"gorm.io/gorm"
)

type EgressPoolService struct {
}

func (s *EgressPoolService) GeneratePoolTag(id int) string {
	return formatN5Tag("n5-pool", id)
}

func (s *EgressPoolService) GenerateStableTag(id int) string {
	return s.GeneratePoolTag(id)
}

func (s *EgressPoolService) Create(pool *n5model.EgressPool) (*n5model.EgressPool, error) {
	if pool == nil {
		return nil, common.NewError("egress pool is nil")
	}

	strategy := normalizeProtocol(pool.Strategy)
	if strategy == "" {
		strategy = "random"
	}
	if !allowedPoolStrategy(strategy) {
		return nil, common.NewError("invalid pool strategy")
	}

	record := &n5model.EgressPool{
		Name:             normalizeName(pool.Name),
		Remark:           strings.TrimSpace(pool.Remark),
		Strategy:         strategy,
		FallbackType:     normalizeTargetType(pool.FallbackType),
		FallbackTargetId: pool.FallbackTargetId,
		Enabled:          pool.Enabled,
		Tag:              fmt.Sprintf("n5-pool-pending-%d", time.Now().UnixNano()),
	}
	if !pool.Enabled {
		record.Enabled = false
	} else {
		record.Enabled = true
	}
	if record.Name == "" {
		return nil, common.NewError("pool name is required")
	}
	if record.FallbackType != "" {
		if err := validateTarget(record.FallbackType, record.FallbackTargetId); err != nil {
			return nil, err
		}
	}

	db := database.GetDB()
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(record).Error; err != nil {
			return err
		}
		record.Tag = s.GeneratePoolTag(record.Id)
		return tx.Save(record).Error
	})
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (s *EgressPoolService) Get(id int) (*n5model.EgressPool, error) {
	if id <= 0 {
		return nil, common.NewError("invalid pool id")
	}
	record := &n5model.EgressPool{}
	if err := database.GetDB().Model(&n5model.EgressPool{}).Where("id = ?", id).First(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (s *EgressPoolService) List() ([]*n5model.EgressPool, error) {
	records := make([]*n5model.EgressPool, 0)
	err := database.GetDB().Model(&n5model.EgressPool{}).Order("id asc").Find(&records).Error
	return records, err
}

func (s *EgressPoolService) AddMember(poolId int, egressId int, weight int, sortOrder int) (*n5model.EgressPoolMember, error) {
	if poolId <= 0 || egressId <= 0 {
		return nil, common.NewError("invalid pool member")
	}
	if weight <= 0 {
		weight = 1
	}

	db := database.GetDB()
	var poolCount int64
	if err := db.Model(&n5model.EgressPool{}).Where("id = ?", poolId).Count(&poolCount).Error; err != nil {
		return nil, err
	}
	if poolCount == 0 {
		return nil, common.NewError("egress pool not found")
	}
	var egressCount int64
	if err := db.Model(&n5model.Egress{}).Where("id = ?", egressId).Count(&egressCount).Error; err != nil {
		return nil, err
	}
	if egressCount == 0 {
		return nil, common.NewError("egress not found")
	}

	member := &n5model.EgressPoolMember{}
	err := db.Model(&n5model.EgressPoolMember{}).
		Where("pool_id = ? and egress_id = ?", poolId, egressId).
		First(member).Error
	if database.IsNotFound(err) {
		member = &n5model.EgressPoolMember{
			PoolId:    poolId,
			EgressId:  egressId,
			Weight:    weight,
			SortOrder: sortOrder,
			Enabled:   true,
		}
		if sortOrder <= 0 {
			var maxSort int
			db.Model(&n5model.EgressPoolMember{}).Where("pool_id = ?", poolId).Select("coalesce(max(sort_order), 0)").Scan(&maxSort)
			member.SortOrder = maxSort + 1
		}
		if err := db.Create(member).Error; err != nil {
			return nil, err
		}
		return member, nil
	}
	if err != nil {
		return nil, err
	}

	member.Weight = weight
	if sortOrder > 0 {
		member.SortOrder = sortOrder
	}
	member.Enabled = true
	if err := db.Save(member).Error; err != nil {
		return nil, err
	}
	return member, nil
}

func (s *EgressPoolService) RemoveMember(poolId int, egressId int) error {
	if poolId <= 0 || egressId <= 0 {
		return common.NewError("invalid pool member")
	}
	return database.GetDB().Where("pool_id = ? and egress_id = ?", poolId, egressId).Delete(&n5model.EgressPoolMember{}).Error
}

func (s *EgressPoolService) ListMembers(poolId int) ([]*n5model.EgressPoolMember, error) {
	if poolId <= 0 {
		return nil, common.NewError("invalid pool id")
	}
	members := make([]*n5model.EgressPoolMember, 0)
	err := database.GetDB().Model(&n5model.EgressPoolMember{}).Where("pool_id = ?", poolId).Order("sort_order asc, id asc").Find(&members).Error
	return members, err
}
