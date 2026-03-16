package model

import (
	"time"
)

const (
	AlertSendRecordStatusSuccess = "success"
	AlertSendRecordStatusFailed  = "failed"
)

// AlertSendRecord 告警发送明细/日志表
type AlertSendRecord struct {
	ID                int             `gorm:"primaryKey;autoIncrement"`
	CreatedAt         time.Time       `gorm:"column:created_at" json:"createdAt,omitempty"`
	UpdatedAt         time.Time       `gorm:"column:updated_at" json:"updatedAt,omitempty"`
	AlertHistory      []*AlertHistory `gorm:"foreignKey:ID" json:"alertHistory"`
	SendStatus        string          `gorm:"type:varchar(32);not null;index;comment:发送状态(success, failed)" json:"sendStatus"`
	ErrorMessage      string          `gorm:"type:text;comment:如果发送失败，记录失败的报错详情(供排查)"`
	ExternalMessageID string          `gorm:"type:varchar(255);index;comment:第三方平台返回的消息ID(如飞书的 message_id)" json:"externalMessageID"`
}

func (*AlertSendRecord) TableName() string {
	return "alert_send_records"
}
