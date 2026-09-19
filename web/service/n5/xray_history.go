package n5

import (
	"x-ui/database"
	n5model "x-ui/database/model/n5"
)

type XrayHistoryService struct {
}

func (s *XrayHistoryService) List(limit int) ([]*n5model.XrayConfigHistory, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	records := make([]*n5model.XrayConfigHistory, 0)
	err := database.GetDB().
		Model(&n5model.XrayConfigHistory{}).
		Order("id desc").
		Limit(limit).
		Find(&records).Error
	return records, err
}
