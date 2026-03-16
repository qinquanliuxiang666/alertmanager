package model

import (
	"time"

	"gorm.io/gorm"
)

// AlertTemplate 告警模板表
type AlertTemplate struct {
	ID                  int            `gorm:"primaryKey"`
	CreatedAt           time.Time      `gorm:"column:created_at" json:"createdAt,omitempty"`
	UpdatedAt           time.Time      `gorm:"column:updated_at" json:"updatedAt,omitempty"`
	DeletedAt           gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
	Name                string         `gorm:"type:varchar(100);not null;uniqueIndex;comment:模板名称"`
	Description         string         `gorm:"type:varchar(255)"`
	Template            string         `gorm:"type:text;not null;comment:单个告警(Markdown/HTML)模板"`
	AggregationTemplate string         `gorm:"type:text;comment:聚合告警(Markdown/HTML)模板"`
}
