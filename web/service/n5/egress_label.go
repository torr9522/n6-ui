package n5

import (
	"strings"
	"x-ui/database"
	n5model "x-ui/database/model/n5"
	"x-ui/util/common"

	"gorm.io/gorm"
)

type EgressLabelService struct {
}

func (s *EgressLabelService) Create(label *n5model.EgressLabel) (*n5model.EgressLabel, error) {
	if label == nil {
		return nil, common.NewError("egress label is nil")
	}

	record := &n5model.EgressLabel{
		Name: strings.TrimSpace(label.Name),
		Type: normalizeLabelType(label.Type),
	}
	if record.Name == "" {
		return nil, common.NewError("label name is required")
	}
	if err := validateLabelType(record.Type); err != nil {
		return nil, err
	}

	if err := database.GetDB().Create(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (s *EgressLabelService) Update(label *n5model.EgressLabel) (*n5model.EgressLabel, error) {
	if label == nil || label.Id <= 0 {
		return nil, common.NewError("invalid egress label")
	}

	record := &n5model.EgressLabel{}
	db := database.GetDB()
	if err := db.Model(&n5model.EgressLabel{}).Where("id = ?", label.Id).First(record).Error; err != nil {
		return nil, err
	}

	name := strings.TrimSpace(label.Name)
	if name == "" {
		return nil, common.NewError("label name is required")
	}
	labelType := normalizeLabelType(label.Type)
	if err := validateLabelType(labelType); err != nil {
		return nil, err
	}

	record.Name = name
	record.Type = labelType
	if err := db.Save(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (s *EgressLabelService) Delete(id int) error {
	if id <= 0 {
		return common.NewError("invalid label id")
	}

	return database.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("label_id = ?", id).Delete(&n5model.EgressLabelRelation{}).Error; err != nil {
			return err
		}
		return tx.Delete(&n5model.EgressLabel{}, id).Error
	})
}

func (s *EgressLabelService) List() ([]*n5model.EgressLabel, error) {
	records := make([]*n5model.EgressLabel, 0)
	err := database.GetDB().Model(&n5model.EgressLabel{}).Order("type asc, id asc").Find(&records).Error
	return records, err
}

func (s *EgressLabelService) Bind(egressId int, labelId int) (*n5model.EgressLabelRelation, error) {
	if egressId <= 0 || labelId <= 0 {
		return nil, common.NewError("invalid egress label relation")
	}

	db := database.GetDB()
	var count int64
	if err := db.Model(&n5model.Egress{}).Where("id = ?", egressId).Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, common.NewError("egress not found")
	}

	count = 0
	if err := db.Model(&n5model.EgressLabel{}).Where("id = ?", labelId).Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, common.NewError("egress label not found")
	}

	record := &n5model.EgressLabelRelation{
		EgressId: egressId,
		LabelId:  labelId,
	}
	if err := db.Where("egress_id = ? and label_id = ?", egressId, labelId).FirstOrCreate(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (s *EgressLabelService) Unbind(egressId int, labelId int) error {
	if egressId <= 0 || labelId <= 0 {
		return common.NewError("invalid egress label relation")
	}
	return database.GetDB().
		Where("egress_id = ? and label_id = ?", egressId, labelId).
		Delete(&n5model.EgressLabelRelation{}).Error
}

func (s *EgressLabelService) ListByEgress(egressId int) ([]*n5model.EgressLabel, error) {
	if egressId <= 0 {
		return nil, common.NewError("invalid egress id")
	}
	records := make([]*n5model.EgressLabel, 0)
	err := database.GetDB().
		Model(&n5model.EgressLabel{}).
		Joins("join n5_egress_label_relations on n5_egress_label_relations.label_id = n5_egress_labels.id").
		Where("n5_egress_label_relations.egress_id = ?", egressId).
		Order("n5_egress_labels.type asc, n5_egress_labels.id asc").
		Find(&records).Error
	return records, err
}
