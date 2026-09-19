package database

import (
	"encoding/json"
	"fmt"
	"regexp"

	n5model "x-ui/database/model/n5"

	"gorm.io/gorm"
)

const n5TagIDWidth = 10

var (
	legacyN5EgressTagPattern = regexp.MustCompile(`^n5-egress-\d+$`)
	legacyN5PoolTagPattern   = regexp.MustCompile(`^n5-pool-\d+$`)
)

func initN5Phase2() error {
	if err := db.AutoMigrate(
		&n5model.Egress{},
		&n5model.EgressTest{},
		&n5model.EgressLabel{},
		&n5model.EgressLabelRelation{},
		&n5model.EgressPool{},
		&n5model.EgressPoolMember{},
		&n5model.TrafficPolicy{},
		&n5model.TrafficPolicyRule{},
		&n5model.TrafficPolicyBinding{},
		&n5model.XrayConfigHistory{},
	); err != nil {
		return err
	}
	if err := migrateN5StableTags(); err != nil {
		return err
	}
	return migrateN5TrafficPolicyBindingUniqueIndex()
}

func formatN5Tag(prefix string, id int) string {
	return fmt.Sprintf("%s-%0*d", prefix, n5TagIDWidth, id)
}

func migrateN5StableTags() error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := migrateN5EgressTags(tx); err != nil {
			return err
		}
		return migrateN5PoolTags(tx)
	})
}

func migrateN5EgressTags(tx *gorm.DB) error {
	records := make([]*n5model.Egress, 0)
	if err := tx.Model(&n5model.Egress{}).Order("id asc").Find(&records).Error; err != nil {
		return err
	}

	for _, record := range records {
		targetTag := formatN5Tag("n5-egress", record.Id)
		if record.Tag == targetTag || !legacyN5EgressTagPattern.MatchString(record.Tag) {
			continue
		}

		obj := make(map[string]interface{})
		if err := json.Unmarshal([]byte(record.OutboundJSON), &obj); err != nil {
			return fmt.Errorf("migrate n5 egress <%d> outbound json failed: %w", record.Id, err)
		}
		obj["tag"] = targetTag
		data, err := json.Marshal(obj)
		if err != nil {
			return fmt.Errorf("marshal migrated n5 egress <%d> outbound json failed: %w", record.Id, err)
		}

		record.Tag = targetTag
		record.OutboundJSON = string(data)
		if err := tx.Save(record).Error; err != nil {
			return err
		}
	}

	return nil
}

func migrateN5PoolTags(tx *gorm.DB) error {
	records := make([]*n5model.EgressPool, 0)
	if err := tx.Model(&n5model.EgressPool{}).Order("id asc").Find(&records).Error; err != nil {
		return err
	}

	for _, record := range records {
		targetTag := formatN5Tag("n5-pool", record.Id)
		if record.Tag == targetTag || !legacyN5PoolTagPattern.MatchString(record.Tag) {
			continue
		}
		record.Tag = targetTag
		if err := tx.Save(record).Error; err != nil {
			return err
		}
	}

	return nil
}

func migrateN5TrafficPolicyBindingUniqueIndex() error {
	if err := db.Exec(`
		DELETE FROM n5_traffic_policy_bindings
		WHERE id NOT IN (
			SELECT MAX(id)
			FROM n5_traffic_policy_bindings
			GROUP BY inbound_id
		)
	`).Error; err != nil {
		return err
	}

	if err := db.Exec("DROP INDEX IF EXISTS idx_n5_traffic_policy_bindings_inbound_id").Error; err != nil {
		return err
	}

	return db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_n5_policy_binding_inbound
		ON n5_traffic_policy_bindings(inbound_id)
	`).Error
}
